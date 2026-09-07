package imagegen

// One retry, for transport failures only.
//
// Measured rather than assumed. The first picture call in the app spent ten
// seconds on `net/http: TLS handshake timeout` against the free endpoint, and
// the second call — made by the MODEL a second later, having read the error —
// succeeded in five (owner's log, 7 ก.ย. 20:37). So the answer was one retry,
// and it cost a whole model turn to arrive at: a failed tool call, a sentence
// of apology, a re-planned prompt, a second call. All of that for a handshake
// that did not complete.
//
// **Transport failures only, and this is the whole discipline of the file.** A
// non-2xx answer is the vendor speaking, not the network failing: a 429 means
// slow down and retrying it immediately is how a rate limit becomes a ban, a
// 400 will be a 400 again, and a 401 will not grow a key. Those go back to the
// caller untouched. What is retried is the case where nothing was ever
// answered — a handshake that timed out, a connection reset, a DNS blip.
//
// One retry, not a loop. A picture call already costs seconds; a second failure
// is a signal the user should see rather than something to keep paying for.

import (
	"net/http"
	"time"
)

// retryPause is long enough for a transient blip to pass and short enough to
// stay inside the wait the user is already in.
const retryPause = 700 * time.Millisecond

// doWithRetry sends req, and sends it once more if the transport itself failed.
func doWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err == nil {
		return resp, nil
	}
	// A cancelled turn is not a blip. Retrying it would keep working for a user
	// who has already stopped waiting.
	if ctxErr := req.Context().Err(); ctxErr != nil {
		return nil, err
	}

	// The first attempt consumed the body. NewRequestWithContext fills GetBody
	// in for the reader types used here, so a fresh one is available; a request
	// that cannot produce one is not retried rather than retried empty.
	retry := req.Clone(req.Context())
	if req.GetBody != nil {
		fresh, bodyErr := req.GetBody()
		if bodyErr != nil {
			return nil, err
		}
		retry.Body = fresh
	} else if req.Body != nil {
		return nil, err
	}

	// Waited on the context, not slept through: a stop during the pause has to
	// end the call rather than be noticed 700ms later.
	timer := time.NewTimer(retryPause)
	defer timer.Stop()
	select {
	case <-req.Context().Done():
		return nil, err
	case <-timer.C:
	}

	resp, retryErr := client.Do(retry)
	if retryErr != nil {
		// The FIRST error is the one returned: it is the one that describes what
		// this call actually ran into, and a second identical sentence with a
		// different socket number tells the reader nothing new.
		return nil, err
	}
	return resp, nil
}
