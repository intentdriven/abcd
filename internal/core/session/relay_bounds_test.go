package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestAConnectionTheRelayClosesStaysCountedUntilItsTransportLeaves: a
// rejoin, a session's end or any other close the relay asks for takes the
// connection out of its session at once, but its place in the connection
// caps is given back only when the transport's final Leave says the socket is
// gone. A close can wait behind a log replay in progress, so uncounting it at
// the close let a guest who rejoined over and over hold sockets nobody counted
// (final review A).
func TestAConnectionTheRelayClosesStaysCountedUntilItsTransportLeaves(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxConnsPerAddr = 3 })
	_, _, code := h.host(t, "tok-ada")
	g1, gp1 := h.join(t, code, "Gus", "198.51.100.7")
	tok := rejoinToken(t, gp1, "joined")

	_, gp2 := h.rejoin(t, tok, "198.51.100.7")
	if !gp1.closed || gp2.last(t, "joined") == nil {
		t.Fatal("the rejoin did not take over the open connection")
	}
	if s := h.r.Stats(); s.Connections != 3 || s.Participants != 2 {
		t.Fatalf("after one rejoin: %+v, want 3 connections (the closed one still held) and 2 participants", s)
	}
	h.rejoin(t, tok, "198.51.100.7")
	// Three sockets from one address are still held: the next is refused,
	// however many times the guest rejoins.
	if _, err := h.r.Connect(&fakePeer{}, "198.51.100.7"); !errors.Is(err, ErrTooManyConnections) {
		t.Fatalf("a fourth connection while three sockets are held: %v, want ErrTooManyConnections", err)
	}
	// The transport ends the first closed socket, and its place, only its,
	// is free.
	g1.Leave()
	g1.Leave()
	if s := h.r.Stats(); s.Connections != 3 {
		t.Fatalf("after the first closed socket ended: %d connections, want 3", s.Connections)
	}
	h.connect(t, &fakePeer{}, "198.51.100.7")
	if _, err := h.r.Connect(&fakePeer{}, "198.51.100.7"); !errors.Is(err, ErrTooManyConnections) {
		t.Fatal("a place was freed twice")
	}
	// A closed connection serves nothing while it waits for its transport.
	gp1.reset()
	send(t, g1, msg{"type": "suggest", "body": msg{"text": "x"}})
	if len(gp1.frames) != 0 {
		t.Fatalf("a closed connection was answered: %v", gp1.texts(t))
	}
}

// TestSuggestionsAreBoundedPerSeat: one participant's suggestions are capped
// in count and in bytes on their seat, so a rejoin does not start them
// afresh and one guest cannot spend the whole session's allowance; the
// session's cap stays as the backstop (final review B).
func TestSuggestionsAreBoundedPerSeat(t *testing.T) {
	d := DefaultLimits()
	if d.MaxSuggestBytesPerSeat != 256<<10 || d.MaxSuggestsPerSeat != 100 || d.MaxSuggestBytesPerSeat >= d.MaxRoomSuggestBytes {
		t.Fatalf("seat defaults = %d bytes, %d suggestions; want 256 KiB, 100, under the session's %d", d.MaxSuggestBytesPerSeat, d.MaxSuggestsPerSeat, d.MaxRoomSuggestBytes)
	}
	delivered := func(p *fakePeer) int {
		n := 0
		for _, m := range p.texts(t) {
			if m["type"] == "suggest" {
				n++
			}
		}
		return n
	}
	t.Run("bytes", func(t *testing.T) {
		h := newHarness(t, func(l *Limits) {
			l.SuggestPerSecond, l.SuggestBurst = 1<<20, 1<<20
			l.MaxSuggestBytesPerSeat = 3000
		})
		_, hp, code := h.host(t, "tok-ada")
		g, gp := h.join(t, code, "Gus", "198.51.100.7")
		tok := rejoinToken(t, gp, "joined")
		body := msg{"replacement": strings.Repeat("y", 1400)}
		send(t, g, msg{"type": "suggest", "body": body})
		send(t, g, msg{"type": "suggest", "body": body})
		send(t, g, msg{"type": "suggest", "body": body})
		if e := gp.last(t, "error"); e == nil || e["code"] != "participant-suggestions-full" || e["kind"] != "suggest" || delivered(hp) != 2 {
			t.Fatalf("past the seat's bytes: %v, delivered %d; want participant-suggestions-full and 2", e, delivered(hp))
		}
		// A rejoin is the same seat.
		g2, gp2 := h.rejoin(t, tok, "198.51.100.7")
		send(t, g2, msg{"type": "suggest", "body": body})
		if e := gp2.last(t, "error"); e == nil || e["code"] != "participant-suggestions-full" || delivered(hp) != 2 {
			t.Fatalf("after a rejoin: %v, delivered %d; want the seat still full", e, delivered(hp))
		}
		// Another guest still has their own allowance.
		o, op := h.join(t, code, "Ida", "198.51.100.8")
		send(t, o, msg{"type": "suggest", "body": body})
		if op.last(t, "suggested") == nil || delivered(hp) != 3 {
			t.Fatalf("another guest was held up: %v", op.texts(t))
		}
	})
	t.Run("count", func(t *testing.T) {
		h := newHarness(t, func(l *Limits) {
			l.SuggestPerSecond, l.SuggestBurst = 1<<20, 1<<20
			l.MaxSuggestsPerSeat = 3
		})
		_, hp, code := h.host(t, "tok-ada")
		g, gp := h.join(t, code, "Gus", "198.51.100.7")
		tok := rejoinToken(t, gp, "joined")
		send(t, g, msg{"type": "suggest", "body": msg{"replacement": "x"}})
		send(t, g, msg{"type": "suggest", "body": msg{"replacement": "x"}})
		g2, gp2 := h.rejoin(t, tok, "198.51.100.7")
		send(t, g2, msg{"type": "suggest", "body": msg{"replacement": "x"}})
		send(t, g2, msg{"type": "suggest", "body": msg{"replacement": "x"}})
		if e := gp2.last(t, "error"); e == nil || e["code"] != "too-many-suggestions" || delivered(hp) != 3 {
			t.Fatalf("the fourth suggestion over a rejoin: %v, delivered %d; want too-many-suggestions and 3", e, delivered(hp))
		}
	})
}

