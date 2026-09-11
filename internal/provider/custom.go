package provider

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Custom is one endpoint the user added themselves: a name of their choosing
// and where it answers. Every other fact about it — wire format, whether a
// key is asked for, what the balance card can say — is borrowed from the
// "openai-compatible" row, because that is the only protocol a user can name
// without Aetox having heard of the host.
//
// This exists because the single "openai-compatible" row could hold ONE base
// URL and ONE key at a time. Pointing it at DeepSeek meant typing over the
// LM Studio port typed there last week, and the picker had no way to say
// which of the two it currently meant (owner, 11 ก.ย. 2569: อยากให้เพิ่มได้เป็นลิสต์
// แบบ + เข้าไป ไม่ใช่มาเปลี่ยนค่าทับ มันจะได้จำค่าได้). A row per endpoint is a
// row per key, a row per model list, and a row per name in the picker.
type Custom struct {
	// ID is the provider name every other layer keys on — the credential
	// store, the per-provider model, the enabled list. Lowercase a-z, 0-9 and
	// dashes, and never a name the static catalog already answers to.
	ID string
	// BaseURL is the endpoint at creation. A later edit on the settings page
	// is stored as an override beside it, the same way the static rows take
	// one, so "reset" comes back here.
	BaseURL string
}

var (
	customMu   sync.RWMutex
	customRows = map[string]Custom{}
)

// customIDPattern is deliberately narrower than what a filename allows: the id
// travels into JSON keys, a URL-safe env-var-like namespace and the sidebar,
// and a dash-joined lowercase word is the shape every catalog name has.
var customIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// SlugCustomID turns whatever the user typed into the id a row will carry, or
// "" when nothing usable survives (a name written entirely in Thai, say —
// the caller says so, rather than silently minting "provider-1").
func SlugCustomID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := true // so a leading dash is never written
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-', r == '_', r == ' ', r == '.', r == '/':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > 40 {
		out = strings.TrimRight(out[:40], "-")
	}
	return out
}

// ValidateCustomID says whether id may name a custom row: well-formed, and not
// a name — canonical or alias — the static catalog already owns. A user who
// adds "deepseek" must not silently shadow DeepSeek for every saved
// preference that names it.
func ValidateCustomID(id string) error {
	if !customIDPattern.MatchString(id) {
		return fmt.Errorf("ชื่อ provider ใช้ได้เฉพาะ a-z 0-9 และ - (ได้ %q)", id)
	}
	if _, ok := catalog[id]; ok {
		return fmt.Errorf("%q เป็นชื่อ provider ที่มีอยู่แล้ว", id)
	}
	for canonical, e := range catalog {
		for _, alias := range e.aliases {
			if id == alias {
				return fmt.Errorf("%q เป็นชื่อเรียกของ %s อยู่แล้ว", id, canonical)
			}
		}
	}
	return nil
}

// SetCustom replaces the whole registered set. Called by whoever loads the
// preference file — the file is the record, this map is only the copy the
// catalog answers from, so it is rewritten on every load rather than edited
// in place and can never drift from what is on disk. Rows that fail
// ValidateCustomID are skipped, not fatal: a hand-edited file must not take
// the catalog down with it.
func SetCustom(rows []Custom) {
	next := make(map[string]Custom, len(rows))
	for _, row := range rows {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if ValidateCustomID(id) != nil {
			continue
		}
		row.ID = id
		row.BaseURL = strings.TrimSuffix(strings.TrimSpace(row.BaseURL), "/")
		next[id] = row
	}
	customMu.Lock()
	customRows = next
	customMu.Unlock()
}

// Customs lists the registered rows, sorted by id.
func Customs() []Custom {
	customMu.RLock()
	out := make([]Custom, 0, len(customRows))
	for _, row := range customRows {
		out = append(out, row)
	}
	customMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// IsCustom reports whether name is a row the user added rather than one the
// catalog ships. Normalized first, so an alias of a static row is never
// mistaken for one.
func IsCustom(name string) bool {
	_, ok := lookupCustom(Normalize(name))
	return ok
}

func lookupCustom(canonical string) (Custom, bool) {
	customMu.RLock()
	defer customMu.RUnlock()
	row, ok := customRows[canonical]
	return row, ok
}

// customSpec is the Spec a custom row answers with: the "openai-compatible"
// row with its own name and endpoint, and no environment variable — a key
// typed for one endpoint must not be read for another just because both are
// OpenAI-shaped.
func customSpec(row Custom) Spec {
	base := catalog["openai-compatible"]
	return Spec{
		Canonical:      row.ID,
		Aliases:        []string{row.ID},
		RequiresAPIKey: base.requiresAPIKey,
		Runtime:        base.runtime,
		BaseURL:        row.BaseURL,
		ModelDefaults:  ModelDefaults{},
		Capabilities:   base.capabilities,
		BalanceKind:    base.balanceKind,
		QuotaSource:    base.quotaSource,
		AcceptsAPIKey:  !base.signInOnly,
	}
}
