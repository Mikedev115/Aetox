package model

import (
	"math"
	"math/rand"
	"testing"
)

// spend prices a window at known rates, the way the vendor would.
func spend(o Observation, in, cache, out float64) float64 {
	return float64(o.UncachedInput)/1e6*in +
		float64(o.CachedInput)/1e6*cache +
		float64(o.Output)/1e6*out
}

// windows builds a set whose token counts vary independently — which is what
// makes the three rates separable, and what a real account produces once it has
// done a few different kinds of work.
func windows(n int, seed int64) []Observation {
	rng := rand.New(rand.NewSource(seed))
	out := make([]Observation, n)
	for i := range out {
		out[i] = Observation{
			UncachedInput: int64(rng.Intn(900_000) + 50_000),
			CachedInput:   int64(rng.Intn(700_000) + 10_000),
			Output:        int64(rng.Intn(300_000) + 5_000),
		}
	}
	return out
}

func TestItRecoversTheRatesItWasChargedAt(t *testing.T) {
	const in, cache, out = 0.55, 0.07, 2.19
	obs := windows(12, 1)
	for i := range obs {
		obs[i].Spent = spend(obs[i], in, cache, out)
	}

	fit, ok := FitPrice(obs)
	if !ok {
		t.Fatal("refused a set it should have solved")
	}
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"input", fit.Price.Input, in},
		{"cacheRead", fit.Price.CacheRead, cache},
		{"output", fit.Price.Output, out},
	} {
		if math.Abs(c.got-c.want) > 1e-6 {
			t.Errorf("%s = %g, want %g", c.name, c.got, c.want)
		}
	}
	if fit.Residual > 1e-9 {
		t.Errorf("residual %g on exact data", fit.Residual)
	}
}

// The case the whole file exists for: the same model, charged at half rate for
// part of the day. Fed only the discounted windows, the fit must come back with
// the discounted rates rather than the table's.
func TestItFindsTheOffPeakRateWhenThatIsWhatWasCharged(t *testing.T) {
	const in, cache, out = 0.135, 0.0135, 0.55
	obs := windows(10, 7)
	for i := range obs {
		obs[i].Spent = spend(obs[i], in, cache, out)
	}
	fit, ok := FitPrice(obs)
	if !ok {
		t.Fatal("refused the off-peak set")
	}
	if math.Abs(fit.Price.Input-in) > 1e-6 || math.Abs(fit.Price.Output-out) > 1e-6 {
		t.Errorf("got in=%g out=%g, want in=%g out=%g", fit.Price.Input, fit.Price.Output, in, out)
	}
}

func TestItSurvivesTheRoundingARealBalanceHas(t *testing.T) {
	const in, cache, out = 0.28, 0.028, 1.1
	obs := windows(40, 3)
	for i := range obs {
		// A balance reported to two decimals: every window's spend arrives
		// rounded, which is exactly what a real reading does.
		obs[i].Spent = math.Round(spend(obs[i], in, cache, out)*100) / 100
	}
	fit, ok := FitPrice(obs)
	if !ok {
		t.Fatal("refused a realistic set")
	}
	// Loose on purpose: the point is that rounding does not move the answer far
	// enough to mislead, not that it does not move it at all.
	if math.Abs(fit.Price.Input-in) > 0.05 || math.Abs(fit.Price.Output-out) > 0.05 {
		t.Errorf("rounding moved the fit too far: in=%g out=%g", fit.Price.Input, fit.Price.Output)
	}
}

func TestItRefusesWhenTheColumnsMoveTogether(t *testing.T) {
	// Every window the same shape, only bigger. Three rates cannot be told
	// apart from this no matter how many samples arrive — any split that adds
	// up explains the data equally well, and picking one would be an invention.
	obs := make([]Observation, 15)
	for i := range obs {
		k := int64(i + 1)
		obs[i] = Observation{UncachedInput: 100_000 * k, CachedInput: 50_000 * k, Output: 20_000 * k}
		obs[i].Spent = spend(obs[i], 0.3, 0.03, 1.2)
	}
	if fit, ok := FitPrice(obs); ok {
		t.Errorf("fitted an unanswerable set: %+v", fit)
	}
}

func TestItRefusesTooFewWindows(t *testing.T) {
	obs := windows(3, 11)
	for i := range obs {
		obs[i].Spent = spend(obs[i], 0.4, 0.04, 1.5)
	}
	// Three points and three unknowns fit exactly and prove nothing.
	if _, ok := FitPrice(obs); ok {
		t.Error("fitted three windows onto three unknowns")
	}
}

func TestWindowsThatTeachNothingAreDropped(t *testing.T) {
	good := windows(6, 5)
	for i := range good {
		good[i].Spent = spend(good[i], 0.5, 0.05, 2)
	}
	// A top-up (balance went UP), an idle window, and a window with no tokens.
	noise := []Observation{
		{UncachedInput: 100_000, Output: 10_000, Spent: -4.20},
		{UncachedInput: 100_000, Output: 10_000, Spent: 0},
		{Spent: 1.5},
	}
	fit, ok := FitPrice(append(noise, good...))
	if !ok {
		t.Fatal("the noise took the set down with it")
	}
	if fit.Samples != len(good) {
		t.Errorf("samples = %d, want %d — the unusable windows were counted", fit.Samples, len(good))
	}
	if math.Abs(fit.Price.Input-0.5) > 1e-6 {
		t.Errorf("input = %g, want 0.5", fit.Price.Input)
	}
}

func TestANegativeRateIsNeverReported(t *testing.T) {
	// Cache reads that are genuinely free, plus noise. The column varies enough
	// to be solvable, so the fit is not declined — and the true rate sitting at
	// zero is exactly where noise pushes an estimate below it. No vendor pays a
	// user to send tokens, and a negative here becomes a negative on a usage
	// page, so it is pinned rather than reported.
	rng := rand.New(rand.NewSource(99))
	obs := windows(60, 21)
	sawFit := false
	for _, scale := range []float64{0.02, 0.08, 0.2} {
		for i := range obs {
			obs[i].Spent = spend(obs[i], 0.4, 0, 1.6) + (rng.Float64()-0.5)*scale
		}
		fit, ok := FitPrice(obs)
		if !ok {
			continue
		}
		sawFit = true
		if fit.Price.Input < 0 || fit.Price.CacheRead < 0 || fit.Price.Output < 0 {
			t.Errorf("noise %g produced a negative rate: %+v", scale, fit.Price)
		}
	}
	if !sawFit {
		t.Fatal("every noise level was declined — the clamp was never exercised")
	}
}
