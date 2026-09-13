package model

// What the ChatGPT backend states about each model it serves, and where that
// statement is kept between runs.
//
// The list endpoint DiscoverResponsesModels already calls answers with far
// more than slugs: every row carries `supported_reasoning_levels` and
// `default_reasoning_level`, per model. Measured 13 ก.ย. 2026 on a Plus
// account — gpt-5.6-luna states low, medium, high, xhigh, max; gpt-5.6-sol and
// -terra add ultra; gpt-5.5 stops at xhigh — while the hand-written table
// (responsesThinkingCapabilities) stopped at xhigh for all of them and folded
// max onto it, so the picker never offered a rung the backend had been
// serving all along. Owner: *"ควรมีตามที่เขาส่งมานะ ไม่ใช่ตั้งเอง"*.
//
// So the statement is kept, keyed by slug, and the codex resolver reads it
// before it reads any table of ours. It is cached to disk the way the
// models.dev table is (model-catalog.json), for the same reason: the picker
// is asked at startup, before anyone has opened the models page, and a
// ladder that exists or does not depending on which screen was opened first
// is the bug §168.21 already fixed once. The table stays as the fallback for
// a machine that has never fetched the list — a conservative ladder every
// model there accepts beats no dial at all.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ResponsesModelFacts is one row of the backend's statement, the fields the
// capability questions read. Slugs are the backend's own ("gpt-5.6-luna").
type ResponsesModelFacts struct {
	Slug string `json:"slug"`
	// ReasoningLevels as stated, shallowest first, in the backend's own words
	// (they are Aetox's words too — thinkingLadder covers every one seen).
	ReasoningLevels []string `json:"reasoning_levels,omitempty"`
	// DefaultReasoningLevel is what the backend runs when no effort is sent.
	DefaultReasoningLevel string `json:"default_reasoning_level,omitempty"`
}

// responsesModelsFile sits beside model-catalog.json under the data root.
const responsesModelsFile = "responses-models.json"

var (
	responsesFactsMu sync.RWMutex
	responsesFacts   map[string]ResponsesModelFacts
	// responsesFactsRoot is where the last install came from, so a fresh
	// discovery can be written back without the discoverer learning what a
	// data root is. Empty until InstallCachedCatalog has run.
	responsesFactsRoot string
	// responsesFactsSeen is the mtime of the file as last read, and
	// responsesFactsChecked when it was last stat'ed. The list is fetched by
	// the SCREEN process (desktop/providers.go) and the ladder is answered by
	// the ENGINE process, so the engine re-reads the file when it has moved —
	// a stat per lookup, throttled, rather than a restart to see a new rung.
	responsesFactsSeen    time.Time
	responsesFactsChecked time.Time
)

// responsesFactsRecheck is how often the file is stat'ed at most. A lookup
// happens per turn and per picker draw; a stat every second is nothing, and
// a rung showing up a second after the models page fetched it is immediate.
const responsesFactsRecheck = time.Second

// SetResponsesModelFacts replaces the in-memory statement. Nil or empty
// clears it, which sends the codex resolver back to its fallback table.
func SetResponsesModelFacts(rows []ResponsesModelFacts) {
	facts := make(map[string]ResponsesModelFacts, len(rows))
	for _, r := range rows {
		slug := strings.ToLower(strings.TrimSpace(r.Slug))
		if slug == "" {
			continue
		}
		facts[slug] = r
	}
	responsesFactsMu.Lock()
	if len(facts) == 0 {
		responsesFacts = nil
	} else {
		responsesFacts = facts
	}
	responsesFactsMu.Unlock()
}

// ResponsesModelFactsFor is what the backend said about one slug, and whether
// it has said anything at all. Picks up a file another process wrote since
// the last read (reloadResponsesFactsIfMoved).
func ResponsesModelFactsFor(modelID string) (ResponsesModelFacts, bool) {
	reloadResponsesFactsIfMoved()
	responsesFactsMu.RLock()
	defer responsesFactsMu.RUnlock()
	f, ok := responsesFacts[strings.ToLower(strings.TrimSpace(modelID))]
	return f, ok
}

// reloadResponsesFactsIfMoved re-reads the cache file when its mtime is newer
// than what this process last loaded. No root, no file, or a recent check:
// nothing happens.
func reloadResponsesFactsIfMoved() {
	responsesFactsMu.RLock()
	root, seen, checked := responsesFactsRoot, responsesFactsSeen, responsesFactsChecked
	responsesFactsMu.RUnlock()
	if root == "" || time.Since(checked) < responsesFactsRecheck {
		return
	}
	responsesFactsMu.Lock()
	responsesFactsChecked = time.Now()
	responsesFactsMu.Unlock()
	info, err := os.Stat(filepath.Join(root, responsesModelsFile))
	if err != nil || !info.ModTime().After(seen) {
		return
	}
	rows, err := LoadResponsesModelFacts(root)
	if err != nil || len(rows) == 0 {
		return
	}
	SetResponsesModelFacts(rows)
	responsesFactsMu.Lock()
	responsesFactsSeen = info.ModTime()
	responsesFactsMu.Unlock()
}

