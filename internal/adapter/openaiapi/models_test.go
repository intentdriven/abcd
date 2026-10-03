package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// listBody is a model list in the standard shape, one entry per id.
func listBody(t *testing.T, ids ...any) string {
	t.Helper()
	data := make([]map[string]any, len(ids))
	for i, id := range ids {
		data[i] = map[string]any{"id": id, "object": "model", "owned_by": "example"}
	}
	b, err := json.Marshal(map[string]any{"object": "list", "data": data})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// lists answers a model list with ids.
func lists(t *testing.T, ids ...any) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	body := listBody(t, ids...)
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
}

// listError is err as the listing's typed failure, or a test failure.
func listError(t *testing.T, err error) *ListError {
	t.Helper()
	var le *ListError
	if !errors.As(err, &le) {
		t.Fatalf("err = %v (%T), want a *ListError", err, err)
	}
	if le.Reason == "" {
		t.Fatalf("the ListError carries no reason: %v", err)
	}
	return le
}

// TestListModelsSendsNoKeyAndFollowsNoRedirect is G1's request rule at the
// adapter: the listing is one GET of {base}/models, carrying no
// Authorization header when the client holds no key, and a redirect, to this
// host or another, is a failure that is never followed, so a key header
// never reaches a second address.
func TestListModelsSendsNoKeyAndFollowsNoRedirect(t *testing.T) {
	f := newFake(t, lists(t, "vendor/coder-large", "vendor/coder-small"))
	got, err := mustClient(t, f.base(), "").Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if want := []string{"vendor/coder-large", "vendor/coder-small"}; fmt.Sprint(got.IDs) != fmt.Sprint(want) || got.Dropped != 0 {
		t.Fatalf("listing = %+v, want %v in the service's order and nothing dropped", got, want)
	}
	last := f.last.Load()
	if n := f.calls.Load(); n != 1 || last == nil {
		t.Fatalf("the service received %d request(s), want exactly one", n)
	}
	if last.method != http.MethodGet || last.path != "/api/v1/models" {
		t.Fatalf("request = %s %s, want GET /api/v1/models", last.method, last.path)
	}
	if last.auth != "" {
		t.Fatalf("a keyless listing sent Authorization %q", last.auth)
	}

	// A key is sent only when one is given, and then only as the bearer token.
	if _, err := mustClient(t, f.base(), testKey).Models(context.Background()); err != nil {
		t.Fatalf("keyed Models: %v", err)
	}
	if auth := f.last.Load().auth; auth != "Bearer "+testKey {
		t.Fatalf("a keyed listing sent Authorization %q, want the bearer token", auth)
	}

	for _, key := range []string{"", testKey} {
		elsewhere := newFake(t, lists(t, "stolen"))
		var auths []string
		moved := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
			auths = append(auths, r.Header.Get("Authorization"))
			http.Redirect(w, r, elsewhere.srv.URL+"/v1/models", http.StatusFound)
		})
		got, err := mustClient(t, moved.base(), key).Models(context.Background())
		if err == nil {
			t.Fatalf("key %t: a redirect was followed to a listing %+v", key != "", got)
		}
		assertNoKey(t, err)
		if le := listError(t, err); le.NeedsKey || !strings.Contains(le.Reason, "redirect") || !strings.Contains(le.Reason, "never follows") {
			t.Fatalf("key %t: ListError = %+v, want a redirect that is never followed", key != "", le)
		}
		if n := elsewhere.calls.Load(); n != 0 {
			t.Fatalf("key %t: the redirect target received %d request(s)", key != "", n)
		}
		if key == "" && (len(auths) != 1 || auths[0] != "") {
			t.Fatalf("a keyless listing sent Authorization %q", auths)
		}
	}
}

