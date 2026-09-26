package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testKey is shaped like a provider key so a leak is unmistakable in any
// message a test inspects.
const testKey = "sk-or-v1-0123456789abcdef-not-a-real-key"

// fake is an OpenAI-compatible server that can fail every call. handler
// answers each request; calls counts them, so a test can prove a refusal
// reached no socket.
type fake struct {
	srv   *httptest.Server
	calls atomic.Int32
	last  atomic.Pointer[seen]
}

// seen is what the fake received on its last call.
type seen struct {
	method, path, auth, contentType string
	body                            map[string]json.RawMessage
}

func newFake(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, body map[string]json.RawMessage)) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var body map[string]json.RawMessage
		_ = json.Unmarshal(raw, &body)
		f.last.Store(&seen{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"), body: body})
		handler(w, r, body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// base is the fake's base URL in the /v1 shape a provider publishes.
func (f *fake) base() string { return f.srv.URL + "/api/v1" }

func completionBody(model, content string) string {
	b, _ := json.Marshal(map[string]any{
		"id": "gen-1", "object": "chat.completion", "model": model,
		"choices": []any{map[string]any{"index": 0, "finish_reason": "stop",
			"message": map[string]any{"role": "assistant", "content": content}}},
	})
	return string(b)
}

func ok(model, content string) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, completionBody(model, content))
	}
}

func status(code int, body string) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}
}

// jsonObject is an output contract that admits one JSON object with a
// "verdict" string, the shape a host sub-agent's payload has.
func jsonObject(b []byte) error {
	var v struct {
		Verdict string `json:"verdict"`
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return err
	}
	if v.Verdict == "" {
		return errors.New("verdict is empty")
	}
	return nil
}

func request() Request {
	return Request{
		Model: "typesafe/jev-1.13",
		Brief: Brief{Instructions: "You are the scribe. Emit one JSON object.", Input: "the request document"},
	}
}

func mustClient(t *testing.T, base, key string, opts ...Option) *Client {
	t.Helper()
	c, err := New(base, key, opts...)
	if err != nil {
		t.Fatalf("New(%q): %v", base, err)
	}
	return c
}

// assertNoKey fails when the key reaches an error a caller could print.
func assertNoKey(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), testKey) {
		t.Fatalf("the key reached an error: %v", err)
	}
}

// TestCompleteSendsTheBriefAndValidatesTheAnswer is criterion 1's call: the
// host's brief goes over the chat-completions protocol as the system and user
// messages, the key as a bearer token, the accepted settings as top-level
// fields, and the answer comes back validated, with the model asked for and
// the model the provider reported.
func TestCompleteSendsTheBriefAndValidatesTheAnswer(t *testing.T) {
	f := newFake(t, ok("typesafe/jev-1.13-20260915", `{"verdict":"yes"}`))
	c := mustClient(t, f.base(), testKey)
	req := request()
	req.Settings = map[string]json.RawMessage{"temperature": json.RawMessage(`0`), "seed": json.RawMessage(`42`)}
	res, err := c.Complete(context.Background(), req, jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("content = %q", res.Content)
	}
	if res.ModelAsked != "typesafe/jev-1.13" || res.ModelReported != "typesafe/jev-1.13-20260915" {
		t.Fatalf("models: asked %q, reported %q", res.ModelAsked, res.ModelReported)
	}
	got := f.last.Load()
	if got.method != http.MethodPost || got.path != "/api/v1/chat/completions" {
		t.Fatalf("request line = %s %s", got.method, got.path)
	}
	if got.auth != "Bearer "+testKey {
		t.Fatal("the key was not sent as a bearer token")
	}
	if !strings.HasPrefix(got.contentType, "application/json") {
		t.Fatalf("content type = %q", got.contentType)
	}
	var msgs []struct{ Role, Content string }
	if err := json.Unmarshal(got.body["messages"], &msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Role != "system" || msgs[0].Content != req.Brief.Instructions ||
		msgs[1].Role != "user" || msgs[1].Content != req.Brief.Input {
		t.Fatalf("messages = %+v", msgs)
	}
	if string(got.body["model"]) != `"typesafe/jev-1.13"` || string(got.body["temperature"]) != "0" ||
		string(got.body["seed"]) != "42" || string(got.body["stream"]) != "false" {
		t.Fatalf("body = %v", got.body)
	}
}

// TestNoKeyMeansNoAuthorizationHeader: a local server needs no key, and none
// is invented.
func TestNoKeyMeansNoAuthorizationHeader(t *testing.T) {
	f := newFake(t, ok("local-model", `{"verdict":"no"}`))
	c := mustClient(t, f.base(), "")
	if _, err := c.Complete(context.Background(), request(), jsonObject); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if a := f.last.Load().auth; a != "" {
		t.Fatalf("Authorization = %q, want none", a)
	}
}

// TestAFencedAnswerIsUnwrapped: a model that wraps its JSON in one code fence
// is read for the document inside it, which the contract then judges.
func TestAFencedAnswerIsUnwrapped(t *testing.T) {
	f := newFake(t, ok("m", "```json\n{\"verdict\":\"yes\"}\n```"))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("content = %q", res.Content)
	}
}

