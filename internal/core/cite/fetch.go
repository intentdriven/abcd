package cite

// The native citation checker: one bounded HTTP attempt per URL.
//
// Three properties are load-bearing, and each is a refusal of something a
// link-checker usually does.
//
// It never retries. A refresh over a reference corpus that retried would become
// a burst against whichever host is having a bad afternoon, and — worse — would
// make the run's duration a function of how many links are failing. One attempt,
// one outcome; a failure is a fact to record, not a reason to try harder.
//
// It never reads a body. Liveness is a property of the status line, so the
// checker takes the response headers and closes the connection. That removes a
// whole class of resource risk (a citation to a huge file costs nothing) and it
// removes the temptation to sniff page content, which is how a "did the page
// really load" heuristic acquires a false-positive rate and a transcript.
//
// It separates "dead" from "not readable by a robot". A 403 is not evidence that
// a source is gone; it is evidence that this fetcher may not look. Recording it
// as broken would put a lie in a committed record the gate then enforces, so
// those statuses route to the manual queue instead.

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/urlguard"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxRedirects bounds the hops one citation may take before the chain is called
// broken. Five matches the bound memory ingest uses; a citation that needs more
// is not a stable address to be citing.
const maxRedirects = 5

// DefaultTimeout bounds one whole attempt — connect, TLS, and response headers.
// It is the ceiling on what a single unreachable citation can cost a run.
const DefaultTimeout = 20 * time.Second

// connectTimeout and headerTimeout bound the two phases separately, so a host
// that accepts a connection and then says nothing cannot consume the whole
// budget while a genuinely slow DNS lookup still has room.
const (
	connectTimeout = 10 * time.Second
	headerTimeout  = 15 * time.Second
)

// userAgent identifies the checker honestly. A fetcher that disguises itself as
// a browser to defeat a publisher's robot policy would be manufacturing the very
// "automatic" receipt the manual queue exists to avoid faking.
const userAgent = "abcd-docs-cite-refresh"

// Status is what one check established about one citation.
type Status string

const (
	// StatusOK means the address resolved to a live document.
	StatusOK Status = "ok"
	// StatusBroken means it did not: a 4xx/5xx, an unresolvable host, a refused
	// connection, a timeout, or a redirect chain past the bound.
	StatusBroken Status = "broken"
	// StatusBlocked means the source refused an automated fetcher. That is not
	// evidence the citation is dead, so it routes to the manual queue rather than
	// into the record.
	StatusBlocked Status = "blocked"
)

// CheckOutcome is one URL's result. It carries a classification, the address the
// request ended at, and a short human-readable reason — never page content, and
// never anything that reaches the committed baseline except FinalURL.
type CheckOutcome struct {
	URL      string `json:"url"`
	Status   Status `json:"status"`
	FinalURL string `json:"final_url,omitempty"`
	// Detail is for the operator's terminal. It is deliberately NOT persisted:
	// the baseline records what is true about a link, not the transcript of
	// finding out.
	Detail string `json:"detail,omitempty"`
	// Answered records that a host replied — any status line at all, including a
	// 404 or a 403. It is the bit that separates "this citation is dead" from
	// "this machine reached nothing", which StatusBroken alone cannot express:
	// a DNS failure, a refused connection and a genuine 404 all land there. The
	// refresh reads it to tell a broken corpus from a broken network, and
	// nothing persists it.
	Answered bool `json:"answered,omitempty"`
}

// Checker is the refresh seam. spc-17 fixes the baseline schema and the lint
// rules as the contract and leaves the producer replaceable, so a specialist
// link checker can later arrive as an adapter satisfying this one method without
// any gate semantics changing.
//
// An implementer owes two things beyond the obvious. The Status must be one of
// the three declared values — anything else stops the run rather than being
// quietly dropped. And on StatusBroken it should set Answered to say whether a
// host actually replied, because that is the bit the refresh uses to tell a dead
// citation from a dead network, and nothing else can recover it. Omitting it on
// StatusOK or StatusBlocked is harmless: those two entail an answer.
type Checker interface {
	Check(rawURL string) CheckOutcome
}

// HTTPChecker is the native, dependency-free implementation of Checker.
type HTTPChecker struct {
	client  *http.Client
	blocked func(net.IP) bool
}

// shippedBlocked is the address policy every shipped checker runs under. It is
// named rather than inlined so the tests can state precisely which single rule
// they relax (loopback, for httptest) and inherit the rest unchanged.
var shippedBlocked = urlguard.BlockedIP

// newShippedHTTPChecker returns the checker as it ships: the full SSRF policy and the
// default timeout.
func newShippedHTTPChecker() *HTTPChecker { return newHTTPChecker(shippedBlocked, DefaultTimeout) }