// rememberResponsesModelFacts is what DiscoverResponsesModels calls with a
// fresh list: install it, and write it beside the catalog cache when a root
// is known. A write failure is swallowed on purpose — the list is already
// live in memory, and a read-only data folder must not turn a successful
// model fetch into an error the picker shows.
func rememberResponsesModelFacts(rows []ResponsesModelFacts) {
	SetResponsesModelFacts(rows)
	responsesFactsMu.RLock()
	root := responsesFactsRoot
	responsesFactsMu.RUnlock()
	if root != "" && SaveResponsesModelFacts(root, rows) == nil {
		if info, err := os.Stat(filepath.Join(root, responsesModelsFile)); err == nil {
			responsesFactsMu.Lock()
			responsesFactsSeen = info.ModTime()
			responsesFactsMu.Unlock()
		}
	}
}

// InstallCachedResponsesModelFacts reads the last statement off disk. Silent
// and networkless, like InstallCachedCatalog, which calls it: a first run
// has no file and that is not a failure.
func InstallCachedResponsesModelFacts(dataRoot string) {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return
	}
	responsesFactsMu.Lock()
	responsesFactsRoot = dataRoot
	responsesFactsSeen, responsesFactsChecked = time.Time{}, time.Time{}
	responsesFactsMu.Unlock()
	rows, err := LoadResponsesModelFacts(dataRoot)
	if err != nil || len(rows) == 0 {
		return
	}
	SetResponsesModelFacts(rows)
	if info, err := os.Stat(filepath.Join(dataRoot, responsesModelsFile)); err == nil {
		responsesFactsMu.Lock()
		responsesFactsSeen = info.ModTime()
		responsesFactsMu.Unlock()
	}
}

// LoadResponsesModelFacts reads the cached statement. A missing or corrupt
// file is nil, nil — the next discovery replaces it.
func LoadResponsesModelFacts(dataRoot string) ([]ResponsesModelFacts, error) {
	raw, err := os.ReadFile(filepath.Join(dataRoot, responsesModelsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rows []ResponsesModelFacts
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, nil
	}
	return rows, nil
}

// SaveResponsesModelFacts writes the statement atomically, the way
// SaveModelCatalog does, so an interrupted write cannot leave a half-file.
func SaveResponsesModelFacts(dataRoot string, rows []ResponsesModelFacts) error {
	if len(rows) == 0 {
		return nil
	}
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dataRoot, responsesModelsFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// resolveResponsesThinkingCapabilities is the codex dial: the backend's own
// statement for this slug when there is one, the hand-written table when the
// list has never been fetched on this machine.
//
// "off" is always the first rung, and it is never a wire value: it is the
// absence of the reasoning parameter, which leaves the model on the default
// the same statement names. Everything else is sent as stated, so a rung the
// backend adds next month is offered the day the list is next fetched, with
// no edit here. Levels the backend does NOT state for this slug fold onto the
// nearest one it does (deriveThinkingAliases) — the same treatment every
// catalog-driven provider gets, and what keeps a saved "max" from becoming a
// 400 on a model whose ladder stops at xhigh.
func resolveResponsesThinkingCapabilities(modelID string) ThinkingCapabilities {
	facts, known := ResponsesModelFactsFor(modelID)
	if !known {
		return cloneThinkingCapabilities(responsesThinkingCapabilities)
	}
	stated := orderByLadder(lowerAll(facts.ReasoningLevels))
	if len(stated) == 0 {
		return cloneThinkingCapabilities(responsesThinkingCapabilities)
	}
	levels := append([]string{"off"}, stated...)
	wire := make(map[string]string, len(stated))
	for _, l := range stated {
		wire[l] = l
	}
	def := strings.ToLower(strings.TrimSpace(facts.DefaultReasoningLevel))
	if !containsLevel(stated, def) {
		def = preferredLevel(levels, responsesDefaultOrder)
	}
	return ThinkingCapabilities{
		Supported: true,
		Native:    true,
		Levels:    levels,
		Default:   def,
		Runtime:   ThinkingRuntimeReasoningEffort,
		Source:    "responses-models",
		Wire:      wire,
		Aliases:   deriveThinkingAliases(levels),
	}
}

// responsesDefaultOrder stands in only when the backend names a default that
// is not in its own list — which it has never done, but a default outside
// the menu is the one shape the picker cannot draw.
var responsesDefaultOrder = []string{"medium", "high", "low"}

func lowerAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
			out = append(out, v)
		}
	}
	return out
}