// TestEveryFailureIsRefusedWithoutTheKey drives the fake through every way a
// provider can fail. Each is an error, none carries the key (even when the
// provider's own body echoes it back), and none is mistaken for an answer.
func TestEveryFailureIsRefusedWithoutTheKey(t *testing.T) {
	huge := strings.Repeat("x", MaxResponseBytes+10)
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
		want    string
	}{
		{"401 echoing the key", status(401, `{"error":{"message":"invalid key `+testKey+`","code":401}}`), "HTTP 401"},
		{"403", status(403, `{"error":{"message":"forbidden"}}`), "HTTP 403"},
		{"429", status(429, `{"error":{"message":"rate limited"}}`), "HTTP 429"},
		{"500", status(500, `upstream exploded`), "HTTP 500"},
		{"502 no body", status(502, ``), "HTTP 502"},
		{"model the provider does not list", status(404, `{"error":{"message":"No endpoints found for typesafe/jev-1.13.","code":404}}`), "HTTP 404"},
		{"bad JSON", status(200, `{"choices": [`), "not a chat completion"},
		{"not an object", status(200, `[1,2,3]`), "not a chat completion"},
		{"huge body", status(200, huge), "larger than"},
		{"huge error body", status(500, huge), "HTTP 500"},
		{"no choices", status(200, `{"model":"m","choices":[]}`), "no choice"},
		{"error object on a 200", status(200, `{"error":{"message":"provider overloaded `+testKey+`","code":502}}`), "reported an error"},
		{"answer fails the output contract", ok("m", `{"verdict":""}`), "output contract"},
		{"answer is prose", ok("m", `I think the answer is yes.`), "output contract"},
		{"answer has an unknown field", ok("m", `{"verdict":"yes","extra":1}`), "output contract"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, tc.handler)
			_, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
			if err == nil {
				t.Fatal("Complete succeeded; want a refusal")
			}
			assertNoKey(t, err)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to name %q", err, tc.want)
			}
			if len(err.Error()) > 1024 {
				t.Fatalf("error is %d bytes; a provider's body must not flood it", len(err.Error()))
			}
		})
	}
}

// TestATimeoutIsRefused: a provider that never answers is refused within the
// client's bound, not waited on.
func TestATimeoutIsRefused(t *testing.T) {
	release := make(chan struct{})
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	defer close(release)
	c := mustClient(t, f.base(), testKey, WithTimeout(200*time.Millisecond))
	start := time.Now()
	_, err := c.Complete(context.Background(), request(), jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded against a server that never answered")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "no answer within") {
		t.Fatalf("error = %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("the timeout took %s", d)
	}
}

// TestARedirectIsNeverFollowed: the base URL is pinned; a provider answering
// with a redirect elsewhere is refused and the other host never sees the key
// or the brief.
func TestARedirectIsNeverFollowed(t *testing.T) {
	elsewhere := newFake(t, ok("m", `{"verdict":"yes"}`))
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		http.Redirect(w, r, elsewhere.srv.URL+"/steal", http.StatusTemporaryRedirect)
	})
	_, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err == nil {
		t.Fatal("Complete succeeded through a redirect")
	}
	assertNoKey(t, err)
	if !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("error = %v", err)
	}
	if n := elsewhere.calls.Load(); n != 0 {
		t.Fatalf("the redirect target received %d request(s)", n)
	}
}

