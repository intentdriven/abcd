// Package openaiapi is abcd's OpenAI-compatible API adapter
// (itd-2609081951381895): one client over the chat-completions protocol, which
// OpenRouter and a local OpenAI-compatible server both speak, so a provider is
// configuration of this adapter and never code (the intent's Decision 2).
//
// The client asks for the model it is given and nothing else: which models a
// provider may serve (its allowlist, and any oracle.denylist entry the
// configuration writes) is internal/core/oracle's to decide before a Client is
// ever built (adr-2609221009491186, adr-2609300107513982). The adapter's own guarantees are the network path's:
//
//   - the base URL is pinned per provider block, plain HTTP is admitted only to
//     this machine (a local server), and a redirect is never followed, so a
//     provider cannot move the key or the brief to another host;
//   - the answer is streamed, every response is bounded (MaxResponseBytes, a
//     stream's events and its whole length besides) and every call is bounded
//     in time, by a first-byte limit, an idle limit between reads, an event
//     limit between a stream's events (a keep-alive comment is not one) and a
//     total cap, so a provider that floods or stalls is refused rather than
//     waited on, while one still sending is never cut off at an arbitrary age;
//     a call that ends early closes its connection, so the server sees it go;
//   - the key travels only as the Authorization header of a request to the
//     pinned base URL. No error, result or log line carries it: a provider's
//     own text (its error, the model it reports and the answer itself) is
//     decoded and scrubbed of every representation of the key (Scrub), the
//     error and the model bounded and sanitised as well, before it can reach
//     an error or a result, because a provider may echo what it was sent, in
//     whatever encoding its stack applies;
//   - a setting the protocol does not take is refused before any call, and the
//     answer is judged by the caller's output contract, the same one the host
//     sub-agent's payload is judged by, so an answer that does not satisfy it is
//     refused rather than used.
//
// It uses net/http and encoding/json alone; it reads no file and no
// credential store: the caller resolves the key by name through
// internal/core/credential and hands the value in. The one environment it
// honours is net/http's own, through the default transport: the standard
// proxy variables for https (HTTPS_PROXY, NO_PROXY) and the platform's trust
// roots. An https call through a proxy is a CONNECT tunnel, so the key
// and the brief stay inside TLS, and a call to this machine (the only one
// plain HTTP may reach) is never proxied.
package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// The answer is streamed, so a call is bounded three ways rather than by
	// one end-to-end deadline, which abandoned a reasoning model's answer that
	// was still arriving (iss-2610030931521214: such answers run past ten
	// minutes).
	//
	// DefaultFirstByteTimeout bounds the wait from sending the brief to the
	// answer's first byte: a local server reads a long prompt before it says
	// anything.
	DefaultFirstByteTimeout = 5 * time.Minute
	// DefaultIdleTimeout bounds the silence between two reads once the answer
	// has begun; a keep-alive comment counts as the server being alive.
	DefaultIdleTimeout = 2 * time.Minute
	// DefaultTotalTimeout caps one call end to end however steadily it
	// streams: connecting, sending the brief and reading the whole answer.
	DefaultTotalTimeout = 30 * time.Minute
	// defaultEventTimeout bounds the time between two events of a stream,
	// keep-alive comments not counting, so a server that keeps the connection
	// alive and never answers is refused well inside the total cap. It is the
	// window the record observed a gateway hold a request that had sent
	// nothing for (about 600 seconds, iss-2610030931521214), and twice the
	// first-byte limit's allowance for a long prompt read.
	defaultEventTimeout = 10 * time.Minute
	// errorBodyWait bounds the read of a failed call's body: the status is the
	// answer, and the body only says why, so it is not waited for long.
	errorBodyWait = 30 * time.Second
	// MaxResponseBytes bounds a successful answer: a plain body, one event of
	// a stream, and the answer a stream assembles. A chat completion carrying a
	// verdict is a few kilobytes; 4 MiB refuses a flood without ever refusing a
	// real one.
	MaxResponseBytes = 4 << 20
	// maxStreamBytes bounds everything a stream may send, framing, keep-alive
	// comments and the reasoning a model streams beside its answer included.
	// Each event repeats the chunk's envelope, so a long answer streamed a
	// token at a time is many times its own size.
	maxStreamBytes = 16 * MaxResponseBytes
	// maxErrorBodyBytes bounds how much of a failed call's body is read at all;
	// a body longer than this is reported as cut off and not quoted.
	maxErrorBodyBytes = 16 << 10
	// maxEcho bounds how much of a provider's own text an error carries.
	maxEcho = 200
	// MaxModelBytes bounds the model a provider reports.
	MaxModelBytes = 200
)

