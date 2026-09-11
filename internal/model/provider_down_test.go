package model

import (
	"net/http"
	"strings"
	"testing"
)

// The statuses that mean the provider failed the request on its own side, and
// the ones that must not be read that way.
//
// 429 is the one that has to stay out even though the transport retries it for
// the same reason it retries a 503: it is this key's pace rather than the
// provider's health, and it already has a sentence of its own.
func TestProviderDownStatusIsTheProvidersFaultsOnly(t *testing.T) {
	for _, status := range []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		529, // Anthropic: overloaded, and in no RFC
	} {
		if !providerDownStatus(status) {
			t.Errorf("status %d is the provider breaking on its own side", status)
		}
		// And the transport has to keep retrying every one of them, which is
		// the half of the answer that must not be lost when the list moves.
		if !retryableStatus(status) {
			t.Errorf("status %d stopped being retried", status)
		}
	}
	for _, status := range []int{
		http.StatusTooManyRequests,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusBadRequest,
		http.StatusOK,
	} {
		if providerDownStatus(status) {
			t.Errorf("status %d is not the provider failing on its own side", status)
		}
	}
	if !retryableStatus(http.StatusTooManyRequests) {
		t.Error("429 stopped being retried, and that is the case the transport exists for")
	}
}

// The sentence, and the one reading it exists to prevent. 2026-09-11: opencode
// answered 500 to a chat turn and the whole message was "opencode request failed
// with status 500: Internal server error", which the owner read as his balance
// having run out.
func TestProviderDownErrorNamesTheirSideAndClearsTheUsersOwn(t *testing.T) {
	err := providerDownError("opencode", "", http.StatusInternalServerError,
		[]byte("Internal server error"), "Internal server error")
	if err == nil {
		t.Fatal("a 500 produced no error at all")
	}
	said := err.Error()
	for _, want := range []string{
		"opencode",              // whose side it was
		"500",                   // what a bug report needs
		"Internal server error", // what the host actually said
		"own servers failed",    // that the request was not at fault
		"credits are all fine",  // and that the money is not either, which is the point
	} {
		if !strings.Contains(said, want) {
			t.Errorf("message is missing %q: %s", want, said)
		}
	}
	if strings.Contains(said, "out of credits") {
		t.Errorf("an outage was reported as an empty wallet: %s", said)
	}

	// A gateway that answers with no body must not leave a colon hanging:
	// "(504: )" reads like something broke on the way to the message.
	bare := providerDownError("opencode", "", http.StatusGatewayTimeout, nil, "").Error()
	if !strings.HasSuffix(bare, "(504)") {
		t.Errorf("want the status alone when the host said nothing: %s", bare)
	}
}

// The other direction, and the reason the body is still read at all: some hosts
// front their billing check with a 5xx of their own. Telling somebody their
// credits are fine when they are not is that first mistake pointing the other
// way, so a balance named in the body wins wherever it turns up.
func TestProviderDownErrorStillHearsABalanceInTheBody(t *testing.T) {
	const said = "Insufficient balance or no resource package. Please recharge."
	const body = `{"error":{"code":"1113","message":"` + said + `"}}`

	err := providerDownError("opencode", "", http.StatusInternalServerError, []byte(body), said)
	if err == nil || !strings.Contains(err.Error(), "out of credits") {
		t.Fatalf("err = %v; want the balance sentence the body names", err)
	}
	if strings.Contains(err.Error(), "credits are all fine") {
		t.Errorf("a 5xx that names a balance was cleared as an outage: %v", err)
	}
}
