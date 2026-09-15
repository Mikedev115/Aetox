// Command enginewinres writes the Windows resource object that makes
// aetox-engine.exe a named binary: version block, application manifest, the
// app's icon. aetox.exe has had all three from the day it existed — wails
// build reads desktop/wails.json and desktop/build/windows/ and links them
// in. The engine is a plain `go build`, so until this ran it shipped with
// none: no CompanyName, no ProductVersion, no manifest, no signature. An
// anonymous stripped executable that spawns powershell.exe and listens on a
// socket is the shape Defender's cloud classifier was trained to dislike,
// and on 2026-09-15 it did — Trojan:Script/Wacatac.C!ml on the v1.7.1
// engine, quarantined mid-session on a user's machine (docs/DECISIONS.md
// §294). The resource block does not make the file trusted; it takes away
// the loudest of the signals that made it look untrusted, the same way
// release.yml already keeps -EncodedCommand off the update script.
//
//	go run ./cmd/enginewinres
//
// from the module root writes cmd/aetox-engine/rsrc_windows_amd64.syso,
// which the Go linker picks up on its own for windows/amd64 and ignores for
// every other target — the Linux engines are untouched. The file is
// generated, not committed (.gitignore): release.yml runs this one step
// before `go build ./cmd/aetox-engine`, and a local build without it is
// merely unnamed, as every build before it was.
//
// Every value is read from where it already lives — internal/version.Current
// and the info block of desktop/wails.json — so the engine can never carry a
// different name or version from the app beside it.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	aetoxversion "github.com/Mikedev115/Aetox/internal/version"
)

// Paths are relative to the module root, which is where go run is invoked
// from; the go.mod check below is what says so out loud when it is not.
const (
	wailsJSON = "desktop/wails.json"
	iconPath  = "desktop/build/windows/icon.ico"
	outPath   = "cmd/aetox-engine/rsrc_windows_amd64.syso"
)

// wailsInfo is the info block of desktop/wails.json — the same fields
// wails build pours into aetox.exe's version resource.
type wailsInfo struct {
	CompanyName string `json:"companyName"`
	ProductName string `json:"productName"`
	Copyright   string `json:"copyright"`
	Comments    string `json:"comments"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "enginewinres:", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run from the module root (no go.mod here)")
	}
	info, err := readInfo(wailsJSON)
	if err != nil {
		return err
	}
	ico, err := os.ReadFile(iconPath)
	if err != nil {
		return err
	}
	rs, err := build(info, aetoxversion.Current, ico)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := rs.WriteObject(&buf, winres.ArchAMD64); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %s %s v%s, %d bytes\n", filepath.ToSlash(outPath), info.CompanyName, info.ProductName, aetoxversion.Current, buf.Len())
	return nil
}

func readInfo(path string) (wailsInfo, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return wailsInfo{}, err
	}
	var cfg struct {
		Info wailsInfo `json:"info"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return wailsInfo{}, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Info.CompanyName == "" || cfg.Info.ProductName == "" {
		return wailsInfo{}, fmt.Errorf("%s: info.companyName and info.productName must both be set", path)
	}
	return cfg.Info, nil
}

// build is the resource set itself, apart from the files, so a test can
// hold it up against what aetox.exe carries.
func build(info wailsInfo, ver string, ico []byte) (*winres.ResourceSet, error) {
	fixed, err := fourPart(ver)
	if err != nil {
		return nil, err
	}
	rs := &winres.ResourceSet{}

	icon, err := winres.LoadICO(bytes.NewReader(ico))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", iconPath, err)
	}
	// Resource name 1 is what Explorer draws for a file; the same slot wails
	// uses, so the two exe icons in the install folder are the one icon.
	if err := rs.SetIcon(winres.ID(1), icon); err != nil {
		return nil, err
	}

	// asInvoker, said explicitly: a manifest that declares its execution
	// level is the difference between an application and a loose binary to
	// the loader (and to the classifier). No UAC, ever — the engine runs
	// as the person who opened the app. Long paths on, because a project
	// root the person picks can be deep and the engine reads it.
	rs.SetManifest(winres.AppManifest{
		Identity:       winres.AssemblyIdentity{Name: "Aetox.Engine", Version: fixed},
		Description:    info.ProductName + " engine",
		ExecutionLevel: winres.AsInvoker,
		LongPathAware:  true,
	})

	vi := version.Info{}
	vi.SetFileVersion(ver)
	vi.SetProductVersion(ver)
	fields := map[string]string{
		version.CompanyName:      info.CompanyName,
		version.ProductName:      info.ProductName,
		version.FileDescription:  info.ProductName + " engine",
		version.InternalName:     "aetox-engine",
		version.OriginalFilename: "aetox-engine.exe",
		version.LegalCopyright:   info.Copyright,
		version.Comments:         info.Comments,
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := vi.Set(version.LangDefault, k, v); err != nil {
			return nil, fmt.Errorf("version %s: %w", k, err)
		}
	}
	rs.SetVersionInfo(vi)
	return rs, nil
}

// fourPart turns "1.7.1" into the [4]uint16 a manifest identity wants
// ("1.7.1.0"); the version resource has its own parser for the string.
func fourPart(ver string) ([4]uint16, error) {
	var out [4]uint16
	parts := strings.Split(ver, ".")
	if len(parts) < 3 || len(parts) > 4 {
		return out, fmt.Errorf("version %q is not major.minor.patch", ver)
	}
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return out, fmt.Errorf("version %q: %w", ver, err)
		}
		out[i] = uint16(n)
	}
	return out, nil
}