// TestListModelsBoundsTimeAndSize: the listing has its own short bound,
// ListTimeout, whatever the client's call limits are, and the body is read
// through MaxResponseBytes plus one byte, a larger one refused unread. At
// most MaxListedModels ids are kept, in the service's order.
func TestListModelsBoundsTimeAndSize(t *testing.T) {
	if ListTimeout != 10*time.Second {
		t.Fatalf("ListTimeout = %s, want 10s (decision 4's short timeout)", ListTimeout)
	}
	if MaxListedModels != 5000 {
		t.Fatalf("MaxListedModels = %d, want 5000", MaxListedModels)
	}
	for _, opts := range [][]Option{nil, {WithTimeout(time.Hour), WithFirstByteTimeout(time.Hour), WithIdleTimeout(time.Hour)}, {WithTimeout(time.Millisecond)}} {
		if c := mustClient(t, "https://api.example.com/v1", "", opts...); c.listWait != ListTimeout {
			t.Fatalf("with %d option(s) the listing's bound is %s, want ListTimeout whatever the call limits", len(opts), c.listWait)
		}
	}

	release := make(chan struct{})
	defer close(release)
	stall := func(headersFirst bool) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
		return func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
			if headersFirst {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":[`)
				w.(http.Flusher).Flush()
			}
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
	}
	for _, headersFirst := range []bool{false, true} {
		f := newFake(t, stall(headersFirst))
		c := mustClient(t, f.base(), testKey, WithTimeout(time.Hour), WithFirstByteTimeout(time.Hour))
		c.listWait = 200 * time.Millisecond
		// The caller's own deadline is far past the listing's bound: a listing
		// that ignored its bound would end on this instead, and say so.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		start := time.Now()
		_, err := c.Models(ctx)
		if d := time.Since(start); d > 3*time.Second {
			t.Fatalf("headers first %t: the listing took %s against a %s bound", headersFirst, d, c.listWait)
		}
		assertNoKey(t, err)
		if le := listError(t, err); !strings.Contains(le.Reason, "no list within") {
			t.Fatalf("headers first %t: ListError = %+v, want the timeout named", headersFirst, le)
		}
	}

	// A body exactly at the cap is read; one byte over is refused unread.
	at := listBody(t, "vendor/coder-large")
	at += strings.Repeat(" ", MaxResponseBytes-len(at))
	for _, tc := range []struct {
		body string
		ok   bool
	}{{at, true}, {at + " ", false}} {
		f := newFake(t, status(http.StatusOK, tc.body))
		got, err := mustClient(t, f.base(), "").Models(context.Background())
		switch {
		case tc.ok && (err != nil || len(got.IDs) != 1):
			t.Fatalf("a %d-byte body: listing %+v, err %v; want it read", len(tc.body), got, err)
		case !tc.ok:
			if le := listError(t, err); !strings.Contains(le.Reason, "larger than") {
				t.Fatalf("a %d-byte body: ListError = %+v, want the size refusal", len(tc.body), le)
			}
		}
	}

	ids := make([]any, MaxListedModels+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("vendor/model-%05d", i)
	}
	f := newFake(t, lists(t, ids...))
	got, err := mustClient(t, f.base(), "").Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(got.IDs) != MaxListedModels || got.IDs[0] != "vendor/model-00000" || got.IDs[MaxListedModels-1] != fmt.Sprintf("vendor/model-%05d", MaxListedModels-1) {
		t.Fatalf("kept %d ids (first %q), want the first %d in the service's order", len(got.IDs), got.IDs[0], MaxListedModels)
	}
}

