package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// opencodeGoLimitReset asks the gateway when its spent window refills.
//
// Only for opencode-go, and only after a 429: the gateway is the one host
// whose limit is stated nowhere on the refusing reply (balance.go says why),
// so the fact has to be fetched. The request goes through the provider's own
// client and applyAuth — the signing transport, the configured headers, the
// key — so it is authorised exactly the way the turn that was refused was.
//
// The window that refused the turn is the one marked rate-limited; the
// earliest reset among those is when a request could work again. A window
// that still has room did not refuse anything and is skipped, so a spent
// five-hour window does not get answered with the weekly reset. Anything
// missing or malformed reports false, and the caller falls back to the
// sentence it always had — a limit this cannot date is a limit it does not
// wait on.
func (p *OpenAICompatibleProvider) opencodeGoLimitReset(ctx context.Context) (time.Time, bool) {
	if NormalizeProvider(p.provider) != "opencode-go" {
		return time.Time{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, opencodeGoUsageTimeout)
	defer cancel()
	// /usage sits beside /chat/completions under the same prefix, which is
	// why it hangs off the base URL rather than the origin (see
	// fetchOpencodeGoUsage).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.baseURL, "/")+"/usage", nil)
	if err != nil {
		return time.Time{}, false
	}
	if err := p.applyAuth(ctx, req); err != nil {
		return time.Time{}, false
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return time.Time{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return time.Time{}, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return time.Time{}, false
	}
	var payload opencodeGoUsage
	if json.Unmarshal(body, &payload) != nil {
		return time.Time{}, false
	}
	return opencodeGoLimitedReset(payload, time.Now())
}

// opencodeGoUsageTimeout bounds the one extra round trip. The turn is already
// refused; a usage endpoint that hangs must not hold the refusal hostage.
const opencodeGoUsageTimeout = 10 * time.Second

// opencodeGoLimitedReset is the earliest stated reset among the windows the
// gateway marks rate-limited. False when none is, or none of those is dated.
func opencodeGoLimitedReset(payload opencodeGoUsage, now time.Time) (time.Time, bool) {
	var earliest time.Time
	for _, stated := range payload.Usage {
		if stated.Status != "rate-limited" {
			continue
		}
		resetAt := parseResetInstant(stated.ResetsAt, now)
		if resetAt.IsZero() || !resetAt.After(now) {
			continue
		}
		if earliest.IsZero() || resetAt.Before(earliest) {
			earliest = resetAt
		}
	}
	return earliest, !earliest.IsZero()
}