// TestAwarenessBytesAreRatedPerConnection: each connection has its own byte
// bucket for awareness, below the session's rate, so one participant cannot
// drain the session's and hold up everyone else's cursors (final review C).
func TestAwarenessBytesAreRatedPerConnection(t *testing.T) {
	d := DefaultLimits()
	if d.AwarenessConnBytesPerSecond != 32<<10 || d.AwarenessConnBytesBurst != 64<<10 {
		t.Fatalf("connection awareness defaults = %d/s burst %d, want 32 KiB/s burst 64 KiB", d.AwarenessConnBytesPerSecond, d.AwarenessConnBytesBurst)
	}
	if d.AwarenessConnBytesPerSecond >= d.AwarenessBytesPerSecond || d.AwarenessConnBytesBurst >= d.AwarenessBytesBurst {
		t.Fatal("one connection's awareness bucket is not below the session's")
	}
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	frame := Frame{Binary: true, Data: append([]byte{TagAwareness}, make([]byte, 8<<10-1)...)}
	hp.reset()
	for i := 0; i < 9; i++ {
		g.Receive(context.Background(), frame)
	}
	if n := len(hp.binaries()); n != 8 {
		t.Fatalf("one connection's burst forwarded %d 8 KiB frames, want 8 (64 KiB)", n)
	}
	if !rateLimitedAs(t, gp, "awareness") {
		t.Fatal("the frame past the connection's byte burst was not refused")
	}
	// The session's bucket still has room for everyone else.
	gp.reset()
	host.Receive(context.Background(), frame)
	if n := len(gp.binaries()); n != 1 {
		t.Fatal("one guest's awareness held up the host's")
	}
	// A second refills the connection's rate, and no more.
	h.clock.advance(time.Second)
	hp.reset()
	for i := 0; i < 5; i++ {
		g.Receive(context.Background(), frame)
	}
	if n := len(hp.binaries()); n != 4 {
		t.Fatalf("a second later %d 8 KiB frames were forwarded, want 4 (32 KiB)", n)
	}
}