// acceptedSettings is the chat-completions request fields a routing row or a
// --route may set. model, messages and stream are the adapter's own and are
// never a setting.
var acceptedSettings = []string{
	"frequency_penalty",
	"max_tokens",
	"presence_penalty",
	"seed",
	"stop",
	"temperature",
	"top_p",
}

// AcceptedSettings returns the settings this adapter accepts, sorted: the
// declaration internal/core/oracle carries on a provider connection, so a
// setting outside it is refused before a step runs (spc-2609251028149555,
// AC 8). The slice is a copy.
func AcceptedSettings() []string { return append([]string(nil), acceptedSettings...) }

// Brief is what the host sub-agent is given, in the protocol's two roles:
// Instructions is the agent's prompt (the system message) and Input is the
// request the verb emitted for it (the user message).
type Brief struct {
	Instructions string
	Input        string
}

// Request is one call: the model asked for, the brief and the settings as
// sent. Settings are JSON scalars keyed by an accepted setting's name.
type Request struct {
	Model    string
	Brief    Brief
	Settings map[string]json.RawMessage
}

// Result is one validated answer: the document the output contract admitted,
// the model asked for, and the model the provider reported (bounded, with any
// hidden or control rune percent-encoded), so a substitution is visible.
type Result struct {
	Content       []byte
	ModelAsked    string
	ModelReported string
	// FinishReason is why the provider stopped ("stop", "length"), bounded
	// and cleaned like the model; empty when it gave none.
	FinishReason string
	// Usage is the token count the provider reported, nil when it sent none.
	Usage *Usage
}

// Usage is the token count a provider reports with its answer.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client is one provider connection. It holds the key; it never prints it.
type Client struct {
	endpoint string
	key      string
	forms    []string // every representation of key the scrub removes
	host     string
	timeout  time.Duration // the total cap
	// firstByte bounds the wait for the answer's first byte, idle the silence
	// between two reads of it once it has begun.
	firstByte time.Duration
	idle      time.Duration
	// events bounds the time between two events of a stream, keep-alive
	// comments not counting; errorWait bounds the read of a failed call's body.
	events    time.Duration
	errorWait time.Duration
	hc        *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout caps one call end to end, however steadily its answer streams;
// d <= 0 keeps DefaultTotalTimeout. A caller with a step it knows to be long
// raises it here; the first-byte and idle limits still apply within it.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithFirstByteTimeout bounds the wait for the answer's first byte; d <= 0
// keeps DefaultFirstByteTimeout.
func WithFirstByteTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.firstByte = d
		}
	}
}

// WithIdleTimeout bounds the silence between two reads once the answer has
// begun; d <= 0 keeps DefaultIdleTimeout.
func WithIdleTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.idle = d
		}
	}
}

// ErrUnreachable is what a call's error wraps when the provider could not be
// reached at all: no connection to it (or to the proxy in front of it) was
// ever made, so nothing of the request, the key or the brief left this
// machine. A caller may leave such a step to another leg. A provider that
// answered, or that took the request and never answered, is not unreachable:
// the brief may already have been sent.
var ErrUnreachable = errors.New("openaiapi: the provider could not be reached")