// TestASettingTheProtocolDoesNotTakeIsRefusedBeforeAnyCall: a setting outside
// the accepted set is refused, naming it, and no request is made.
func TestASettingTheProtocolDoesNotTakeIsRefusedBeforeAnyCall(t *testing.T) {
	f := newFake(t, ok("m", `{"verdict":"yes"}`))
	req := request()
	req.Settings = map[string]json.RawMessage{"model": json.RawMessage(`"anthropic/claude-opus"`)}
	_, err := mustClient(t, f.base(), testKey).Complete(context.Background(), req, jsonObject)
	if err == nil || !strings.Contains(err.Error(), `"model"`) {
		t.Fatalf("error = %v, want a refusal naming the setting", err)
	}
	if n := f.calls.Load(); n != 0 {
		t.Fatalf("a refused request reached the provider %d time(s)", n)
	}
}

// TestAnEmptyModelIsRefusedBeforeAnyCall: the adapter asks for the model it
// is given, and it is never given none.
func TestAnEmptyModelIsRefusedBeforeAnyCall(t *testing.T) {
	f := newFake(t, ok("m", `{"verdict":"yes"}`))
	req := request()
	req.Model = ""
	if _, err := mustClient(t, f.base(), testKey).Complete(context.Background(), req, jsonObject); err == nil {
		t.Fatal("an empty model was sent")
	}
	if n := f.calls.Load(); n != 0 {
		t.Fatalf("reached the provider %d time(s)", n)
	}
}

// TestTheReportedModelIsBoundedAndClean: the provider's model field is
// untrusted, so it is bounded and a hidden or control rune cannot ride it.
func TestTheReportedModelIsBoundedAndClean(t *testing.T) {
	f := newFake(t, ok("evil‮model\x1b[31m"+strings.Repeat("m", 500), `{"verdict":"yes"}`))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(res.ModelReported) > MaxModelBytes+3 || strings.ContainsAny(res.ModelReported, "‮\x1b") {
		t.Fatalf("model reported = %q", res.ModelReported)
	}
}

// TestValidateBaseURL: the base URL is pinned per provider block. Plain HTTP
// is admitted only to this machine (a local server), and a URL carrying
// credentials, a query or a fragment is refused.
func TestValidateBaseURL(t *testing.T) {
	good := []string{
		"https://openrouter.ai/api/v1",
		"https://api.example.com/v1/",
		"http://127.0.0.1:8080/v1",
		"http://localhost:11434/v1",
		"http://[::1]:8000/v1",
	}
	for _, u := range good {
		if err := ValidateBaseURL(u); err != nil {
			t.Errorf("ValidateBaseURL(%q) = %v, want nil", u, err)
		}
	}
	bad := []string{
		"", "openrouter.ai/api/v1", "ftp://example.com/v1",
		"http://example.com/v1", "http://192.0.2.10/v1",
		"https://user:" + testKey + "@example.com/v1",
		"https://example.com/v1?key=" + testKey,
		"https://example.com/v1#frag",
		"https:///v1",
	}
	for _, u := range bad {
		err := ValidateBaseURL(u)
		if err == nil {
			t.Errorf("ValidateBaseURL(%q) = nil, want a refusal", u)
		}
		assertNoKey(t, err)
	}
}

// TestAcceptedSettingsIsACopy: the declaration cannot be widened by a caller.
func TestAcceptedSettingsIsACopy(t *testing.T) {
	a := AcceptedSettings()
	a[0] = "model"
	if AcceptedSettings()[0] == "model" {
		t.Fatal("AcceptedSettings returned the package's own slice")
	}
	for _, k := range AcceptedSettings() {
		if k == "model" || k == "messages" || k == "stream" {
			t.Fatalf("AcceptedSettings admits %q, which the adapter itself sets", k)
		}
	}
}

