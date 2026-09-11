package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mikedev115/Aetox/internal/agentpkg"
	"github.com/Mikedev115/Aetox/internal/subagent"
)

// Sending a worker somewhere else.
//
// The export button exists before the install button on purpose: it is the test
// of the package standard. Anything that fails to travel is something still
// coupled to this app rather than to the worker, and the only way to find out
// which is which is to pack one up and look
// (docs/architecture/agent-package-standard-2026-08-08.md, v2).
//
// Thin, like every binding on this page: what a package is lives in
// internal/agentpkg, and the rule that a worker is an overlay of the user's
// folder over the shipped one lives in internal/subagent. This assembles the
// two and names the file.

// AgentPackageBytes packs one worker into a .zip and hands the bytes back,
// with the summary the export dialog shows. The engine's half of
// ExportAgentPackage (screen_doors.go): the package is built from folders on
// the engine's host, and where the user saves it is the screen's business.
func (a *App) AgentPackageBytes(name string) (ExportFile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ExportFile{}, fmt.Errorf("ไม่รู้ว่าจะส่งออกเอเจนตัวไหน")
	}
	// Only agents. A helper is the assistant's own hands in a second context and
	// has no folder to pack, so the honest answer is the reason rather than an
	// empty archive (COMPANY.md §4).
	if kind := subagent.KindOf(name); kind != subagent.KindAgent {
		return ExportFile{}, fmt.Errorf("%s ไม่ใช่เอเจน จึงไม่มีโฟลเดอร์ให้ส่งออก", name)
	}
	sources := subagent.PackageSources(name)
	if len(sources) == 0 {
		return ExportFile{}, fmt.Errorf("ไม่พบโฟลเดอร์ของ %s", name)
	}
	// A broken mcp-servers.json should say so before anything is packed.
	servers, err := agentpkg.PlacedServers(name)
	if err != nil {
		return ExportFile{}, err
	}
	// agentpkg writes a file; the bytes are what travel. A temp file on this
	// host, read back and removed, keeps the packer's one contract intact.
	tmp, err := os.CreateTemp("", "aetox-agent-*.zip")
	if err != nil {
		return ExportFile{}, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)
	res, err := agentpkg.Export(tmpPath, agentpkg.Options{Name: name, Sources: sources, Servers: servers})
	if err != nil {
		return ExportFile{}, err
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return ExportFile{}, err
	}
	return ExportFile{Name: name + "-agent.zip", Data: data, Note: exportSummary(res)}, nil
}

// exportSummary says what is in the file and, just as importantly, what is not.
//
// The three lines after the first are the whole reason this returns prose
// rather than a path: a seller has to be able to check that their own memory
// stayed behind and their own token did not travel, and the moment to show them
// that is the moment it happened.
func exportSummary(res agentpkg.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ส่งออก %s แล้ว %d ไฟล์\nไฟล์: %s", res.Name, res.Files, res.Path)
	if len(res.Servers) > 0 {
		fmt.Fprintf(&b, "\nเซิร์ฟเวอร์ที่ติดไปด้วย: %s", strings.Join(res.Servers, ", "))
	}
	if len(res.Asked) > 0 {
		keys := make([]string, 0, len(res.Asked))
		for _, f := range res.Asked {
			keys = append(keys, f.Key)
		}
		fmt.Fprintf(&b, "\nค่าที่ถอดออกและจะถามคนติดตั้งแทน: %s", strings.Join(keys, ", "))
	}
	if len(res.Left) > 0 {
		fmt.Fprintf(&b, "\nไม่ได้ติดไปด้วย: %s (ความจำเป็นของเครื่องนี้ ไม่ใช่ของที่ขาย)", strings.Join(res.Left, ", "))
	}
	return b.String()
}