// TestListModelsNamesWhyThereIsNoList is G7 at the adapter: 401 and 403 mean
// the service lists its models only for a key, and every other way there is
// no list (a status, an empty list, a body that is not a list, no usable id,
// a host that cannot be reached) carries its own reason in plain words. An
// id the adapter cannot keep is dropped and counted, never passed on.
func TestListModelsNamesWhyThereIsNoList(t *testing.T) {
	closed := newFake(t, lists(t, "m"))
	unreachable := closed.base()
	closed.srv.Close()

	cases := []struct {
		name     string
		handler  func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
		base     string
		needsKey bool
		reason   string
	}{
		{name: "401", handler: status(401, `{"error":{"message":"missing key"}}`), needsKey: true, reason: "only for a key"},
		{name: "403", handler: status(403, `forbidden`), needsKey: true, reason: "only for a key"},
		{name: "404", handler: status(404, `{"error":"no such route"}`), reason: "answered not found"},
		{name: "500", handler: status(500, `{"error":{"message":"down"}}`), reason: "answered HTTP 500"},
		{name: "empty list", handler: status(200, `{"object":"list","data":[]}`), reason: "listed no models"},
		{name: "no data member", handler: status(200, `{"object":"list"}`), reason: "not a model list"},
		{name: "data not a list", handler: status(200, `{"data":{"id":"m"}}`), reason: "not a model list"},
		{name: "not JSON", handler: status(200, `<html>models</html>`), reason: "not a model list"},
		{name: "no usable id", handler: lists(t, "", 42, "esc\x1b[31m", strings.Repeat("m", 129)), reason: "listed no usable models"},
		{name: "unreachable", base: unreachable, reason: "could not be reached"},
	}
	reasons := map[string]string{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := tc.base
			if base == "" {
				base = newFake(t, tc.handler).base()
			}
			got, err := mustClient(t, base, "").Models(context.Background())
			if err == nil {
				t.Fatalf("Models listed %+v; want no list", got)
			}
			le := listError(t, err)
			if le.NeedsKey != tc.needsKey || !strings.Contains(le.Reason, tc.reason) {
				t.Fatalf("ListError = %+v, want NeedsKey %t and a reason naming %q", le, tc.needsKey, tc.reason)
			}
			if !strings.Contains(err.Error(), le.Reason) || !strings.HasPrefix(err.Error(), "openaiapi: ") {
				t.Fatalf("Error() = %q, want the adapter's prefix and the reason", err.Error())
			}
			if strings.ContainsAny(le.Reason, "\n\x1b") {
				t.Fatalf("the reason carries a control character: %q", le.Reason)
			}
			reasons[tc.name] = le.Reason
		})
	}
	if reasons["404"] == reasons["500"] || reasons["empty list"] == reasons["not JSON"] || reasons["not JSON"] == reasons["no usable id"] {
		t.Fatalf("two ways of having no list share a reason: %v", reasons)
	}

	// Usable ids are kept in order; every other entry is dropped and counted.
	long := "vendor/" + strings.Repeat("m", 128-len("vendor/"))
	f := newFake(t, lists(t, "vendor/a", "", 7, nil, "esc\x1b[31m", "bidi‮name", "zw​name", long, long+"x", "  ", "vendor/b"))
	got, err := mustClient(t, f.base(), "").Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if want := []string{"vendor/a", long, "vendor/b"}; fmt.Sprint(got.IDs) != fmt.Sprint(want) || got.Dropped != 8 {
		t.Fatalf("listing = %+v, want %v kept and 8 dropped", got, want)
	}
}

// TestListModelsScrubsTheKey: a service may echo the key, in any encoding,
// in an error body and in a listed id. Neither reaches the result: no error
// and no reason carries any form of it, and an id that carries it is
// dropped, never kept redacted.
func TestListModelsScrubsTheKey(t *testing.T) {
	echoes := append([]string{}, escapedForms[:8]...)
	echoes = append(echoes, jsonEscapeAll(awkwardKey), htmlNumericAll(awkwardKey), jsonEscapeMixed(awkwardKey),
		// Short enough to pass the length bound, so only a reading drops them:
		// one character reference, and one written as a JSON escape.
		"&#115;"+awkwardKey[1:], "\x5cu0026#115;"+awkwardKey[1:])
	for _, code := range []int{401, 403, 404, 500} {
		for i, e := range echoes {
			for _, body := range []string{`{"error":{"message":"bad key ` + e + `"}}`, `bad key ` + e} {
				f := newFake(t, status(code, body))
				_, err := mustClient(t, f.base(), awkwardKey).Models(context.Background())
				le := listError(t, err)
				assertNoKeyForm(t, fmt.Sprintf("HTTP %d, form %d: the error", code, i), err.Error())
				assertNoKeyForm(t, fmt.Sprintf("HTTP %d, form %d: the reason", code, i), le.Reason)
			}
		}
	}

	cases := []struct {
		key string
		ids []any
	}{
		{key: testKey, ids: []any{"vendor/good", "vendor/" + testKey, testKey, "vendor/also-good"}},
	}
	awkward := []any{"vendor/good"}
	for _, e := range echoes {
		awkward = append(awkward, e, "vendor/"+e, e+"/m")
	}
	cases = append(cases, struct {
		key string
		ids []any
	}{awkwardKey, append(awkward, "vendor/also-good")})
	for _, tc := range cases {
		f := newFake(t, lists(t, tc.ids...))
		got, err := mustClient(t, f.base(), tc.key).Models(context.Background())
		if err != nil {
			t.Fatalf("Models: %v", err)
		}
		joined := strings.Join(got.IDs, "\n")
		assertNoKeyForm(t, "the listed ids", joined)
		if strings.Contains(joined, testKey) || strings.Contains(joined, "[credential]") {
			t.Fatalf("the listed ids carry the key or its redaction: %q", got.IDs)
		}
		if want := []string{"vendor/good", "vendor/also-good"}; fmt.Sprint(got.IDs) != fmt.Sprint(want) || got.Dropped != len(tc.ids)-2 {
			t.Fatalf("listing = %q, %d dropped; want %v and %d dropped", got.IDs, got.Dropped, want, len(tc.ids)-2)
		}
	}
}
