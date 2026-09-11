package main

import "testing"

// WHAT A CHAT PUTS IN FRONT OF THE USER IS HANDED OVER ONLY WHEN THE CHAT MADE
// IT. The artifact flag is drawn under the answer and listed under ชิ้นงาน, and
// both are about what this chat produced — so a deck or a page in the chat's
// own output folder is flagged, and a project file opened for a look is not.
// The browser's version of the same question starts from the file:/// address
// the open resolved to.
func TestOnlyAFileOfThisChatsOutputFolderIsHandedOver(t *testing.T) {
	cases := []struct {
		subdir, rel string
		want        bool
	}{
		{"output/s1", "output/s1/deck.html", true},
		{"output/s1", "output/s1/work/page-1.png", true},
		{"output/s1", "output/s2/deck.html", false},
		{"output/s1", "README.md", false},
		{"", "output/s1/deck.html", false}, // a focused project has no output folder
	}
	for _, c := range cases {
		if got := handedOver(c.subdir, c.rel); got != c.want {
			t.Errorf("handedOver(%q, %q) = %v, want %v", c.subdir, c.rel, got, c.want)
		}
	}
}

func TestABrowserOpenOfAnOutputPageNamesThePathWriteReported(t *testing.T) {
	root := `C:\Users\me\aetox`
	cases := []struct {
		url  string
		want string
	}{
		{"file:///C:/Users/me/aetox/output/s1/dashboard.html", "output/s1/dashboard.html"},
		{"file:///C:/Users/me/aetox/README.md", ""},
		{"file:///C:/Users/other/page.html", ""},
		{"https://example.com/output/s1/dashboard.html", ""},
	}
	for _, c := range cases {
		if got := handedOverFile(root, "output/s1", c.url); got != c.want {
			t.Errorf("handedOverFile(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}
