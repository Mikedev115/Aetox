package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// RecurringRequest represents a request or habit that has appeared repeatedly
// across multiple sessions (docs/architecture/user-profile-and-habits-2026-09-06.md §4.2).
//
// Computed deterministically from `messages` in SQLite without an LLM: the model
// reads a count; it does not invent one.
type RecurringRequest struct {
	Text        string   `json:"text"`        // Representative text from the latest session
	Normalized  string   `json:"normalized"`  // Cleaned key for clustering
	Count       int      `json:"count"`       // Number of sessions asking for this
	SessionIDs  []string `json:"sessionIds"`  // IDs of sessions that contained this request
	LastAskedAt string   `json:"lastAskedAt"` // Timestamp of the most recent occurrence
}

// Regex for attachment wrappers injected into prompt text.
var attachmentPattern = regexp.MustCompile(`(?i)\n*\[attachment: [^\]]*\]\s*\S*|\[Attached (?:file|image): [^\]]+\]`)

// Thai polite particles and request fillers stripped during normalization.
var thaiFillers = []string{
	"หน่อยครับ", "หน่อยค่ะ", "หน่อยนะ", "หน่อยดิ", "หน่อยที", "หน่อยจ้า",
	"นะครับ", "นะคะ", "ครับผม", "ครับ", "ค่ะ", "จ้า", "หน่อย",
	"ด้วยครับ", "ด้วยค่ะ", "ด้วยนะ", "ด้วย", "ให้ผมหน่อย", "ให้หน่อย",
	"ให้ที", "ทีครับ", "ทีค่ะ", "ที", "นะ", "ดิ", "หน่อยซิ", "ซิ",
	"มาหน่อย", "มา", "ให้", "เท่าไหร่", "ยังไง", "บ้าง",
}

// thaiPrefixes stripped when identifying the core action.
var thaiPrefixes = []string{
	"ตรวจการ", "ตรวจสอบ", "ตรวจ", "เช็คการ", "เช็ค",
	"อยากได้", "รบกวน", "ช่วย", "ฝาก", "ขอให้", "ขอ",
	"สร้าง", "ทำ", "ลอง",
}

// normalizeMessage cleans user text to extract the core intent for recurrence clustering.
func normalizeMessage(text string) string {
	// 1. Remove attachments notation
	text = attachmentPattern.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// 2. Lowercase for case-insensitive matching
	text = strings.ToLower(text)

	// 3. Remove punctuation and control characters (preserve dots between digits for versions like 1.5.22)
	runes := []rune(text)
	var b strings.Builder
	for i, r := range runes {
		if r == '.' && i > 0 && i+1 < len(runes) && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]) {
			b.WriteRune(r)
		} else if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	text = b.String()

	// 4. Collapse spaces and strip fillers from words
	fields := strings.Fields(text)
	for i, f := range fields {
		for _, filler := range thaiFillers {
			if strings.HasSuffix(f, filler) && len(f) > len(filler) {
				fields[i] = strings.TrimSuffix(f, filler)
				break
			}
		}
	}
	text = strings.Join(fields, " ")

	// 5. Strip Thai polite particles and prefixes/suffixes iteratively
	changed := true
	for changed {
		changed = false
		for _, filler := range thaiFillers {
			if strings.HasSuffix(text, filler) {
				text = strings.TrimSpace(strings.TrimSuffix(text, filler))
				changed = true
			}
		}
		for _, prefix := range thaiPrefixes {
			if strings.HasPrefix(text, prefix) {
				text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
				changed = true
			}
		}
	}

	// 6. Common synonym alignment
	text = strings.ReplaceAll(text, "กำลังไฟ", "กินไฟ")

	return strings.TrimSpace(text)
}

