// Command gen writes the two files that keep the screen and the engine in
// step (§248 B1): the engine's API interface — every exported method of
// *Engine — and the screen's forwarders, one binding per API method that the
// screen does not implement itself.
//
// Source of truth is the engine's method set, read from its Go files with
// go/ast; nothing here is hand-listed. A method the screen defines on its own
// App (a dialog, a reveal, the browser) is left alone; everything else the
// frontend can call reaches the engine through a one-line forwarder. In phase
// 2 the same reading emits the RPC client and server, and the forwarders call
// a client instead of a value — which is why the API is an interface.
//
// Run from the repository root:
//
//	go run ./internal/engine/gen
//
// or through `go generate ./internal/engine`. A test in the engine package
// regenerates into a temp dir and fails when the committed files are stale.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "", "write generated files under this directory instead of the tree (tests)")
	flag.Parse()
	if err := run(*root, *out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

type method struct {
	name     string
	params   []param
	results  []ast.Expr
	variadic bool
}

type param struct {
	name string
	typ  ast.Expr
}

func run(root, out string) error {
	fset := token.NewFileSet()
	engineDir := filepath.Join(root, "internal", "engine")
	desktopDir := filepath.Join(root, "desktop")

	engineFiles, err := goFiles(engineDir)
	if err != nil {
		return err
	}
	// What the engine package names: its types (to qualify), its imports (to
	// carry the qualifiers' paths), and the methods on *Engine.
	types := map[string]bool{}
	imports := map[string]string{} // selector name → import path
	var methods []method
	for _, f := range engineFiles {
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				name = filepath.Base(path)
				// go-sdk style paths (…/v2) name the package by the element
				// before the version; every import here that matters is
				// aliased or plainly named, so the base is enough.
			}
			imports[name] = path
		}
		for _, d := range file.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					if ts, ok := sp.(*ast.TypeSpec); ok {
						types[ts.Name.Name] = true
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || !d.Name.IsExported() {
					continue
				}
				if receiverType(d.Recv) != "Engine" {
					continue
				}
				m := method{name: d.Name.Name}
				n := 0
				for _, field := range d.Type.Params.List {
					names := field.Names
					if len(names) == 0 {
						names = []*ast.Ident{{Name: fmt.Sprintf("p%d", n)}}
					}
					for _, id := range names {
						name := id.Name
						if name == "_" {
							name = fmt.Sprintf("p%d", n)
						}
						m.params = append(m.params, param{name: name, typ: field.Type})
						n++
					}
					if _, ok := field.Type.(*ast.Ellipsis); ok {
						m.variadic = true
					}
				}
				if d.Type.Results != nil {
					for _, field := range d.Type.Results.List {
						count := len(field.Names)
						if count == 0 {
							count = 1
						}
						for i := 0; i < count; i++ {
							m.results = append(m.results, field.Type)
						}
					}
				}
				methods = append(methods, m)
			}
		}
	}
	// One method per name: a file pair split by build tag (computer_icon_*.go)
	// declares the same method twice, and only one of them is ever compiled.
	seen := map[string]bool{}
	var unique []method
	for _, m := range methods {
		if seen[m.name] {
			continue
		}
		seen[m.name] = true
		unique = append(unique, m)
	}
	methods = unique
	sort.Slice(methods, func(i, j int) bool { return methods[i].name < methods[j].name })

	// What the screen already implements itself.
	own := map[string]bool{}
	desktopFiles, err := goFiles(desktopDir)
	if err != nil {
		return err
	}
	for _, f := range desktopFiles {
		if strings.HasSuffix(f, "_gen.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return err
		}
		for _, d := range file.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv != nil && receiverType(fd.Recv) == "App" {
				own[fd.Name.Name] = true
			}
		}
	}

	// Unexported engine types cannot cross a package boundary; say which
	// method carries one rather than emitting code that does not compile.
	var unexported []string
	for _, m := range methods {
		for _, p := range m.params {
			unexported = append(unexported, unexportedIn(p.typ, types, m.name)...)
		}
		for _, r := range m.results {
			unexported = append(unexported, unexportedIn(r, types, m.name)...)
		}
	}
	if len(unexported) > 0 {
		return fmt.Errorf("exported methods carry unexported types; export them first:\n  %s", strings.Join(unexported, "\n  "))
	}

	api, err := renderAPI(fset, methods, imports)
	if err != nil {
		return err
	}
	fwd, err := renderForwarders(fset, methods, own, types, imports)
	if err != nil {
		return err
	}
	apiPath := filepath.Join(engineDir, "api_gen.go")
	fwdPath := filepath.Join(desktopDir, "engine_forwarders_gen.go")
	if out != "" {
		apiPath = filepath.Join(out, "api_gen.go")
		fwdPath = filepath.Join(out, "engine_forwarders_gen.go")
	}
	if err := os.WriteFile(apiPath, api, 0o644); err != nil {
		return err
	}
	return os.WriteFile(fwdPath, fwd, 0o644)
}

