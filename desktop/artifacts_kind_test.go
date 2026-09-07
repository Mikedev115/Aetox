package main

// What the ผลงาน gallery shows, and what it refuses to show.
//
// Both halves are one complaint, made on 8 ก.ย. 2569 while v1.5.19 was being
// cut: *"มันไม่กรองอะไรเลย ไฟล์อันไหนที่ Aetox แก้แม่งไปรวมกันหมด"*. The page
// had two chips — files and video — and no opinion at all about what belongs
// on it, so one session that had copied a source tree into its output folder
// contributed 8,267 rows against 300 real artifacts from every other session
// put together.
//
// The tests are written from the disk that produced that complaint: a session
// holding two real deliverables beside a checkout, and one HTML file of each of
// the three kinds that share the extension.

import (
	"os"
	"path/filepath"
	"testing"
)

// writeInSession puts one file at a path under a session's output folder,
// creating whatever directories it needs. Content matters here in a way it does
// not in artifacts_test.go: three of these files are told apart by their bytes.
func writeInSession(t *testing.T, a *App, session, rel, body string) string {
	t.Helper()
	path := filepath.Join(a.cur().cfg.SandboxRoot, "output", session, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// kindsByName is the gallery's answer, keyed by file name, for asserting
// against without depending on the order the walk happened to produce.
func kindsByName(files []Artifact) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		out[f.Name] = f.Kind
	}
	return out
}

// The three that share an extension. A deck, a scene and a page are all .html
// and nothing in the name separates them — `index.html` is a plausible name for
// all three — which is the whole reason the kind is decided by opening the file
// rather than by a regex over the name in the window.
func TestTheThreeKindsOfHTMLAreToldApart(t *testing.T) {
	app := bootGalleryApp(t)

	writeInSession(t, app, "s1", "deck/index.html",
		"<!doctype html><html><body><section class=\"slide\">หนึ่ง</section>"+
			"<section class=\"slide\">สอง</section></body></html>")
	writeInSession(t, app, "s1", "clip/index.html",
		"<!doctype html><html><body data-composition-id=\"clip\" data-duration=\"6\">"+
			"<div class=\"title\">ชื่อเรื่อง</div></body></html>")
	writeInSession(t, app, "s1", "site/index.html",
		"<!doctype html><html><body><h1>หน้าแรก</h1><p>ข้อความ</p></body></html>")

	files := app.ListArtifacts()
	if len(files) != 3 {
		t.Fatalf("gallery returned %d files, want 3: %v", len(files), kindsByName(files))
	}
	// Keyed by folder rather than by name, because all three ARE index.html —
	// which is the thing being demonstrated, and would quietly collapse a
	// name-keyed map to one entry.
	byFolder := map[string]string{}
	for _, f := range files {
		byFolder[f.Folder] = f.Kind
	}
	for folder, want := range map[string]string{
		"deck": artifactKindSlides,
		"clip": artifactKindScene,
		"site": artifactKindPage,
	} {
		if byFolder[folder] != want {
			t.Errorf("%s/index.html is %q, want %q", folder, byFolder[folder], want)
		}
	}
}

// A fragment carrying slide markers is not a deck, and the slides room is the
// one that decides that (readDeckRow asks deck.Whole before deck.Count). Two
// rooms disagreeing would put a file on this page's สไลด์ shelf that the room
// itself refuses to open.
func TestAFragmentIsNotADeckHereEither(t *testing.T) {
	app := bootGalleryApp(t)
	writeInSession(t, app, "s1", "piece.html",
		"<section class=\"slide\">ชิ้นส่วน</section><section class=\"slide\">อีกชิ้น</section>")

	got := kindsByName(app.ListArtifacts())
	if got["piece.html"] != artifactKindPage {
		t.Errorf("a fragment reads as %q, want %q", got["piece.html"], artifactKindPage)
	}
}

