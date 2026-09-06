package turn

// Which delegation a `task` call is redeeming, on its way to the UI.
//
// The window draws a delegate as a card with a face, a brief and a clock, and
// draws the agent's wait for it as a row of its own. Those are two objects
// describing one worker, and until this id was stamped there was nothing
// joining them: the row read "รอผลงาน · 13s" above a card that showed no time
// at all, because a `task` START returns the instant the worker is spawned and
// its own seconds are therefore always about zero. The thirteen seconds were
// the delegate's, on the wrong object, next to the right one.

import "testing"

func TestCollectedTaskNamesTheDelegationBeingRedeemed(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
		args string
		want string
	}{
		{"a collect", "task", `{"action":"collect","task_id":"task_1"}`, "task_1"},
		// Unsticking a parked delegate is about one delegation too, and the
		// window joins it the same way.
		{"an answer", "task", `{"action":"answer","task_id":"task_7","text":"yes"}`, "task_7"},
		{"shouted and padded", "task", `{"action":" Collect ","task_id":" task_2 "}`, "task_2"},
		// A start HAS no id yet — the register mints one and hands it back in
		// the result. Stamping the argument of a start would put a worker's name
		// on the row that hired them.
		{"a start", "task", `{"action":"start","agent":"doc","prompt":"go"}`, ""},
		{"a start with no action key at all", "task", `{"agent":"doc"}`, ""},
		// One call may redeem several. A row claiming to be about one of them is
		// worse than a row claiming nothing: the seconds it took belong to no
		// single card.
		{"several at once", "task", `{"action":"collect","task_id":"task_1,task_2"}`, ""},
		{"nothing to redeem", "task", `{"action":"collect"}`, ""},
		{"not a delegation tool", "read", `{"action":"collect","task_id":"task_1"}`, ""},
		{"arguments that are not JSON", "task", `collect task_1`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := collectedTask(tc.tool, tc.args); got != tc.want {
				t.Fatalf("collectedTask(%q, %q) = %q, want %q", tc.tool, tc.args, got, tc.want)
			}
		})
	}
}

// The two readers of a `task` call must not disagree about what it is. A row
// that is stamped with a delegation id AND reported as a delegation would draw
// a second card for a worker nobody hired, beside the real one it just redeemed.
func TestACollectIsNeverAlsoReportedAsHiringAnybody(t *testing.T) {
	const args = `{"action":"collect","task_id":"task_1"}`
	if _, _, isTask := delegationOf("task", args); isTask {
		t.Fatal("a collect reported itself as a delegation")
	}
	if collectedTask("task", args) == "" {
		t.Fatal("a collect carried no delegation id to join it to its card")
	}
	const start = `{"action":"start","agent":"doc","prompt":"go"}`
	if _, _, isTask := delegationOf("task", start); !isTask {
		t.Fatal("a start stopped reporting itself as a delegation")
	}
	if id := collectedTask("task", start); id != "" {
		t.Fatalf("a start carried a delegation id %q before the register had minted one", id)
	}
}