// unreachableError is a scrubbed refusal that is ErrUnreachable.
type unreachableError struct{ msg string }

func (e *unreachableError) Error() string { return e.msg }
func (e *unreachableError) Unwrap() error { return ErrUnreachable }

// errRedirect is what the client answers a redirect with.
var errRedirect = errors.New("redirect refused")

// New returns a client for the provider at baseURL (validated by
// ValidateBaseURL), sending key as a bearer token; an empty key sends none,
// for a local server that needs none.
func New(baseURL, key string, opts ...Option) (*Client, error) {
	if err := ValidateBaseURL(baseURL); err != nil {
		return nil, err
	}
	u, _ := url.Parse(baseURL)
	// net/http never proxies a call to localhost or a loopback address, but it
	// matches localhost in lower case alone, so LOCALHOST would be sent
	// through HTTP_PROXY with the key in cleartext. A host name is
	// case-insensitive, so spelling it in lower case changes nothing else
	// about where the call goes. Only localhost is rewritten: net/http already
	// compares NO_PROXY without case, and an IPv6 zone is not lower-cased.
	if strings.EqualFold(u.Hostname(), "localhost") {
		u.Host = strings.ToLower(u.Host)
	}
	c := &Client{
		endpoint:  strings.TrimSuffix(u.String(), "/") + "/chat/completions",
		key:       key,
		forms:     keyForms(key),
		host:      u.Host,
		timeout:   DefaultTotalTimeout,
		firstByte: DefaultFirstByteTimeout,
		idle:      DefaultIdleTimeout,
		events:    defaultEventTimeout,
		errorWait: errorBodyWait,
	}
	for _, o := range opts {
		o(c)
	}
	c.hc = &http.Client{
		// No Timeout: it would bound the whole streamed answer as one
		// deadline. Complete bounds the call itself (the first-byte, idle and
		// total limits), through the request's context.
		//
		// The base URL is pinned: a redirect, to this host or another, is never
		// followed, so the key and the brief go only where the block says.
		CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect },
	}
	return c, nil
}

