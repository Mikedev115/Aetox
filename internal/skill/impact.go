package skill

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/lsp"
	"github.com/Mikedev115/Aetox/internal/model"
)

// impactSkill answers "if this symbol changes, what is the blast radius:
// production, test, and other references, generated/RPC boundaries, and related narrow checks".
// It is read-only and queries the language server via lsp.Shared without any background indexing.
type impactSkill struct {
	root         string
	outputSubdir func() string
}

func (*impactSkill) Name() string { return "impact" }

func (*impactSkill) Description() string {
	return "วิเคราะห์ผลกระทบก่อนแก้ไข: จุดประกาศ, references แยกตาม role, boundary generated/RPC, และการทดสอบที่เกี่ยวข้อง"
}

func (*impactSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of a file where the identifier is used or declared.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "The identifier to analyze, exactly as written.",
			},
		},
		"required":             []string{"path", "name"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name: "impact",
			Description: "Analyze the blast radius of an identifier before modifying it: declaration, references categorized by role (production/tests/other), generated/RPC surfaces, and related narrow checks.",
			Parameters: payload,
		},
	}
}

func (s *impactSkill) Execute(ctx context.Context, input Input) (Output, error) {
	args := stringSlice(input["args"])
	if len(args) < 2 {
		err := errors.New("usage: impact <path> <name>")
		return newToolOutput("impact", "impact", "", time.Now(), false, err), err
	}
	return s.ExecuteTool(ctx, map[string]any{"path": args[0], "name": strings.Join(args[1:], " ")})
}

func (s *impactSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	res, early, err := resolveSymbolTarget(ctx, s.root, s.outputSubdir, "impact", args)
	if early != nil {
		return *early, err
	}

	outText := formatImpact(s.root, res.name, res.path, res.info)
	out, truncated := limitLines(strings.TrimRight(outText, "\n"), defaultToolOutputLineLimit)
	o := newToolOutput("impact", res.command, out, res.start, truncated, nil)
	o.ResultCount = len(res.info.Refs)
	return o, nil
}

func impactSymbolTitle(defPath, name, hover string) string {
	pkg := ""
	if defPath != "" {
		dir := filepath.Base(filepath.Dir(defPath))
		if dir != "." && dir != "/" && dir != "\\" && dir != "" {
			pkg = dir
		}
	}
	recv := ""
	for _, rawLine := range strings.Split(hover, "\n") {
		line := strings.TrimSpace(rawLine)
		line = strings.TrimPrefix(line, "```go")
		line = strings.TrimPrefix(line, "```")
		line = strings.TrimSpace(line)

		funcIdx := strings.Index(line, "func ")
		if funcIdx == -1 {
			continue
		}
		funcPart := line[funcIdx+5:]
		if !strings.HasPrefix(funcPart, "(") {
			continue
		}
		endIdx := strings.Index(funcPart, ")")
		if endIdx <= 1 {
			continue
		}
		afterParen := strings.TrimSpace(funcPart[endIdx+1:])
		if strings.HasPrefix(afterParen, name) || strings.HasPrefix(afterParen, "."+name) {
			recvPart := funcPart[1:endIdx]
			recvPart = strings.TrimPrefix(recvPart, "*")
			fields := strings.Fields(recvPart)
			if len(fields) == 1 {
				recv = strings.TrimPrefix(fields[0], "*")
			} else if len(fields) >= 2 {
				recv = strings.TrimPrefix(fields[len(fields)-1], "*")
			}
			if recv != "" {
				break
			}
		}
	}
	if pkg != "" && recv != "" {
		return fmt.Sprintf("%s.%s.%s", pkg, recv, name)
	}
	if pkg != "" {
		return fmt.Sprintf("%s.%s", pkg, name)
	}
	return name
}

func isFileHeaderGenerated(root, path string) bool {
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, path)
	}
	f, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() && lines < 5 {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if strings.HasPrefix(line, "// code generated") || strings.Contains(line, "@generated") || strings.HasPrefix(line, "/* code generated") {
			return true
		}
		lines++
	}
	return false
}