// awkwardKey carries every character an encoder escapes: '/' (PHP's
// json_encode), '<', '>', '&' and '\” (Go's HTML-safe JSON and HTML), '"'
// and '\\' (every JSON encoder and Go's %q), '+' and ' ' (a URL's query) and
// a non-ASCII letter (\u escapes and QuoteToASCII). A throwaway value, not a
// real credential.
const awkwardKey = `sk-aw/k+w<a>r&d'k"e\y é`

// escapedForms are representations of awkwardKey a provider may echo, each
// hand-written rather than derived from the code under test.
// Every backslash below is written \x5c, so the escapes are the text a
// provider sends rather than a Go escape.
var escapedForms = []string{
	awkwardKey,
	"sk-aw\x5c/k+w<a>r&d'k\x5c\x22e\x5c\x5cy \u00e9",                     // PHP json_encode, unicode kept
	"sk-aw\x5c/k+w<a>r&d'k\x5c\x22e\x5c\x5cy \x5cu00e9",                  // PHP json_encode default
	"sk-aw/k+w\x5cu003ca\x5cu003er\x5cu0026d'k\x5c\x22e\x5c\x5cy \u00e9", // Go encoding/json
	"sk-aw/k+w<a>r&d'k\x5c\x22e\x5c\x5cy \x5cu00e9",                      // Go %+q / QuoteToASCII
	"sk-aw/k+w&lt;a&gt;r&amp;d&#39;k&#34;e\x5cy \u00e9",                  // html.EscapeString
	"sk-aw%2Fk%2Bw%3Ca%3Er%26d%27k%22e%5Cy+%C3%A9",                       // url.QueryEscape
	"sk-aw%2Fk+w%3Ca%3Er&d%27k%22e%5Cy%20%C3%A9",                         // url.PathEscape
	"\x5cu0073\x5cu006b\x5cu002d\x5cu0061\x5cu0077",                      // every rune \u-escaped (prefix)
	"&#115;&#107;&#45;&#97;&#119;",                                       // every rune a numeric entity (prefix)
}

// assertNoKeyForm fails when any representation of awkwardKey, or its
// distinctive prefix, reaches text a caller could print or record.
func assertNoKeyForm(t *testing.T, where, s string) {
	t.Helper()
	for _, f := range escapedForms {
		if strings.Contains(s, f) {
			t.Fatalf("%s carries the key as %q: %s", where, f, s)
		}
	}
	for _, prefix := range []string{"sk-aw", "sk\x5cu002daw"} {
		if strings.Contains(s, prefix) {
			t.Fatalf("%s carries the key's prefix %q: %s", where, prefix, s)
		}
	}
}

// TestNoRepresentationOfTheKeySurvivesInAnError: a provider may echo the key
// in any field, in any encoding its stack applies, and in a body that is not
// JSON at all. Whatever the shape, the key reaches no error and no result.
func TestNoRepresentationOfTheKeySurvivesInAnError(t *testing.T) {
	var cases []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
		want    string
	}
	add := func(name string, h func(http.ResponseWriter, *http.Request, map[string]json.RawMessage), want string) {
		cases = append(cases, struct {
			name    string
			handler func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
			want    string
		}{name, h, want})
	}
	for i, f := range escapedForms[1:5] {
		add(fmt.Sprintf("401 detail field, JSON form %d", i+1), status(401, `{"detail":"bad key `+f+`"}`), "HTTP 401")
		add(fmt.Sprintf("401 error.message, JSON form %d", i+1), status(401, `{"error":{"message":"bad key `+f+`"}}`), "HTTP 401")
		add(fmt.Sprintf("200 error beside the message, JSON form %d", i+1), status(200, `{"error":{"code":401,"key":"`+f+`"}}`), "reported an error")
	}
	add("401 every rune \\u-escaped", status(401, `{"detail":"`+jsonEscapeAll(awkwardKey)+`"}`), "HTTP 401")
	add("401 HTML body", status(401, `<p>bad key `+escapedForms[5]+`</p>`), "HTTP 401")
	add("401 HTML body, numeric entities", status(401, `<p>bad key `+htmlNumericAll(awkwardKey)+`</p>`), "HTTP 401")
	add("401 query-escaped", status(401, `rejected GET /v1/models?key=`+escapedForms[6]), "HTTP 401")
	add("401 path-escaped", status(401, `rejected /keys/`+escapedForms[7]), "HTTP 401")
	add("contract quotes the answer", ok("m", awkwardKey), "output contract")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, tc.handler)
			quoting := func(b []byte) error { return fmt.Errorf("want a verdict, got %q and %+q", b, b) }
			_, err := mustClient(t, f.base(), awkwardKey).Complete(context.Background(), request(), quoting)
			if err == nil {
				t.Fatal("Complete succeeded; want a refusal")
			}
			assertNoKeyForm(t, "the error", err.Error())
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to name %q", err, tc.want)
			}
		})
	}
}