// ValidateBaseURL admits an absolute https URL, or an http URL to this machine
// (localhost or a loopback address), with a host and no credentials, query or
// fragment. The refusal never quotes the URL, which may carry a secret.
func ValidateBaseURL(raw string) error {
	if raw == "" {
		return errors.New("base_url is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("base_url is not a URL")
	}
	switch {
	case u.User != nil:
		return errors.New("base_url carries credentials; a key is named in the provider block, never written into its URL")
	case u.RawQuery != "" || u.ForceQuery:
		return errors.New("base_url carries a query; a provider's base URL takes none")
	case u.Fragment != "" || strings.Contains(raw, "#"):
		return errors.New("base_url carries a fragment; a provider's base URL takes none")
	case u.Host == "" || u.Hostname() == "":
		return errors.New("base_url names no host")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if loopback(u.Hostname()) {
			return nil
		}
		return errors.New("base_url is plain http to another machine; a key and a brief leave this machine only over https (http is admitted for a local server on localhost or a loopback address)")
	}
	return errors.New("base_url is not an https URL")
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Complete sends one brief and returns the answer the contract admits. The
// request is refused before any call when it names no model or carries a
// setting the protocol does not take. contract is the output contract the
// host sub-agent's payload is judged by; nil admits any answer (a
// verification call, which judges only that the provider answered).
//
// The call asks for a stream and assembles its events into the answer
// (stream.go); a server that answers with one chat-completion body instead is
// read as that. It is bounded by ctx and by the first-byte, idle and total
// limits, and whichever ends it cancels the request, which closes the
// connection, so a server still generating sees the client go away.
func (c *Client) Complete(ctx context.Context, req Request, contract func([]byte) error) (Result, error) {
	body, err := c.render(req)
	if err != nil {
		return Result{}, err
	}
	l := c.limit(ctx)
	defer l.stop()
	httpReq, err := http.NewRequestWithContext(l.ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, c.fail("the request could not be built")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream, application/json")
	if c.key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return Result{}, c.callError(l, err)
	}
	// Closed on every path before the limits are released: an answer not read
	// to its end leaves the connection closed, never parked.
	defer resp.Body.Close()
	live := &liveBody{r: resp.Body, l: l}

	if resp.StatusCode != http.StatusOK {
		return Result{}, c.failedStatus(l, resp)
	}
	if eventStream(resp.Header.Get("Content-Type")) {
		return c.readStream(l, live, req.Model, contract)
	}
	raw, err := io.ReadAll(io.LimitReader(live, MaxResponseBytes+1))
	if err != nil {
		return Result{}, c.callError(l, err)
	}
	if len(raw) > MaxResponseBytes {
		return Result{}, c.fail(fmt.Sprintf("%s answered with a body larger than %d bytes, so it is refused unread", c.host, MaxResponseBytes))
	}
	return c.decode(raw, req.Model, contract)
}

// failedStatus reports a non-200 answer on its status, quoting what its body
// says when the whole body arrives within errorWait and maxErrorBodyBytes. The
// body is read past liveBody, so its bytes re-arm no limit: the first-byte,
// idle and total limits still run, and whichever ends the read first, the
// status is reported. A body cut off, by the wait, the size bound or a read
// error, is not quoted at all: the cut can end inside a key the provider
// echoed, and the scrub matches whole key forms only (iss-2610031210435016).
func (c *Client) failedStatus(l *limits, resp *http.Response) error {
	wait := time.AfterFunc(c.errorWait, func() { l.cancel(errErrorBody) })
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
	wait.Stop()
	msg := fmt.Sprintf("%s answered HTTP %d", c.host, resp.StatusCode)
	if err != nil || len(raw) > maxErrorBodyBytes {
		return c.fail(msg + " (its body was cut off, so it is not quoted)")
	}
	if said := c.providerSaid(raw); said != "" {
		msg += ": " + said
	}
	return c.fail(msg)
}

// render builds the request body: the adapter's own fields, then the
// settings, each refused unless the protocol takes it.
func (c *Client) render(req Request) ([]byte, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("openaiapi: the request names no model; the adapter asks for the model it is given, and it was given none")
	}
	fields := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": req.Brief.Instructions},
			{"role": "user", "content": req.Brief.Input},
		},
		"stream": true,
	}
	keys := make([]string, 0, len(req.Settings))
	for k := range req.Settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !accepted(k) {
			return nil, fmt.Errorf("openaiapi: setting %q is not one the chat-completions adapter accepts (%s); it is refused before the call, never dropped",
				termsafe.Sanitize(bound(k)), strings.Join(acceptedSettings, ", "))
		}
		v := req.Settings[k]
		if !json.Valid(v) {
			return nil, fmt.Errorf("openaiapi: setting %s is not JSON", k)
		}
		fields[k] = json.RawMessage(v)
	}
	return json.Marshal(fields)
}

func accepted(k string) bool {
	for _, a := range acceptedSettings {
		if a == k {
			return true
		}
	}
	return false
}

