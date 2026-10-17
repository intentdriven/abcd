// Package repoprompt is the RepoPrompt reviewer route's adapter (itd-6,
// spc-2609211950427074 scope 2): a client of the Model Context Protocol's
// Streamable HTTP transport, spoken with the standard library alone, that
// sends one review request to the MCP server the person configured for
// RepoPrompt on their own machine and returns the text it answered.
// RepoPrompt answers with whichever model the person configured in it; the
// adapter names none.
//
// The adapter connects and nothing else. It never starts RepoPrompt or any
// process, never reads or writes RepoPrompt's configuration, and never looks
// for a server: the one address it reaches is the URL its caller hands it,
// which the person set (oracle.rp.url in the machine's configuration).
//
// One review is one MCP session: initialize, the initialized notification,
// tools/list (to confirm the tool is offered and read whether it takes
// new_chat), tools/call, and a best-effort DELETE of the session. The
// request is the tool's message argument, in a new chat when the tool takes
// one, so every reviewer is fresh.
//
// The address is pinned to this machine: CheckURL admits http or https to
// localhost or a loopback address only, since what a review request carries
// is the lane's diff and its record; a redirect is never followed and no
// proxy is ever used. Every response is bounded in size, the connection and
// the session's set-up in time (ConnectTimeout), and the call itself in time
// (CallTimeout).
//
// A failure is an *Error whose Kind says why: unreachable (nothing answered
// at the address, or the connection closed before an answer), timeout (a
// bound ran out), refused (the server answered with an error, or does not
// offer the tool as a review needs it), unparsable (the answer is not the
// protocol's) or failed (anything else). Its Detail is abcd's own account
// and never carries the server's text. A context the caller ends is returned
// as its own error, never as an *Error.
package repoprompt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultTool is the RepoPrompt tool a review request is sent to when the
// configuration names none (oracle.rp.tool): the one that sends a message to
// the model the person configured in RepoPrompt.
const DefaultTool = "oracle_send"