func isConfirmedGenerated(root, path string) bool {
	slashPath := filepath.ToSlash(path)
	base := filepath.Base(slashPath)
	lowerPath := strings.ToLower(slashPath)
	lowerBase := strings.ToLower(base)

	if strings.HasSuffix(lowerBase, "_gen.go") || strings.Contains(lowerBase, ".gen.") ||
		strings.HasSuffix(lowerBase, "_gen.ts") || strings.HasSuffix(lowerBase, ".gen.ts") {
		return true
	}
	if strings.Contains(lowerPath, "wailsjs/") {
		return true
	}
	if strings.HasSuffix(lowerBase, ".pb.go") || strings.HasSuffix(lowerBase, "_grpc.pb.go") || strings.HasSuffix(lowerBase, "_pb.go") {
		return true
	}
	return isFileHeaderGenerated(root, path)
}

func isRPCConvention(path string) bool {
	slashPath := filepath.ToSlash(path)
	lowerPath := strings.ToLower(slashPath)
	return strings.Contains(lowerPath, "/rpc/") || strings.HasPrefix(lowerPath, "rpc/")
}

func isOther(root, path string) bool {
	if isConfirmedGenerated(root, path) {
		return true
	}
	slashPath := filepath.ToSlash(path)
	base := filepath.Base(slashPath)
	lowerPath := strings.ToLower(slashPath)
	lowerBase := strings.ToLower(base)

	if strings.Contains(lowerPath, "/gen/") || strings.HasPrefix(lowerPath, "gen/") ||
		strings.Contains(lowerPath, "/generated/") || strings.HasPrefix(lowerPath, "generated/") {
		return true
	}
	if strings.HasSuffix(lowerBase, ".md") || strings.HasSuffix(lowerBase, ".txt") || strings.HasSuffix(lowerBase, ".rst") ||
		strings.HasPrefix(lowerPath, "docs/") || strings.Contains(lowerPath, "/docs/") {
		return true
	}
	if strings.Contains(lowerPath, "/testdata/") || strings.HasPrefix(lowerPath, "testdata/") ||
		strings.Contains(lowerPath, "/fixtures/") || strings.HasPrefix(lowerPath, "fixtures/") ||
		strings.Contains(lowerPath, "/mocks/") || strings.HasPrefix(lowerPath, "mocks/") ||
		strings.Contains(lowerPath, "/mock/") || strings.HasPrefix(lowerBase, "mock_") || strings.HasPrefix(lowerBase, "fake_") {
		return true
	}
	return isFileHeaderGenerated(root, path)
}

func isTest(path string) bool {
	slashPath := filepath.ToSlash(path)
	base := filepath.Base(slashPath)
	lowerPath := strings.ToLower(slashPath)
	lowerBase := strings.ToLower(base)

	if strings.HasSuffix(lowerBase, "_test.go") {
		return true
	}
	if strings.HasSuffix(lowerBase, ".test.ts") || strings.HasSuffix(lowerBase, ".test.js") ||
		strings.HasSuffix(lowerBase, ".test.tsx") || strings.HasSuffix(lowerBase, ".test.jsx") {
		return true
	}
	if strings.HasSuffix(lowerBase, ".spec.ts") || strings.HasSuffix(lowerBase, ".spec.js") ||
		strings.HasSuffix(lowerBase, ".spec.tsx") || strings.HasSuffix(lowerBase, ".spec.jsx") {
		return true
	}
	if strings.HasPrefix(lowerBase, "test_") || strings.HasSuffix(lowerBase, "_test.py") {
		return true
	}
	if strings.Contains(lowerPath, "/tests/") || strings.HasPrefix(lowerPath, "tests/") ||
		strings.Contains(lowerPath, "/test/") || strings.HasPrefix(lowerPath, "test/") ||
		strings.Contains(lowerPath, "/__tests__/") {
		return true
	}
	return false
}

func commonDir(dirs []string) string {
	if len(dirs) == 0 {
		return ""
	}
	parts := strings.Split(dirs[0], "/")
	common := parts
	for _, d := range dirs[1:] {
		cur := strings.Split(d, "/")
		i := 0
		for i < len(common) && i < len(cur) && common[i] == cur[i] {
			i++
		}
		common = common[:i]
	}
	return strings.Join(common, "/")
}

func deriveNarrowTestCommand(testFiles []string) string {
	if len(testFiles) == 0 {
		return ""
	}
	allGo := true
	for _, tf := range testFiles {
		if !strings.HasSuffix(tf, "_test.go") {
			allGo = false
			break
		}
	}
	if allGo {
		dirs := make([]string, 0, len(testFiles))
		for _, tf := range testFiles {
			dirs = append(dirs, filepath.ToSlash(filepath.Dir(tf)))
		}
		common := commonDir(dirs)
		if common == "" || common == "." {
			return "go test ./..."
		}
		return fmt.Sprintf("go test ./%s/...", common)
	}
	return ""
}

