package cite

import (
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// loopbackOK is the address policy the fetch tests run under: everything the
// shipped guard refuses, EXCEPT loopback — which is where httptest binds. The
// shipped constructor is never relaxed; only these tests are.
func loopbackOK(ip net.IP) bool { return !ip.IsLoopback() && shippedBlocked(ip) }

// testChecker builds a checker against the local test policy with a short
// timeout, so a hung server fails a test in a second rather than half a minute.
func testChecker(timeout time.Duration) *HTTPChecker {
	return newHTTPChecker(loopbackOK, timeout)
}

// TestCheckAliveRecordsTheFinalAddress is the ordinary case: a 200 is alive and
// the recorded final address is the one that answered.
func TestCheckAliveRecordsTheFinalAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	got := testChecker(5 * time.Second).Check(srv.URL + "/page")
	if got.Status != StatusOK {
		t.Fatalf("status = %q (%s), want %q", got.Status, got.Detail, StatusOK)
	}
	if got.FinalURL != srv.URL+"/page" {
		t.Errorf("final URL = %q, want %q", got.FinalURL, srv.URL+"/page")
	}
}

// TestCheckRecordsRedirectDrift is the whole point of recording a final address:
// a citation that still names the pre-redirect URL is alive today and dead the
// day the redirect is retired, so the baseline must capture where it lands.
func TestCheckRedirectDriftIsRecorded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusMovedPermanently)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	got := testChecker(5 * time.Second).Check(srv.URL + "/old")
	if got.Status != StatusOK {
		t.Fatalf("status = %q (%s), want %q", got.Status, got.Detail, StatusOK)
	}
	if got.FinalURL != srv.URL+"/new" {
		t.Errorf("final URL = %q, want the redirect target %q", got.FinalURL, srv.URL+"/new")
	}
}

// TestCheckRefusesARedirectLoop pins the bound: a self-redirect must terminate as
// an outcome, never spin.
func TestCheckRefusesARedirectLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer srv.Close()

	got := testChecker(5 * time.Second).Check(srv.URL + "/loop")
	if got.Status != StatusBroken {
		t.Fatalf("status = %q (%s), want %q", got.Status, got.Detail, StatusBroken)
	}
	if !strings.Contains(got.Detail, "redirect") {
		t.Errorf("detail = %q, want it to name the redirect bound", got.Detail)
	}
}

// TestCheckAnsweredSeparatesADeadLinkFromADeadNetwork pins the bit the refresh's
// wholesale-failure guard depends on. StatusBroken alone cannot tell "the host
// replied 404" from "nothing replied", and the guard destroys or preserves a
// committed baseline on that difference — so the checker, which is the only code
// that knows, has to carry it.
//
// The redirect cases are the subtle ones: `client.Do` returns an ERROR when the
// chain is refused, yet a host plainly answered with a 3xx to get there.
func TestCheckAnsweredSeparatesADeadLinkFromADeadNetwork(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	forbidden := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer forbidden.Close()
	loop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer loop.Close()

	answered := []string{ok.URL, notFound.URL, forbidden.URL, loop.URL}
	for _, target := range answered {
		if got := testChecker(5 * time.Second).Check(target); !got.Answered {
			t.Errorf("%s: Answered = false, but a host replied (status %q, %s)", target, got.Status, got.Detail)
		}
	}

	// Nothing on the other end: a closed port is a transport failure, and the
	// guard must be able to see that no host answered.
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()
	if got := testChecker(2 * time.Second).Check(deadURL); got.Answered {
		t.Errorf("%s: Answered = true, but nothing was listening", deadURL)
	}
	// An SSRF refusal never reaches a host at all.
	if got := newShippedHTTPChecker().Check("http://169.254.169.254/x"); got.Answered {
		t.Error("a guard refusal reported that a host answered")
	}
}

// TestCheckBlockedStatusesRouteToTheManualQueue pins the exact classification.
// These four say "an automated fetcher may not read this", which is a different
// fact from "this link is dead" — recording them as broken would be a lie the
// gate then enforces.
func TestCheckBlockedStatusesRouteToTheManualQueue(t *testing.T) {
	for _, code := range []int{
		http.StatusUnauthorized,    // 401
		http.StatusForbidden,       // 403
		http.StatusNotAcceptable,   // 406
		http.StatusTooManyRequests, // 429
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		got := testChecker(5 * time.Second).Check(srv.URL)
		srv.Close()
		if got.Status != StatusBlocked {
			t.Errorf("HTTP %d: status = %q (%s), want %q", code, got.Status, got.Detail, StatusBlocked)
		}
	}
}

// TestCheckBrokenStatuses pins the other side of the same line: a 404 or a 500 is
// a broken citation, not a blocked one.
func TestCheckBrokenStatuses(t *testing.T) {
	for _, code := range []int{
		http.StatusNotFound,
		http.StatusGone,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		got := testChecker(5 * time.Second).Check(srv.URL)
		srv.Close()
		if got.Status != StatusBroken {
			t.Errorf("HTTP %d: status = %q (%s), want %q", code, got.Status, got.Detail, StatusBroken)
		}
	}
}

// TestCheckTimesOutAsAnOutcome pins the no-hang guarantee: a server that never
// answers costs one timeout, and the run continues.
func TestCheckTimesOutAsAnOutcome(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	defer func() { close(release); srv.Close() }()

	start := time.Now()
	got := testChecker(150 * time.Millisecond).Check(srv.URL)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("check took %s; the timeout is not bounding the attempt", elapsed)
	}
	if got.Status != StatusBroken {
		t.Fatalf("status = %q (%s), want %q", got.Status, got.Detail, StatusBroken)
	}
}

// TestCheckOneAttemptPerURL pins "a failure is an outcome, not a retry": the
// checker must hit the server exactly once, so a refresh over a large corpus
// cannot turn into a storm against a struggling host.
func TestCheckOneAttemptPerURL(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	testChecker(5 * time.Second).Check(srv.URL)
	if hits != 1 {
		t.Fatalf("server saw %d requests, want exactly 1", hits)
	}
}

// TestCheckRefusesInternalAddressesUnderTheShippedPolicy is the SSRF regression
// for THIS fetch path. The shipped constructor — not the relaxed test one — must
// refuse a loopback or metadata address that a committed document cites.
func TestCheckRefusesInternalAddressesUnderTheShippedPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	for _, target := range []string{srv.URL, "http://169.254.169.254/latest/meta-data/", "http://svc.internal/x"} {
		got := newShippedHTTPChecker().Check(target)
		if got.Status != StatusBroken {
			t.Errorf("%s: status = %q, want %q", target, got.Status, StatusBroken)
		}
		if !strings.Contains(got.Detail, "refusing") {
			t.Errorf("%s: detail = %q, want an SSRF-guard refusal", target, got.Detail)
		}
	}
}

// TestCheckOutcomeCarriesNoContentByConstruction pins a deliberate narrowing:
// liveness is judged from the status line alone, so no cited page's bytes ever
// enter the process or the record. A citation to a multi-gigabyte file costs a
// response header, not a download — and no fetch transcript can leak into the
// committed baseline through this type.
func TestCheckOutcomeCarriesNoContentByConstruction(t *testing.T) {
	banned := map[string]bool{
		"body": true, "content": true, "headers": true, "transcript": true,
		"response": true, "html": true, "bytes": true,
	}
	typ := reflect.TypeOf(CheckOutcome{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if banned[name] {
			t.Errorf("CheckOutcome declares a %s field; the checker must carry no page content", name)
		}
	}
}