// CheckURL admits the MCP server's URL: an absolute http or https URL to
// localhost or a loopback address, with no credentials, query or fragment. A
// refusal never quotes the URL, which may carry a credential.
func CheckURL(raw string) error {
	if raw == "" {
		return errors.New("the URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("it is not a URL")
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return errors.New("it is not an http or https URL")
	case u.User != nil:
		return errors.New("it carries credentials, and the MCP server's address takes none")
	case u.RawQuery != "" || u.ForceQuery:
		return errors.New("it carries a query, and the MCP server's address takes none")
	case u.Fragment != "" || strings.Contains(raw, "#"):
		return errors.New("it carries a fragment, and the MCP server's address takes none")
	case u.Hostname() == "":
		return errors.New("it names no host")
	case !loopback(u.Hostname()):
		return errors.New("it names another machine; a review request (the lane's diff and its record) is sent only to this machine, to localhost or a loopback address")
	}
	return nil
}

// loopback reports whether host is localhost or a loopback address.
func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// The protocol's fixed points.
const (
	// ProtocolVersion is the MCP revision the client asks for. A server that
	// answers with another revision the client speaks is accepted.
	ProtocolVersion = "2025-06-18"
	// DefaultConnectTimeout bounds the connection and the session's set-up
	// (initialize and tools/list).
	DefaultConnectTimeout = 10 * time.Second
	// DefaultCallTimeout bounds the review call itself: a model reading a
	// lane's diff can take many minutes.
	DefaultCallTimeout = 30 * time.Minute
	// MaxResponseBytes bounds every response body or stream.
	MaxResponseBytes = 8 << 20
	// closeTimeout bounds the session's closing DELETE.
	closeTimeout = 2 * time.Second
)

// supported are the protocol revisions whose Streamable HTTP transport this
// client speaks.
var supported = map[string]bool{ProtocolVersion: true, previousVersion: true}

// previousVersion is the revision before ProtocolVersion, whose Streamable
// HTTP transport is the same in every way this client uses.
const previousVersion = "2025-03-26"

// Kind is why a review did not come back.
type Kind string

// The kinds a failure takes. The names match the fallback reasons the loop
// records (internal/core/runner) where they overlap.
const (
	KindUnreachable Kind = "unreachable"
	KindTimeout     Kind = "timeout"
	KindRefused     Kind = "refused"
	KindUnparsable  Kind = "unparsable"
	KindFailed      Kind = "failed"
)

// Error is a review that did not come back: its kind, and abcd's own account
// of it, which is its text.
type Error struct {
	Kind   Kind
	Detail string
}

func (e *Error) Error() string { return e.Detail }

// Reason is the failure's kind as a fallback receipt records it.
func (e *Error) Reason() string { return string(e.Kind) }

func failf(k Kind, format string, a ...any) *Error {
	return &Error{Kind: k, Detail: fmt.Sprintf(format, a...)}
}

// Client reaches one MCP server.
type Client struct {
	url  string
	tool string
	// ConnectTimeout and CallTimeout are the bounds; the defaults when zero.
	ConnectTimeout time.Duration
	CallTimeout    time.Duration
	hc             *http.Client
}

// New returns a client for the MCP server at rawURL (checked by CheckURL)
// that sends a review to tool, DefaultTool when tool is empty.
func New(rawURL, tool string) (*Client, error) {
	if err := CheckURL(rawURL); err != nil {
		return nil, fmt.Errorf("repoprompt: the MCP server's URL: %w", err)
	}
	if tool == "" {
		tool = DefaultTool
	}
	return &Client{url: rawURL, tool: tool, hc: &http.Client{
		// Never proxied (a nil Proxy), and dialled only at a loopback address,
		// whatever a host name resolves to: the address is this machine.
		Transport: &http.Transport{DialContext: dialLoopback(net.DefaultResolver.LookupIPAddr)},
		// The address is pinned: a redirect is never followed.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

// errNotLoopback is a dial refused because the address is not this machine's.
var errNotLoopback = errors.New("the MCP server's address resolves to no loopback address")

// dialLoopback dials only a loopback address: a host name is resolved here,
// through lookup, and only its loopback addresses are dialled, so a name that
// resolves elsewhere (localhost in a hosts file, a search domain) never takes a
// request off the machine.
func dialLoopback(lookup func(ctx context.Context, host string) ([]net.IPAddr, error)) func(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		var ips []net.IP
		if ip := net.ParseIP(host); ip != nil {
			ips = []net.IP{ip}
		} else {
			found, err := lookup(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, a := range found {
				ips = append(ips, a.IP)
			}
		}
		err = errNotLoopback
		for _, ip := range ips {
			if !ip.IsLoopback() {
				continue
			}
			conn, derr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if derr == nil {
				return conn, nil
			}
			err = derr
		}
		return nil, err
	}
}

func (c *Client) connectTimeout() time.Duration {
	if c.ConnectTimeout > 0 {
		return c.ConnectTimeout
	}
	return DefaultConnectTimeout
}

func (c *Client) callTimeout() time.Duration {
	if c.CallTimeout > 0 {
		return c.CallTimeout
	}
	return DefaultCallTimeout
}

// Review sends request to the tool in one session of its own and returns the
// text the tool answered, its text parts joined. ctx bounds everything; a
// ctx the caller ends is returned as its own error, never as an *Error.
func (c *Client) Review(ctx context.Context, request string) (string, error) {
	defer c.hc.CloseIdleConnections()
	s := &session{c: c}
	defer s.close()
	setup, cancel := context.WithTimeout(ctx, c.connectTimeout())
	tl, err := s.open(setup)
	cancel()
	if err != nil {
		return "", classify(ctx, err, "setting up the session")
	}
	args := map[string]any{"message": request}
	if tl.newChat {
		args["new_chat"] = true
	}
	call, cancel := context.WithTimeout(ctx, c.callTimeout())
	defer cancel()
	raw, err := s.request(call, "tools/call", map[string]any{"name": c.tool, "arguments": args})
	if err != nil {
		return "", classify(ctx, err, "the "+c.tool+" call")
	}
	return toolText(c.tool, raw)
}

// toolInfo is what tools/list says of the tool a review is sent to.
type toolInfo struct {
	newChat bool
}

// session is one MCP session over the Streamable HTTP transport.
type session struct {
	c       *Client
	id      string // Mcp-Session-Id, once the server assigns one
	version string // the negotiated protocol revision
	next    int64
}

// open initializes the session and reads the tool's declaration.
func (s *session) open(ctx context.Context) (toolInfo, error) {
	raw, err := s.request(ctx, "initialize", map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "abcd", "version": "1"},
	})
	if err != nil {
		return toolInfo{}, err
	}
	var ans struct {
		ProtocolVersion string                     `json:"protocolVersion"`
		Capabilities    map[string]json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &ans); err != nil || ans.ProtocolVersion == "" {
		return toolInfo{}, failf(KindUnparsable, "the initialize answer is not the protocol's")
	}
	if !supported[ans.ProtocolVersion] {
		return toolInfo{}, failf(KindRefused, "the server answered with an MCP revision this client does not speak (it speaks %s and %s)", ProtocolVersion, previousVersion)
	}
	s.version = ans.ProtocolVersion
	if _, ok := ans.Capabilities["tools"]; !ok {
		return toolInfo{}, failf(KindRefused, "the server offers no tools")
	}
	if err := s.notify(ctx, "notifications/initialized"); err != nil {
		return toolInfo{}, err
	}
	return s.findTool(ctx)
}

// findTool pages through tools/list for the client's tool.
func (s *session) findTool(ctx context.Context) (toolInfo, error) {
	var cursor string
	for range 32 {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := s.request(ctx, "tools/list", params)
		if err != nil {
			return toolInfo{}, err
		}
		var list struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return toolInfo{}, failf(KindUnparsable, "the tools/list answer is not the protocol's")
		}
		for _, t := range list.Tools {
			if t.Name != s.c.tool {
				continue
			}
			if _, ok := t.InputSchema.Properties["message"]; !ok {
				return toolInfo{}, failf(KindRefused, "the %s tool takes no message argument, so it cannot take a review request", s.c.tool)
			}
			_, nc := t.InputSchema.Properties["new_chat"]
			return toolInfo{newChat: nc}, nil
		}
		if list.NextCursor == "" {
			break
		}
		cursor = list.NextCursor
	}
	return toolInfo{}, failf(KindRefused, "the server offers no %s tool", s.c.tool)
}

// rpcError is a JSON-RPC error object.
type rpcError struct {
	Code int `json:"code"`
}

// message is one JSON-RPC message as the transport carries it: a response
// (id with result or error), or a server's own request or notification
// (method), which the client does not serve.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

// request sends one request and returns its result.
func (s *session) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.next++
	id := s.next
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		return nil, failf(KindFailed, "the %s request could not be encoded", method)
	}
	resp, err := s.post(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if method == "initialize" {
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			if !visibleASCII(sid) {
				return nil, failf(KindUnparsable, "the session id the server assigned is not visible ASCII")
			}
			s.id = sid
		}
	}
	if err := status(resp, method); err != nil {
		return nil, err
	}
	m, err := readResponse(resp, id, method)
	if err != nil {
		return nil, err
	}
	if m.Error != nil {
		return nil, failf(KindRefused, "the server answered the %s request with JSON-RPC error %d", method, m.Error.Code)
	}
	if len(m.Result) == 0 {
		return nil, failf(KindUnparsable, "the %s answer carries no result", method)
	}
	return m.Result, nil
}