// completion is the part of a chat completion the adapter reads.
type completion struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
		} `json:"message"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage          `json:"usage"`
	Error json.RawMessage `json:"error"`
}

// decode reads one chat-completion body: a server's answer to a call that
// asked for a stream and got one body instead.
func (c *Client) decode(raw []byte, asked string, contract func([]byte) error) (Result, error) {
	var cc completion
	if err := json.Unmarshal(raw, &cc); err != nil {
		// The decoder's message can quote the body; it is dropped.
		return Result{}, c.fail(c.host + " answered with a body that is not a chat completion (not a JSON object of the protocol's shape)")
	}
	if isError(cc.Error) {
		return Result{}, c.reportedError(raw)
	}
	if len(cc.Choices) == 0 {
		return Result{}, c.noChoice()
	}
	a := assembled{model: cc.Model, usage: cc.Usage}
	if p := cc.Choices[0].Message.Content; p != nil {
		a.content.WriteString(*p)
	}
	if p := cc.Choices[0].FinishReason; p != nil {
		a.finish = *p
	}
	return c.admit(&a, asked, contract)
}

// assembled is an answer as read, from one body or from a stream's chunks.
type assembled struct {
	model   string
	content strings.Builder
	finish  string
	usage   *Usage
}

// admit holds an answer as read to the adapter's rules and the caller's
// output contract, and returns the result a caller records.
func (c *Client) admit(a *assembled, asked string, contract func([]byte) error) (Result, error) {
	// An answer that names no model cannot show what answered: the record would
	// carry an empty model and no oracle.denylist entry could match it, so it is
	// a contract failure, refused rather than used (iss-2609300805371434).
	if strings.TrimSpace(a.model) == "" {
		return Result{}, c.fail(c.host + " reported no model with its answer, so what answered cannot be shown; the answer is refused rather than used")
	}
	// The reported model and the finish reason are the provider's own text,
	// recorded and quoted in a denylist refusal, so they are scrubbed like any
	// other, bounded and cleaned of hidden runes.
	res := Result{ModelAsked: asked, ModelReported: cleanModel(c.scrub(a.model)),
		FinishReason: cleanModel(c.scrub(a.finish)), Usage: a.usage}
	// The answer is the payload a verb records, so it is scrubbed like every
	// other text the provider sends, before the contract judges it.
	res.Content = []byte(c.scrubAnswer(unfence(a.content.String())))
	if contract != nil {
		if err := contract(res.Content); err != nil {
			return Result{}, c.fail("the answer does not satisfy the output contract, so it is refused rather than used: " +
				termsafe.Sanitize(bound(c.scrub(err.Error()))))
		}
	}
	return res, nil
}

// isError reports whether a protocol error member is present and not null.
func isError(e json.RawMessage) bool { return len(e) > 0 && string(e) != "null" }

// reportedError is the refusal of a successful answer that carries the
// protocol's error member, quoting what the provider said, scrubbed.
func (c *Client) reportedError(raw []byte) error {
	msg := c.host + " reported an error in a successful answer"
	if said := c.providerSaid(raw); said != "" {
		msg += ": " + said
	}
	return c.fail(msg)
}

func (c *Client) noChoice() error {
	return c.fail(c.host + " answered with no choice, so there is no answer to read")
}

// scrubAnswer is an answer with every representation of the key removed,
// including the ones a reader of the payload would undo: a JSON answer is
// decoded, and each string in it, field names included, is judged with its
// HTML character references resolved; any other answer is judged with its
// JSON escapes undone and its character references resolved. An answer that
// reveals no key that way keeps the provider's bytes, the literal forms
// scrubbed; one that does is replaced by its decoded text, scrubbed, a JSON
// answer rendered again from its decoded values. As in providerSaid, the scrub
// runs before each decoding step as well as after it.
func (c *Client) scrubAnswer(s string) string {
	s = c.scrub(s)
	if len(c.forms) == 0 {
		return s
	}
	// clean is v as a reader would take it, scrubbed, and whether that
	// differs from the unscrubbed reading, which is when v carries the key.
	clean := func(v string) (string, bool) {
		out := c.scrub(html.UnescapeString(c.scrub(v)))
		return out, out != html.UnescapeString(v)
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil || dec.More() {
		out, revealed := clean(unescapeJSONText(s))
		if !revealed {
			return s
		}
		return out
	}
	var walk func(any) (any, bool)
	walk = func(v any) (any, bool) {
		switch t := v.(type) {
		case string:
			if out, revealed := clean(t); revealed {
				return out, true
			}
			return t, false
		case []any:
			changed := false
			for i, e := range t {
				var ch bool
				t[i], ch = walk(e)
				changed = changed || ch
			}
			return t, changed
		case map[string]any:
			out, changed := make(map[string]any, len(t)), false
			for k, e := range t {
				nk, kc := clean(k)
				if !kc {
					nk = k
				}
				ne, ec := walk(e)
				out[nk] = ne
				changed = changed || kc || ec
			}
			return out, changed
		}
		return v, false
	}
	w, revealed := walk(v)
	if !revealed {
		return s
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(w) != nil {
		out, _ := clean(unescapeJSONText(s))
		return out
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// unfence returns the document inside one surrounding Markdown code fence, or
// s trimmed when it carries none. Models often wrap a JSON answer in one; the
// contract judges what is inside.
func unfence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") || !strings.HasSuffix(t, "```") || len(t) < 6 {
		return t
	}
	nl := strings.IndexByte(t, '\n')
	if nl < 0 {
		return t
	}
	inner := t[nl+1 : len(t)-3]
	if strings.Contains(inner, "```") {
		return t
	}
	return strings.TrimSpace(inner)
}

