package engine

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

// The model's own memory of a conversation, written down.
//
// Until 16 ก.ย. 2026 a chat's memory lived in exactly one place: the
// memory.Context of the engine holding it. That engine is let go of the
// moment the user looks at another chat (showConversation), and when they
// came back the memory was rebuilt from the text transcript — the question
// and the answer, and nothing between them. Every tool call, every tool
// result, every summary a compaction had left behind was gone, and the
// comment on LoadSession said so in as many words. Two bills came due at
// once: the model no longer knew what it had read, so it read it again; and
// the request it sent next diverged from the one the provider had cached at
// the first assistant turn that ever called a tool, so the whole history was
// billed as a miss (DeepSeek: ten times the hit price). The owner's screen on
// the day this was written: 19.1k of a 28k window missed on a reopened chat,
// and the model grepping for a file it had found the turn before.
//
// So the memory is written at the end of every turn — everything after the
// system prompt, in the shape the engine holds it — and a reopened chat is
// given that back instead of a transcript. One row per session, replaced
// each time: a context is bounded by the engine's own char budget, and a
// compaction shrinks the row rather than growing it.
//
// The row is only trusted while it still describes the transcript. Both are
// written at the end of a turn, but the transcript can move on its own: an
// edited question drops two rows, a crash leaves an unanswered one that the
// next launch closes. The row remembers which message it was written after,
// and a reader whose transcript ends anywhere else falls back to the
// transcript rebuild — the loss this whole file exists to end, taken once,
// on the row that could not be trusted, rather than a memory of a turn the
// user deleted.
//
// Images and documents do not survive the trip (`json:"-"` on model.Message,
// and each provider spells them differently anyway). Everything a provider
// insists on getting back — tool call ids, Gemini's thought signatures,
// Responses reasoning items — is on the message with a tag, and comes back.

const contextStoreSchema = `
CREATE TABLE IF NOT EXISTS session_context (
  session_id      TEXT PRIMARY KEY,
  last_message_id INTEGER NOT NULL,
  messages        TEXT NOT NULL,
  written_at      TEXT NOT NULL
);`

// storeContext writes down what conv's engine remembers, stamped with the
// last message row of its transcript. Called at the end of every turn, once
// the turn's rows are in the store and after any rebuild endTurn itself
// performed, so what is written is what the next turn would have run on.
//
// Nothing to write is not an error: a conversation with no engine, or one
// whose memory holds only the system prompt, leaves whatever row there was.
func (a *Engine) storeContext(conv *conversation) {
	if conv == nil || conv.agent == nil || conv.id == "" {
		return
	}
	held := conv.agent.ContextMessages()
	if len(held) <= 1 {
		return
	}
	db, err := a.database()
	if err != nil {
		return
	}
	writeContext(db, conv.id, lastMessageID(conv.transcript), held[1:])
}

// writeContext is storeContext without the engine: the row, the stamp and the
// messages, for the store. A memory that fails to encode is not written,
// and the stale row (if any) goes with it — a reader must never be handed a
// memory older than the transcript it is asked to trust it against.
func writeContext(db *sql.DB, sessionID string, lastID int64, messages []model.Message) {
	encoded, err := json.Marshal(messages)
	if err != nil {
		_, _ = db.Exec(`DELETE FROM session_context WHERE session_id = ?`, sessionID)
		return
	}
	_, _ = db.Exec(`
		INSERT INTO session_context(session_id, last_message_id, messages, written_at)
		VALUES(?,?,?,?)
		ON CONFLICT(session_id) DO UPDATE SET
		  last_message_id = excluded.last_message_id,
		  messages        = excluded.messages,
		  written_at      = excluded.written_at`,
		sessionID, lastID, string(encoded), time.Now().Format(time.RFC3339))
}

// historyFor is what a freshly built engine is given as this session's past:
// the memory written at its last turn when that memory still describes the
// transcript, and the transcript itself otherwise. Every caller that used to
// hand RestoreHistory a transcriptToModelMessages goes through here now,
// except the ones that are deliberately rewinding (regenerate.restoreContext)
// — a rewind is a transcript the store has not caught up with yet, and the
// stamp would refuse the row anyway.
func (a *Engine) historyFor(sessionID string, transcript []SessionMessage) []model.Message {
	if held := a.storedContext(sessionID, transcript); held != nil {
		return held
	}
	return transcriptToModelMessages(transcript)
}

// storedContext reads the session's written memory, or nil when there is
// none or it no longer describes the transcript.
//
// Two stamps are accepted. The last row outright is the ordinary case: the
// memory was written at the end of the turn that wrote that row. The last row
// not part of a failed pair covers the turn the app died inside: openTurn had
// stored the question, the next launch closed it with a marker row, and the
// memory written one turn earlier is still exactly what the transcript
// rebuild would produce, since that rebuild skips the failed pair too.
func (a *Engine) storedContext(sessionID string, transcript []SessionMessage) []model.Message {
	if sessionID == "" || len(transcript) == 0 {
		return nil
	}
	db, err := a.database()
	if err != nil {
		return nil
	}
	var stamp int64
	var encoded string
	if db.QueryRow(`SELECT last_message_id, messages FROM session_context WHERE session_id = ?`,
		sessionID).Scan(&stamp, &encoded) != nil {
		return nil
	}
	if stamp != lastMessageID(transcript) && stamp != lastSettledMessageID(transcript) {
		return nil
	}
	var messages []model.Message
	if json.Unmarshal([]byte(encoded), &messages) != nil || len(messages) == 0 {
		return nil
	}
	return messages
}

// lastMessageID is the row id of the transcript's last message; 0 for none.
func lastMessageID(transcript []SessionMessage) int64 {
	if len(transcript) == 0 {
		return 0
	}
	return transcript[len(transcript)-1].ID
}

// lastSettledMessageID is the row id of the last message that is not part of
// a failed pair — the last row the context rebuild would keep; 0 for none.
func lastSettledMessageID(transcript []SessionMessage) int64 {
	for i := len(transcript) - 1; i >= 0; i-- {
		if !failedPairAt(transcript, i) {
			return transcript[i].ID
		}
	}
	return 0
}

// forgetContext drops a session's written memory. The delete paths call it
// inside their own transaction; this is the one-off for callers with a bare
// handle.
func forgetContext(db sqlExecQuerier, sessionID string) error {
	_, err := db.Exec(`DELETE FROM session_context WHERE session_id = ?`, sessionID)
	return err
}
