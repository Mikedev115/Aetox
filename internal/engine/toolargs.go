package engine

// Small helpers the engine's own tool packs share with the window's — kept
// here in the engine's own words after the browser and computer packs moved
// to desktop/ (§248 B1), where they keep a copy. Two copies of ten lines is
// cheaper than a third package for them.

import (
	"encoding/json"
	"regexp"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// The names the window's packs register under, as the engine knows them: the
// tool event stamp (recordToolAction), the computer switch and the guide's
// desk (workbenchSkills) judge by name, never by type.
const (
	browserToolName  = "browser"
	computerToolName = "computer"
	guideToolName    = "guide"
)

var (
	driveLetterRe = regexp.MustCompile(`^[a-zA-Z]:[\\/]`)
	urlSchemeRe   = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	bareSchemeRe  = regexp.MustCompile(`(?i)^(about|data|mailto|javascript):`)
)

func toolDef(name, description string, schema map[string]any) model.ToolDefinition {
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  payload,
		},
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// Both of these used to be written out here, and both were a shape short: an
// int and a float64 but not a quoted number, a bool but not a quoted bool.
// Models send the quoted forms — `{"action":"click","ref":"1"}` is in this
// machine's tool_runs twelve times — and each one arrived as a zero-value that
// no longer resembled what was asked for. internal/skill has had the right rule
// since `read` needed it; this defers to it rather than agreeing with it.
func intArg(v any) int { return skill.IntArg(v) }