// longestCommonSubstringRunes finds the length of the longest contiguous common rune slice.
func longestCommonSubstringRunes(a, b []rune) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	maxLen := 0
	dp := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		prev := 0
		for j := 1; j <= len(b); j++ {
			temp := dp[j]
			if a[i-1] == b[j-1] {
				dp[j] = prev + 1
				if dp[j] > maxLen {
					maxLen = dp[j]
				}
			} else {
				dp[j] = 0
			}
			prev = temp
		}
	}
	return maxLen
}

// areRequestsSimilar determines if two normalized requests belong to the same cluster.
func areRequestsSimilar(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}

	runesA := []rune(a)
	runesB := []rune(b)
	lenA, lenB := len(runesA), len(runesB)
	minRunes := lenA
	maxRunes := lenB
	if lenB < minRunes {
		minRunes = lenB
		maxRunes = lenA
	}

	if minRunes == 0 {
		return false
	}

	// If one contains the other: require the shorter one to be at least 50% of the longer one
	// or minRunes >= 15 runes so that short words do not absorb long completely different sentences.
	if strings.Contains(a, b) || strings.Contains(b, a) {
		if float64(minRunes)/float64(maxRunes) >= 0.5 || minRunes >= 15 {
			return true
		}
	}

	// Check longest common contiguous substring as a proportion of both texts
	lcs := longestCommonSubstringRunes(runesA, runesB)
	if minRunes >= 6 && float64(lcs)/float64(minRunes) >= 0.55 && lcs >= 6 {
		if float64(lcs)/float64(maxRunes) >= 0.35 {
			return true
		}
	}

	// Word/token overlap check for multi-word phrases.
	// Single characters, pure numbers (e.g. "1", "5"), and duplicate matches must not trigger false clusters.
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) > 1 && len(wordsB) > 1 {
		setB := make(map[string]bool, len(wordsB))
		for _, w := range wordsB {
			if len([]rune(w)) > 1 && !isAllDigits(w) {
				setB[w] = true
			}
		}

		uniqueA := make(map[string]bool, len(wordsA))
		for _, w := range wordsA {
			if len([]rune(w)) > 1 && !isAllDigits(w) {
				uniqueA[w] = true
			}
		}

		common := 0
		for w := range uniqueA {
			if setB[w] {
				common++
			}
		}

		minWords := len(uniqueA)
		if len(setB) < minWords {
			minWords = len(setB)
		}

		if minWords > 0 && common >= 2 && float64(common)/float64(minWords) >= 0.5 {
			return true
		}
	}

	return false
}

type sessionUserMsg struct {
	sessionID string
	rawText   string
	time      string
}

// clusterRecurringMessages groups first-turn messages into recurring clusters.
func clusterRecurringMessages(msgs []sessionUserMsg, minCount int) []RecurringRequest {
	if len(msgs) == 0 {
		return nil
	}

	var clusters []*RecurringRequest

	for _, m := range msgs {
		norm := normalizeMessage(m.rawText)
		if len(norm) < 3 {
			continue
		}

		matched := false
		for _, c := range clusters {
			if areRequestsSimilar(norm, c.Normalized) {
				c.Count++
				c.SessionIDs = append(c.SessionIDs, m.sessionID)
				if c.LastAskedAt == "" || m.time > c.LastAskedAt {
					c.LastAskedAt = m.time
					c.Text = m.rawText
				}
				matched = true
				break
			}
		}

		if !matched {
			clusters = append(clusters, &RecurringRequest{
				Text:        m.rawText,
				Normalized:  norm,
				Count:       1,
				SessionIDs:  []string{m.sessionID},
				LastAskedAt: m.time,
			})
		}
	}

	var out []RecurringRequest
	for _, c := range clusters {
		if c.Count >= minCount {
			out = append(out, *c)
		}
	}

	// Sort by count descending, then by last asked timestamp
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].LastAskedAt > out[j].LastAskedAt
	})

	return out
}