// TestTheReportedModelNeverCarriesTheKey: the model a provider reports is
// recorded and quoted in a denylist refusal, so a provider that reports the
// key as its model has it scrubbed like any other text it sends.
func TestTheReportedModelNeverCarriesTheKey(t *testing.T) {
	f := newFake(t, ok("vendor/"+awkwardKey, `{"verdict":"yes"}`))
	res, err := mustClient(t, f.base(), awkwardKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	assertNoKeyForm(t, "the reported model", res.ModelReported)
	if !strings.HasPrefix(res.ModelReported, "vendor/") {
		t.Fatalf("model reported = %q, want the provider's text around the key kept", res.ModelReported)
	}
}

// TestScrubKeepsTextWithoutTheKey: the scrub removes the key and nothing
// else, and an empty key scrubs nothing.
func TestScrubKeepsTextWithoutTheKey(t *testing.T) {
	if got := Scrub("rate limited: slow down", awkwardKey); got != "rate limited: slow down" {
		t.Fatalf("Scrub changed text without the key: %q", got)
	}
	if got := Scrub("anything", ""); got != "anything" {
		t.Fatalf("Scrub with no key = %q", got)
	}
	for _, f := range escapedForms[:8] {
		if got := Scrub("x "+f+" y", awkwardKey); got != "x [credential] y" {
			t.Fatalf("Scrub(%q) = %q", f, got)
		}
	}
}

func jsonEscapeAll(s string) string {
	var b strings.Builder
	for _, r := range s {
		fmt.Fprintf(&b, `\u%04x`, r)
	}
	return b.String()
}

func htmlNumericAll(s string) string {
	var b strings.Builder
	for _, r := range s {
		fmt.Fprintf(&b, "&#%d;", r)
	}
	return b.String()
}

// jsonEscapeEveryRune writes every rune of s as a JSON \u escape, a rune
// above the Basic Multilingual Plane as a surrogate pair. Every backslash is
// written \x5c, so the escape is the text a provider sends.
func jsonEscapeEveryRune(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r > 0xFFFF {
			r -= 0x10000
			fmt.Fprintf(&b, "\x5cu%04x\x5cu%04X", 0xD800+(r>>10), 0xDC00+(r&0x3FF))
			continue
		}
		fmt.Fprintf(&b, "\x5cu%04x", r)
	}
	return b.String()
}

