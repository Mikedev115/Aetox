package main

import (
	"runtime"
	"strings"
	"testing"
)

// The sentence for an engine that is not beside the app. Packaged, it names
// the likely cause and what to press (§294: Defender quarantined the v1.7.1
// engine and the person read a bare "ไม่พบ" with nothing to do about it);
// in a development tree it stays bare, because nothing was quarantined.
func TestMissingEngineNamesQuarantineOnlyWhenPackaged(t *testing.T) {
	packaged := missingEngine("aetox-engine.exe", `C:\Program Files\Aetox\Aetox`, true).Error()
	wants := []string{"aetox-engine.exe", `C:\Program Files\Aetox\Aetox`, "กัก", "Restore", "Allow on device", "ติดตั้ง Aetox ซ้ำ"}
	if runtime.GOOS == "windows" {
		wants = append(wants, "Windows Security › Protection history")
	}
	for _, want := range wants {
		if !strings.Contains(packaged, want) {
			t.Errorf("packaged sentence lacks %q:\n%s", want, packaged)
		}
	}
	dev := missingEngine("aetox-engine.exe", `D:\Aetox\Aetox\desktop\build\bin`, false).Error()
	if strings.Contains(dev, "กัก") || strings.Contains(dev, "Protection history") {
		t.Errorf("development sentence blames an antivirus:\n%s", dev)
	}
	if !strings.HasPrefix(dev, "ไม่พบ aetox-engine.exe ข้างโปรแกรม") {
		t.Errorf("development sentence changed shape:\n%s", dev)
	}
}