func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	sort.Strings(out)
	return out, nil
}

func receiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// unexportedIn lists the unexported package-level types a signature type
// mentions, each tagged with the method it appears in.
func unexportedIn(t ast.Expr, types map[string]bool, method string) []string {
	var out []string
	ast.Inspect(t, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			_ = sel // a qualified name is another package's business
			return false
		}
		if id, ok := n.(*ast.Ident); ok && types[id.Name] && !id.IsExported() {
			out = append(out, method+": "+id.Name)
		}
		return true
	})
	return out
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, e)
	return buf.String()
}

// qualify rewrites a signature type for use from the desktop package: every
// identifier that names an engine type becomes engine.X, and every qualifier
// it mentions is recorded so the file can import it.
func qualify(fset *token.FileSet, e ast.Expr, types map[string]bool, used map[string]bool) string {
	clone := cloneExpr(e)
	ast.Inspect(clone, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := n.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
			return false
		case *ast.Ident:
			if types[n.Name] && n.IsExported() {
				n.Name = "engine." + n.Name
				used["engine"] = true
			}
		}
		return true
	})
	return exprString(fset, clone)
}

// cloneExpr copies the parts of a type expression qualify may rewrite. Only
// identifiers are edited, so a shallow copy of the containers around them is
// enough — but those containers must be copied, or the first qualification
// would edit the engine's own AST and the API interface would print engine.X
// inside package engine.
func cloneExpr(e ast.Expr) ast.Expr {
	switch e := e.(type) {
	case *ast.Ident:
		return &ast.Ident{Name: e.Name}
	case *ast.StarExpr:
		return &ast.StarExpr{X: cloneExpr(e.X)}
	case *ast.ArrayType:
		return &ast.ArrayType{Len: e.Len, Elt: cloneExpr(e.Elt)}
	case *ast.Ellipsis:
		return &ast.Ellipsis{Elt: cloneExpr(e.Elt)}
	case *ast.MapType:
		return &ast.MapType{Key: cloneExpr(e.Key), Value: cloneExpr(e.Value)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: cloneExpr(e.X), Sel: &ast.Ident{Name: e.Sel.Name}}
	case *ast.ChanType:
		return &ast.ChanType{Dir: e.Dir, Value: cloneExpr(e.Value)}
	case *ast.FuncType:
		return &ast.FuncType{Params: cloneFields(e.Params), Results: cloneFields(e.Results)}
	case *ast.InterfaceType:
		return e // `any`/`interface{}` — nothing to qualify inside
	case *ast.StructType:
		return e
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: cloneExpr(e.X)}
	default:
		return e
	}
}

func cloneFields(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{}
	for _, f := range fl.List {
		out.List = append(out.List, &ast.Field{Names: f.Names, Type: cloneExpr(f.Type)})
	}
	return out
}

