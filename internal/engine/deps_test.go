package engine

// The line this package holds (§248 B1, design doc §10).
//
// Two rules, both about what the engine must never be able to do, so that
// moving it to another machine changes nothing:
//
//  1. No file here imports the window (wailsapp) or the provider key store
//     (internal/credentials). The screen is reached through Screen and
//     nothing else; a key is something the screen signs with, never
//     something the engine holds.
//  2. No file here reads a credential out of the sign-in store: the four
//     oauth doors that hand back a token, an endpoint, or signed headers are
//     the screen's (desktop/provider_forward.go). The engine may still start
//     and finish a sign-in, list methods, and show status — those write the
//     store, or read what is not a secret.
//
// Narrower than `go list -deps` on purpose: imagegen, stt, tts and the
// automation clients read their own per-host keys (rule 5 of §248), and
// bootstrap reads `${connect:}` for MCP headers, so the package's transitive
// closure legitimately links both stores. What the rule can hold, and what
// it holds, is that the engine's own code never does.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

var bannedImports = []string{
	"github.com/wailsapp/",
	"github.com/Mikedev115/Aetox/internal/credentials",
}

// bannedOAuthCalls are the selectors on the oauth package that answer with a
// secret, or with something only the holder of one should be trusted for.
var bannedOAuthCalls = map[string]bool{
	"TokenSource": true,
	"Endpoint":    true,
	"Headers":     true,
	"Token":       true,
}

func engineFiles(t *testing.T) map[string]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*ast.File{}
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly|parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		files[p] = f
	}
	return files
}

func TestEngineFilesNeverImportTheWindowOrTheKeyStore(t *testing.T) {
	for name, f := range engineFiles(t) {
		if strings.HasSuffix(name, "_test.go") {
			// A test may name the key store's path to check a document; the
			// shipped engine is what the rule is about.
			continue
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, banned := range bannedImports {
				if strings.HasPrefix(path, banned) {
					t.Errorf("%s imports %s — the engine reaches the window through Screen and holds no key (§248)", name, path)
				}
			}
		}
	}
}

func TestEngineFilesNeverReadACredential(t *testing.T) {
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "oauth" || !bannedOAuthCalls[sel.Sel.Name] {
				return true
			}
			t.Errorf("%s: oauth.%s at %s — a credential is read on the screen, never in the engine (§248)", p, sel.Sel.Name, fset.Position(sel.Pos()))
			return true
		})
	}
}