// TestRateLimitedFramesAreAnsweredOncePerSecondWithoutTheLock: an update or
// awareness frame over its connection's rate is answered once a second at
// most, the rest dropped silently, and the decision takes no relay lock, so
// a flood of them cannot contend with every other connection. A rate limit
// still counts as no refusal (final review E).
func TestRateLimitedFramesAreAnsweredOncePerSecondWithoutTheLock(t *testing.T) {
	h := newHarness(t, nil)
	l := h.r.Limits()
	host, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	aw := Frame{Binary: true, Data: []byte{TagAwareness, 1}}
	limited := func(p *fakePeer, kind string) int {
		n := 0
		for _, m := range p.texts(t) {
			if m["type"] == "error" && m["code"] == "rate-limited" && m["kind"] == kind {
				n++
			}
		}
		return n
	}
	gp.reset()
	for i := 0; i < l.AwarenessBurst+5000; i++ {
		g.Receive(context.Background(), aw)
	}
	if n := limited(gp, "awareness"); n != 1 || gp.closed {
		t.Fatalf("5000 frames over the rate were answered %d times (closed=%v), want once and open", n, gp.closed)
	}
	// Past its rate a frame is dropped without the relay's lock.
	h.r.mu.Lock()
	done := make(chan struct{})
	go func() {
		g.Receive(context.Background(), aw)
		close(done)
	}()
	select {
	case <-done:
		h.r.mu.Unlock()
	case <-time.After(2 * time.Second):
		h.r.mu.Unlock()
		<-done
		t.Fatal("a frame over its rate waited for the relay's lock")
	}
	// A second later the next drop is answered again.
	h.clock.advance(time.Second)
	gp.reset()
	for i := 0; i < l.AwarenessPerSecond+10; i++ {
		g.Receive(context.Background(), aw)
	}
	if n := limited(gp, "awareness"); n != 1 {
		t.Fatalf("a second later the drops were answered %d times, want once", n)
	}
	// Updates are answered the same way.
	h.clock.advance(time.Second)
	hp.reset()
	for i := 0; i < l.UpdateBurst+500; i++ {
		host.Receive(context.Background(), update(byte(i)))
	}
	if n := limited(hp, "update"); n != 1 || hp.closed {
		t.Fatalf("500 updates over the rate were answered %d times (closed=%v), want once and open", n, hp.closed)
	}
}

// TestARefusedSuggestionIsAlwaysAnsweredWithItsKind: a guest's page holds
// its one suggestion in flight until an answer comes, so every refusal of a
// suggest names kind "suggest", and a suggest dropped past the message budget
// is answered too, once a second (final review F).
func TestARefusedSuggestionIsAlwaysAnsweredWithItsKind(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	outside := &fakePeer{}
	oc := h.connect(t, outside, "203.0.113.9")
	send(t, oc, msg{"type": "suggest", "body": msg{"text": "x"}})
	if e := outside.last(t, "error"); e == nil || e["code"] != "not-in-session" || e["kind"] != "suggest" {
		t.Fatalf("a suggest outside a session: %v, want not-in-session of kind suggest", e)
	}
	send(t, host, msg{"type": "suggest", "body": msg{"text": "x"}})
	if e := hp.last(t, "error"); e == nil || e["code"] != "not-a-guest" || e["kind"] != "suggest" {
		t.Fatalf("a member's suggest: %v, want not-a-guest of kind suggest", e)
	}
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	send(t, g, msg{"type": "suggest", "body": "not an object"})
	if e := gp.last(t, "error"); e == nil || e["code"] != "bad-suggestion" || e["kind"] != "suggest" {
		t.Fatalf("a malformed suggestion: %v, want bad-suggestion of kind suggest", e)
	}
	// Spend the message budget, then suggest twice.
	for i := 0; i < h.r.Limits().MessageBurst+10; i++ {
		send(t, g, msg{"type": "nonsense"})
	}
	gp.reset()
	send(t, g, msg{"type": "suggest", "body": msg{"text": "x"}})
	send(t, g, msg{"type": "suggest", "body": msg{"text": "x"}})
	texts := gp.texts(t)
	if len(texts) != 1 || texts[0]["code"] != "rate-limited" || texts[0]["kind"] != "suggest" {
		t.Fatalf("two suggests past the message budget were answered %v, want one rate-limited of kind suggest", texts)
	}
}

// TestAnExpiredSessionHoldsNoSessionSlot: a session past its lifetime does
// not count against MaxSessions while it waits for the sweep (final review
// G).
func TestAnExpiredSessionHoldsNoSessionSlot(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxSessions = 1 })
	_, hp, _ := h.host(t, "tok-ada")
	h.clock.advance(h.r.Limits().MaxSessionLife)
	p := h.hostFrom(t, "198.51.100.20")
	if p.last(t, "hosted") == nil {
		t.Fatalf("a new session beside an expired one: %v, want hosted", p.texts(t))
	}
	if c := hp.last(t, "closed"); c == nil || c["reason"] != "expired" {
		t.Fatalf("the expired session's host was not told: %v", hp.last(t, "closed"))
	}
	if s := h.r.Stats(); s.Sessions != 1 {
		t.Fatalf("sessions = %d, want the new one alone", s.Sessions)
	}
}