// newHTTPChecker builds a checker with an explicit address policy and timeout.
// The policy is a parameter for exactly one reason: it lets the fetch paths be
// exercised against an httptest server, which binds loopback, without ever
// relaxing what newShippedHTTPChecker ships.
func newHTTPChecker(blocked func(net.IP) bool, timeout time.Duration) *HTTPChecker {
	dialer := &net.Dialer{
		Timeout: connectTimeout,
		// The connect-time re-check closes the DNS-rebinding gap between the
		// name guard below and the transport's own resolution.
		Control: urlguard.DialControl(blocked),
	}
	return &HTTPChecker{
		blocked: blocked,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:           dialer.DialContext,
				TLSHandshakeTimeout:   connectTimeout,
				ResponseHeaderTimeout: headerTimeout,
			},
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				// Both refusals below are wrapped in redirectRefusal. Reaching
				// this function at all means a host answered with a 3xx, but
				// client.Do reports the refusal as an ERROR — so without the
				// marker the outcome would claim nothing replied, and the
				// refresh's wholesale-failure guard reads exactly that bit.
				if len(via) >= maxRedirects {
					return &redirectRefusal{errors.New("more than " + strconv.Itoa(maxRedirects) + " redirects")}
				}
				// A chain that has reached https never leaves it. The hop's
				// scheme is the redirecting host's choice, so without this pin
				// an https citation walked down to plaintext is followed, and
				// whatever answers on the plaintext leg — which anyone on the
				// path can forge — supplies the final address the baseline
				// records (iss-2609012037440084; memory ingest and update pin
				// the same way). It stands ahead of the host guard so the
				// refusal names the scheme. An http citation, and an http hop
				// before any https one, are followed as before: the pin is
				// against a downgrade, not against http.
				if r.URL.Scheme != "https" && reachedHTTPS(via) {
					return &redirectRefusal{errHTTPSDowngrade}
				}
				// Every hop is re-guarded: a public address that redirects to
				// 169.254.169.254 must not be followed.
				if err := urlguard.CheckHostWith(r.URL.Hostname(), blocked); err != nil {
					return &redirectRefusal{err}
				}
				return nil
			},
		},
	}
}

// Check performs one bounded attempt and classifies the result.
func (c *HTTPChecker) Check(rawURL string) CheckOutcome {
	out := CheckOutcome{URL: rawURL}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		out.Status, out.Detail = StatusBroken, "not a parsable URL: "+err.Error()
		return out
	}
	if err := urlguard.CheckHostWith(parsed.Hostname(), c.blocked); err != nil {
		out.Status, out.Detail = StatusBroken, err.Error()
		return out
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		out.Status, out.Detail = StatusBroken, err.Error()
		return out
	}
	req.Header.Set("User-Agent", userAgent)
	// Ask for a document. A source that refuses this Accept is refusing a robot,
	// which the status classification below reads correctly.
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if errors.Is(err, errHTTPSDowngrade) {
		// Not evidence the source is dead — the host answered, and pointed
		// somewhere this checker will not follow — so it goes to the manual
		// queue, never into the baseline as broken. The detail names the
		// scheme only: the plaintext hop's URL is redirect-controlled and may
		// carry a credential in its query.
		out.Status, out.Answered = StatusBlocked, true
		out.Detail = "the redirect chain left https for plaintext http, which is not followed — confirm the source by hand"
		return out
	}
	if err != nil {
		out.Status, out.Detail = StatusBroken, transportDetail(err)
		// A refused redirect chain is still a chain: a host answered 3xx to get
		// here, so this is a broken citation rather than a broken network.
		var rr *redirectRefusal
		out.Answered = errors.As(err, &rr)
		return out
	}
	// The body is never read: liveness is the status line. Closing without
	// draining forecloses the connection, which is the correct trade when the
	// next request in a run is to a different host anyway.
	defer resp.Body.Close()

	// A status line came back, whatever it says.
	out.Answered = true
	out.FinalURL = rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		// The final address is redirect-controlled: net/url preserves raw
		// non-ASCII in the query, and encoding/json escapes neither C1 nor
		// bidi/zero-width runes, so an unencoded value would reach the --json
		// surface and the committed baseline verbatim (iss-359).
		// Percent-encoding is lossless where the terminal sanitizer's masking
		// would corrupt a record that must round-trip.
		out.FinalURL = termsafe.EncodeHiddenRunes(resp.Request.URL.String())
	}
	out.Status, out.Detail = classify(resp.StatusCode)
	return out
}

// classify maps a status code onto the three outcomes.
//
// The blocked set is deliberately small and status-code-only: 401 and 403 are a
// credential or robot refusal, 406 is a content-negotiation refusal aimed at
// non-browsers, and 429 is rate limiting. Everything else that is not 2xx is
// broken. No body is sniffed for challenge-page markers — a heuristic over page
// text would be both unreliable and a reason to start reading content, and the
// challenge pages that matter answer with one of these codes anyway.
func classify(code int) (Status, string) {
	switch {
	case code >= 200 && code < 300:
		return StatusOK, ""
	case code == http.StatusUnauthorized, code == http.StatusForbidden,
		code == http.StatusNotAcceptable, code == http.StatusTooManyRequests:
		return StatusBlocked, "HTTP " + strconv.Itoa(code) + " " + http.StatusText(code) +
			" — the source refuses automated fetching"
	default:
		return StatusBroken, "HTTP " + strconv.Itoa(code) + " " + http.StatusText(code)
	}
}

// errHTTPSDowngrade is the redirect policy's refusal of a hop that leaves
// https once the chain has reached it.
var errHTTPSDowngrade = errors.New("a redirect left https for plaintext http")

// reachedHTTPS reports whether any request already made in the chain was
// https, so the next hop may not be anything else.
func reachedHTTPS(via []*http.Request) bool {
	for _, r := range via {
		if r.URL != nil && r.URL.Scheme == "https" {
			return true
		}
	}
	return false
}

// redirectRefusal marks an error raised from CheckRedirect — one where a host
// DID answer, with a 3xx, before abcd declined to follow it. It exists so Check
// can set Answered on a path where client.Do reports only an error.
type redirectRefusal struct{ err error }

func (e *redirectRefusal) Error() string { return e.err.Error() }
func (e *redirectRefusal) Unwrap() error { return e.err }

// transportDetail renders a transport failure without the *url.Error wrapper,
// which repeats the URL the outcome already names.
func transportDetail(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		if ue.Timeout() {
			return "timed out: " + ue.Err.Error()
		}
		return ue.Err.Error()
	}
	return err.Error()
}
