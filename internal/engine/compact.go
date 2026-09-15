package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const compactWait = 2 * time.Minute

// CompactSession triggers manual on-demand compaction on the given session:
// it micro-sweeps stale tool outputs past the recent tail and summarizes older
// conversation turns if eligible. Returns the updated ContextBreakdown.
func (a *Engine) CompactSession(sessionID string) (ContextBreakdown, error) {
	conv := a.cur()
	if id := strings.TrimSpace(sessionID); id != "" && id != conv.id {
		conv = a.liveConversation(id)
	}
	if conv == nil || conv.id == "" {
		return a.GetContextBreakdown(), fmt.Errorf("เปิดแชทนั้นก่อนแล้วค่อยคอมแพ็ค")
	}
	if a.turnRunningIn(conv.id) {
		return a.GetContextBreakdown(), fmt.Errorf("รอให้คำตอบนี้จบก่อน")
	}
	if conv.agent == nil {
		return a.GetContextBreakdown(), fmt.Errorf("ยังไม่มีโมเดล")
	}

	ctx, cancel := context.WithTimeout(context.Background(), compactWait)
	defer cancel()

	if _, _, err := conv.agent.CompactContext(ctx); err != nil {
		return a.GetContextBreakdown(), fmt.Errorf("บีบอัดคอนเท็กซ์ไม่สำเร็จ: %w", err)
	}

	return a.GetContextBreakdown(), nil
}
