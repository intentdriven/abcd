package repoprompt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
)

func review(t *testing.T, url string, tune func(*Client)) (string, error) {
	t.Helper()
	c, err := New(url, "")
	if err != nil {
		t.Fatal(err)
	}
	if tune != nil {
		tune(c)
	}
	return c.Review(context.Background(), "the request")
}

func wantKind(t *testing.T, err error, k Kind) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Kind != k {
		t.Fatalf("want a %s failure, got %v", k, err)
	}
	if e.Reason() != string(k) {
		t.Fatalf("the reason is the kind: %q", e.Reason())
	}
}

// TestAReviewIsOneMCPSession is criterion 1's adapter half: the request is
// sent to the tool in a session of its own over the Streamable HTTP
// transport, a new chat when the tool takes one, and the tool's text comes
// back; the session's id and the negotiated revision ride every later request,
// and the session is closed.
func TestAReviewIsOneMCPSession(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			f := &mcptest.Server{Stream: stream, Tools: map[string][]string{DefaultTool: {"message", "new_chat"}, "read_file": {"path"}}}
			got, err := review(t, mcptest.Start(t, f), nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != "echo: the request" {
				t.Fatalf("the tool's text comes back: %q", got)
			}
			if m := strings.Join(f.Methods(), ","); m != "initialize,notifications/initialized,tools/list,tools/call" {
				t.Fatalf("one session, in the protocol's order: %v", m)
			}
			calls := f.Calls()
			if len(calls) != 1 || calls[0].Name != DefaultTool {
				t.Fatalf("the review goes to the tool: %v", calls)
			}
			if a := calls[0].Arguments; a["message"] != "the request" || a["new_chat"] != true || len(a) != 2 {
				t.Fatalf("the request is the message, in a new chat: %v", a)
			}
			for i, h := range f.Headers() {
				if !strings.Contains(h.Get("Accept"), "application/json") || !strings.Contains(h.Get("Accept"), "text/event-stream") {
					t.Fatalf("request %d accepts both answer shapes: %v", i, h)
				}
				if i == 0 {
					continue
				}
				if h.Get("Mcp-Session-Id") != mcptest.SessionID || h.Get("Mcp-Protocol-Version") != ProtocolVersion {
					t.Fatalf("request %d carries the session and the revision: %v", i, h)
				}
			}
			if !f.Deleted() {
				t.Fatal("the session is closed when the review is done")
			}
		})
	}
}

// TestNoNewChatIsSentToAToolThatDeclaresNone: an argument the tool does not
// declare is not sent.
func TestNoNewChatIsSentToAToolThatDeclaresNone(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}}
	if _, err := review(t, mcptest.Start(t, f), nil); err != nil {
		t.Fatal(err)
	}
	calls := f.Calls()
	if len(calls) != 1 {
		t.Fatalf("one call: %v", calls)
	}
	if _, ok := calls[0].Arguments["new_chat"]; ok || len(calls[0].Arguments) != 1 {
		t.Fatalf("only the declared argument is sent: %v", calls[0].Arguments)
	}
}

// TestAConfiguredToolIsTheOneCalled: the tool the configuration names is
// called, and a server that does not offer the tool, or offers it without a
// message argument, is refused before any call.
func TestAConfiguredToolIsTheOneCalled(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{"chat_send": {"message"}}}
	c, err := New(mcptest.Start(t, f), "chat_send")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Review(context.Background(), "x"); err != nil || len(f.Calls()) != 1 || f.Calls()[0].Name != "chat_send" {
		t.Fatalf("the configured tool is called: %v %v", f.Calls(), err)
	}
	g := &mcptest.Server{Tools: map[string][]string{"chat_send": {"message"}}}
	_, err = review(t, mcptest.Start(t, g), nil)
	wantKind(t, err, KindRefused)
	if len(g.Calls()) != 0 || !strings.Contains(err.Error(), "offers no "+DefaultTool) {
		t.Fatalf("a tool the server does not offer is never called: %v %v", g.Calls(), err)
	}
	h := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"prompt"}}}
	_, err = review(t, mcptest.Start(t, h), nil)
	wantKind(t, err, KindRefused)
	if len(h.Calls()) != 0 {
		t.Fatalf("a tool that takes no message is never called: %v", h.Calls())
	}
}

