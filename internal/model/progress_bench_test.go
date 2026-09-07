package model

import (
	"fmt"
	"strings"
	"testing"
)

// The accumulator sees every streamed fragment of a tool call, and everything
// it does with one touches the arguments accumulated *so far* — which grow with
// the file being written. Two separate quadratics lived here, and this is the
// benchmark that found both: the helpers re-scanning the whole buffer per
// fragment, fixed by pacing them on the clock in toolProgressTracker.report,
// and the buffer itself being re-copied per fragment, fixed by the Builder in
// streamToolAccumulator. Together they took an 800-line write from 2.7ms and
// 18MB to 68us and 215KB. It stays on the hot path, so it stays measured.
func BenchmarkStreamAccumulatorLargeWrite(b *testing.B) {
	// An 800-line HTML file, arriving the way a model streams one.
	var frags []string
	frags = append(frags, `{"path": "landing.html", "content": "`)
	for i := 0; i < 800; i++ {
		frags = append(frags, fmt.Sprintf(`  <div class=\"row-%d\">line of content here</div>\n`, i))
	}
	frags = append(frags, `"}`)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		acc := newStreamToolAccumulator(func(string, string, string, int) {})
		for _, f := range frags {
			acc.add([]streamToolCallDelta{{Index: 0, Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{Name: "write", Arguments: f}}})
		}
	}
}

// The same 800-line file with the arguments in the other order. When "content"
// arrives first the subject stays unresolved for the whole write, so the regex
// keeps looking instead of matching once and stopping. Argument order is the
// model's choice, so this ordering is not hypothetical — it used to cost 8x the
// other one, and the point of keeping both benchmarks is that the gap between
// them is what a re-scan per fragment looks like when it comes back.
func BenchmarkStreamAccumulatorLargeWriteContentFirst(b *testing.B) {
	var frags []string
	frags = append(frags, `{"content": "`)
	for i := 0; i < 800; i++ {
		frags = append(frags, fmt.Sprintf(`  <div class=\"row-%d\">line of content here</div>\n`, i))
	}
	frags = append(frags, `", "path": "landing.html"}`)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		acc := newStreamToolAccumulator(func(string, string, string, int) {})
		for _, f := range frags {
			acc.add([]streamToolCallDelta{{Index: 0, Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{Name: "write", Arguments: f}}})
		}
	}
}

func BenchmarkSubjectFromPartialArgs(b *testing.B) {
	args := `{"path": "internal/skill/edit.go", "content": "` + strings.Repeat(`x\n`, 5000)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		SubjectFromPartialArgs(args)
	}
}
