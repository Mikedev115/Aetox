package model

// What a provider ACTUALLY charged, solved for from money that actually moved.
//
// The published table is wrong in two different ways and this answers both.
// It is wrong by omission — 933 of the 7,443 rows in models.dev carry no input
// price at all — and it is wrong by nature for any provider whose price is not
// a constant: DeepSeek discounts off-peak hours, so `deepseek-v4-pro` has no
// single number to be right about, and a table that quotes one is wrong twice a
// day (owner, 7 ก.ย.: "บางเวลาราคามันไม่เท่ากันผมเลยคิดว่าควรทำเอง").
//
// **Nothing here costs a request.** The app already reads a provider's balance
// when the user opens its card. Two of those readings are the ends of a window;
// the usage database already knows every token spent between them. So an
// observation is a subtraction over data the app was collecting anyway, and the
// price falls out of enough of them.
//
// **Three rates, not one factor.** A single correction factor on the table
// would have been a third of this code and would have folded a cache-read
// discount, an output premium and an off-peak cut into one number that explains
// none of them. Input, cached input and output are priced separately by every
// vendor here, so they are solved for separately.
//
// The hard part is not the algebra, it is knowing when to REFUSE. Ordinary chat
// moves the three token counts together — a long prompt tends to get a long
// answer — and columns that move together cannot be told apart no matter how
// many samples arrive. A fit taken from such a window set is three confident
// numbers built on nothing, which is worse than the table it replaced. So the
// solver is asked for its own condition, and a poorly-conditioned system is
// declined rather than rounded off.

import (
	"math"
)

// Observation is one window between two balance readings: what the account was
// charged, and what it bought.
type Observation struct {
	// Token counts over the window, already split the way ModelPrice.Cost
	// wants them — cached input is not a subset of uncached input.
	UncachedInput, CachedInput, Output int64
	// Spent is the money the balance actually moved over the window, positive.
	// A top-up in the middle makes this meaningless, which is the caller's
	// problem to detect (a negative or absurd delta is dropped, not fitted).
	Spent float64
}

// Fit is what a solve produced, and how much it should be believed.
type Fit struct {
	Price ModelPrice
	// Samples is how many observations went in.
	Samples int
	// Residual is the mean absolute error between what the fitted rates predict
	// for each window and what that window actually cost, in money. Small
	// against the windows' own size means the three rates explain the spending.
	Residual float64
	// Spread is the smallest pivot the elimination met, relative to the
	// largest. It is the "can these columns be told apart" number: near zero
	// means the token counts moved together and the split between them is
	// guesswork. Reported so a caller can be stricter than the floor below.
	Spread float64
}

// Minimums a fit must clear to be offered at all.
//
// Four samples for three unknowns: three would fit exactly, residual zero, and
// say nothing about whether the answer is right — an exact fit through the
// minimum number of points is a restatement of the points.
const (
	minFitSamples = 4
	// Below this the columns are not independent enough to be split. Measured
	// against the alternative: a wrong split is asserted with the same
	// confidence as a right one, and nothing downstream can tell them apart.
	minFitSpread = 1e-3
)

// FitPrice solves for the input, cached-input and output rates that best
// explain a set of windows, in the same per-million units ModelPrice uses.
//
// The bool is false when the observations cannot answer the question: too few,
// all of one shape, or a solve that came back with the columns indistinguishable.
// A false here means "keep using the table", never "the price is zero".
func FitPrice(obs []Observation) (Fit, bool) {
	usable := make([]Observation, 0, len(obs))
	for _, o := range obs {
		// A window that spent nothing, or that spans a top-up, teaches nothing.
		// Negative spending is a top-up; zero is either rounding or an idle
		// window, and both are silent about rates.
		if o.Spent <= 0 {
			continue
		}
		if o.UncachedInput <= 0 && o.CachedInput <= 0 && o.Output <= 0 {
			continue
		}
		usable = append(usable, o)
	}
	if len(usable) < minFitSamples {
		return Fit{}, false
	}

	// Normal equations for the least-squares problem, in per-million units so
	// the three unknowns come out at the scale ModelPrice is written in and the
	// matrix is not built out of numbers a million apart.
	var ata [3][3]float64
	var atb [3]float64
	for _, o := range usable {
		row := [3]float64{
			float64(o.UncachedInput) / 1e6,
			float64(o.CachedInput) / 1e6,
			float64(o.Output) / 1e6,
		}
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				ata[i][j] += row[i] * row[j]
			}
			atb[i] += row[i] * o.Spent
		}
	}

	solution, spread, ok := solve3(ata, atb)
	if !ok || spread < minFitSpread {
		return Fit{}, false
	}

	// A negative rate is not a discount, it is the fit absorbing noise into a
	// column that barely varied. Pinned to zero rather than reported: no vendor
	// pays a user to send tokens, and a negative in a price table would flow
	// straight into a negative cost on somebody's usage page.
	for i := range solution {
		if solution[i] < 0 {
			solution[i] = 0
		}
	}

	price := ModelPrice{Input: solution[0], CacheRead: solution[1], Output: solution[2]}
	return Fit{
		Price:    price,
		Samples:  len(usable),
		Residual: meanAbsError(usable, price),
		Spread:   spread,
	}, true
}

// meanAbsError is what the fitted rates get wrong per window, in money.
func meanAbsError(obs []Observation, p ModelPrice) float64 {
	if len(obs) == 0 {
		return 0
	}
	total := 0.0
	for _, o := range obs {
		// Cost's own fallback — cache reads billed as input when no cache rate
		// is published — must not apply here: a fitted zero means the solve
		// found no cache cost, which is an answer, not an absence.
		predicted := float64(o.UncachedInput)/1e6*p.Input +
			float64(o.CachedInput)/1e6*p.CacheRead +
			float64(o.Output)/1e6*p.Output
		total += math.Abs(predicted - o.Spent)
	}
	return total / float64(len(obs))
}

// solve3 solves a symmetric 3x3 system by Gaussian elimination with partial
// pivoting, and reports how well-separated the columns were.
//
// Spread is the smallest pivot over the largest, taken during elimination. It
// is not a true condition number and does not need to be: what a caller has to
// know is whether the three columns were independent enough to be split, and a
// pivot that collapsed says they were not.
func solve3(a [3][3]float64, b [3]float64) ([3]float64, float64, bool) {
	m := [3][4]float64{}
	for i := 0; i < 3; i++ {
		copy(m[i][:3], a[i][:])
		m[i][3] = b[i]
	}

	largest, smallest := 0.0, math.Inf(1)
	for col := 0; col < 3; col++ {
		pivot := col
		for r := col + 1; r < 3; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[pivot][col]) {
				pivot = r
			}
		}
		m[col], m[pivot] = m[pivot], m[col]

		p := math.Abs(m[col][col])
		if p == 0 || math.IsNaN(p) {
			return [3]float64{}, 0, false
		}
		largest = math.Max(largest, p)
		smallest = math.Min(smallest, p)

		for r := col + 1; r < 3; r++ {
			factor := m[r][col] / m[col][col]
			for c := col; c < 4; c++ {
				m[r][c] -= factor * m[col][c]
			}
		}
	}

	var x [3]float64
	for i := 2; i >= 0; i-- {
		sum := m[i][3]
		for j := i + 1; j < 3; j++ {
			sum -= m[i][j] * x[j]
		}
		x[i] = sum / m[i][i]
	}
	for _, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return [3]float64{}, 0, false
		}
	}

	spread := 0.0
	if largest > 0 {
		spread = smallest / largest
	}
	return x, spread, true
}