func formatImpact(root, name, queriedPath string, info *lsp.SymbolInfo) string {
	var b strings.Builder
	title := impactSymbolTitle(info.DefPath, name, info.Hover)
	fmt.Fprintf(&b, "Impact: %s\n", title)

	if info.DefPath != "" {
		fmt.Fprintf(&b, "Declared at %s:%d\n\n", relativeToRoot(root, info.DefPath), info.DefLine)
	} else {
		fmt.Fprintf(&b, "Declared at %s:%d\n\n", queriedPath, info.Occurrence)
	}

	if info.RefsTruncated {
		fmt.Fprintf(&b, "Warning: references capped at %d by language server; blast radius is incomplete and additional callers/tests exist beyond this list\n\n", len(info.Refs))
	}

	var prodRefs []string
	var testRefs []string
	var otherRefs []string
	var genRPCEvidence []string
	testFilesMap := make(map[string]bool)

	for _, loc := range info.Refs {
		rel := relativeToRoot(root, loc.Path)
		locStr := fmt.Sprintf("%s:%d", rel, loc.Line)

		if isConfirmedGenerated(root, rel) {
			genRPCEvidence = append(genRPCEvidence, locStr)
			otherRefs = append(otherRefs, locStr)
		} else if isTest(rel) {
			testRefs = append(testRefs, locStr)
			testFilesMap[rel] = true
		} else if isOther(root, rel) {
			otherRefs = append(otherRefs, locStr)
		} else {
			prodRefs = append(prodRefs, locStr)
			if isRPCConvention(rel) {
				genRPCEvidence = append(genRPCEvidence, fmt.Sprintf("%s (path convention only, unconfirmed runtime boundary)", locStr))
			}
		}
	}

	prodCount := fmt.Sprintf("%d", len(prodRefs))
	testCount := fmt.Sprintf("%d", len(testRefs))
	otherCount := fmt.Sprintf("%d", len(otherRefs))
	if info.RefsTruncated {
		prodCount = fmt.Sprintf("%d, partial", len(prodRefs))
		testCount = fmt.Sprintf("%d, partial", len(testRefs))
		otherCount = fmt.Sprintf("%d, partial", len(otherRefs))
	}

	fmt.Fprintf(&b, "Production references (%s)\n", prodCount)
	if len(prodRefs) > 0 {
		for _, ref := range prodRefs {
			fmt.Fprintf(&b, "- %s\n", ref)
		}
	} else {
		b.WriteString("- (none)\n")
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "Test references (%s)\n", testCount)
	if len(testRefs) > 0 {
		for _, ref := range testRefs {
			fmt.Fprintf(&b, "- %s\n", ref)
		}
	} else {
		b.WriteString("- (none)\n")
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "Other references (%s)\n", otherCount)
	if len(otherRefs) > 0 {
		for _, ref := range otherRefs {
			fmt.Fprintf(&b, "- %s\n", ref)
		}
	} else {
		b.WriteString("- (none)\n")
	}
	b.WriteString("\n")

	b.WriteString("Generated/RPC surface evidence\n")
	if len(genRPCEvidence) > 0 {
		for _, ref := range genRPCEvidence {
			fmt.Fprintf(&b, "- %s\n", ref)
		}
	} else {
		b.WriteString("- (none)\n")
	}
	b.WriteString("\n")

	b.WriteString("Related narrow checks\n")
	testFiles := make([]string, 0, len(testFilesMap))
	for tf := range testFilesMap {
		testFiles = append(testFiles, tf)
	}
	sort.Strings(testFiles)

	cmd := deriveNarrowTestCommand(testFiles)
	if len(testFiles) > 0 {
		for _, tf := range testFiles {
			fmt.Fprintf(&b, "- %s\n", tf)
		}
		if cmd != "" {
			fmt.Fprintf(&b, "- %s\n", cmd)
		} else {
			b.WriteString("- No narrow automated check was identified.\n")
		}
		if info.RefsTruncated {
			b.WriteString("- Warning: additional test files may exist outside the capped references\n")
		}
	} else {
		if info.RefsTruncated {
			b.WriteString("- Warning: references were capped at limit; no test files identified in partial set\n")
		} else {
			b.WriteString("- No narrow automated check was identified.\n")
		}
	}

	return b.String()
}
