package main

import (
	"strings"
	"testing"
)

// The attach menu's rows and the dialog's filters have to describe the same
// app. The bug that produced the menu was .docx living in every part of the
// attachment path except one hand-written pattern string, where nobody could
// see it was missing.
func TestAttachFiltersCoverEveryGroup(t *testing.T) {
	everything := attachFilters("")[0].Pattern
	for _, tc := range []struct {
		group string
		want  string
	}{
		{attachGroupImage, "*.png"},
		{attachGroupMedia, "*.mp4"},
		{attachGroupDocument, "*.docx"},
	} {
		filters := attachFilters(tc.group)
		if len(filters) == 0 {
			t.Fatalf("group %q offers no filters", tc.group)
		}
		if !strings.Contains(filters[0].Pattern, tc.want) {
			t.Errorf("group %q opens on %q, which does not offer %s", tc.group, filters[0].Pattern, tc.want)
		}
		// A row that narrows past what the widest filter admits would hide a
		// file the app can actually read, which is the original bug wearing a
		// different hat.
		for _, ext := range strings.Split(filters[0].Pattern, ";") {
			if !strings.Contains(everything, ext) {
				t.Errorf("group %q offers %s, missing from the everything filter", tc.group, ext)
			}
		}
		// Nobody is trapped in the row they pressed.
		last := filters[len(filters)-1]
		if last.Pattern != "*.*" {
			t.Errorf("group %q ends on %q, not the every-file filter", tc.group, last.Pattern)
		}
	}
}

// An unknown row asks for nothing, rather than for nothing at all: a menu that
// grows a row before the Go side knows its name must still open a dialog.
func TestAttachFiltersUnknownGroupFiltersNothingAway(t *testing.T) {
	for _, group := range []string{"", "tab-context-someday"} {
		filters := attachFilters(group)
		if len(filters) < 2 || !strings.Contains(filters[0].Pattern, "*.docx") {
			t.Errorf("group %q does not open on the everything filter: %+v", group, filters)
		}
	}
}

// SaveDrawing accepts exactly one shape: the PNG data URL the drawing button
// rendered a moment ago. Anything else is not that button.
func TestDecodePNGDataURLIsStrict(t *testing.T) {
	if _, err := decodePNGDataURL("data:image/png;base64,aGVsbG8="); err != nil {
		t.Fatalf("a well-formed PNG data URL was refused: %v", err)
	}
	for _, bad := range []string{
		"", "hello", "data:image/svg+xml;base64,aGVsbG8=",
		"data:image/png;base64,!!!not-base64!!!", "data:image/png;base64,",
	} {
		if _, err := decodePNGDataURL(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
