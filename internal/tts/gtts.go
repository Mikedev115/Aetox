package tts

// Google Translate's voice, through the gtts-cli CLI (pip install gTTS) — the
// second free cloud vendor. It has no named voices, only languages, so its
// "voices" are language rows ("th: Thai") and the picked voice ID is a
// language code. Same honesty as edge: the text goes to Google.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mikedev115/Aetox/internal/proc"
)

// swapped in tests.
var runGTTS = execCLI

type gttsVoice struct {
	binPath string
	lang    string // "th", "en", ... ; "" = the CLI's default (en)
}

func newGTTS(desc Descriptor, opts Options) (Engine, error) {
	binPath, err := lookBinary(desc)
	if err != nil {
		return nil, err
	}
	return &gttsVoice{binPath: binPath, lang: strings.TrimSpace(opts.Voice)}, nil
}

func (*gttsVoice) ID() string { return "gtts" }

func (*gttsVoice) Mime() string { return "audio/mpeg" }

// Voices runs `gtts-cli --all` and reads its "  th: Thai" lines. One voice per
// language is the vendor's own shape, not a simplification.
func (g *gttsVoice) Voices(ctx context.Context) ([]Voice, error) {
	out, err := runGTTS(ctx, g.binPath, "--all")
	if err != nil {
		return nil, err
	}
	return parseGTTSLanguages(out), nil
}

func (g *gttsVoice) Synthesize(ctx context.Context, text, outPath string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("ไม่มีข้อความให้อ่าน")
	}
	args := []string{text, "--output", outPath}
	if g.lang != "" {
		args = append(args, "--lang", g.lang)
	}
	if _, err := runGTTS(ctx, g.binPath, args...); err != nil {
		return fmt.Errorf("gTTS ไม่สำเร็จ — ตัวนี้ต้องต่อเน็ต (%s)", firstLine(err.Error(), err.Error()))
	}
	return nil
}

func parseGTTSLanguages(raw string) []Voice {
	var voices []Voice
	for _, line := range strings.Split(raw, "\n") {
		code, name, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		code = strings.TrimSpace(code)
		name = strings.TrimSpace(name)
		if code == "" || name == "" || strings.ContainsAny(code, " \t") {
			continue
		}
		voices = append(voices, Voice{ID: code, Name: name + " (" + code + ")", Lang: code})
	}
	// Alphabetical by code so th sits where a Thai eye scans for it.
	sort.Slice(voices, func(i, j int) bool { return voices[i].ID < voices[j].ID })
	return voices
}

// execCLI runs a pip-installed vendor's program and returns its stdout. Its
// stderr is the message when it fails; a program that is not there names the
// install.
func execCLI(ctx context.Context, binPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binPath, args...)
	proc.HideConsole(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("ไม่พบโปรแกรม %s — ติดตั้งด้วย: pip install gTTS", filepath.Base(binPath))
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s ไม่สำเร็จ (%s)", filepath.Base(binPath), firstLine(msg, msg))
	}
	return stdout.String(), nil
}

// lookBinary resolves a CLI vendor's program: PATH first, then the pip --user
// Scripts folders, which pip fills WITHOUT putting them on PATH — measured on
// the owner's machine 2026-09-01, where `pip install edge-tts` landed in
// %APPDATA%\Python\Python313\Scripts and LookPath alone called it missing.
// Written for the edge and gtts engines; edge has since stopped needing a
// program at all (edge.go), so this is gtts's now.
func lookBinary(desc Descriptor) (string, error) {
	for _, name := range desc.Binaries {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		for _, name := range desc.Binaries {
			matches, _ := filepath.Glob(filepath.Join(appData, "Python", "*", "Scripts", name+".exe"))
			sort.Sort(sort.Reverse(sort.StringSlice(matches))) // newest Python first
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && !info.IsDir() {
					return match, nil
				}
			}
		}
	}
	return "", fmt.Errorf("ไม่พบโปรแกรม %s ในเครื่อง — %s", strings.Join(desc.Binaries, "/"), desc.Install)
}
