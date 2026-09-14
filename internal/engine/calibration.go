package engine

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// promptCalibration is what this model's tokenizer has been measured to make
// of Aetox's chars/4 guess: multiply the fixed part of a request (system
// prompt + tool block) by Fixed and the conversation by Var, and the guess
// lands where the provider's count did.
//
// It exists because the forecast on a fresh chat was a number the first reply
// then contradicted — 17.0k "will cost", 13.7k "cost" — and a meter that shrinks
// on contact with reality reads as a meter that was lying. The guess was: chars
// over four is an English rule, the tool block is JSON with the same twenty
// keys repeated, and the system prompt is half Thai. Nobody can count tokens
// here (the tokenizers are the providers' own, and only Anthropic publishes a
// counting endpoint), but every round this model ever answered was recorded
// beside the guess that went out with it, and the ratio between the two IS a
// count — one that has already been paid for.
//
// Two coefficients rather than one because the two parts do not err alike, and
// the first message is nearly all fixed part: a single ratio learned on long
// sessions (mostly conversation) would be the wrong ratio for the number this
// was built to fix.
type promptCalibration struct {
	Fixed, Var float64
	// Rounds is how many measured rounds the coefficients rest on; zero means
	// none and the coefficients are 1 — the bare guess.
	Rounds int
}

// noCalibration is the bare chars/4 guess, unchanged.
var noCalibration = promptCalibration{Fixed: 1, Var: 1}

// apply scales a guess into the calibrated slices. System and Tools share one
// coefficient: nobody reports them apart, and the tool block is the larger of
// the two on every desk.
func (c promptCalibration) apply(e model.PromptEstimate) (system, tools, messages int) {
	round := func(v float64) int { return int(v + 0.5) }
	return round(float64(e.System) * c.Fixed), round(float64(e.Tools) * c.Fixed), round(float64(e.Messages) * c.Var)
}

// calibrationRow is one recorded round: what the provider counted beside what
// Aetox guessed for the same request.
type calibrationRow struct {
	real, fixed, variable int
}

// calibrationWindow is how many recent rounds the fit reads. Enough that one
// odd round cannot swing it; recent, because a tool block that grew last week
// is the one on the wire now.
const calibrationWindow = 200

// promptCalibration reads the model's recent rounds and fits them. Nothing
// recorded — a model never used, a database that will not open — is the bare
// guess, not an error: the meter must draw something either way.
func (a *Engine) promptCalibration(provider, modelName string) promptCalibration {
	if strings.TrimSpace(modelName) == "" {
		return noCalibration
	}
	db, err := a.database()
	if err != nil {
		return noCalibration
	}
	var sample []calibrationRow
	err = eachRow(db, "context meter: calibration rounds",
		`SELECT prompt_tokens, est_fixed_tokens, est_var_tokens
		 FROM token_usage
		 WHERE model = ? AND provider = ? AND est_fixed_tokens IS NOT NULL
		 ORDER BY id DESC LIMIT ?`,
		[]any{modelName, model.NormalizeProvider(provider), calibrationWindow},
		func(rows *sql.Rows) error {
			var r calibrationRow
			if err := rows.Scan(&r.real, &r.fixed, &r.variable); err != nil {
				return err
			}
			sample = append(sample, r)
			return nil
		})
	// A read that broke partway is a sample that is short, and a fit on a
	// short sample is still a fit — but not one to trust over the bare guess.
	if err != nil {
		return noCalibration
	}
	return fitCalibration(sample)
}