// notify sends one notification, which the server accepts without an answer.
func (s *session) notify(ctx context.Context, method string) error {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method})
	resp, err := s.post(ctx, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxResponseBytes))
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return status(resp, method)
	}
	return nil
}

// post sends one message to the endpoint.
func (s *session) post(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.c.url, bytes.NewReader(body))
	if err != nil {
		return nil, failf(KindFailed, "the request could not be built")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	s.headers(req)
	resp, err := s.c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *session) headers(req *http.Request) {
	if s.id != "" {
		req.Header.Set("Mcp-Session-Id", s.id)
	}
	if s.version != "" {
		req.Header.Set("MCP-Protocol-Version", s.version)
	}
}

// close ends the session the server assigned an id, best effort.
func (s *session) close() {
	if s.id == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.c.url, nil)
	if err != nil {
		return
	}
	s.headers(req)
	if resp, err := s.c.hc.Do(req); err == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxResponseBytes))
		resp.Body.Close()
	}
}

// status refuses a response that is not a success.
func status(resp *http.Response, method string) error {
	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return failf(KindRefused, "the server answered the %s request with a redirect (HTTP %d), which is never followed", method, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound:
		return failf(KindRefused, "the server answered the %s request with HTTP 404: no MCP endpoint at that path, or the session ended", method)
	case resp.StatusCode >= 400:
		return failf(KindRefused, "the server answered the %s request with HTTP %d", method, resp.StatusCode)
	case resp.StatusCode < 200:
		return failf(KindUnparsable, "the server answered the %s request with HTTP %d", method, resp.StatusCode)
	}
	return nil
}

