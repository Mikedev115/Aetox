package engine

import (
	"math"
	"testing"

	"github.com/Mikedev115/Aetox/internal/cognitive"
	"github.com/Mikedev115/Aetox/internal/model"
)

func near(got, want, tol float64) bool { return math.Abs(got-want) <= tol }

// The fit recovers two different tokenizer habits from rounds where both
// parts vary — the shape a few sessions of one model leave behind.
func TestFitCalibrationSeparatesFixedFromConversation(t *testing.T) {
	// The provider packs the tool block at 0.8 of the guess and Thai prose at
	// 1.5 of it. Two sessions with different tool blocks, growing histories.
	var sample []calibrationRow
	for _, fixed := range []int{16_000, 12_000} {
		for _, v := range []int{100, 900, 2_500, 6_000, 14_000} {
			sample = append(sample, calibrationRow{
				real: int(0.8*float64(fixed) + 1.5*float64(v)), fixed: fixed, variable: v,
			})
		}
	}
	c := fitCalibration(sample)
	if c.Rounds != len(sample) {
		t.Errorf("Rounds = %d, want %d", c.Rounds, len(sample))
	}
	if !near(c.Fixed, 0.8, 0.01) || !near(c.Var, 1.5, 0.01) {
		t.Errorf("fit = (%.3f, %.3f), want (0.8, 1.5)", c.Fixed, c.Var)
	}
	// What the meter shows for a fresh chat on that model: the fixed part at
	// the tokenizer's rate, not chars/4.
	system, tools, messages := c.apply(model.PromptEstimate{System: 9_800, Tools: 7_200, Messages: 50})
	if system != 7_840 || tools != 5_760 || messages != 75 {
		t.Errorf("apply = %d/%d/%d, want 7840/5760/75", system, tools, messages)
	}
}

// A model that has only ever seen first messages cannot say what a
// conversation costs; the fit says one ratio, not two invented ones.
func TestFitCalibrationFallsBackToOneRatio(t *testing.T) {
	sample := []calibrationRow{
		{real: 13_700, fixed: 17_000, variable: 0},
		{real: 13_720, fixed: 17_000, variable: 20},
		{real: 13_690, fixed: 17_000, variable: 10},
		{real: 13_700, fixed: 17_000, variable: 0},
	}
	c := fitCalibration(sample)
	want := (13_700.0 + 13_720 + 13_690 + 13_700) / (17_000.0*4 + 30)
	if !near(c.Fixed, want, 0.001) || c.Fixed != c.Var {
		t.Errorf("degenerate fit = (%.4f, %.4f), want both %.4f", c.Fixed, c.Var, want)
	}
	if c.Rounds != 4 {
		t.Errorf("Rounds = %d, want 4", c.Rounds)
	}

	// Two rounds is one ratio too, however well they differ.
	c = fitCalibration(sample[:2])
	if c.Fixed != c.Var {
		t.Errorf("two rounds fit two coefficients (%.3f, %.3f); one ratio is all they hold", c.Fixed, c.Var)
	}
}

// Nothing recorded, or nothing usable, is the bare guess — never a divide by
// zero and never a coefficient the meter would draw as an empty request.
func TestFitCalibrationOnNothing(t *testing.T) {
	for name, sample := range map[string][]calibrationRow{
		"empty":      nil,
		"zero real":  {{real: 0, fixed: 100, variable: 10}},
		"zero guess": {{real: 100, fixed: 0, variable: 0}},
	} {
		if c := fitCalibration(sample); c != noCalibration {
			t.Errorf("%s: fit = %+v, want the bare guess", name, c)
		}
	}
	// A row recording something else entirely (uncached-only tokens, once)
	// is held to the clamp rather than believed.
	c := fitCalibration([]calibrationRow{{real: 10, fixed: 10_000, variable: 0}})
	if c.Fixed != 0.25 {
		t.Errorf("Fixed = %.3f on a 0.001 ratio, want the 0.25 floor", c.Fixed)
	}
}

// The bug as reported on 14 ก.ย. 2026, end to end: a fresh chat forecast
// 17.0k, the first reply measured 13.7k, and the meter that had said 17.0k
// looked broken. After this model's rounds are on record, the forecast for the
// next fresh chat is the number the provider will count, and the panel can
// say how many rounds it rests on.
func TestGetContextBreakdownForecastsFromMeasuredRounds(t *testing.T) {
	const systemPrompt = "you are a test system prompt, long enough to weigh something on the meter"
	fresh := func(a *Engine, id string) {
		agent := cognitive.NewAgent(cognitive.AgentConfig{SystemPrompt: systemPrompt})
		conv := &conversation{agent: agent, id: id}
		conv.cfg = a.cfg
		conv.cfg.ModelProvider = "deepseek"
		conv.cfg.ModelName = "deepseek-v4-flash"
		seed(a, conv)
	}
	a := &Engine{dbDir: t.TempDir()}
	fresh(a, "s1")
	t.Cleanup(func() {
		if a.db != nil {
			_ = a.db.Close()
		}
	})

	before := a.GetContextBreakdown()
	if before.Measured || before.CalibratedRounds != 0 {
		t.Fatalf("fresh chat on an unseen model: Measured=%v CalibratedRounds=%d, want the bare guess", before.Measured, before.CalibratedRounds)
	}
	guess := before.UsedTokens
	if guess <= 0 {
		t.Fatal("expected a non-zero guess from the system prompt")
	}

	// Four rounds of one session: the provider counts the fixed part at 0.8
	// of the guess and the conversation at 1.5, and the agent stamped each
	// round with the guess it went out with.
	for _, v := range []int{20, 300, 900, 2_000} {
		est := model.PromptEstimate{System: guess, Tools: 0, Messages: v}
		real := int(0.8*float64(guess) + 1.5*float64(v))
		a.recordTokenUsage(a.cur(), model.Usage{PromptTokens: real, CompletionTokens: 5, Estimate: est})
	}
	// A title request on the same session: counted, but stamped with no guess,
	// and so not in the fit.
	a.recordTokenUsage(a.cur(), model.Usage{PromptTokens: 999, CompletionTokens: 2})

	fresh(a, "s2")
	after := a.GetContextBreakdown()
	if after.Measured {
		t.Fatal("a fresh chat has sent nothing; Measured must be false")
	}
	if after.CalibratedRounds != 4 {
		t.Errorf("CalibratedRounds = %d, want the 4 stamped rounds (the title round has no guess)", after.CalibratedRounds)
	}
	want := int(0.8*float64(guess) + 0.5)
	if !near(float64(after.UsedTokens), float64(want), 2) {
		t.Errorf("forecast = %d, want the provider's rate %d (bare guess was %d)", after.UsedTokens, want, guess)
	}

	// Another model has its own tokenizer: nothing learned here applies.
	a.cur().cfg.ModelName = "deepseek-v4-pro"
	if other := a.GetContextBreakdown(); other.CalibratedRounds != 0 || other.UsedTokens != guess {
		t.Errorf("other model: CalibratedRounds=%d UsedTokens=%d, want 0 and the bare guess %d", other.CalibratedRounds, other.UsedTokens, guess)
	}
}