// jsonEscapeMixed writes s the way a mixed encoder may: ASCII letters as \u
// escapes in upper-case hex, '/', '"' and the backslash as short escapes, and
// every other rune as it is.
func jsonEscapeMixed(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '"' || r == 0x5c:
			b.WriteByte(0x5c)
			b.WriteRune(r)
		case r < 0x80 && ('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z'):
			fmt.Fprintf(&b, "\x5cu%04X", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// TestAnEscapedKeyInAnUndecodableBodyIsScrubbed: a body that is not
// decodable JSON (plain text, or a JSON body cut at the error-body bound) is
// scrubbed as thoroughly as one that is: the key written as JSON escapes,
// ordinary ASCII runes included, reaches no error, and neither does a key
// that itself carries a backslash sequence or a character reference written
// literally, which a decoding step would otherwise rewrite before the scrub.
func TestAnEscapedKeyInAnUndecodableBodyIsScrubbed(t *testing.T) {
	const astralKey = "sk-astral-\U0001F600-key"
	const backslashKey = "sk-back\x5cnslash\x5cu0041-key"
	const ampKey = "sk-amp&amp;&#65;-key"
	type tc struct{ name, key, prefix, body string }
	var cases []tc
	for _, k := range []struct{ key, prefix string }{{awkwardKey, "sk-aw"}, {astralKey, "sk-astral"}} {
		every, mixed := jsonEscapeEveryRune(k.key), jsonEscapeMixed(k.key)
		cases = append(cases,
			tc{"plain text, every rune escaped", k.key, k.prefix, "bad key " + every + " was refused"},
			tc{"plain text, mixed escapes", k.key, k.prefix, "bad key " + mixed + " was refused"},
			tc{"JSON cut at the bound, every rune escaped", k.key, k.prefix,
				`{"detail":"bad key ` + every + strings.Repeat("x", maxErrorBodyBytes) + `"}`},
			tc{"JSON cut short, mixed escapes", k.key, k.prefix, `{"detail":"bad key ` + mixed},
		)
	}
	cases = append(cases,
		tc{"plain text, a key with a literal backslash sequence", backslashKey, "sk-back", "bad key " + backslashKey + " was refused"},
		tc{"plain text, a key with a literal character reference", ampKey, "sk-amp", "bad key " + ampKey + " was refused"},
		tc{"JSON, a key with a literal character reference", ampKey, "sk-amp", `{"detail":"bad key ` + ampKey + `"}`},
	)
	for _, c := range cases {
		t.Run(c.name+" "+c.prefix, func(t *testing.T) {
			f := newFake(t, status(401, c.body))
			_, err := mustClient(t, f.base(), c.key).Complete(context.Background(), request(), nil)
			if err == nil {
				t.Fatal("Complete succeeded; want a refusal")
			}
			msg := err.Error()
			if !strings.Contains(msg, "HTTP 401") {
				t.Fatalf("error = %v, want it to name HTTP 401", err)
			}
			for _, leak := range []string{c.prefix, jsonEscapeEveryRune(c.prefix), jsonEscapeMixed(c.prefix)} {
				if strings.Contains(msg, leak) {
					t.Fatalf("the error carries the key as %q: %s", leak, msg)
				}
			}
			if !strings.Contains(msg, "[credential]") {
				t.Fatalf("error = %s, want the key replaced by [credential] and the provider's text around it kept", msg)
			}
		})
	}
}

// TestUnescapeJSONText: every JSON string escape is undone once, wherever it
// stands in the text, and anything that is not a well-formed escape is left
// exactly as it is. Every backslash below is written \x5c.
func TestUnescapeJSONText(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"plain text", "plain text"},
		{"\x5cu006B\x5cu0065y", "key"},
		{"\x5cu00e9 \x5cu00E9", "é é"},
		{"\x5cud83d\x5cude00", "\U0001F600"},
		{"\x5cuD83D\x5cuDE00!", "\U0001F600!"},
		{"\x5c/ \x5c\x22 \x5c\x5c \x5cb\x5cf\x5cn\x5cr\x5ct", "/ \x22 \x5c \b\f\n\r\t"},
		{"\x5c\x5cu006b", "\x5cu006b"},
		{"\x5cq \x5cx41", "\x5cq \x5cx41"},
		{"\x5cu12", "\x5cu12"},
		{"\x5cuZZZZ", "\x5cuZZZZ"},
		{"\x5cu12G4", "\x5cu12G4"},
		{"end\x5c", "end\x5c"},
		{"\x5cud83d alone", "\x5cud83d alone"},
		{"\x5cude00 alone", "\x5cude00 alone"},
		{"\x5cud83d\x5cu0041", "\x5cud83dA"},
		{"\x5cu0000", "\x00"},
		{"\xff\x5cu0041", "\xffA"},
	} {
		if got := unescapeJSONText(c.in); got != c.want {
			t.Errorf("unescapeJSONText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