// fitCalibration is least squares of real against (fixed, variable) with no
// intercept — a request with nothing in it costs nothing.
//
// The fit degrades honestly. Two coefficients need the sample to vary in both
// directions; a model that has only ever seen first messages (variable ≈ 0
// everywhere) cannot say what conversation costs, and forcing the normal
// equations through would divide by nothing and return anything. Then, and
// with fewer than three rounds, both parts get the one ratio the sample can
// support: total counted over total guessed. Each coefficient is clamped to
// [0.25, 4]: outside that the sample is not a tokenizer's habit but a row that
// recorded something else (a provider that reported uncached tokens only, once).
func fitCalibration(sample []calibrationRow) promptCalibration {
	var sff, svv, sfv, sfr, svr, sumReal, sumGuess, sumVar float64
	n := 0
	for _, r := range sample {
		if r.real <= 0 || r.fixed+r.variable <= 0 {
			continue
		}
		f, v, y := float64(r.fixed), float64(r.variable), float64(r.real)
		sff += f * f
		svv += v * v
		sfv += f * v
		sfr += f * y
		svr += v * y
		sumReal += y
		sumGuess += f + v
		sumVar += v
		n++
	}
	if n == 0 || sumGuess <= 0 {
		return noCalibration
	}
	clamp := func(k float64) float64 {
		switch {
		case k < 0.25:
			return 0.25
		case k > 4:
			return 4
		}
		return k
	}
	single := clamp(sumReal / sumGuess)
	c := promptCalibration{Fixed: single, Var: single, Rounds: n}
	det := sff*svv - sfv*sfv
	// Well-conditioned means two things. The columns are not (nearly) one
	// column: det is sff·svv·(1 − cos²θ) between them, so this asks that they
	// be at least ~6° apart. And the conversation has actually weighed
	// something: rounds where it was a few dozen tokens under a 17k floor are
	// at any angle to the floor and still say nothing about its rate — the
	// provider's own count wobbles by more than that between two identical
	// requests. Below either, one ratio is all the data holds.
	if n < 3 || det <= 0.01*sff*svv || sumVar < 0.05*sumGuess {
		return c
	}
	c.Fixed = clamp((sfr*svv - svr*sfv) / det)
	c.Var = clamp((svr*sff - sfr*sfv) / det)
	return c
}

// floorTimeout bounds one floor measurement. Longer than the connection probe's
// 15 s because this request is the whole fixed part — fourteen thousand tokens
// on a local runtime is prompt processing measured in tens of seconds, and the
// meter would rather wait than record nothing.
const floorTimeout = 90 * time.Second

// MeasureContextFloor sends the current chat's floor — system prompt and tool
// block, one word, one token back — and records what the provider counted as
// the first calibration row for this provider+model. The forecast that reads
// it next is the provider's own number, not chars/4.
//
// Owner, 14 ก.ย. 2026, on being told the count only comes back after a message
// is sent: "มันตรวจได้ครั้งแรกครับ ตรวจแล้วรู้เลยว่าค่ามันเท่าไหร่ แล้วก็เอาค่านั้นมาใช้
// เป็นค่าเริ่มต้นเลย" — the connection test already spends a request; spend it
// on the request that matters. This is that, done for the chat's own model
// the first time its meter is drawn (measureFloorOnce), and callable on its
// own.
//
// It costs what one message's fixed part costs, once per model, and on a
// provider with a prompt cache it is not even lost: the same bytes go out with
// the first real message a moment later, and hit.
func (a *Engine) MeasureContextFloor() (ContextBreakdown, error) {
	conv := a.cur()
	if conv.agent == nil {
		return a.GetContextBreakdown(), errors.New("no chat to measure")
	}
	var defs []model.ToolDefinition
	if conv.registry != nil {
		defs = a.deskTools().ToolDefinitions()
	}
	ctx, cancel := context.WithTimeout(context.Background(), floorTimeout)
	defer cancel()
	// Thinking off, whatever the chat's dial says: the dial is a request
	// parameter and moves no prompt token, and a one-token answer under a
	// thinking budget is a request some providers refuse outright.
	u, err := conv.agent.MeasureFloor(ctx, defs, turn.TurnOptions{ThinkLevel: think.LevelNone})
	if err != nil {
		return a.GetContextBreakdown(), err
	}
	a.storeTokenUsage(conv, u)
	debuglog.Msg("context floor: %s/%s counted %d for a guess of %d (%d rows on record next)",
		conv.cfg.ModelProvider, conv.cfg.ModelName, u.PromptTokens, u.Estimate.Total(), calibrationWindow)
	a.emitEvent("context:measured", conv.id)
	return a.GetContextBreakdown(), nil
}

// measureFloorOnce runs MeasureContextFloor in the background, once per
// provider+model for the life of this process. Tried, not succeeded: a
// provider that is down or unpaid fails the same way on every keystroke, and
// the meter refreshes on every keystroke.
func (a *Engine) measureFloorOnce(conv *conversation) {
	key := model.NormalizeProvider(conv.cfg.ModelProvider) + "\x00" + strings.TrimSpace(conv.cfg.ModelName)
	if strings.TrimSpace(conv.cfg.ModelName) == "" || conv.agent == nil {
		return
	}
	a.floorMu.Lock()
	if a.floorTried == nil {
		a.floorTried = map[string]bool{}
	}
	tried := a.floorTried[key]
	a.floorTried[key] = true
	a.floorMu.Unlock()
	if tried {
		return
	}
	go func() {
		if _, err := a.MeasureContextFloor(); err != nil {
			debuglog.Msg("context floor: %s/%s not measured: %v", conv.cfg.ModelProvider, conv.cfg.ModelName, err)
		}
	}()
}