// providerSaid is a provider's own error message, bounded, sanitised and
// scrubbed of the key: the protocol's error.message when the body carries
// one, else the body itself. A provider may echo the key in any field and in
// any encoding its stack applies, so the text is decoded before the scrub: a
// JSON body is re-rendered from its decoded values (every \u, \/ and other
// escape undone), a body that does not decode (plain text, or JSON the
// provider cut short) has its JSON escapes undone where they stand
// (unescapeJSONText), and HTML character references are resolved, which
// leaves the key, wherever it was, in the one literal form the scrub
// matches. The scrub runs before each decoding step as well as after it, so
// a key written literally is removed before a step could rewrite it, and it
// removes the key's escaped forms besides.
func (c *Client) providerSaid(raw []byte) string {
	var env struct {
		Error json.RawMessage `json:"error"`
	}
	said := ""
	if json.Unmarshal(raw, &env) == nil && len(env.Error) > 0 {
		var obj struct {
			Message string `json:"message"`
		}
		var str string
		switch {
		case json.Unmarshal(env.Error, &obj) == nil && obj.Message != "":
			said = obj.Message
		case json.Unmarshal(env.Error, &str) == nil:
			said = str
		}
	}
	if said == "" {
		var decoded bool
		if said, decoded = decodedBody(raw); !decoded {
			said = unescapeJSONText(c.scrub(said))
		}
	}
	said = c.scrub(html.UnescapeString(c.scrub(said)))
	return termsafe.Sanitize(bound(strings.TrimSpace(said)))
}

// decodedBody is a body as text with its JSON escapes undone: a JSON document
// is decoded and rendered again without escaping anything JSON does not
// require, and decoded is true; anything else is returned as it is, and
// decoded is false.
func decodedBody(raw []byte) (text string, decoded bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil || dec.More() {
		return string(raw), false
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(v) != nil {
		return string(raw), false
	}
	return strings.TrimSuffix(b.String(), "\n"), true
}

// unescapeJSONText undoes, once and wherever it stands, every well-formed
// JSON string escape in s: the short escapes (\" \\ \/ \b \f \n \r \t) and
// \uXXXX in either case of hex, a surrogate pair joined into its rune. It is
// lenient where a decoder is strict: anything that is not a well-formed
// escape (an unknown letter, short or non-hex digits, a trailing backslash,
// a surrogate without its partner) is kept exactly as it is, and so is every
// byte outside an escape, so it never fails and never drops text. That makes
// it total over a body a decoder refuses: a key escaped rune by rune in any
// mix of these forms comes out literal, which is the form the scrub matches.
func unescapeJSONText(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		if r, ok := shortEscape(s[i+1]); ok {
			b.WriteByte(r)
			i += 2
			continue
		}
		r, ok := hex4(s, i)
		if !ok {
			b.WriteByte(s[i])
			i++
			continue
		}
		switch {
		case utf16.IsSurrogate(r):
			// A surrogate is a rune only with its partner; alone it is kept
			// as written.
			if lo, ok := hex4(s, i+6); ok && r < 0xDC00 {
				if joined := utf16.DecodeRune(r, lo); joined != utf8.RuneError {
					b.WriteRune(joined)
					i += 12
					continue
				}
			}
			b.WriteString(s[i : i+6])
		default:
			b.WriteRune(r)
		}
		i += 6
	}
	return b.String()
}

