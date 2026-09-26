// Package openaiapi is abcd's OpenAI-compatible API adapter
// (itd-2609081951381895): one client over the chat-completions protocol, which
// OpenRouter and a local OpenAI-compatible server both speak, so a provider is
// configuration of this adapter and never code (the intent's Decision 2).
//
// The client asks for the model it is given and nothing else: which models a
// provider may serve, and the vendor denylist above them, are
// internal/core/oracle's to decide before a Client is ever built
// (adr-2609221009491186). The adapter's own guarantees are the network path's:
//
//   - the base URL is pinned per provider block, plain HTTP is admitted only to
//     this machine (a local server), and a redirect is never followed, so a
//     provider cannot move the key or the brief to another host;
//   - every response is bounded (MaxResponseBytes) and every call is bounded in
//     time (WithTimeout), so a provider that floods or stalls is refused rather
//     than waited on;
//   - the key travels only as the Authorization header of a request to the
//     pinned base URL. No error, result or log line carries it: a provider's
//     own error text is bounded, sanitised and scrubbed of the key before it
//     can reach an error, because a provider may echo what it was sent;
//   - a setting the protocol does not take is refused before any call, and the
//     answer is judged by the caller's output contract, the same one the host
//     sub-agent's payload is judged by, so an answer that does not satisfy it is
//     refused rather than used.
//
// It uses net/http and encoding/json alone; it reads no file, no environment
// and no credential store: the caller resolves the key by name through
// internal/core/credential and hands the value in.
package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// DefaultTimeout bounds one call end to end: connecting, sending the brief
	// and reading the whole answer.
	DefaultTimeout = 120 * time.Second
	// MaxResponseBytes bounds a successful answer. A chat completion carrying a
	// verdict is a few kilobytes; 4 MiB refuses a flood without ever refusing a
	// real one.
	MaxResponseBytes = 4 << 20
	// maxErrorBodyBytes bounds how much of a failed call's body is read at all.
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
}

// Client is one provider connection. It holds the key; it never prints it.
type Client struct {
	endpoint string
	key      string
	host     string
	timeout  time.Duration
	hc       *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout bounds one call; d <= 0 keeps DefaultTimeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}

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
	c := &Client{
		endpoint: strings.TrimSuffix(u.String(), "/") + "/chat/completions",
		key:      key,
		host:     u.Host,
		timeout:  DefaultTimeout,
	}
	for _, o := range opts {
		o(c)
	}
	c.hc = &http.Client{
		Timeout: c.timeout,
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
func (c *Client) Complete(ctx context.Context, req Request, contract func([]byte) error) (Result, error) {
	body, err := c.render(req)
	if err != nil {
		return Result{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, c.fail("the request could not be built")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return Result{}, c.transportError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		msg := fmt.Sprintf("%s answered HTTP %d", c.host, resp.StatusCode)
		if said := c.providerSaid(raw); said != "" {
			msg += ": " + said
		}
		return Result{}, c.fail(msg)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return Result{}, c.transportError(err)
	}
	if len(raw) > MaxResponseBytes {
		return Result{}, c.fail(fmt.Sprintf("%s answered with a body larger than %d bytes, so it is refused unread", c.host, MaxResponseBytes))
	}
	return c.decode(raw, req.Model, contract)
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
		"stream": false,
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
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func (c *Client) decode(raw []byte, asked string, contract func([]byte) error) (Result, error) {
	var cc completion
	if err := json.Unmarshal(raw, &cc); err != nil {
		// The decoder's message can quote the body; it is dropped.
		return Result{}, c.fail(c.host + " answered with a body that is not a chat completion (not a JSON object of the protocol's shape)")
	}
	if len(cc.Error) > 0 && string(cc.Error) != "null" {
		msg := c.host + " reported an error in a successful answer"
		if said := c.providerSaid(raw); said != "" {
			msg += ": " + said
		}
		return Result{}, c.fail(msg)
	}
	if len(cc.Choices) == 0 {
		return Result{}, c.fail(c.host + " answered with no choice, so there is no answer to read")
	}
	res := Result{ModelAsked: asked, ModelReported: cleanModel(cc.Model)}
	content := ""
	if p := cc.Choices[0].Message.Content; p != nil {
		content = *p
	}
	res.Content = []byte(unfence(content))
	if contract != nil {
		if err := contract(res.Content); err != nil {
			return Result{}, c.fail("the answer does not satisfy the output contract, so it is refused rather than used: " +
				termsafe.Sanitize(bound(err.Error())))
		}
	}
	return res, nil
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
// one, else the body's first bytes.
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
		said = string(raw)
	}
	said = c.scrub(said)
	return termsafe.Sanitize(bound(strings.TrimSpace(said)))
}

func (c *Client) transportError(err error) error {
	var ne net.Error
	switch {
	case errors.Is(err, errRedirect):
		return c.fail(c.host + " answered with a redirect, and abcd never follows one: the base URL is pinned, so the key and the brief go nowhere else")
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return c.fail(fmt.Sprintf("no answer within %s from %s, so the call is abandoned", c.timeout, c.host))
	case errors.Is(err, context.Canceled):
		return c.fail("the call to " + c.host + " was cancelled")
	}
	// The *url.Error text names the endpoint (no secret) and the transport's
	// own fault; it is scrubbed and bounded all the same.
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return c.fail("could not reach " + c.host + ": " + termsafe.Sanitize(bound(c.scrub(err.Error()))))
}

// fail is every error the client returns: prefixed, and scrubbed of the key a
// last time, whatever built it.
func (c *Client) fail(msg string) error {
	return errors.New("openaiapi: " + c.scrub(msg))
}

// scrub replaces the key wherever it appears.
func (c *Client) scrub(s string) string {
	if c.key == "" {
		return s
	}
	return strings.ReplaceAll(s, c.key, "[credential]")
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
