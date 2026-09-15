package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// The resource set the engine ships with: an icon, a manifest and a version
// block whose name and version are the app's. This is the whole reason the
// package exists, so the test asks for each one by resource type — a set
// that lost one would still write a valid .syso and a nameless engine.
func TestBuildCarriesIconManifestAndVersion(t *testing.T) {
	ico, err := os.ReadFile("../../" + iconPath)
	if err != nil {
		t.Fatal(err)
	}
	info := wailsInfo{CompanyName: "Aetox", ProductName: "Aetox", Copyright: "© 2026", Comments: "c"}
	rs, err := build(info, "1.7.2", ico)
	if err != nil {
		t.Fatal(err)
	}

	seen := map[winres.Identifier]bool{}
	rs.Walk(func(typeID, _ winres.Identifier, _ uint16, _ []byte) bool {
		seen[typeID] = true
		return true
	})
	for _, want := range []winres.Identifier{winres.RT_ICON, winres.RT_GROUP_ICON, winres.RT_MANIFEST, winres.RT_VERSION} {
		if !seen[want] {
			t.Errorf("resource type %v missing from the set", want)
		}
	}

	raw := rs.Get(winres.RT_VERSION, winres.ID(1), 0)
	if raw == nil {
		t.Fatal("no version resource at the standard slot (RT_VERSION, 1, neutral)")
	}
	vi, err := version.FromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := vi.ProductVersion; got != [4]uint16{1, 7, 2, 0} {
		t.Errorf("ProductVersion = %v, want 1.7.2.0", got)
	}
	// One translation only (LangDefault, split per resource language on
	// the way out), so the single table is the table.
	var table version.StringTable
	for _, tb := range vi.Table() {
		table = *tb
	}
	for key, want := range map[string]string{
		version.CompanyName:      "Aetox",
		version.ProductName:      "Aetox",
		version.OriginalFilename: "aetox-engine.exe",
	} {
		if table[key] != want {
			t.Errorf("%s = %q, want %q", key, table[key], want)
		}
	}

	// And it links: WriteObject is what the go toolchain will consume.
	var buf bytes.Buffer
	if err := rs.WriteObject(&buf, winres.ArchAMD64); err != nil {
		t.Fatal(err)
	}
	if buf.Len() < len(ico) {
		t.Errorf("object is %d bytes, smaller than the icon it embeds (%d)", buf.Len(), len(ico))
	}
}

func TestFourPart(t *testing.T) {
	got, err := fourPart("1.7.1")
	if err != nil || got != [4]uint16{1, 7, 1, 0} {
		t.Errorf("fourPart(1.7.1) = %v, %v", got, err)
	}
	for _, bad := range []string{"1.7", "1.7.1.0.0", "1.x.1", "70000.0.0"} {
		if _, err := fourPart(bad); err == nil {
			t.Errorf("fourPart(%q) accepted", bad)
		}
	}
}