// shortEscape is the byte a JSON short escape \c stands for.
func shortEscape(c byte) (byte, bool) {
	switch c {
	case '"', '\\', '/':
		return c, true
	case 'b':
		return '\b', true
	case 'f':
		return '\f', true
	case 'n':
		return '\n', true
	case 'r':
		return '\r', true
	case 't':
		return '\t', true
	}
	return 0, false
}

// hex4 reads a \uXXXX escape starting at s[i], and reports whether one
// stands there.
func hex4(s string, i int) (rune, bool) {
	if i+6 > len(s) || s[i] != '\\' || s[i+1] != 'u' {
		return 0, false
	}
	v, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
	if err != nil {
		return 0, false
	}
	return rune(v), true
}

// callError is a call that ended without an answer: refused, cancelled,
// never connected, or ended by one of its limits, each said in its own words.
// A limit is read from the cause it cancelled the call with, never from the
// transport's error, which says only that the request was cancelled.
func (c *Client) callError(l *limits, err error) error {
	var ne net.Error
	switch {
	case errors.Is(err, errRedirect):
		return c.fail(c.host + " answered with a redirect, and abcd never follows one: the base URL is pinned, so the key and the brief go nowhere else")
	case errors.Is(l.parent.Err(), context.Canceled):
		return c.fail("the call to " + c.host + " was cancelled")
	case errors.Is(l.parent.Err(), context.DeadlineExceeded) && !ours(context.Cause(l.ctx)):
		// The caller's own deadline, not one of the call's limits.
		return c.fail("the caller's deadline for the call to " + c.host + " passed, so the call is abandoned")
	case neverConnected(err):
		return &unreachableError{msg: c.reachFailure(err).Error()}
	}
	switch context.Cause(l.ctx) {
	case errFirstByte:
		return c.fail(fmt.Sprintf("no answer within %s from %s, so the call is abandoned", c.firstByte, c.host))
	case errIdle:
		return c.fail(fmt.Sprintf("%s sent nothing for %s in the middle of its answer, so the call is abandoned", c.host, c.idle))
	case errEvents:
		switch {
		case !l.begun.Load():
			return c.fail(fmt.Sprintf("no answer within %s from %s, so the call is abandoned", l.events, c.host))
		case l.progress.Load():
			return c.fail(fmt.Sprintf("%s sent only keep-alives for %s in the middle of its answer, so the call is abandoned", c.host, l.events))
		}
		return c.fail(fmt.Sprintf("%s sent only keep-alives for %s and never began its answer, so the call is abandoned", c.host, l.events))
	case errTotal:
		switch {
		case !l.begun.Load():
			return c.fail(fmt.Sprintf("no answer within %s from %s, so the call is abandoned", c.timeout, c.host))
		case l.quiet != nil && !l.progress.Load():
			return c.fail(fmt.Sprintf("%s sent only keep-alives for %s, the call's total limit, and never began its answer, so the call is abandoned", c.host, c.timeout))
		}
		return c.fail(fmt.Sprintf("%s was still answering at the call's total limit of %s, so the call is abandoned", c.host, c.timeout))
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || (errors.As(err, &ne) && ne.Timeout()) {
		return c.fail(fmt.Sprintf("no answer within %s from %s, so the call is abandoned", c.timeout, c.host))
	}
	return c.reachFailure(err)
}

// reachFailure is a transport fault in the transport's own words. The
// *url.Error text names the endpoint (no secret) and the fault; it is
// scrubbed and bounded all the same.
func (c *Client) reachFailure(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return c.fail("could not reach " + c.host + ": " + termsafe.Sanitize(bound(c.scrub(err.Error()))))
}

// neverConnected reports whether err is a failure to connect at all: a name
// that did not resolve, or a dial (to the provider, or to the proxy in front
// of it) that did not complete, so no byte of the request was sent.
func neverConnected(err error) bool {
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return true
	}
	var op *net.OpError
	return errors.As(err, &op) && (op.Op == "dial" || op.Op == "proxyconnect")
}