func renderAPI(fset *token.FileSet, methods []method, imports map[string]string) ([]byte, error) {
	// The qualifiers the signatures mention, for the import block.
	used := map[string]bool{}
	for _, m := range methods {
		for _, p := range m.params {
			qualifiersIn(p.typ, used)
		}
		for _, r := range m.results {
			qualifiersIn(r, used)
		}
	}
	block, err := importBlock(used, imports, false)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("// Code generated by internal/engine/gen. DO NOT EDIT.\n\npackage engine\n\n")
	b.WriteString(block)
	b.WriteString("// API is every exported method of *Engine: what the screen may ask of the\n")
	b.WriteString("// engine, in one process today and across a socket in phase 2 (§248).\n")
	b.WriteString("type API interface {\n")
	for _, m := range methods {
		b.WriteString("\t" + m.name + "(")
		for i, p := range m.params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.name + " " + exprString(fset, p.typ))
		}
		b.WriteString(")")
		b.WriteString(resultsString(fset, m.results, nil, nil))
		b.WriteString("\n")
	}
	b.WriteString("}\n\nvar _ API = (*Engine)(nil)\n")
	return format.Source([]byte(b.String()))
}

func resultsString(fset *token.FileSet, results []ast.Expr, types map[string]bool, used map[string]bool) string {
	if len(results) == 0 {
		return ""
	}
	render := func(e ast.Expr) string {
		if types == nil {
			return exprString(fset, e)
		}
		return qualify(fset, e, types, used)
	}
	if len(results) == 1 {
		return " " + render(results[0])
	}
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = render(r)
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func renderForwarders(fset *token.FileSet, methods []method, own, types map[string]bool, imports map[string]string) ([]byte, error) {
	used := map[string]bool{}
	var body strings.Builder
	for _, m := range methods {
		if own[m.name] {
			continue
		}
		body.WriteString("func (a *App) " + m.name + "(")
		var args []string
		for i, p := range m.params {
			if i > 0 {
				body.WriteString(", ")
			}
			body.WriteString(p.name + " " + qualify(fset, p.typ, types, used))
			arg := p.name
			if _, ok := p.typ.(*ast.Ellipsis); ok {
				arg += "..."
			}
			args = append(args, arg)
		}
		body.WriteString(")")
		body.WriteString(resultsString(fset, m.results, types, used))
		body.WriteString(" {\n\t")
		if len(m.results) > 0 {
			body.WriteString("return ")
		}
		body.WriteString("a.eng." + m.name + "(" + strings.Join(args, ", ") + ")\n}\n\n")
	}

	var b strings.Builder
	b.WriteString("// Code generated by internal/engine/gen. DO NOT EDIT.\n\npackage main\n\n")
	b.WriteString("// Every engine binding the frontend calls, forwarded (§248 B1). A binding\n")
	b.WriteString("// the screen implements itself is not here — the generator skips any\n")
	b.WriteString("// method desktop/ defines on App.\n\n")
	block, err := importBlock(used, imports, true)
	if err != nil {
		return nil, err
	}
	b.WriteString(block)
	b.WriteString(body.String())
	return format.Source([]byte(b.String()))
}

// qualifiersIn records the package qualifiers a type mentions.
func qualifiersIn(t ast.Expr, used map[string]bool) {
	ast.Inspect(t, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
			return false
		}
		return true
	})
}

// importBlock renders the imports a generated file needs. withEngine is for
// the desktop side, where the engine package itself is one of them.
func importBlock(used map[string]bool, imports map[string]string, withEngine bool) (string, error) {
	var paths []string
	for name := range used {
		path, ok := imports[name]
		if name == "engine" {
			if !withEngine {
				continue
			}
			path, ok = "github.com/Mikedev115/Aetox/internal/engine", true
		}
		if !ok {
			return "", fmt.Errorf("signature uses qualifier %q with no import in the engine package", name)
		}
		alias := ""
		if filepath.Base(path) != name {
			alias = name + " "
		}
		paths = append(paths, alias+strconv.Quote(path))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("import (\n")
	for _, p := range paths {
		b.WriteString("\t" + p + "\n")
	}
	b.WriteString(")\n\n")
	return b.String(), nil
}
