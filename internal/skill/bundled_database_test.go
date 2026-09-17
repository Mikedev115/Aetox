package skill

import (
	"strings"
	"testing"
)

func TestDatabaseSkillShipsAsALocalFirstPack(t *testing.T) {
	s := bundledDoc(t, "aetox-database")
	if !s.Bundled {
		t.Error("aetox-database is not marked bundled")
	}
	if strings.TrimSpace(s.Before) == "" {
		t.Error("aetox-database has no before trigger, so the prompt cannot route database work to it")
	}
	if !strings.Contains(s.body, "calls together in one reply") {
		t.Error("aetox-database makes independent reference reads consume separate model rounds")
	}

	want := map[string]bool{
		"references/local-workflow.md": false,
		"references/migrations.md":     false,
		"references/sql-safety.md":     false,
		"references/sqlite.md":         false,
		"references/postgres.md":       false,
		"references/mysql.md":          false,
	}
	for _, name := range supportingFiles(s) {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("aetox-database does not ship %s", name)
		}
	}

	for name := range want {
		content, err := readSkillFile(s, name, 0)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(strings.ToLower(content), "mcp") {
			t.Errorf("%s crosses the local-only boundary by discussing MCP", name)
		}
	}
	if strings.Contains(strings.ToLower(s.body), "mcp") {
		t.Error("the database skill crosses the local-only boundary by discussing MCP")
	}
}
