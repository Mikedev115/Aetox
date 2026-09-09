package main

import (
	"database/sql"
	"regexp"
	"sort"
	"strings"
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
var attachmentPattern = regexp.MustCompile(`\[Attached (?:file|image): [^\]]+\]`)

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

	// 3. Remove punctuation and control characters
	var b strings.Builder
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	text = b.String()

	// 4. Collapse spaces
	fields := strings.Fields(text)
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
	minRunes := len(runesA)
	if len(runesB) < minRunes {
		minRunes = len(runesB)
	}

	if minRunes == 0 {
		return false
	}

	// If one contains the other
	if strings.Contains(a, b) || strings.Contains(b, a) {
		if minRunes >= 3 {
			return true
		}
	}

	// Check longest common contiguous substring as a proportion of the shorter text
	lcs := longestCommonSubstringRunes(runesA, runesB)
	if minRunes >= 6 && float64(lcs)/float64(minRunes) >= 0.55 && lcs >= 6 {
		return true
	}

	// Word/token overlap check for multi-word phrases
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) > 1 && len(wordsB) > 1 {
		common := 0
		setB := make(map[string]bool, len(wordsB))
		for _, w := range wordsB {
			setB[w] = true
		}
		for _, w := range wordsA {
			if setB[w] {
				common++
			}
		}
		if common >= 2 || (float64(common)/float64(len(wordsA)) >= 0.5 && common > 0) {
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

// detectRecurringRequests queries SQLite for the first user message of recent sessions
// and returns clusters that cross minCount.
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

	return clusterRecurringMessages(msgs, minCount)
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
