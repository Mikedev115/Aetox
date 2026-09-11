package main

// The attach pickers and the drawing's save (§248 B1): dialogs on this
// machine, so the screen's. What comes back is a path on this machine — the
// engine copies it into the project (SaveChatImage, SaveChatFile) — or, for
// the drawing, bytes a canvas in this window rendered a moment ago.

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// PickAttachmentImage prompts the user to pick an image file (native dialog)
// for chat attachment, returning its absolute OS path, or "" if cancelled.
func (a *App) PickAttachmentImage() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "แนบรูปภาพ",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Images (*.png, *.jpg, *.jpeg, *.gif, *.webp, *.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp"},
		},
	})
}

// What the attach menu offers, one list per row.
//
// Lists rather than ready-made patterns because the rows and the
// "everything" line have to agree: the bug that started this was a type
// (.docx) present everywhere in the app except in one hand-written pattern
// string, which made a file the app could read invisible in the dialog and
// gave nobody a way to find out why.
//
// The legacy Office trio (.doc/.xls/.ppt) is deliberately absent: `read`
// opens the OOXML three (skill.officeExt) and refuses those, so listing them
// here would be a promise the reader cannot keep.
var (
	imageAttachExt = []string{"png", "jpg", "jpeg", "gif", "webp", "bmp"}
	mediaAttachExt = []string{"mp4", "mov", "mkv", "webm", "avi", "mp3", "wav", "m4a", "flac", "ogg"}
	docAttachExt   = []string{"pdf", "docx", "pptx", "xlsx", "txt", "md", "csv", "json"}
)

// The rows the composer's attach menu can ask for. Anything else, the empty
// string included, is the "ไฟล์อื่น" row and filters nothing away.
const (
	attachGroupImage    = "image"
	attachGroupMedia    = "media"
	attachGroupDocument = "document"
)

func attachPattern(exts ...[]string) string {
	var parts []string
	for _, list := range exts {
		for _, ext := range list {
			parts = append(parts, "*."+ext)
		}
	}
	return strings.Join(parts, ";")
}

// attachFilters is the one list the attach dialog offers, narrowed to the row
// the user picked in the menu. It sits apart from the picker so the
// multi-select dialog and any future single-file caller cannot drift into
// offering different file types.
//
// Every group still carries the wider two filters under its own: the menu
// chooses what the dialog opens on, and never what the person is allowed to
// come back with.
func attachFilters(group string) []wailsruntime.FileFilter {
	var (
		image      = wailsruntime.FileFilter{DisplayName: "รูปภาพ", Pattern: attachPattern(imageAttachExt)}
		media      = wailsruntime.FileFilter{DisplayName: "วิดีโอ และเสียง", Pattern: attachPattern(mediaAttachExt)}
		document   = wailsruntime.FileFilter{DisplayName: "เอกสาร", Pattern: attachPattern(docAttachExt)}
		everything = wailsruntime.FileFilter{DisplayName: "ไฟล์ที่แนบได้ทั้งหมด", Pattern: attachPattern(imageAttachExt, mediaAttachExt, docAttachExt)}
		any        = wailsruntime.FileFilter{DisplayName: "ทุกไฟล์", Pattern: "*.*"}
	)
	switch group {
	case attachGroupImage:
		return []wailsruntime.FileFilter{image, everything, any}
	case attachGroupMedia:
		return []wailsruntime.FileFilter{media, everything, any}
	case attachGroupDocument:
		return []wailsruntime.FileFilter{document, everything, any}
	}
	return []wailsruntime.FileFilter{everything, image, media, document, any}
}

// PickAttachments prompts for files to attach — images, clips, documents —
// and allows picking several at once. The composer stages a list, so a
// single-file dialog was the only reason one question could carry one file.
// The image-only picker stays for the paths that specifically want one.
//
// `group` is the menu row that was pressed, and it only decides which filter
// the dialog opens on.
func (a *App) PickAttachments(group string) ([]string, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "แนบไฟล์",
		Filters: attachFilters(group),
	})
	if err != nil {
		return []string{}, err
	}
	// Cancelling gives nil, which marshals to null and is what the frontend
	// would then call .length on (ARCHITECTURE.md §34).
	if paths == nil {
		return []string{}, nil
	}
	return paths, nil
}

// SaveDrawing is the screen's alone: the bytes come from a canvas this window
// rendered a moment ago, and they go to a file on this machine.
func (a *App) SaveDrawing(dataURL string) (string, error) {
	data, err := decodePNGDataURL(dataURL)
	if err != nil {
		return "", err
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "บันทึกภาพ",
		DefaultFilename: "aetox-drawing.png",
		Filters:         []wailsruntime.FileFilter{{DisplayName: "PNG (*.png)", Pattern: "*.png"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// decodePNGDataURL is the checked half of SaveDrawing: only a PNG data URL,
// because the bytes come from a canvas this app rendered a moment ago — any
// other shape means the caller is not the drawing button.
func decodePNGDataURL(dataURL string) ([]byte, error) {
	raw, ok := strings.CutPrefix(strings.TrimSpace(dataURL), "data:image/png;base64,")
	if !ok {
		return nil, fmt.Errorf("not a PNG data URL")
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("the image data does not decode")
	}
	return data, nil
}
