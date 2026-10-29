package session

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	core "github.com/intentdriven/abcd/internal/core/session"
)

// livePeers is how many sockets the transport is serving now.
func (s *Server) livePeers() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.peers)
}

// bigLog has the host log n updates of size bytes each, and returns the join
// code once the relay has handled them all.
func bigLog(t *testing.T, host *client, n, size int) string {
	t.Helper()
	for i := 0; i < n; i++ {
		b := make([]byte, size)
		b[0] = core.TagUpdate
		b[1] = byte(i)
		host.sendBinary(b)
	}
	host.sendJSON(map[string]string{"type": "rotate"})
	got, _ := host.until("code")
	return got["code"].(string)
}

// joinSlow joins by code, or rejoins by token when code is empty, and reads
// nothing past the joined answer: its log replay waits on a socket nobody
// reads. It returns the client and its rejoin token.
func (f *fixture) joinSlow(t *testing.T, code, token string) (*client, string) {
	t.Helper()
	c := f.dial(t, origin)
	c.ws.SetReadLimit(1 << 20)
	if code != "" {
		c.sendJSON(map[string]string{"type": "join", "code": code, "name": "Gus"})
	} else {
		c.sendJSON(map[string]string{"type": "rejoin", "token": token})
	}
	j, _ := c.until("joined")
	tok, _ := j["rejoin"].(string)
	return c, tok
}

// slowLogFixture is a relay whose log is far larger than a socket's buffers,
// so a replay to a page that reads nothing stays in progress.
func slowLogFixture(t *testing.T, mod func(*Config)) *fixture {
	return newFixture(t, func(c *Config) {
		c.Limits.UpdatesPerSecond, c.Limits.UpdateBurst = 1<<20, 1<<20
		c.Limits.MaxLogBytes = 64 << 20
		if mod != nil {
			mod(c)
		}
	})
}

// TestARejoinDuringASlowReplayStaysBounded reproduces the final review's
// finding A: a guest whose log replay is in progress rejoins on a new socket,
// again and again. The relay closes each old connection, but the close waits
// behind the replay; it used to give the old socket's place back at once, so
// the sockets held grew past every cap, counted nowhere. Now each stays
// counted until its transport ends it, past the per-address cap the next is
// refused, and each old socket ends within the close grace.
func TestARejoinDuringASlowReplayStaysBounded(t *testing.T) {
	const perAddr = 4
	f := slowLogFixture(t, func(c *Config) {
		c.Limits.MaxConnsPerAddr = perAddr
		c.CloseGrace = 500 * time.Millisecond
	})
	host, _ := f.hostSession(t)
	code := bigLog(t, host, 60, 250<<10)
	_, tok := f.joinSlow(t, code, "")
	refused := false
	for i := 0; i < 6; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ws, resp, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {origin}}})
		cancel()
		if err != nil {
			if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("rejoin %d: %v", i+1, err)
			}
			refused = true
			break
		}
		t.Cleanup(func() { _ = ws.CloseNow() })
		c := &client{t: t, ws: ws}
		c.sendJSON(map[string]string{"type": "rejoin", "token": tok})
		j, _ := c.until("joined")
		tok, _ = j["rejoin"].(string)
		if live, counted := f.srv.livePeers(), f.srv.Relay().Stats().Connections; live > perAddr || counted < live {
			t.Fatalf("after rejoin %d: %d sockets served, %d counted; want at most %d and every one counted", i+1, live, counted, perAddr)
		}
	}
	if !refused {
		t.Fatalf("six rejoins in a burst were never refused: the old sockets are not counted (%d served)", f.srv.livePeers())
	}
	// Each old socket ends within the grace, its replay abandoned, and gives
	// its place back: the host and the newest rejoin are left.
	deadline := time.Now().Add(3 * time.Second)
	for f.srv.livePeers() > 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if live, counted := f.srv.livePeers(), f.srv.Relay().Stats().Connections; live != 2 || counted != 2 {
		t.Fatalf("after the close grace: %d sockets served, %d counted; want 2 and 2", live, counted)
	}
}

// TestAClosedConnectionStopsItsReplay: a page reading its replay slowly is
// closed by a rejoin elsewhere; the rest of its replay is dropped and the
// close comes next, well before the log would have ended, without waiting for
// the close grace.
func TestAClosedConnectionStopsItsReplay(t *testing.T) {
	const n = 60
	f := slowLogFixture(t, func(c *Config) { c.CloseGrace = time.Minute })
	host, _ := f.hostSession(t)
	code := bigLog(t, host, n, 250<<10)
	slow, tok := f.joinSlow(t, code, "")
	// A few entries read, then the participant rejoins elsewhere.
	for i := 0; i < 3; i++ {
		if _, b, err := slow.next(); err != nil || b == nil {
			t.Fatalf("replay entry %d: %v", i, err)
		}
	}
	f.joinSlow(t, "", tok)
	start := time.Now()
	read := 3
	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err = slow.ws.Read(ctx)
		cancel()
		if err != nil {
			break
		}
		read++
		time.Sleep(10 * time.Millisecond)
	}
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("the slow page ended with %v, want the relay's policy close", err)
	}
	if read >= n {
		t.Fatalf("the slow page read all %d entries before its close: the replay was not stopped", read)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("the close took %v to arrive", took)
	}
}

// TestAReplayHasATimeBound: a replay that has not finished within its bound
// ends the connection, however slowly the page keeps reading.
func TestAReplayHasATimeBound(t *testing.T) {
	const n = 60
	f := slowLogFixture(t, func(c *Config) { c.ReplayWithin = 300 * time.Millisecond })
	host, _ := f.hostSession(t)
	code := bigLog(t, host, n, 250<<10)
	slow, _ := f.joinSlow(t, code, "")
	start := time.Now()
	read := 0
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := slow.ws.Read(ctx)
		cancel()
		if err != nil {
			break
		}
		read++
		time.Sleep(50 * time.Millisecond)
	}
	if read >= n {
		t.Fatalf("the slow page read all %d entries: the replay had no bound", read)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("the bounded replay took %v to end", took)
	}
	deadline := time.Now().Add(3 * time.Second)
	for f.srv.livePeers() > 1 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if live, counted := f.srv.livePeers(), f.srv.Relay().Stats().Connections; live != 1 || counted != 1 {
		t.Fatalf("after the bounded replay: %d sockets served, %d counted; want the host alone", live, counted)
	}
}