// loadIgnoredHabits loads normalized patterns that the user dismissed.
func loadIgnoredHabits(db *sql.DB) map[string]bool {
	ignored := make(map[string]bool)
	if db == nil {
		return ignored
	}
	_ = eachRow(db, "habits: reading ignored habits", `SELECT normalized FROM ignored_habits`, nil, func(rows *sql.Rows) error {
		var norm string
		if err := rows.Scan(&norm); err == nil && norm != "" {
			ignored[norm] = true
		}
		return nil
	})
	return ignored
}

// detectRecurringRequests queries SQLite for the first user message of recent sessions
// and returns clusters that cross minCount, excluding dismissed habits.
func detectRecurringRequests(db *sql.DB, minCount, limitSessions int) []RecurringRequest {
	if db == nil {
		return nil
	}
	if limitSessions <= 0 {
		limitSessions = 150
	}

	// Get the first user message of each session
	var msgs []sessionUserMsg
	err := eachRow(db, "habits: reading first messages", `
		SELECT m.session_id, m.text, m.time
		  FROM messages m
		  INNER JOIN (
		      SELECT session_id, MIN(id) AS first_id
		        FROM messages
		       WHERE role = 'user'
		       GROUP BY session_id
		  ) f ON m.id = f.first_id
		 ORDER BY m.id DESC
		 LIMIT ?`,
		[]any{limitSessions},
		func(rows *sql.Rows) error {
			var m sessionUserMsg
			if err := rows.Scan(&m.sessionID, &m.rawText, &m.time); err != nil {
				return err
			}
			msgs = append(msgs, m)
			return nil
		},
	)
	if err != nil {
		return nil
	}

	clusters := clusterRecurringMessages(msgs, minCount)
	ignored := loadIgnoredHabits(db)
	if len(ignored) == 0 {
		return clusters
	}

	var filtered []RecurringRequest
	for _, req := range clusters {
		isIgnored := false
		for ign := range ignored {
			if ign == req.Normalized || areRequestsSimilar(ign, req.Normalized) {
				isIgnored = true
				break
			}
		}
		if !isIgnored {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

// checkFirstTurnRecurrence checks if the current message matches a recurring pattern
// in earlier sessions (excluding currentSessionID).
func checkFirstTurnRecurrence(db *sql.DB, currentSessionID, text string) (count int, representative string) {
	norm := normalizeMessage(text)
	if len(norm) < 3 {
		return 0, ""
	}

	all := detectRecurringRequests(db, 2, 100)
	for _, req := range all {
		// Verify this request is similar
		if areRequestsSimilar(norm, req.Normalized) {
			// Count how many earlier sessions had this request
			priorCount := 0
			for _, sid := range req.SessionIDs {
				if sid != currentSessionID {
					priorCount++
				}
			}
			if priorCount >= 2 {
				return priorCount, req.Text
			}
		}
	}

	return 0, ""
}

// ListRecurringRequests exposes recurring requests to Wails for Settings UI.
// Default minCount is 2 so user can see things asked even twice.
func (a *App) ListRecurringRequests() []RecurringRequest {
	db, err := a.database()
	if err != nil {
		return nil
	}
	return detectRecurringRequests(db, 2, 100)
}

// DismissRecurringRequest records a habit pattern in ignored_habits so it won't appear again.
func (a *App) DismissRecurringRequest(normalized, sampleText string) error {
	db, err := a.database()
	if err != nil {
		return err
	}
	if normalized == "" && sampleText != "" {
		normalized = normalizeMessage(sampleText)
	}
	if normalized == "" {
		return fmt.Errorf("normalized habit pattern is empty")
	}
	_, err = db.Exec(`
		INSERT INTO ignored_habits(normalized, sample_text, ignored_at)
		VALUES (?, ?, ?)
		ON CONFLICT(normalized) DO UPDATE SET ignored_at = excluded.ignored_at, sample_text = excluded.sample_text`,
		normalized, sampleText, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// RestoreRecurringRequest removes a habit pattern from ignored_habits.
func (a *App) RestoreRecurringRequest(normalized string) error {
	db, err := a.database()
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM ignored_habits WHERE normalized = ?`, normalized)
	return err
}
