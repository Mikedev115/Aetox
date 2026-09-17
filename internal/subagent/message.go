package subagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// task_message is soft steering for work already in flight. It does not answer
// ask_main and it does not restart the delegate; it becomes input at the
// child's next model boundary with all of that child's context intact.
type taskMessageTool struct{ runner *Delegations }

func (t *taskMessageTool) Name() string { return "task_message" }

func (t *taskMessageTool) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	started := time.Now()
	id := strings.TrimSpace(stringArg(args, "task_id"))
	message := strings.TrimSpace(stringArg(args, "message"))
	if id == "" {
		return t.fail(started, "task_id is required — it is the id of the running sub-agent")
	}
	if message == "" {
		return t.fail(started, "message is required — send only the update that changes this worker's job")
	}
	if err := t.runner.message(id, message); err != nil {
		return t.fail(started, err.Error())
	}
	content := fmt.Sprintf("sent an update to %s — it will read it at its next model boundary. The task keeps running; collect it when you are ready.", id)
	return skill.Output{
		Name:       t.Name(),
		Command:    "task_message " + id,
		Content:    content,
		RawOutput:  content,
		Success:    true,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func (t *taskMessageTool) fail(started time.Time, reason string) (skill.Output, error) {
	if outstanding := t.runner.running(); len(outstanding) > 0 {
		ids := make([]string, 0, len(outstanding))
		for _, task := range outstanding {
			ids = append(ids, fmt.Sprintf("%s (%s)", task.id, task.profile))
		}
		reason += "\nstill running: " + strings.Join(ids, ", ")
	}
	return skill.Output{
		Name:       t.Name(),
		Command:    "task_message",
		Content:    reason,
		RawOutput:  reason,
		Stderr:     reason,
		Success:    false,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}