// TestNothingAnsweringIsUnreachable is criterion 2's first trigger: an
// address where nothing listens refuses the connection, and the failure says
// so in abcd's words.
func TestNothingAnsweringIsUnreachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	_, err = review(t, "http://"+addr+"/mcp", nil)
	wantKind(t, err, KindUnreachable)
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("the detail says the connection was refused: %v", err)
	}
}

// TestASlowServerIsATimeout is criterion 2's second trigger: the call and the
// session's set-up each run out of time under their own bound.
func TestASlowServerIsATimeout(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, Delay: 2 * time.Second}
	_, err := review(t, mcptest.Start(t, f), func(c *Client) { c.CallTimeout = 100 * time.Millisecond })
	wantKind(t, err, KindTimeout)

	// A server that takes the connection and never answers initialize.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var conns []net.Conn
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			c.Close()
		}
	})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	_, err = review(t, "http://"+ln.Addr().String()+"/mcp", func(c *Client) { c.ConnectTimeout = 100 * time.Millisecond })
	wantKind(t, err, KindTimeout)
}

// TestACallerThatStopsIsNotAFailure: a context the caller ends comes back as
// its own error, so the caller can tell an interrupt from a route that did not
// answer.
func TestACallerThatStopsIsNotAFailure(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, Delay: 2 * time.Second}
	c, err := New(mcptest.Start(t, f), "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = c.Review(ctx, "x")
	var e *Error
	if errors.As(err, &e) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the caller's own deadline comes back as itself: %v", err)
	}
}

// TestTheServersErrorsAreRefusalsInAbcdsWords: a JSON-RPC error, a tool error
// and a server offering no tools are refused, and the detail never carries the
// server's text.
func TestTheServersErrorsAreRefusalsInAbcdsWords(t *testing.T) {
	tools := map[string][]string{DefaultTool: {"message"}}
	for name, f := range map[string]*mcptest.Server{
		"rpc error on the call":   {Tools: tools, RPCError: "tools/call"},
		"rpc error on initialize": {Tools: tools, RPCError: "initialize"},
		"tool error": {Tools: tools, Answer: func(map[string]any) any {
			return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": "the server's own words"}}}
		}},
		"no tools capability":         {NoTools: true},
		"rpc error on the tools list": {Tools: tools, RPCError: "tools/list"},
	} {
		_, err := review(t, mcptest.Start(t, f), nil)
		var e *Error
		if !errors.As(err, &e) || e.Kind != KindRefused {
			t.Fatalf("%s: want refused, got %v", name, err)
		}
		if strings.Contains(err.Error(), "own words") {
			t.Fatalf("%s: the detail is abcd's own: %v", name, err)
		}
	}
}

