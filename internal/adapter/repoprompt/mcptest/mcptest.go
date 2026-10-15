// Package mcptest is an MCP server over the Streamable HTTP transport that a
// test owns, on the loopback address, for the RepoPrompt reviewer route's
// tests (itd-6): no test reaches a real RepoPrompt or the network. It answers
// initialize, the initialized notification, tools/list, tools/call and the
// session's closing DELETE, and records what it was sent.
package mcptest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Version is the protocol revision the server answers initialize with.
const Version = "2025-06-18"

// SessionID is the session id the server assigns.
const SessionID = "session-1"

// Call is one tools/call the server took.
type Call struct {
	Name      string
	Arguments map[string]any
}

// Server is the fake. Its zero value offers no tools.
type Server struct {
	// Tools are the tools tools/list offers, each with the names of the
	// properties its input schema declares.
	Tools map[string][]string
	// Answer is the tools/call result; nil answers with the message echoed.
	Answer func(args map[string]any) any
	// Stream answers every request as an event stream rather than a JSON
	// body, with a notification and a keep-alive comment before the response.
	Stream bool
	// RPCError answers the named method with a JSON-RPC error.
	RPCError string
	// Delay holds the tools/call answer this long.
	Delay time.Duration
	// NoTools leaves the tools capability out of initialize.
	NoTools bool
	// ProtocolVersion is the revision initialize answers with; Version when
	// empty.
	ProtocolVersion string
	// Session is the session id initialize assigns; SessionID when empty.
	Session string

	mu      sync.Mutex
	methods []string
	calls   []Call
	headers []http.Header
	deleted bool
}

// Start serves s on the loopback address until the test ends and returns its
// MCP endpoint.
func Start(t testing.TB, s *Server) string {
	t.Helper()
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	return srv.URL + "/mcp"
}

// TextResult is a tools/call result carrying text.
func TextResult(text string) any {
	return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
}

// Methods are the methods the server was sent, in order.
func (s *Server) Methods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.methods...)
}

// Calls are the tools/call requests the server took.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.calls...)
}

// Headers are the headers of each POST, in order.
func (s *Server) Headers() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]http.Header(nil), s.headers...)
}

// Deleted reports whether the client closed the session it was assigned.
func (s *Server) Deleted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleted
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.mu.Lock()
		s.deleted = r.Header.Get("Mcp-Session-Id") == SessionID
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	var m struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(w, "not JSON", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.methods = append(s.methods, m.Method)
	s.headers = append(s.headers, r.Header.Clone())
	s.mu.Unlock()
	if len(m.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if m.Method == s.RPCError {
		s.reply(w, m.ID, nil, map[string]any{"code": -32603, "message": "the server's own words"})
		return
	}
	switch m.Method {
	case "initialize":
		caps := map[string]any{"tools": map[string]any{}}
		if s.NoTools {
			caps = map[string]any{}
		}
		session, version := s.Session, s.ProtocolVersion
		if session == "" {
			session = SessionID
		}
		if version == "" {
			version = Version
		}
		w.Header().Set("Mcp-Session-Id", session)
		s.reply(w, m.ID, map[string]any{"protocolVersion": version, "capabilities": caps,
			"serverInfo": map[string]any{"name": "mcptest", "version": "0"}}, nil)
	case "tools/list":
		tools := []any{}
		for name, props := range s.Tools {
			p := map[string]any{}
			for _, k := range props {
				p[k] = map[string]any{"type": "string"}
			}
			tools = append(tools, map[string]any{"name": name, "inputSchema": map[string]any{"type": "object", "properties": p}})
		}
		s.reply(w, m.ID, map[string]any{"tools": tools}, nil)
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(m.Params, &p)
		s.mu.Lock()
		s.calls = append(s.calls, Call{Name: p.Name, Arguments: p.Arguments})
		s.mu.Unlock()
		if s.Delay > 0 {
			select {
			case <-time.After(s.Delay):
			case <-r.Context().Done():
				return
			}
		}
		res := TextResult("echo: " + fmt.Sprint(p.Arguments["message"]))
		if s.Answer != nil {
			res = s.Answer(p.Arguments)
		}
		s.reply(w, m.ID, res, nil)
	default:
		s.reply(w, m.ID, nil, map[string]any{"code": -32601, "message": "no such method"})
	}
}

func (s *Server) reply(w http.ResponseWriter, id json.RawMessage, result, rpcErr any) {
	msg := map[string]any{"jsonrpc": "2.0", "id": id}
	if rpcErr != nil {
		msg["error"] = rpcErr
	} else {
		msg["result"] = result
	}
	b, _ := json.Marshal(msg)
	if !s.Stream {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	// A server's own notification, then a keep-alive comment, come before
	// the response; the client reads past both.
	fmt.Fprint(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{}}\n\n")
	fmt.Fprint(w, ": keep-alive\n\n")
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", b)
}
