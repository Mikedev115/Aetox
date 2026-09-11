package rpc

// The numbers §248 promised its record: what the wire costs. Logged, not
// asserted — a machine under load would fail a threshold for no reason of
// the code's — and read into the design doc §10 by hand.

import (
	"context"
	"encoding/json"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func percentiles(d []time.Duration) (p50, p95, max time.Duration) {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)/2], d[len(d)*95/100], d[len(d)-1]
}

func TestMeasureTheWire(t *testing.T) {
	_, c, log := wired(t)

	// A binding's round trip: the smallest call there is, two thousand times.
	const n = 2000
	took := make([]time.Duration, 0, n)
	begin := time.Now()
	for i := 0; i < n; i++ {
		start := time.Now()
		if v := c.AppVersion(); v == "" {
			t.Fatal("AppVersion answered empty")
		}
		took = append(took, time.Since(start))
	}
	total := time.Since(begin)
	p50, p95, max := percentiles(took)
	// The mean off the whole run, because Windows' clock is coarser than one
	// call and a p50 of 0s says only that.
	t.Logf("round trip, loopback tcp, %d calls: mean %s  p50 %s  p95 %s  max %s", n, total/n, p50, p95, max)

	// A turn on the test model, in-process against over the wire, on the same
	// engine. The model is a fixture, so the difference is the wire's own:
	// the call, and every event crossing as a frame.
	if _, err := c.OpenProjectPath(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SwitchProvider("aetox"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SwitchModel("aetox-render:test"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SwitchApprovalMode("full-access"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.NewSession(); err != nil {
		t.Fatal(err)
	}
	// Once each way to warm, then measured.
	srv := serverOf(c)
	if _, err := srv.Engine().SendMessage("warm", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SendMessage("warm", ""); err != nil {
		t.Fatal(err)
	}
	var direct, wire []time.Duration
	var frames atomic.Int64
	var bytes atomic.Int64
	for i := 0; i < 5; i++ {
		start := time.Now()
		if _, err := srv.Engine().SendMessage("เทสๆ", ""); err != nil {
			t.Fatal(err)
		}
		direct = append(direct, time.Since(start))

		before := len(log.all())
		start = time.Now()
		if _, err := c.SendMessage("เทสๆ", ""); err != nil {
			t.Fatal(err)
		}
		wire = append(wire, time.Since(start))
		events := log.all()[before:]
		frames.Add(int64(len(events)))
		for _, e := range events {
			b, _ := json.Marshal(e)
			bytes.Add(int64(len(b)))
		}
	}
	d50, _, _ := percentiles(direct)
	w50, _, _ := percentiles(wire)
	t.Logf("turn on aetox-render:test, 5 each: in-process p50 %s, over the wire p50 %s, delta %s", d50, w50, w50-d50)
	t.Logf("events per wire turn: %d frames, %d bytes", frames.Load()/5, bytes.Load()/5)
	_ = context.Background
}

// serverOf is the test's way back from a client to the server wired() made:
// wired keeps them paired in a table because the client cannot know.
func serverOf(c *Client) *Server {
	pairsMu.Lock()
	defer pairsMu.Unlock()
	return pairs[c]
}