// TestAnAnswerThatIsNotTheProtocolsIsUnparsable.
func TestAnAnswerThatIsNotTheProtocolsIsUnparsable(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"html": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html></html>")
		},
		"another request's id": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":99,"result":{}}`)
		},
		"too large": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"pad":"`+strings.Repeat("x", MaxResponseBytes)+`"}}`)
		},
	} {
		srv := httptest.NewServer(h)
		_, err := review(t, srv.URL, nil)
		srv.Close()
		var e *Error
		if !errors.As(err, &e) || e.Kind != KindUnparsable {
			t.Fatalf("%s: want unparsable, got %v", name, err)
		}
	}
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, Answer: func(map[string]any) any {
		return map[string]any{"content": []any{map[string]any{"type": "image", "data": "AA=="}}}
	}}
	_, err := review(t, mcptest.Start(t, f), nil)
	wantKind(t, err, KindUnparsable)
}

// TestAnAnswerAtTheBoundIsRead: the size bound refuses only what passes it.
func TestAnAnswerAtTheBoundIsRead(t *testing.T) {
	for _, n := range []int{10, 11} {
		c := &capped{r: strings.NewReader(strings.Repeat("x", n)), left: 10}
		got, err := io.ReadAll(c)
		switch {
		case n == 10 && (err != nil || len(got) != 10):
			t.Fatalf("an answer of exactly the bound is read: %d %v", len(got), err)
		case n == 11 && !errors.Is(err, errTooLarge):
			t.Fatalf("an answer past the bound is refused: %d %v", len(got), err)
		}
	}
}

// TestARedirectIsNeverFollowed: the address is pinned.
func TestARedirectIsNeverFollowed(t *testing.T) {
	reached := false
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	t.Cleanup(other.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)
	_, err := review(t, srv.URL, nil)
	wantKind(t, err, KindRefused)
	if reached {
		t.Fatal("the redirect was followed")
	}
}

// TestNewRefusesAnAddressOffThisMachine: the client is never built for an
// address CheckURL refuses.
func TestNewRefusesAnAddressOffThisMachine(t *testing.T) {
	for _, bad := range []string{"http://example.com/mcp", "http://u:p@127.0.0.1/", "ftp://127.0.0.1/"} {
		if _, err := New(bad, ""); err == nil {
			t.Fatalf("New refuses %q", bad)
		}
	}
}

// TestTheRevisionIsNegotiated: the previous revision, whose transport is the
// same, is accepted and named on every later request; any other is refused
// before a tool is listed.
func TestTheRevisionIsNegotiated(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, ProtocolVersion: "2025-03-26"}
	if _, err := review(t, mcptest.Start(t, f), nil); err != nil {
		t.Fatal(err)
	}
	if h := f.Headers(); h[len(h)-1].Get("Mcp-Protocol-Version") != "2025-03-26" {
		t.Fatalf("the negotiated revision rides the call: %v", h[len(h)-1])
	}
	g := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, ProtocolVersion: "2024-11-05"}
	_, err := review(t, mcptest.Start(t, g), nil)
	wantKind(t, err, KindRefused)
	if strings.Join(g.Methods(), ",") != "initialize" {
		t.Fatalf("nothing follows a revision the client does not speak: %v", g.Methods())
	}
}

// TestASessionIdThatIsNotVisibleASCIIIsRefused: a header value the client
// would send back is held to the transport's shape.
func TestASessionIdThatIsNotVisibleASCIIIsRefused(t *testing.T) {
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, Session: "a session"}
	_, err := review(t, mcptest.Start(t, f), nil)
	wantKind(t, err, KindUnparsable)
}

// TestAStreamThatEndsBeforeTheResponseIsUnreachable, and a missing endpoint
// is refused naming the status.
func TestAStreamThatEndsBeforeTheResponseIsUnreachable(t *testing.T) {
	cut := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\n")
	}))
	t.Cleanup(cut.Close)
	_, err := review(t, cut.URL, nil)
	wantKind(t, err, KindUnreachable)

	missing := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(missing.Close)
	_, err = review(t, missing.URL, nil)
	wantKind(t, err, KindRefused)
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("the refusal names the status: %v", err)
	}
}

// TestOnlyALoopbackAddressIsDialled: a host name is resolved at dial time and
// only its loopback addresses are dialled, so a name that resolves elsewhere
// (localhost in a hosts file, a search domain) never takes a request off the
// machine; localhost itself reaches a server on the loopback address.
func TestOnlyALoopbackAddressIsDialled(t *testing.T) {
	elsewhere := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("10.0.0.7")}}, nil
	}
	if _, err := dialLoopback(elsewhere)(context.Background(), "tcp", "localhost:7777"); !errors.Is(err, errNotLoopback) {
		t.Fatalf("a name resolving off this machine is never dialled: %v", err)
	}
	if _, err := dialLoopback(elsewhere)(context.Background(), "tcp", "10.0.0.7:7777"); !errors.Is(err, errNotLoopback) {
		t.Fatalf("an address off this machine is never dialled: %v", err)
	}
	f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}}
	url := strings.Replace(mcptest.Start(t, f), "127.0.0.1", "localhost", 1)
	if _, err := review(t, url, nil); err != nil {
		t.Fatalf("localhost reaches the loopback server: %v", err)
	}
}

// TestAToolAnswerPastTheBoundIsUnparsable: the review call's answer is held
// to the same bound as every other: one just inside it is read, one just past
// it is refused as unparsable, never truncated into a return.
func TestAToolAnswerPastTheBoundIsUnparsable(t *testing.T) {
	for _, c := range []struct {
		text int
		ok   bool
	}{{MaxResponseBytes - 4096, true}, {MaxResponseBytes, false}} {
		f := &mcptest.Server{Tools: map[string][]string{DefaultTool: {"message"}}, Answer: func(map[string]any) any {
			return mcptest.TextResult(strings.Repeat("x", c.text))
		}}
		got, err := review(t, mcptest.Start(t, f), nil)
		if c.ok {
			if err != nil || len(got) != c.text {
				t.Fatalf("an answer inside the bound is read whole: %d %v", len(got), err)
			}
			continue
		}
		wantKind(t, err, KindUnparsable)
		if !strings.Contains(err.Error(), "larger than") {
			t.Fatalf("the refusal names the bound: %v", err)
		}
	}
}
