package main

import (
	"strings"
	"testing"
)

// A source file is a download, not a page. WebView2 aborts the navigation and
// the tab reports "not found, or unreachable" — which sends the model looking
// for a path bug when the file was right there.
func TestUnrenderableFile(t *testing.T) {
	refused := []string{
		"file:///C:/proj/test-hello.ts",
		"file:///C:/proj/main.go",
		"file:///C:/proj/style.scss",
	}
	for _, url := range refused {
		why := unrenderableFile(url)
		if why == "" {
			t.Errorf("unrenderableFile(%q) = \"\", want a reason the browser cannot show it", url)
			continue
		}
		if !strings.Contains(why, "read") {
			t.Errorf("the refusal must name what to use instead, got %q", why)
		}
	}

	allowed := []string{
		"file:///C:/proj/index.html",
		"file:///C:/proj/diagram.svg",
		"file:///C:/proj/report.pdf",
		"file:///C:/proj/notes.txt",
		"file:///C:/proj/README", // no extension: let the browser decide
		"https://example.com/app.ts",
		"about:blank",
	}
	for _, url := range allowed {
		if why := unrenderableFile(url); why != "" {
			t.Errorf("unrenderableFile(%q) = %q, want it opened", url, why)
		}
	}
}