// The rest of the shelves, which the extension answers on its own.
func TestTheOrdinaryKindsComeOffTheExtension(t *testing.T) {
	app := bootGalleryApp(t)
	for name, want := range map[string]string{
		"cover.png":   artifactKindImage,
		"promo.mp4":   artifactKindVideo,
		"voice.mp3":   artifactKindAudio,
		"budget.xlsx": artifactKindSheet,
		"สรุป.md":     artifactKindDoc,
		"convert.py":  artifactKindDoc,
		"weights.bin": artifactKindOther,
	} {
		writeInSession(t, app, "s1", name, "x")
		if got := kindsByName(app.ListArtifacts())[name]; got != want {
			t.Errorf("%s is %q, want %q", name, got, want)
		}
	}
}

// The flood, reproduced at the shape it arrived in: two real deliverables and a
// checkout dropped beside them.
//
// Named `Aetox` on the owner's disk, which is why the rule cannot be a list of
// directory names — no list would have had that one on it. What gives a
// checkout away is the `.git` inside it.
func TestACheckoutDroppedInASessionIsNotTheSessionsWork(t *testing.T) {
	app := bootGalleryApp(t)

	writeInSession(t, app, "s1", "โพสต์-1.md", "ข้อความ")
	writeInSession(t, app, "s1", "โพสต์-2.md", "ข้อความ")
	// The checkout. Its own name is ordinary, and it holds the sorts of files a
	// repository holds — including two that would have passed for deliverables.
	writeInSession(t, app, "s1", "Aetox/.git/config", "[core]")
	writeInSession(t, app, "s1", "Aetox/README.md", "# Aetox")
	writeInSession(t, app, "s1", "Aetox/docs/index.html", "<!doctype html><html><body>docs</body></html>")
	writeInSession(t, app, "s1", "Aetox/third_party/x/vendor.js", "x")

	got := kindsByName(app.ListArtifacts())
	if len(got) != 2 {
		t.Fatalf("gallery returned %d files, want the 2 the session actually produced: %v", len(got), got)
	}
	for _, name := range []string{"โพสต์-1.md", "โพสต์-2.md"} {
		if got[name] != artifactKindDoc {
			t.Errorf("%s is %q, want %q", name, got[name], artifactKindDoc)
		}
	}
}

// The machinery of other people's tools, in a folder that is otherwise a real
// deliverable. This is the second rule and it is not the same as the first: an
// exported site legitimately brings a `node_modules` and is still a site, so
// the folder is kept and the noise inside it is not.
func TestVendorFoldersAreNotDeliverables(t *testing.T) {
	app := bootGalleryApp(t)

	writeInSession(t, app, "s1", "site/index.html", "<!doctype html><html><body>หน้าแรก</body></html>")
	writeInSession(t, app, "s1", "site/about.html", "<!doctype html><html><body>เกี่ยวกับ</body></html>")
	writeInSession(t, app, "s1", "site/node_modules/left-pad/index.js", "x")
	writeInSession(t, app, "s1", "site/__pycache__/build.pyc", "x")

	got := kindsByName(app.ListArtifacts())
	if len(got) != 2 {
		t.Fatalf("gallery returned %d files, want the 2 pages: %v", len(got), got)
	}
	if got["index.html"] != artifactKindPage || got["about.html"] != artifactKindPage {
		t.Errorf("the site's own pages did not survive: %v", got)
	}
}

// A session whose whole contents are a checkout answers as an empty session
// rather than as an error or a hole in the sweep.
func TestASessionThatIsOnlyACheckoutContributesNothing(t *testing.T) {
	app := bootGalleryApp(t)
	writeInSession(t, app, "s1", "clone/.git/HEAD", "ref: refs/heads/main")
	writeInSession(t, app, "s1", "clone/main.go", "package main")
	writeInSession(t, app, "s2", "จริง.md", "ข้อความ")

	files := app.ListArtifacts()
	if len(files) != 1 || files[0].Name != "จริง.md" {
		t.Fatalf("gallery = %v, want only the one real file", kindsByName(files))
	}
}