// readResponse reads the response to request id from a JSON body or an event
// stream.
func readResponse(resp *http.Response, id int64, method string) (message, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]))
	body := &capped{r: resp.Body, left: MaxResponseBytes}
	switch ct {
	case "application/json":
		data, err := io.ReadAll(body)
		if err != nil {
			return message{}, err
		}
		var m message
		if err := json.Unmarshal(data, &m); err != nil || !m.answers(id) {
			return message{}, failf(KindUnparsable, "the %s answer is not a JSON-RPC response to it", method)
		}
		return m, nil
	case "text/event-stream":
		return readStream(body, id, method)
	}
	return message{}, failf(KindUnparsable, "the %s answer is neither JSON nor an event stream", method)
}

// readStream reads server-sent events until one carries the response to id.
// A server's own requests and notifications on the stream are read past.
func readStream(r io.Reader, id int64, method string) (message, error) {
	br := bufio.NewReader(r)
	var data bytes.Buffer
	for {
		line, err := br.ReadString('\n')
		if err != nil && line == "" {
			if errors.Is(err, io.EOF) {
				return message{}, failf(KindUnreachable, "the %s answer's stream ended before the response", method)
			}
			return message{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if data.Len() > 0 {
				var m message
				if json.Unmarshal(data.Bytes(), &m) == nil && m.answers(id) {
					return m, nil
				}
				data.Reset()
			}
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
}

// answers reports whether m is the response to request id.
func (m message) answers(id int64) bool {
	if m.Method != "" || len(m.ID) == 0 {
		return false
	}
	n, err := strconv.ParseInt(string(bytes.TrimSpace(m.ID)), 10, 64)
	return err == nil && n == id && (len(m.Result) > 0 || m.Error != nil)
}

// toolText is the text a tools/call result carries.
func toolText(tool string, raw json.RawMessage) (string, error) {
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", failf(KindUnparsable, "the %s answer is not a tool result", tool)
	}
	if res.IsError {
		return "", failf(KindRefused, "the %s tool answered with an error", tool)
	}
	var parts []string
	for _, c := range res.Content {
		if c.Type == "text" && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	if len(parts) == 0 {
		return "", failf(KindUnparsable, "the %s answer carries no text", tool)
	}
	return strings.Join(parts, "\n"), nil
}

// classify turns a transport error into an *Error. A ctx the caller ended is
// returned as itself.
func classify(ctx context.Context, err error, what string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	var ne net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &ne) && ne.Timeout():
		return failf(KindTimeout, "%s ran out of time", what)
	case errors.Is(err, errTooLarge):
		return failf(KindUnparsable, "%s: the answer is larger than %d bytes", what, MaxResponseBytes)
	case errors.Is(err, errNotLoopback):
		return failf(KindRefused, "%s: the MCP server's address resolves to no address on this machine, so nothing was sent", what)
	}
	var oe *net.OpError
	if errors.As(err, &oe) && oe.Op == "dial" {
		return failf(KindUnreachable, "nothing answered at the MCP server's address: %s", dialCause(oe))
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &oe) {
		return failf(KindUnreachable, "%s: the connection closed before an answer", what)
	}
	return failf(KindFailed, "%s did not complete", what)
}

// dialCause names why a dial failed, in abcd's words.
func dialCause(oe *net.OpError) string {
	msg := oe.Err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return "connection refused"
	case strings.Contains(msg, "no such host"):
		return "no such host"
	}
	return "the connection could not be made"
}

// errTooLarge is what a capped read answers past its bound.
var errTooLarge = errors.New("response too large")

// capped reads at most left bytes, then errTooLarge if more follow.
type capped struct {
	r    io.Reader
	left int64
}

func (c *capped) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var one [1]byte
		if n, err := c.r.Read(one[:]); n == 0 && errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		return 0, errTooLarge
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}

// visibleASCII reports whether s is the visible ASCII a session id is.
func visibleASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return len(s) <= 256
}