// fail is every error the client returns: prefixed, and scrubbed of the key a
// last time, whatever built it.
func (c *Client) fail(msg string) error {
	return errors.New("openaiapi: " + c.scrub(msg))
}

// scrub replaces every representation of the key wherever it appears.
func (c *Client) scrub(s string) string { return replaceForms(s, c.forms) }

// Scrub replaces every representation of key in s with "[credential]": the
// key itself and the forms an encoder in a provider's stack or in abcd's own
// error path may give it (JSON with and without HTML escaping, with '/'
// escaped and with non-ASCII escaped, Go's quoting, HTML escaping, URL query
// and path escaping, and the terminal-safe renderings). An empty key scrubs
// nothing. A front door that formats an error built from a provider's text
// scrubs it with this a last time.
func Scrub(s, key string) string { return replaceForms(s, keyForms(key)) }

func replaceForms(s string, forms []string) string {
	for _, f := range forms {
		s = strings.ReplaceAll(s, f, "[credential]")
	}
	return s
}

// keyForms is key and each escaped form of it, distinct and longest first, so
// a longer form is replaced whole before a shorter one could cut into it.
func keyForms(key string) []string {
	if key == "" {
		return nil
	}
	unquote := func(q string) string { return q[1 : len(q)-1] }
	jsonForm := func(escapeHTML bool) string {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(escapeHTML)
		_ = enc.Encode(key)
		return unquote(strings.TrimSuffix(b.String(), "\n"))
	}
	var quoted []string
	for _, j := range []string{jsonForm(true), jsonForm(false)} {
		quoted = append(quoted, j, asciiEscape(j))
	}
	quoted = append(quoted, unquote(strconv.Quote(key)), unquote(strconv.QuoteToASCII(key)))
	forms := []string{key, html.EscapeString(key), url.QueryEscape(key), url.PathEscape(key),
		termsafe.Sanitize(key), termsafe.EncodeHiddenRunes(key)}
	for _, q := range quoted {
		forms = append(forms, q, strings.ReplaceAll(q, "/", `\/`))
	}
	seen := map[string]bool{}
	out := forms[:0]
	for _, f := range forms {
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// asciiEscape writes every non-ASCII rune of a JSON string body as a \u
// escape (a surrogate pair above the Basic Multilingual Plane), the form an
// encoder that emits ASCII only gives it.
func asciiEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		case r > 0xFFFF:
			r -= 0x10000
			fmt.Fprintf(&b, `\u%04x\u%04x`, 0xD800+(r>>10), 0xDC00+(r&0x3FF))
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
	}
	return b.String()
}

// cleanModel bounds a provider-reported model and percent-encodes any hidden
// or control rune, so it is recorded but cannot reorder or escape the record.
func cleanModel(m string) string {
	m = termsafe.EncodeHiddenRunes(m)
	if len(m) > MaxModelBytes {
		cut := MaxModelBytes
		for cut > 0 && m[cut]&0xC0 == 0x80 {
			cut--
		}
		m = m[:cut] + "..."
	}
	return m
}

// bound cuts s to maxEcho bytes on a rune boundary.
func bound(s string) string {
	if len(s) <= maxEcho {
		return s
	}
	cut := maxEcho
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut] + "..."
}
