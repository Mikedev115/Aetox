package mode

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// A Dial is the stance a running engine is actually filtering through, held
// somewhere both the dispatcher and the assistant can reach it.
//
// **Why this is not just a field.** bootstrap.Engine used to copy
// Options.Stance into a local and close over the copy, which was right while
// the only way to change a stance was to re-bootstrap: the value could not move
// under a running turn because nothing could move it. Letting the assistant
// narrow its own stance breaks that, and it breaks it in the one place
// that matters — the dispatcher's filter runs at request time, so a live value
// there means a tool withheld half way through a turn is refused for the rest
// of it, with no engine rebuilt and no turn interrupted.
//
// What a Dial deliberately does NOT reach is the system prompt, which is built
// once per bootstrap and stays as it was for the turn. That is not a gap being
// tolerated: the stance tool hands the new stance's Direction() back as its own
// result, which lands later in the context than the prompt does and therefore
// outweighs it (§106.4). One mechanism, in the place the model reads next.
type Dial struct {
	mu       sync.RWMutex
	cur      Stance
	onChange func(Stance)
}

// NewDial holds a stance that can be narrowed while the engine runs. onChange
// may be nil; a host passes one when something outside the engine — a picker on
// screen, a row in a database — has a second copy of this fact to keep.
//
// Called with the lock released, because a host's handler re-reads the app and
// a handler that called back in would deadlock the dispatcher mid-turn.
func NewDial(start Stance, onChange func(Stance)) *Dial {
	return &Dial{cur: start, onChange: onChange}
}

// Stance is what this engine is filtering through right now. A nil Dial answers
// ลงมือ, so every caller that has no dial at all reads the stance that withholds
// nothing — the same zero-value reading Stance itself is built on.
func (d *Dial) Stance() Stance {
	if d == nil {
		return StanceAct
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cur
}

// ErrWiden is what a Dial answers when asked to hand something back.
//
// Its own error rather than a message built at the call site, because the
// *caller* that must not be able to fudge this is a tool the model calls: the
// refusal has to read the same every time, and it has to be testable as a fact
// rather than as a sentence.
var ErrWiden = errors.New("a stance may only narrow: switching back is the user's press")

// Narrow moves the dial to a stance that withholds strictly more than the one
// it is on, and refuses anything else.
//
// **The refusal is the feature.** COMPANY.md §6.3 is safe under a moving stance
// for exactly one reason — a stance can only ever subtract, so no context can
// gain a tool it was not opened with. An assistant that could widen its own
// stance would be the first thing in this app able to hand itself a tool, and
// it would be doing it inside a turn the user is already watching run. It would
// also undo the plainest sentence วางแผน says: *the user turned this dial
// deliberately and turning it back is one press* — theirs, not the model's.
//
// Same-stance is refused too, and separately. "Already there" is not an error
// in the world, but it IS a wasted round and a model that cannot see it wasted
// one; saying so plainly is what stops it being repeated.
func (d *Dial) Narrow(to Stance) error {
	if d == nil {
		return errors.New("no stance dial on this engine")
	}
	d.mu.Lock()
	from := d.cur
	if to == from {
		d.mu.Unlock()
		return fmt.Errorf("already in %q", stanceID(from))
	}
	if !from.Narrows(to) {
		d.mu.Unlock()
		return fmt.Errorf("%w (%s → %s)", ErrWiden, stanceID(from), stanceID(to))
	}
	d.cur = to
	onChange := d.onChange
	d.mu.Unlock()
	if onChange != nil {
		onChange(to)
	}
	return nil
}

// Carries and AllowsAction are the two questions the dispatcher asks, forwarded
// to whatever stance the dial is on at the moment of the asking. They are the
// whole reason this type exists — see the note above about request-time
// filtering — and they are methods rather than fields so a caller cannot
// accidentally snapshot one into a closure.
func (d *Dial) Carries(name string, source skill.Source) bool {
	return d.Stance().Carries(name, source)
}

func (d *Dial) AllowsAction(tool, action string) bool {
	return d.Stance().AllowsAction(tool, action)
}
