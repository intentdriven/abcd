package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakePeer is the transport's half of one connection, as a test sees it: every
// frame the relay sent, and whether the relay closed it.
type fakePeer struct {
	mu     sync.Mutex
	frames []Frame
	closed bool
	policy bool
	reason string
	full   bool
	logs   int // log replays queued, each one item
	// entered counts the times the relay said the connection entered a
	// session.
	entered int
}

func (p *fakePeer) Send(f Frame) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.full {
		return false
	}
	p.frames = append(p.frames, Frame{Binary: f.Binary, Data: append([]byte(nil), f.Data...)})
	return true
}

func (p *fakePeer) SendLog(entries [][]byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.full {
		return false
	}
	p.logs++
	for _, e := range entries {
		p.frames = append(p.frames, Frame{Binary: true, Data: append([]byte(nil), e...)})
	}
	return true
}

func (p *fakePeer) Entered() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entered++
}

func (p *fakePeer) Close(policy bool, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed, p.policy, p.reason = true, policy, reason
}

// texts decodes every text frame the peer received.
func (p *fakePeer) texts(t *testing.T) []map[string]any {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []map[string]any
	for _, f := range p.frames {
		if f.Binary {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(f.Data, &m); err != nil {
			t.Fatalf("a text frame that is not JSON: %q", f.Data)
		}
		out = append(out, m)
	}
	return out
}

// last returns the last text message of type typ, or nil.
func (p *fakePeer) last(t *testing.T, typ string) map[string]any {
	t.Helper()
	var got map[string]any
	for _, m := range p.texts(t) {
		if m["type"] == typ {
			got = m
		}
	}
	return got
}

func (p *fakePeer) binaries() [][]byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out [][]byte
	for _, f := range p.frames {
		if f.Binary {
			out = append(out, f.Data)
		}
	}
	return out
}

func (p *fakePeer) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frames = nil
}

// fakeVerifier answers for tokens it was told about: token -> login and the
// repositories that login may push to.
type fakeVerifier struct {
	mu     sync.Mutex
	logins map[string]string
	push   map[string]map[string]bool
	calls  int
}

func (v *fakeVerifier) Verify(_ context.Context, token, repo string) (string, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls++
	login, ok := v.logins[token]
	if !ok {
		return "", false, errors.New("bad credentials")
	}
	return login, v.push[login][strings.ToLower(repo)], nil
}

// fakeClock is the injectable clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type harness struct {
	r     *Relay
	clock *fakeClock
	ver   *fakeVerifier
}

func newHarness(t *testing.T, mod func(*Limits)) *harness {
	t.Helper()
	l := DefaultLimits()
	// The tests' fake connections never end on their own, so the connection
	// caps are lifted unless a test sets them.
	l.MaxConnections, l.MaxConnsPerAddr = 1<<20, 1<<20
	if mod != nil {
		mod(&l)
	}
	clock := &fakeClock{now: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)}
	ver := &fakeVerifier{
		logins: map[string]string{"tok-ada": "ada", "tok-bea": "bea", "tok-cy": "cy"},
		push: map[string]map[string]bool{
			"ada": {"acme/brief": true},
			"bea": {"acme/brief": true},
			"cy":  {}, // signed in, but may not change the repository
		},
	}
	return &harness{r: New(Options{Limits: l, Verifier: ver, Now: clock.Now}), clock: clock, ver: ver}
}

// connect admits a connection, failing the test if the relay refuses it.
func (h *harness) connect(t *testing.T, p *fakePeer, addr string) *Conn {
	t.Helper()
	c, err := h.r.Connect(p, addr)
	if err != nil {
		t.Fatalf("connect from %s: %v", addr, err)
	}
	return c
}

func send(t *testing.T, c *Conn, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	c.Receive(context.Background(), Frame{Data: data})
}

type msg map[string]any

// host connects a peer, verifies it with token and opens a session for
// acme/brief, returning the connection, its peer and the join code.
func (h *harness) host(t *testing.T, token string) (*Conn, *fakePeer, string) {
	t.Helper()
	p := &fakePeer{}
	c := h.connect(t, p, "192.0.2.1")
	send(t, c, msg{"type": "verify", "token": token, "repo": "acme/brief"})
	send(t, c, msg{"type": "host", "repo": "acme/brief"})
	hosted := p.last(t, "hosted")
	if hosted == nil {
		t.Fatalf("no hosted answer; got %v", p.texts(t))
	}
	return c, p, hosted["code"].(string)
}

func (h *harness) join(t *testing.T, code, name, addr string) (*Conn, *fakePeer) {
	t.Helper()
	p := &fakePeer{}
	c := h.connect(t, p, addr)
	send(t, c, msg{"type": "join", "code": code, "name": name})
	return c, p
}

func update(b ...byte) Frame { return Frame{Binary: true, Data: append([]byte{TagUpdate}, b...)} }

// attributed is an update as the relay sends it on: tag 2, the sender's
// participant number as four big-endian bytes, then the update's bytes.
func attributed(id string, b ...byte) []byte {
	n, err := strconv.ParseUint(strings.TrimPrefix(id, "p"), 10, 32)
	if err != nil || !strings.HasPrefix(id, "p") {
		panic("not a participant id: " + id)
	}
	return append([]byte{TagAttributedUpdate, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}, b...)
}

// attributedAwareness is an awareness message as the relay sends it on: tag 3,
// the sender's participant number as four big-endian bytes, then the
// message's bytes.
func attributedAwareness(id string, b ...byte) []byte {
	out := attributed(id, b...)
	out[0] = TagAttributedAwareness
	return out
}

func TestHostOpensASessionAndGetsACode(t *testing.T) {
	h := newHarness(t, nil)
	_, p, code := h.host(t, "tok-ada")
	if len(code) != 6 {
		t.Fatalf("code %q is not six characters", code)
	}
	for _, r := range code {
		if !strings.ContainsRune(CodeAlphabet, r) {
			t.Fatalf("code %q has %q, outside the alphabet", code, r)
		}
	}
	roster := p.last(t, "roster")
	if roster == nil {
		t.Fatal("the host got no roster")
	}
	parts := roster["participants"].([]any)
	if len(parts) != 1 || parts[0].(map[string]any)["login"] != "ada" || parts[0].(map[string]any)["host"] != true {
		t.Fatalf("roster = %v, want ada alone, as host", roster)
	}
	if s := h.r.Stats(); s.Sessions != 1 || s.Codes != 1 {
		t.Fatalf("stats = %+v, want one session and one code", s)
	}
}

func TestHostingNeedsAVerifiedMember(t *testing.T) {
	h := newHarness(t, nil)
	for _, c := range []struct {
		name  string
		token string
		want  string
	}{
		{"unverified", "", "not-verified"},
		{"verified without push", "tok-cy", "not-a-member"},
	} {
		p := &fakePeer{}
		conn := h.connect(t, p, "192.0.2.1")
		if c.token != "" {
			send(t, conn, msg{"type": "verify", "token": c.token, "repo": "acme/brief"})
		}
		send(t, conn, msg{"type": "host", "repo": "acme/brief"})
		if e := p.last(t, "error"); e == nil || e["code"] != c.want {
			t.Errorf("%s: error = %v, want %s", c.name, e, c.want)
		}
	}
	if s := h.r.Stats(); s.Sessions != 0 {
		t.Fatalf("stats = %+v, want no session", s)
	}
}

func TestCodesAreDrawnFromTheAlphabet(t *testing.T) {
	r := New(Options{Rand: bytes.NewReader(bytes.Repeat([]byte{0, 30, 247, 248, 255, 31}, 10))})
	code, err := r.newCode()
	if err != nil {
		t.Fatal(err)
	}
	// 0 -> A, 30 -> 9, 247 -> 247%31 = 30 -> 9, 248 and 255 rejected, 31 -> A.
	if code != "A99AA9" {
		t.Fatalf("code = %q, want A99AA9 (bytes 248 and above are rejected, not folded)", code)
	}
}

func TestGuestJoinsByCodeAndReceivesTheReplayedLog(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	host.Receive(context.Background(), update(1, 2, 3))
	host.Receive(context.Background(), update(4, 5))

	// The code is read loosely, as a person types it.
	_, gp := h.join(t, strings.ToLower(code[:3])+"-"+strings.ToLower(code[3:]), "Gus", "198.51.100.7")
	joined := gp.last(t, "joined")
	if joined == nil || joined["role"] != "guest" || joined["name"] != "Gus" || joined["repo"] != "acme/brief" {
		t.Fatalf("joined = %v, want a guest named Gus in acme/brief", joined)
	}
	got := gp.binaries()
	if len(got) != 2 || !bytes.Equal(got[0], attributed(host.id, 1, 2, 3)) || !bytes.Equal(got[1], attributed(host.id, 4, 5)) {
		t.Fatalf("replayed = %v, want the two updates in order", got)
	}
	// The replay sits between joined and synced.
	var order []string
	gp.mu.Lock()
	for _, f := range gp.frames {
		if f.Binary {
			order = append(order, "update")
			continue
		}
		var m map[string]any
		_ = json.Unmarshal(f.Data, &m)
		order = append(order, m["type"].(string))
	}
	gp.mu.Unlock()
	if strings.Join(order, ",") != "joined,update,update,synced,roster" {
		t.Fatalf("order = %v", order)
	}
}

func TestGuestUpdateIsRefusedWhileAMembersIsForwarded(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	guest, gp := h.join(t, code, "Gus", "198.51.100.7")
	hp.reset()
	gp.reset()

	guest.Receive(context.Background(), update(9, 9))
	if e := gp.last(t, "error"); e == nil || e["code"] != "read-only" {
		t.Fatalf("guest's update: error = %v, want read-only", e)
	}
	if len(hp.binaries()) != 0 {
		t.Fatal("a guest's update reached the host")
	}
	if s := h.r.Stats(); s.LogBytes != 0 {
		t.Fatalf("a guest's update was logged: %+v", s)
	}

	host.Receive(context.Background(), update(7))
	if got := gp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributed(host.id, 7)) {
		t.Fatalf("the guest received %v, want the member's update", got)
	}
	if len(hp.binaries()) != 0 {
		t.Fatal("the sender's own update was echoed back")
	}

	// Awareness is forwarded from anyone, attributed, and never logged.
	guest.Receive(context.Background(), Frame{Binary: true, Data: []byte{TagAwareness, 1}})
	if got := hp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributedAwareness(guest.id, 1)) {
		t.Fatalf("the host received %v, want the guest's awareness", got)
	}
	if s := h.r.Stats(); s.LogBytes != LogEntryOverhead+2 {
		t.Fatalf("log bytes = %d, want %d (the one update)", s.LogBytes, LogEntryOverhead+2)
	}
}

func TestVerifyPromotesATokenWithPushAndNotOneWithout(t *testing.T) {
	h := newHarness(t, nil)
	_, hp, code := h.host(t, "tok-ada")
	g1, p1 := h.join(t, code, "Bea", "198.51.100.7")
	g2, p2 := h.join(t, code, "Cy", "198.51.100.8")

	send(t, g1, msg{"type": "verify", "token": "tok-bea"})
	if v := p1.last(t, "verified"); v == nil || v["login"] != "bea" || v["member"] != true {
		t.Fatalf("bea: verified = %v, want a member", v)
	}
	send(t, g2, msg{"type": "verify", "token": "tok-cy"})
	if v := p2.last(t, "verified"); v == nil || v["login"] != "cy" || v["member"] != false {
		t.Fatalf("cy: verified = %v, want not a member", v)
	}
	send(t, g2, msg{"type": "verify", "token": "tok-nobody"})
	if e := p2.last(t, "error"); e == nil || e["code"] != "verify-failed" {
		t.Fatalf("a bad token: error = %v, want verify-failed", e)
	}

	roles := map[string]string{}
	for _, e := range hp.last(t, "roster")["participants"].([]any) {
		m := e.(map[string]any)
		roles[m["name"].(string)] = m["role"].(string)
	}
	if roles["bea"] != "member" || roles["Cy"] != "guest" {
		t.Fatalf("roster roles = %v, want bea a member and Cy a guest", roles)
	}

	// The promoted member's update now goes through; cy's still does not.
	hp.reset()
	g1.Receive(context.Background(), update(1))
	g2.Receive(context.Background(), update(2))
	if got := hp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributed(g1.id, 1)) {
		t.Fatalf("host received %v, want bea's update alone", got)
	}

	// The relay holds no token: nothing it can report carries one.
	if strings.Contains(h.r.debugState(), "tok-") {
		t.Fatal("the relay kept a token after verifying it")
	}
}

func TestVerifyIsBoundedPerConnection(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxVerifies = 2 })
	_, _, code := h.host(t, "tok-ada")
	g, p := h.join(t, code, "Cy", "198.51.100.8")
	for i := 0; i < 3; i++ {
		send(t, g, msg{"type": "verify", "token": "tok-cy"})
	}
	if e := p.last(t, "error"); e == nil || e["code"] != "too-many-verifies" {
		t.Fatalf("error = %v, want too-many-verifies", e)
	}
	if h.ver.calls != 1+2 { // the host's own, then two of cy's
		t.Fatalf("GitHub was asked %d times, want 3", h.ver.calls)
	}
}

func TestSuggestionGoesOnlyToMembersStampedWithTheGuestsName(t *testing.T) {
	h := newHarness(t, nil)
	_, hp, code := h.host(t, "tok-ada")
	m2, mp := h.join(t, code, "", "198.51.100.6")
	send(t, m2, msg{"type": "verify", "token": "tok-bea"})
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	_, op := h.join(t, code, "Other", "198.51.100.9")
	hp.reset()
	mp.reset()
	op.reset()

	send(t, g, msg{"type": "suggest", "name": "ada", "body": msg{"path": "brief.md", "text": "better words"}})
	for who, p := range map[string]*fakePeer{"host": hp, "member": mp} {
		s := p.last(t, "suggest")
		if s == nil || s["from"] != "Gus" {
			t.Fatalf("%s: suggest = %v, want one stamped from Gus", who, s)
		}
		if body := s["body"].(map[string]any); body["text"] != "better words" {
			t.Fatalf("%s: body = %v", who, body)
		}
	}
	if s := op.last(t, "suggest"); s != nil {
		t.Fatalf("another guest received a suggestion: %v", s)
	}
	if a := gp.last(t, "suggested"); a == nil || a["id"] == "" {
		t.Fatalf("the guest got no receipt: %v", gp.texts(t))
	}

	// A suggestion that is not an object is refused; a member does not suggest.
	send(t, g, msg{"type": "suggest", "body": "text"})
	if e := gp.last(t, "error"); e == nil || e["code"] != "bad-suggestion" {
		t.Fatalf("error = %v, want bad-suggestion", e)
	}
	send(t, m2, msg{"type": "suggest", "body": msg{}})
	if e := mp.last(t, "error"); e == nil || e["code"] != "not-a-guest" {
		t.Fatalf("error = %v, want not-a-guest", e)
	}
}

func TestGuestNameIsCleaned(t *testing.T) {
	h := newHarness(t, nil)
	_, _, code := h.host(t, "tok-ada")
	_, p := h.join(t, code, "  Gus‮\n"+strings.Repeat("x", 100), "198.51.100.7")
	name := p.last(t, "joined")["name"].(string)
	if strings.ContainsAny(name, "‮\n") || len([]rune(name)) > 40 || !strings.HasPrefix(name, "Gus") {
		t.Fatalf("name = %q, want it trimmed, without control or format characters, at most 40", name)
	}
	_, p2 := h.join(t, code, "​", "198.51.100.8")
	if n := p2.last(t, "joined")["name"]; n != "Guest" {
		t.Fatalf("an empty name became %q, want Guest", n)
	}
}

func TestWrongCodesAreLimitedPerAddress(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxFailedJoins = 3 })
	_, _, code := h.host(t, "tok-ada")
	var last *fakePeer
	for i := 0; i < 3; i++ {
		_, last = h.join(t, "ZZZZZZ", "Mal", "203.0.113.5")
		if e := last.last(t, "error"); e == nil || e["code"] != "bad-code" {
			t.Fatalf("try %d: error = %v, want bad-code", i, e)
		}
	}
	if !last.closed || !last.policy {
		t.Fatal("the third wrong code did not close the connection")
	}
	// Past the limit, even the right code is refused from that address, and
	// the answer tells nothing about the code.
	_, p := h.join(t, code, "Mal", "203.0.113.5")
	if e := p.last(t, "error"); e == nil || e["code"] != "too-many-tries" || !p.closed {
		t.Fatalf("error = %v, want too-many-tries and a close", e)
	}
	// An IPv6 visitor is counted by its /48: a site holds 65,536 /64s.
	for i := 0; i < 3; i++ {
		h.join(t, "ZZZZZZ", "Mal", "2001:db8:1:"+string(rune('a'+i))+"::1")
	}
	if _, p := h.join(t, code, "Mal", "2001:db8:1:ff::1"); p.last(t, "error")["code"] != "too-many-tries" {
		t.Fatal("rotating networks inside one /48 escaped the limit")
	}
	// Another address is not held up, and the window passes.
	if _, p := h.join(t, code, "Gus", "198.51.100.7"); p.last(t, "joined") == nil {
		t.Fatal("another address was held up by the limit")
	}
	h.clock.advance(DefaultLimits().FailedJoinWindow + time.Second)
	if _, p := h.join(t, code, "Mal", "203.0.113.5"); p.last(t, "joined") == nil {
		t.Fatal("the limit outlived its window")
	}
}

func TestSizeLimits(t *testing.T) {
	h := newHarness(t, func(l *Limits) {
		l.MaxMessageBytes = 64
		l.MaxLogBytes = LogEntryOverhead + 12
		l.MaxParticipants = 2
		l.MaxSessions = 1
	})
	host, hp, code := h.host(t, "tok-ada")

	host.Receive(context.Background(), Frame{Binary: true, Data: append([]byte{TagUpdate}, make([]byte, 64)...)})
	if e := hp.last(t, "error"); e == nil || e["code"] != "too-big" {
		t.Fatalf("an oversized frame: error = %v, want too-big", e)
	}
	host.Receive(context.Background(), update(make([]byte, 8)...))
	host.Receive(context.Background(), update(1, 2))
	if e := hp.last(t, "error"); e == nil || e["code"] != "log-full" {
		t.Fatalf("past the log limit: error = %v, want log-full", e)
	}
	if s := h.r.Stats(); s.LogBytes != LogEntryOverhead+9 {
		t.Fatalf("log bytes = %d, want %d (the one update and its overhead)", s.LogBytes, LogEntryOverhead+9)
	}

	h.join(t, code, "One", "198.51.100.7")
	if _, p := h.join(t, code, "Two", "198.51.100.8"); p.last(t, "error")["code"] != "full" {
		t.Fatal("a third participant got into a session of two")
	}

	// One session per relay: a second host is refused.
	p := &fakePeer{}
	c := h.connect(t, p, "192.0.2.2")
	send(t, c, msg{"type": "verify", "token": "tok-bea", "repo": "acme/brief"})
	send(t, c, msg{"type": "host", "repo": "acme/brief"})
	if e := p.last(t, "error"); e == nil || e["code"] != "too-many-sessions" {
		t.Fatalf("error = %v, want too-many-sessions", e)
	}
}

func TestOnlyTheHostRotatesAndCloses(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	send(t, g, msg{"type": "rotate"})
	send(t, g, msg{"type": "close"})
	if e := gp.last(t, "error"); e == nil || e["code"] != "not-host" {
		t.Fatalf("error = %v, want not-host", e)
	}
	send(t, host, msg{"type": "rotate"})
	next := hp.last(t, "code")
	if next == nil || next["code"] == code {
		t.Fatalf("rotate answered %v, want a new code", next)
	}
	if _, p := h.join(t, code, "Late", "198.51.100.8"); p.last(t, "error")["code"] != "bad-code" {
		t.Fatal("the old code still works after a rotation")
	}
	if _, p := h.join(t, next["code"].(string), "Late", "198.51.100.8"); p.last(t, "joined") == nil {
		t.Fatal("the new code does not work")
	}
}

func TestHostClosePurgesTheSession(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	send(t, host, msg{"type": "register", "client": 12345})
	host.Receive(context.Background(), update(1, 2, 3))
	_, gp := h.join(t, code, "Gus", "198.51.100.7")

	send(t, host, msg{"type": "close"})
	if c := gp.last(t, "closed"); c == nil || !gp.closed {
		t.Fatalf("the guest was not told and closed: %v", gp.texts(t))
	}
	// Nothing of the session is held; the two sockets are counted until
	// their transports end them.
	if s := h.r.Stats(); s != (Stats{Connections: 2}) {
		t.Fatalf("after the host closed, the relay still holds %+v", s)
	}
	if _, p := h.join(t, code, "Late", "198.51.100.8"); p.last(t, "error")["code"] != "bad-code" {
		t.Fatal("a closed session's code still works")
	}
}

func TestIdleSessionIsPurged(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	host.Receive(context.Background(), update(1))
	g, _ := h.join(t, code, "Gus", "198.51.100.7")
	host.Leave()
	h.clock.advance(time.Hour)
	h.r.Sweep()
	if s := h.r.Stats(); s.Sessions != 1 {
		t.Fatal("a session with someone still connected was purged")
	}
	g.Leave()
	h.clock.advance(DefaultLimits().IdleAfter - time.Second)
	h.r.Sweep()
	if s := h.r.Stats(); s.Sessions != 1 {
		t.Fatal("purged before the idle limit")
	}
	h.clock.advance(2 * time.Second)
	h.r.Sweep()
	if s := h.r.Stats(); s != (Stats{}) {
		t.Fatalf("after the idle limit, the relay still holds %+v", s)
	}
}

func TestHostRejoiningByLoginKeepsTheHostRole(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	host.Leave()
	p := &fakePeer{}
	c := h.connect(t, p, "192.0.2.1")
	send(t, c, msg{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	send(t, c, msg{"type": "join", "code": code})
	if j := p.last(t, "joined"); j == nil || j["role"] != "member" || j["host"] != true {
		t.Fatalf("joined = %v, want the host back as a member and host", j)
	}
	send(t, c, msg{"type": "rotate"})
	if p.last(t, "code") == nil {
		t.Fatal("the returning host could not rotate")
	}
}

func TestRegisterCreditsAClientToAMember(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	m, mp := h.join(t, code, "", "198.51.100.6")
	send(t, m, msg{"type": "verify", "token": "tok-bea"})
	g, gp := h.join(t, code, "Gus", "198.51.100.7")

	send(t, host, msg{"type": "register", "client": 4294967295})
	clients := hp.last(t, "roster")["clients"].(map[string]any)
	if clients["4294967295"] != "ada" {
		t.Fatalf("clients = %v, want the id credited to ada", clients)
	}
	send(t, m, msg{"type": "register", "client": 4294967295})
	if e := mp.last(t, "error"); e == nil || e["code"] != "client-taken" {
		t.Fatalf("error = %v, want client-taken", e)
	}
	send(t, g, msg{"type": "register", "client": 7})
	if e := gp.last(t, "error"); e == nil || e["code"] != "read-only" {
		t.Fatalf("a guest registered a client: %v", e)
	}
	send(t, host, msg{"type": "register", "client": -1})
	if e := hp.last(t, "error"); e == nil || e["code"] != "bad-client" {
		t.Fatalf("error = %v, want bad-client", e)
	}
}

func TestMalformedMessagesAreRefused(t *testing.T) {
	h := newHarness(t, nil)
	p := &fakePeer{}
	c := h.connect(t, p, "192.0.2.1")
	for _, raw := range []string{`not json`, `{"type":"dance"}`, `[]`} {
		c.Receive(context.Background(), Frame{Data: []byte(raw)})
	}
	var codes []string
	for _, m := range p.texts(t) {
		codes = append(codes, m["code"].(string))
	}
	if strings.Join(codes, ",") != "bad-message,unknown-message,bad-message" {
		t.Fatalf("codes = %v", codes)
	}
	c.Receive(context.Background(), update(1))
	if e := p.last(t, "error"); e["code"] != "not-in-session" {
		t.Fatalf("error = %v, want not-in-session", e)
	}
	send(t, c, msg{"type": "verify", "token": "tok-ada", "repo": "../etc"})
	if e := p.last(t, "error"); e["code"] != "bad-repo" {
		t.Fatalf("error = %v, want bad-repo", e)
	}
}

func TestSlowPeerIsDropped(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	gp.mu.Lock()
	gp.full = true
	gp.mu.Unlock()
	host.Receive(context.Background(), update(1))
	if !gp.closed {
		t.Fatal("a peer whose queue is full was not closed")
	}
	if s := h.r.Stats(); s.Participants != 1 {
		t.Fatalf("participants = %d, want the slow peer gone", s.Participants)
	}
}

// TestAddressKeyCountsATailnetAddressAlone: Tailscale gives each device one
// address, and every device of a tailnet shares a /64 of its IPv6 range, so a
// tailnet address is its own key while other IPv6 addresses are counted by /64.
func TestAddressKeyCountsATailnetAddressAlone(t *testing.T) {
	for in, want := range map[string]string{
		"fd7a:115c:a1e0::a1":                      "fd7a:115c:a1e0::a1",
		"fd7a:115c:a1e0:ab12:4843:cd96:6258:b240": "fd7a:115c:a1e0:ab12:4843:cd96:6258:b240",
		"100.101.102.103":                         "100.101.102.103",
		"2001:db8:1:2::a":                         "2001:db8:1::/48",
		"2001:db8:1:ffff::a":                      "2001:db8:1::/48",
		"::ffff:192.0.2.1":                        "192.0.2.1",
	} {
		if got := AddressKey(in); got != want {
			t.Errorf("AddressKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// hostFrom connects at addr, verifies as ada and asks to host acme/brief,
// returning the peer.
func (h *harness) hostFrom(t *testing.T, addr string) *fakePeer {
	t.Helper()
	p := &fakePeer{}
	c := h.connect(t, p, addr)
	send(t, c, msg{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	send(t, c, msg{"type": "host", "repo": "acme/brief"})
	return p
}

// rateLimited reports whether p's last error is rate-limited, with p still
// open: an answer over the limit is dropped, not the connection.
func rateLimited(t *testing.T, p *fakePeer) bool {
	t.Helper()
	e := p.last(t, "error")
	return e != nil && e["code"] == "rate-limited" && !p.closed
}

// rateLimitedAs reports whether p's last error is rate-limited for a message
// of kind, so a page can tell a dropped cursor from a dropped suggestion.
func rateLimitedAs(t *testing.T, p *fakePeer, kind string) bool {
	t.Helper()
	e := p.last(t, "error")
	return rateLimited(t, p) && e["kind"] == kind
}

func TestAwarenessIsRateLimitedPerConnection(t *testing.T) {
	h := newHarness(t, nil)
	l := DefaultLimits()
	if l.AwarenessPerSecond != 20 || l.AwarenessBurst != 40 {
		t.Fatalf("awareness default = %d/s burst %d, want 20/s burst 40", l.AwarenessPerSecond, l.AwarenessBurst)
	}
	hc, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	hp.reset()
	gp.reset()
	aw := Frame{Binary: true, Data: []byte{TagAwareness, 1}}
	for i := 0; i < l.AwarenessBurst; i++ {
		g.Receive(context.Background(), aw)
	}
	if n := len(hp.binaries()); n != l.AwarenessBurst {
		t.Fatalf("the burst forwarded %d frames, want %d", n, l.AwarenessBurst)
	}
	if e := gp.last(t, "error"); e != nil {
		t.Fatalf("an error inside the burst: %v", e)
	}
	g.Receive(context.Background(), aw)
	if n := len(hp.binaries()); n != l.AwarenessBurst {
		t.Fatal("a frame past the burst was forwarded")
	}
	if !rateLimitedAs(t, gp, "awareness") {
		t.Fatalf("past the burst: %v, closed=%v; want rate-limited and still open", gp.last(t, "error"), gp.closed)
	}
	// A second refills the rate, and no more.
	h.clock.advance(time.Second)
	hp.reset()
	gp.reset()
	for i := 0; i < l.AwarenessPerSecond+1; i++ {
		g.Receive(context.Background(), aw)
	}
	if n := len(hp.binaries()); n != l.AwarenessPerSecond {
		t.Fatalf("a second later %d frames were forwarded, want %d", n, l.AwarenessPerSecond)
	}
	if !rateLimitedAs(t, gp, "awareness") {
		t.Fatal("the frame past the refill was not refused")
	}
	// Each connection has its own bucket.
	gp.reset()
	hc.Receive(context.Background(), aw)
	if len(gp.binaries()) != 1 {
		t.Fatal("the host's awareness was held up by the guest's")
	}
}

func TestSuggestIsRateLimitedPerConnection(t *testing.T) {
	h := newHarness(t, nil)
	l := DefaultLimits()
	if l.SuggestPerSecond != 1 || l.SuggestBurst != 5 {
		t.Fatalf("suggest default = %d/s burst %d, want 1/s burst 5", l.SuggestPerSecond, l.SuggestBurst)
	}
	_, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	count := func() int {
		n := 0
		for _, m := range hp.texts(t) {
			if m["type"] == "suggest" {
				n++
			}
		}
		return n
	}
	for i := 0; i < l.SuggestBurst+1; i++ {
		send(t, g, msg{"type": "suggest", "body": msg{"text": "x"}})
	}
	if n := count(); n != l.SuggestBurst {
		t.Fatalf("%d suggestions delivered, want the burst of %d", n, l.SuggestBurst)
	}
	if !rateLimitedAs(t, gp, "suggest") {
		t.Fatalf("past the burst: %v, closed=%v; want rate-limited and still open", gp.last(t, "error"), gp.closed)
	}
	h.clock.advance(time.Second)
	send(t, g, msg{"type": "suggest", "body": msg{"text": "y"}})
	if n := count(); n != l.SuggestBurst+1 {
		t.Fatal("a second later the next suggestion was not delivered")
	}
	gp.reset()
	send(t, g, msg{"type": "suggest", "body": msg{"text": "z"}})
	if !rateLimitedAs(t, gp, "suggest") || count() != l.SuggestBurst+1 {
		t.Fatal("the suggestion past the refill was not refused")
	}
}

func TestHostingIsRateLimitedPerAddress(t *testing.T) {
	h := newHarness(t, nil)
	l := DefaultLimits()
	if l.MaxHostsPerAddr != 3 || l.HostWindow != 10*time.Minute {
		t.Fatalf("host default = %d per %v, want 3 per 10m", l.MaxHostsPerAddr, l.HostWindow)
	}
	// A refused host does not count towards the limit.
	p := &fakePeer{}
	c := h.connect(t, p, "203.0.113.5")
	send(t, c, msg{"type": "host", "repo": "acme/brief"})
	if e := p.last(t, "error"); e == nil || e["code"] != "not-verified" {
		t.Fatalf("unverified host: %v", e)
	}
	for i := 0; i < l.MaxHostsPerAddr; i++ {
		if p := h.hostFrom(t, "203.0.113.5"); p.last(t, "hosted") == nil {
			t.Fatalf("session %d was not opened: %v", i, p.texts(t))
		}
	}
	p = h.hostFrom(t, "203.0.113.5")
	if p.last(t, "hosted") != nil || !rateLimited(t, p) {
		t.Fatalf("a fourth session from one address: %v, closed=%v; want rate-limited and still open", p.texts(t), p.closed)
	}
	if h.r.Stats().Sessions != l.MaxHostsPerAddr {
		t.Fatalf("sessions = %d, want %d", h.r.Stats().Sessions, l.MaxHostsPerAddr)
	}
	if p := h.hostFrom(t, "198.51.100.7"); p.last(t, "hosted") == nil {
		t.Fatal("another address was held up by the limit")
	}
	h.clock.advance(l.HostWindow)
	if p := h.hostFrom(t, "203.0.113.5"); p.last(t, "hosted") == nil {
		t.Fatal("the limit outlived its window")
	}
}

// rejoinToken returns the rejoin token a hosted or joined answer carried.
func rejoinToken(t *testing.T, p *fakePeer, typ string) string {
	t.Helper()
	m := p.last(t, typ)
	if m == nil {
		t.Fatalf("no %s answer; got %v", typ, p.texts(t))
	}
	tok, _ := m["rejoin"].(string)
	// 128 random bits, base64url without padding.
	if len(tok) != 22 || strings.Trim(tok, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_") != "" {
		t.Fatalf("%s carried rejoin token %q, want 22 base64url characters", typ, tok)
	}
	return tok
}

func (h *harness) rejoin(t *testing.T, token, addr string) (*Conn, *fakePeer) {
	t.Helper()
	p := &fakePeer{}
	c := h.connect(t, p, addr)
	send(t, c, msg{"type": "rejoin", "token": token})
	return c, p
}

func TestRejoinAfterARotationKeepsTheParticipant(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	hostTok := rejoinToken(t, hp, "hosted")
	host.Receive(context.Background(), update(1, 2))
	guest, gp := h.join(t, code, "Gus", "198.51.100.7")
	guestTok := rejoinToken(t, gp, "joined")
	guestID := gp.last(t, "joined")["id"]
	member, mp := h.join(t, code, "", "198.51.100.8")
	send(t, member, msg{"type": "verify", "token": "tok-bea"})
	memberTok := rejoinToken(t, mp, "joined")
	if hostTok == guestTok || guestTok == memberTok {
		t.Fatal("two participants were given the same rejoin token")
	}

	send(t, host, msg{"type": "rotate"})
	if next := hp.last(t, "code"); next == nil || next["code"] == code {
		t.Fatal("the code did not rotate")
	}
	guest.Leave()
	member.Leave()
	host.Leave()

	// The guest comes back with the token alone: same id, role and name, the
	// log replayed, and no code needed.
	_, gp2 := h.rejoin(t, guestTok, "198.51.100.9")
	j := gp2.last(t, "joined")
	if j == nil || j["role"] != "guest" || j["name"] != "Gus" || j["id"] != guestID || j["host"] == true {
		t.Fatalf("guest rejoined = %v, want guest Gus as %v", j, guestID)
	}
	if got := gp2.binaries(); len(got) != 1 || got[0][0] != TagAttributedUpdate || !bytes.Equal(got[0][5:], []byte{1, 2}) {
		t.Fatalf("the rejoined guest was replayed %v", got)
	}
	if gp2.last(t, "synced") == nil {
		t.Fatal("the rejoined guest was not told it is synced")
	}

	// The member comes back a member, with the login GitHub verified.
	m2, mp2 := h.rejoin(t, memberTok, "198.51.100.10")
	if j := mp2.last(t, "joined"); j == nil || j["role"] != "member" || j["name"] != "bea" || j["host"] == true {
		t.Fatalf("member rejoined = %v, want member bea", j)
	}
	m2.Receive(context.Background(), update(3))
	if e := mp2.last(t, "error"); e != nil {
		t.Fatalf("the rejoined member cannot edit: %v", e)
	}

	// The host comes back the host, and is told the current code.
	h2, hp2 := h.rejoin(t, hostTok, "192.0.2.1")
	if j := hp2.last(t, "joined"); j == nil || j["role"] != "member" || j["host"] != true {
		t.Fatalf("host rejoined = %v, want the host", j)
	}
	if c := hp2.last(t, "code"); c == nil || c["code"] == code {
		t.Fatalf("the rejoined host was not told the current code: %v", c)
	}
	send(t, h2, msg{"type": "rotate"})
	if e := hp2.last(t, "error"); e != nil {
		t.Fatalf("the rejoined host cannot rotate: %v", e)
	}
	roster := hp2.last(t, "roster")["participants"].([]any)
	if len(roster) != 3 {
		t.Fatalf("roster = %v, want the three back", roster)
	}
}

func TestRejoinTakesOverAConnectionStillOpen(t *testing.T) {
	h := newHarness(t, nil)
	_, _, code := h.host(t, "tok-ada")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	tok := rejoinToken(t, gp, "joined")
	_, gp2 := h.rejoin(t, tok, "198.51.100.7")
	if gp2.last(t, "joined") == nil {
		t.Fatalf("rejoin over an open connection: %v", gp2.texts(t))
	}
	if !gp.closed || !gp.policy {
		t.Fatal("the connection the token was taken from was left open")
	}
	roster := gp2.last(t, "roster")["participants"].([]any)
	if len(roster) != 2 {
		t.Fatalf("roster = %v, want the host and one Gus", roster)
	}
}

func TestForgedRejoinTokensAreRefusedAndLimited(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxFailedJoins = 3 })
	_, _, code := h.host(t, "tok-ada")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	real := rejoinToken(t, gp, "joined")

	var last *fakePeer
	for i, forged := range []string{"AAAAAAAAAAAAAAAAAAAAAA", real[:21] + "x", strings.Repeat("y", 5000)} {
		_, last = h.rejoin(t, forged, "203.0.113.5")
		if e := last.last(t, "error"); e == nil || e["code"] != "bad-rejoin" || last.last(t, "joined") != nil {
			t.Fatalf("forged token %d: error = %v, want bad-rejoin", i, e)
		}
	}
	if !last.closed || !last.policy {
		t.Fatal("the third failed rejoin did not close the connection")
	}
	// Past the limit even the real token is refused from that address, as a
	// right code is after too many wrong ones.
	if _, p := h.rejoin(t, real, "203.0.113.5"); p.last(t, "error")["code"] != "too-many-tries" || !p.closed {
		t.Fatal("failed rejoins are not limited like wrong codes")
	}
	// Wrong codes and forged tokens share one count.
	h.join(t, "ZZZZZZ", "Mal", "203.0.113.6")
	h.join(t, "ZZZZZZ", "Mal", "203.0.113.6")
	if _, p := h.rejoin(t, "AAAAAAAAAAAAAAAAAAAAAA", "203.0.113.6"); !p.closed {
		t.Fatal("wrong codes and forged tokens are counted apart")
	}
	if s := h.r.Stats(); s.RejoinTokens != 2 {
		t.Fatalf("stats = %+v, want the two real tokens", s)
	}
}

func TestRejoinTokenDiesWithItsSession(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	hostTok := rejoinToken(t, hp, "hosted")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	tok := rejoinToken(t, gp, "joined")
	send(t, host, msg{"type": "close"})
	if s := h.r.Stats(); s != (Stats{Connections: 2}) {
		t.Fatalf("after the host closed, the relay still holds %+v", s)
	}
	for _, tk := range []string{tok, hostTok} {
		if _, p := h.rejoin(t, tk, "198.51.100.7"); p.last(t, "error")["code"] != "bad-rejoin" {
			t.Fatal("a closed session's rejoin token still works")
		}
	}

	// And with a session purged for idleness.
	host2, hp2, _ := h.host(t, "tok-bea")
	tok2 := rejoinToken(t, hp2, "hosted")
	host2.Leave()
	h.clock.advance(DefaultLimits().IdleAfter + time.Second)
	h.r.Sweep()
	if _, p := h.rejoin(t, tok2, "192.0.2.1"); p.last(t, "error")["code"] != "bad-rejoin" {
		t.Fatal("a purged session's rejoin token still works")
	}
	if s := h.r.Stats(); s.RejoinTokens != 0 {
		t.Fatalf("stats = %+v, want no token held", s)
	}
}

func TestRejoinTokensAreBoundedPerSession(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxParticipants = 2; l.MaxRejoinTokens = 3 })
	_, _, code := h.host(t, "tok-ada")
	var toks []string
	for i := 0; i < 4; i++ {
		g, p := h.join(t, code, "G", "198.51.100.7")
		toks = append(toks, rejoinToken(t, p, "joined"))
		g.Leave()
	}
	if s := h.r.Stats(); s.RejoinTokens != 3 {
		t.Fatalf("stats = %+v, want 3 tokens held", s)
	}
	// The oldest disconnected participant's token went first; the host's,
	// still connected, stays.
	if _, p := h.rejoin(t, toks[0], "198.51.100.7"); p.last(t, "error")["code"] != "bad-rejoin" {
		t.Fatal("the oldest token outlived the bound")
	}
	if _, p := h.rejoin(t, toks[3], "198.51.100.7"); p.last(t, "joined") == nil {
		t.Fatal("the newest token was dropped")
	}
	// A token is never echoed into the relay's state, only its digest.
	if strings.Contains(h.r.debugState(), toks[3]) {
		t.Fatal("the relay holds a rejoin token in the clear")
	}
}

// TestReplayIsOneQueuedItem: a newcomer's replay is handed to the transport
// as one item, whatever the log's length, so a log longer than any send queue
// never drops the newcomer; synced follows it.
func TestReplayIsOneQueuedItem(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.UpdatesPerSecond, l.UpdateBurst = 1<<20, 1<<20 })
	host, _, code := h.host(t, "tok-ada")
	const n = 3000
	for i := 0; i < n; i++ {
		host.Receive(context.Background(), update(byte(i), byte(i>>8)))
	}
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	if gp.logs != 1 {
		t.Fatalf("the replay was queued as %d items, want 1", gp.logs)
	}
	if got := len(gp.binaries()); got != n {
		t.Fatalf("replayed %d entries, want %d", got, n)
	}
	if gp.last(t, "synced") == nil || gp.closed {
		t.Fatalf("the newcomer was not synced, or was closed")
	}
}

func TestDefaultConnectionLimits(t *testing.T) {
	l := DefaultLimits()
	if l.MaxConnections != 256 || l.MaxConnsPerAddr != 16 || l.AdmitWithin != 30*time.Second {
		t.Fatalf("defaults = %d connections, %d per address, %v to enter; want 256, 16, 30s", l.MaxConnections, l.MaxConnsPerAddr, l.AdmitWithin)
	}
}

// TestConnectionsAreCappedRelayWideAndPerAddress: past either cap a new
// connection is refused and nothing else is touched; a connection that ends
// gives its place back.
func TestConnectionsAreCappedRelayWideAndPerAddress(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxConnections, l.MaxConnsPerAddr = 4, 2 })
	a1 := h.connect(t, &fakePeer{}, "203.0.113.5")
	h.connect(t, &fakePeer{}, "203.0.113.5")
	if _, err := h.r.Connect(&fakePeer{}, "203.0.113.5"); !errors.Is(err, ErrTooManyConnections) {
		t.Fatalf("a third connection from one address: %v, want ErrTooManyConnections", err)
	}
	h.connect(t, &fakePeer{}, "2001:db8:1:1::1")
	h.connect(t, &fakePeer{}, "2001:db8:1:2::1")
	if _, err := h.r.Connect(&fakePeer{}, "198.51.100.7"); !errors.Is(err, ErrTooManyConnections) {
		t.Fatalf("a fifth connection on the relay: %v, want ErrTooManyConnections", err)
	}
	if s := h.r.Stats(); s.Connections != 4 {
		t.Fatalf("connections = %d, want 4", s.Connections)
	}
	a1.Leave()
	a1.Leave()
	h.connect(t, &fakePeer{}, "198.51.100.7")
	if s := h.r.Stats(); s.Connections != 4 {
		t.Fatalf("connections = %d after one left and one came, want 4", s.Connections)
	}
}

// TestClosingASessionGivesItsConnectionsBack: the host's close ends every
// connection in the session at once, and each one's place is free again once
// its transport has ended it: a close can wait behind a log replay, and the
// socket is held until then (final review A).
func TestClosingASessionGivesItsConnectionsBack(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	guest, _ := h.join(t, code, "Gus", "198.51.100.7")
	send(t, host, msg{"type": "close"})
	if s := h.r.Stats(); s != (Stats{Connections: 2}) {
		t.Fatalf("after the close the relay holds %+v, want only the two sockets", s)
	}
	host.Leave()
	guest.Leave()
	if s := h.r.Stats(); s != (Stats{}) {
		t.Fatalf("after both transports left the relay holds %+v", s)
	}
}

// TestAConnectionThatEntersNoSessionExpires: a connection that has not
// hosted, joined or rejoined when the transport's grace period ends is closed
// (not as a refusal: the page may simply open another), and one that has is
// left alone.
func TestAConnectionThatEntersNoSessionExpires(t *testing.T) {
	h := newHarness(t, nil)
	idle := &fakePeer{}
	c := h.connect(t, idle, "198.51.100.7")
	send(t, c, msg{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	c.Expire()
	if !idle.closed || idle.policy {
		t.Fatalf("an idle connection: closed=%v policy=%v, want a normal close", idle.closed, idle.policy)
	}
	if e := idle.last(t, "error"); e == nil || e["code"] != "join-timeout" {
		t.Fatalf("error = %v, want join-timeout", e)
	}
	if s := h.r.Stats(); s.Connections != 1 {
		t.Fatalf("connections = %d after the expiry, want the socket held until its transport ends it", s.Connections)
	}
	c.Leave()
	if s := h.r.Stats(); s.Connections != 0 {
		t.Fatalf("connections = %d after the transport left, want 0", s.Connections)
	}

	host, hp, code := h.host(t, "tok-ada")
	guest, gp := h.join(t, code, "Gus", "198.51.100.8")
	host.Expire()
	guest.Expire()
	if hp.closed || gp.closed {
		t.Fatal("a connection in a session expired")
	}
	if hp.entered != 1 || gp.entered != 1 {
		t.Fatalf("entered = %d and %d, want each told once", hp.entered, gp.entered)
	}
}

// TestTheLogChargesEachEntryAndCapsTheirCount: an entry costs its bytes and
// a fixed overhead against the log limit, so a log of tiny updates cannot
// hold many times its limit, and the entries are capped in number too.
func TestTheLogChargesEachEntryAndCapsTheirCount(t *testing.T) {
	if d := DefaultLimits(); d.MaxLogEntries != 50000 || LogEntryOverhead != 64 {
		t.Fatalf("defaults: %d entries, %d bytes of overhead; want 50000 and 64", d.MaxLogEntries, LogEntryOverhead)
	}
	h := newHarness(t, func(l *Limits) { l.MaxLogBytes = 150 })
	host, hp, _ := h.host(t, "tok-ada")
	host.Receive(context.Background(), update(1))
	host.Receive(context.Background(), update(2))
	if e := hp.last(t, "error"); e != nil {
		t.Fatalf("two one-byte updates: error %v", e)
	}
	host.Receive(context.Background(), update(3))
	if e := hp.last(t, "error"); e == nil || e["code"] != "log-full" {
		t.Fatalf("a third one-byte update under 150 bytes: error = %v, want log-full", e)
	}

	h = newHarness(t, func(l *Limits) { l.MaxLogEntries = 3 })
	host, hp, _ = h.host(t, "tok-ada")
	for i := 0; i < 3; i++ {
		host.Receive(context.Background(), update(byte(i)))
	}
	if e := hp.last(t, "error"); e != nil {
		t.Fatalf("three updates: error %v", e)
	}
	host.Receive(context.Background(), update(9))
	if e := hp.last(t, "error"); e == nil || e["code"] != "log-full" {
		t.Fatalf("a fourth entry past the cap: error = %v, want log-full", e)
	}
}

// TestUpdatesAreRateLimitedPerConnection: a member's updates past the bucket
// are dropped with rate-limited, neither logged nor forwarded, and the
// connection stays open.
func TestUpdatesAreRateLimitedPerConnection(t *testing.T) {
	l := DefaultLimits()
	if l.UpdatesPerSecond != 60 || l.UpdateBurst != 200 {
		t.Fatalf("update default = %d/s burst %d, want 60/s burst 200", l.UpdatesPerSecond, l.UpdateBurst)
	}
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	gp.reset()
	for i := 0; i < l.UpdateBurst; i++ {
		host.Receive(context.Background(), update(byte(i)))
	}
	if e := hp.last(t, "error"); e != nil {
		t.Fatalf("within the burst: error %v", e)
	}
	host.Receive(context.Background(), update(1))
	if !rateLimited(t, hp) {
		t.Fatalf("past the burst: want rate-limited with the connection open, got %v", hp.last(t, "error"))
	}
	if n := len(gp.binaries()); n != l.UpdateBurst {
		t.Fatalf("forwarded %d updates, want %d", n, l.UpdateBurst)
	}
	if s := h.r.Stats(); s.LogBytes != l.UpdateBurst*(LogEntryOverhead+2) {
		t.Fatalf("logged %d bytes, want the burst alone", s.LogBytes)
	}
	h.clock.advance(time.Second)
	hp.reset()
	for i := 0; i < l.UpdatesPerSecond; i++ {
		host.Receive(context.Background(), update(byte(i)))
	}
	if e := hp.last(t, "error"); e != nil {
		t.Fatalf("a second later, %d more: error %v", l.UpdatesPerSecond, e)
	}
}

// TestUpdatesAreAttributedToTheirSender: the relay sends every member's
// update on as tag 2 with the sender's participant number, live and in the
// replay; the roster names each author's login, also after they left; and a
// participant cannot send tag 2 itself.
func TestUpdatesAreAttributedToTheirSender(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	bea, bp := h.join(t, code, "", "198.51.100.8")
	send(t, bea, msg{"type": "verify", "token": "tok-bea"})
	beaTok := rejoinToken(t, bp, "joined")
	guest, gp := h.join(t, code, "Gus", "198.51.100.7")
	gp.reset()

	host.Receive(context.Background(), update(1))
	bea.Receive(context.Background(), update(2))
	got := gp.binaries()
	if len(got) != 2 || !bytes.Equal(got[0], attributed(host.id, 1)) || !bytes.Equal(got[1], attributed(bea.id, 2)) {
		t.Fatalf("the guest received %v, want each update attributed to its sender", got)
	}
	authors := func(p *fakePeer) map[string]any {
		r := p.last(t, "roster")
		a, _ := r["authors"].(map[string]any)
		return a
	}
	num := func(id string) string { return strings.TrimPrefix(id, "p") }
	if a := authors(gp); a[num(host.id)] != "ada" || a[num(bea.id)] != "bea" || len(a) != 2 {
		t.Fatalf("roster authors = %v, want ada and bea by participant number", a)
	}

	// A participant cannot claim another's attribution.
	forged := Frame{Binary: true, Data: attributed(host.id, 9)}
	bea.Receive(context.Background(), forged)
	if e := bp.last(t, "error"); e == nil || e["code"] != "unknown-tag" {
		t.Fatalf("a client's tag 2: error = %v, want unknown-tag", e)
	}

	// Bea leaves; a newcomer's replay still names her, and so does the roster.
	beaID := bea.id
	bea.Leave()
	_, np := h.join(t, code, "Nia", "198.51.100.9")
	replay := np.binaries()
	if len(replay) != 2 || !bytes.Equal(replay[1], attributed(beaID, 2)) {
		t.Fatalf("the newcomer's replay = %v", replay)
	}
	if a := authors(np); a[num(beaID)] != "bea" {
		t.Fatalf("roster authors after bea left = %v", a)
	}

	// Rejoining keeps the participant number, so the attribution holds.
	back, _ := h.rejoin(t, beaTok, "198.51.100.8")
	if back.id != beaID {
		t.Fatalf("bea rejoined as %s, want %s", back.id, beaID)
	}
	gp.reset()
	back.Receive(context.Background(), update(3))
	if got := gp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributed(beaID, 3)) {
		t.Fatalf("after the rejoin the guest received %v", got)
	}
	_ = guest
	_ = hp
}

// TestAwarenessIsAttributedToItsSender: the relay sends every awareness
// message on as tag 3 with the sender's participant number, exactly as it
// sends an update on as tag 2, from a guest as from a member; it logs none of
// them; and a participant cannot send tag 3 itself.
func TestAwarenessIsAttributedToItsSender(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	guest, gp := h.join(t, code, "Gus", "198.51.100.7")
	gusTok := rejoinToken(t, gp, "joined")
	hp.reset()
	gp.reset()

	guest.Receive(context.Background(), Frame{Binary: true, Data: []byte{TagAwareness, 4, 5}})
	host.Receive(context.Background(), Frame{Binary: true, Data: []byte{TagAwareness, 6}})
	if got := hp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributedAwareness(guest.id, 4, 5)) {
		t.Fatalf("the host received %v, want the guest's awareness attributed to %s", got, guest.id)
	}
	if got := gp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributedAwareness(host.id, 6)) {
		t.Fatalf("the guest received %v, want the host's awareness attributed to %s", got, host.id)
	}
	if s := h.r.Stats(); s.LogBytes != 0 {
		t.Fatalf("awareness was logged: %+v", s)
	}

	// A participant cannot claim another's attribution.
	hp.reset()
	guest.Receive(context.Background(), Frame{Binary: true, Data: attributedAwareness(host.id, 9)})
	if e := gp.last(t, "error"); e == nil || e["code"] != "unknown-tag" {
		t.Fatalf("a client's tag 3: error = %v, want unknown-tag", e)
	}
	if got := hp.binaries(); len(got) != 0 {
		t.Fatalf("a client's tag 3 was forwarded: %v", got)
	}

	// Rejoining keeps the participant number, so the attribution holds.
	gusID := guest.id
	guest.Leave()
	back, _ := h.rejoin(t, gusTok, "198.51.100.7")
	hp.reset()
	back.Receive(context.Background(), Frame{Binary: true, Data: []byte{TagAwareness, 7}})
	if got := hp.binaries(); len(got) != 1 || !bytes.Equal(got[0], attributedAwareness(gusID, 7)) {
		t.Fatalf("after the rejoin the host received %v", got)
	}
}

// TestRosterNamesTheHostsLogin: the roster names the login that opened the
// session, also while the host is not connected, so a page can tell the
// host's writes from anyone else's.
func TestRosterNamesTheHostsLogin(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	if r := hp.last(t, "roster"); r == nil || r["host"] != "ada" {
		t.Fatalf("the host's roster = %v, want host ada", r)
	}
	bea, bp := h.join(t, code, "", "198.51.100.8")
	send(t, bea, msg{"type": "verify", "token": "tok-bea"})
	host.Leave()
	if r := bp.last(t, "roster"); r == nil || r["host"] != "ada" {
		t.Fatalf("after the host left, the roster = %v, want host ada", r)
	}
	_, np := h.join(t, code, "Nia", "198.51.100.9")
	if r := np.last(t, "roster"); r == nil || r["host"] != "ada" {
		t.Fatalf("a newcomer's roster while the host is away = %v, want host ada", r)
	}
}

// TestAuthorsAreBoundedPerSession: a session names at most MaxCredits
// authors; past it a new author's update is refused and not logged.
func TestAuthorsAreBoundedPerSession(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.MaxCredits = 1 })
	host, _, code := h.host(t, "tok-ada")
	host.Receive(context.Background(), update(1))
	bea, bp := h.join(t, code, "", "198.51.100.8")
	send(t, bea, msg{"type": "verify", "token": "tok-bea"})
	bea.Receive(context.Background(), update(2))
	if e := bp.last(t, "error"); e == nil || e["code"] != "too-many-authors" {
		t.Fatalf("a second author past the bound: error = %v, want too-many-authors", e)
	}
	if s := h.r.Stats(); s.LogBytes != LogEntryOverhead+2 {
		t.Fatalf("log bytes = %d, want the first update alone", s.LogBytes)
	}
}

// TestWrongCodesAreBudgetedRelayWide: past the relay-wide budget of wrong
// codes, joining by code pauses for everybody until the window turns. During
// the pause every code gets the same answer, right or wrong, and counts for
// its own address but not towards the pause. A rejoin token and a verified
// member are not held up.
func TestWrongCodesAreBudgetedRelayWide(t *testing.T) {
	if d := DefaultLimits(); d.RelayFailedJoins != 600 || d.FailedJoinWindow != 10*time.Minute {
		t.Fatalf("defaults: %d wrong codes in %v, want 600 in 10m", d.RelayFailedJoins, d.FailedJoinWindow)
	}
	h := newHarness(t, func(l *Limits) {
		l.RelayFailedJoins = 5
		l.MaxFailedJoins = 3
	})
	_, _, code := h.host(t, "tok-ada")
	_, gp := h.join(t, code, "Gus", "198.51.100.200")
	gusTok := rejoinToken(t, gp, "joined")
	for i := 0; i < 5; i++ {
		_, p := h.join(t, "ZZZZZZ", "Mal", "203.0.113."+strconv.Itoa(i+1))
		if e := p.last(t, "error"); e == nil || e["code"] != "bad-code" {
			t.Fatalf("wrong code %d: %v, want bad-code", i, e)
		}
	}
	// The pause: a right code and a wrong one get the same answer.
	for _, c := range []string{code, "ZZZZZZ"} {
		_, p := h.join(t, c, "Ann", "198.51.100.1")
		if e := p.last(t, "error"); e == nil || e["code"] != "paused" || p.last(t, "joined") != nil {
			t.Fatalf("joining with %s during the pause: %v, want paused", c, p.texts(t))
		}
	}
	// Each try during the pause counts for its own address.
	_, p := h.join(t, code, "Ann", "198.51.100.1")
	if !p.closed {
		t.Fatal("an address past its own limit during the pause stayed open")
	}
	// A rejoin token and a verified member carry on.
	if _, rp := h.rejoin(t, gusTok, "198.51.100.200"); rp.last(t, "joined") == nil {
		t.Fatalf("a rejoin during the pause: %v", rp.texts(t))
	}
	mp := &fakePeer{}
	m := h.connect(t, mp, "198.51.100.50")
	send(t, m, msg{"type": "verify", "token": "tok-bea", "repo": "acme/brief"})
	send(t, m, msg{"type": "join", "code": code})
	if mp.last(t, "joined") == nil {
		t.Fatalf("a verified member during the pause: %v", mp.texts(t))
	}
	// The tries during the pause did not extend it: once the window turns,
	// joining by code works again.
	h.clock.advance(DefaultLimits().FailedJoinWindow)
	if _, p := h.join(t, code, "Bo", "198.51.100.2"); p.last(t, "joined") == nil {
		t.Fatalf("after the pause: %v", p.texts(t))
	}
}

// TestForgedRejoinTokensDoNotPauseJoining: wrong rejoin tokens count for
// their address alone, never towards the relay-wide budget.
func TestForgedRejoinTokensDoNotPauseJoining(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.RelayFailedJoins = 2 })
	_, _, code := h.host(t, "tok-ada")
	for i := 0; i < 4; i++ {
		h.rejoin(t, "forged-token-"+strconv.Itoa(i), "203.0.113."+strconv.Itoa(i+1))
	}
	if _, p := h.join(t, code, "Gus", "198.51.100.7"); p.last(t, "joined") == nil {
		t.Fatalf("joining after forged rejoin tokens: %v", p.texts(t))
	}
}

// TestOnlyServedRepositoriesAreVerifiedOrHosted: a relay serving a list of
// repositories refuses to verify against, or host, any other, before asking
// GitHub anything; an owner on the list serves every repository it holds.
func TestOnlyServedRepositoriesAreVerifiedOrHosted(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)}
	ver := &fakeVerifier{
		logins: map[string]string{"tok-ada": "ada", "tok-mal": "mal"},
		push: map[string]map[string]bool{
			"ada": {"acme/brief": true, "team/notes": true},
			"mal": {"mal/own": true},
		},
	}
	r := New(Options{Verifier: ver, Now: clock.Now, Repos: []string{"Acme/Brief"}, Owners: []string{"TEAM"}})
	if r.Serves("mal/own") || !r.Serves("acme/brief") || !r.Serves("team/notes") || r.Serves("acme/other") {
		t.Fatal("Serves does not follow the lists")
	}

	mp := &fakePeer{}
	mal, _ := r.Connect(mp, "203.0.113.5")
	send(t, mal, msg{"type": "verify", "token": "tok-mal", "repo": "mal/own"})
	if e := mp.last(t, "error"); e == nil || e["code"] != "repo-not-served" {
		t.Fatalf("verify against an unserved repository: %v, want repo-not-served", e)
	}
	if ver.calls != 0 {
		t.Fatalf("GitHub was asked %d times about an unserved repository", ver.calls)
	}
	send(t, mal, msg{"type": "host", "repo": "mal/own"})
	if e := mp.last(t, "error"); e == nil || e["code"] != "repo-not-served" {
		t.Fatalf("host an unserved repository: %v, want repo-not-served", e)
	}
	if s := r.Stats(); s.Sessions != 0 {
		t.Fatalf("sessions = %d, want none", s.Sessions)
	}

	for _, repo := range []string{"acme/brief", "Team/Notes"} {
		p := &fakePeer{}
		c, _ := r.Connect(p, "192.0.2.1")
		send(t, c, msg{"type": "verify", "token": "tok-ada", "repo": repo})
		send(t, c, msg{"type": "host", "repo": repo})
		if p.last(t, "hosted") == nil {
			t.Fatalf("hosting %s: %v", repo, p.texts(t))
		}
	}

	// No list at all serves every repository.
	open := New(Options{Verifier: ver, Now: clock.Now})
	if !open.Serves("mal/own") {
		t.Fatal("a relay with no list does not serve every repository")
	}
}

func TestValidOwner(t *testing.T) {
	for _, s := range []string{"acme", "a", "Acme-Co", "a1"} {
		if !ValidOwner(s) {
			t.Errorf("%q refused", s)
		}
	}
	for _, s := range []string{"", "-acme", "acme/brief", "ac me", "a_b", strings.Repeat("a", 40)} {
		if ValidOwner(s) {
			t.Errorf("%q accepted", s)
		}
	}
}

// TestAwarenessFramesAreCappedInSize: an awareness frame is a cursor and a
// name, a few hundred bytes, so one over MaxAwarenessBytes is refused and
// never forwarded (security review 2, item 1).
func TestAwarenessFramesAreCappedInSize(t *testing.T) {
	if d := DefaultLimits(); d.MaxAwarenessBytes != 8<<10 {
		t.Fatalf("MaxAwarenessBytes = %d, want 8 KiB", d.MaxAwarenessBytes)
	}
	h := newHarness(t, nil)
	_, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	hp.reset()
	g.Receive(context.Background(), Frame{Binary: true, Data: append([]byte{TagAwareness}, make([]byte, 8<<10)...)})
	if e := gp.last(t, "error"); e == nil || e["code"] != "too-big" {
		t.Fatalf("an awareness frame over 8 KiB: %v, want too-big", e)
	}
	if n := len(hp.binaries()); n != 0 {
		t.Fatalf("an oversized awareness frame was forwarded %d times", n)
	}
	g.Receive(context.Background(), Frame{Binary: true, Data: append([]byte{TagAwareness}, make([]byte, 8<<10-1)...)})
	if n := len(hp.binaries()); n != 1 {
		t.Fatalf("an awareness frame of exactly 8 KiB was forwarded %d times, want once", n)
	}
}

// TestAwarenessBytesAreRatedPerRoom: every participant's awareness together
// is held to a byte rate per session, so many connections each inside their
// own frame rate cannot flood the room's queues.
func TestAwarenessBytesAreRatedPerRoom(t *testing.T) {
	d := DefaultLimits()
	if d.AwarenessBytesPerSecond != 128<<10 || d.AwarenessBytesBurst != 512<<10 {
		t.Fatalf("room awareness defaults = %d/s burst %d, want 128 KiB/s burst 512 KiB", d.AwarenessBytesPerSecond, d.AwarenessBytesBurst)
	}
	h := newHarness(t, func(l *Limits) {
		l.AwarenessPerSecond, l.AwarenessBurst = 1<<20, 1<<20
		l.AwarenessBytesPerSecond, l.AwarenessBytesBurst = 4096, 4096*4
	})
	host, hp, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	frame := Frame{Binary: true, Data: append([]byte{TagAwareness}, make([]byte, 4095)...)}
	hp.reset()
	for i := 0; i < 4; i++ {
		g.Receive(context.Background(), frame)
	}
	if n := len(hp.binaries()); n != 4 {
		t.Fatalf("the room's burst forwarded %d frames, want 4", n)
	}
	g.Receive(context.Background(), frame)
	if !rateLimited(t, gp) || len(hp.binaries()) != 4 {
		t.Fatal("a frame past the room's byte burst was not refused")
	}
	// The budget is the room's: the host is held to it too.
	host.Receive(context.Background(), frame)
	if !rateLimited(t, hp) {
		t.Fatal("the host's awareness was not held to the room's byte rate")
	}
	h.clock.advance(time.Second)
	gp.reset()
	hp.reset()
	g.Receive(context.Background(), frame)
	if n := len(hp.binaries()); n != 1 {
		t.Fatalf("a second later %d frames were forwarded, want 1", n)
	}
}

// TestSuggestionsAreBoundedInSizeCountAndRoomBytes: a guest's suggestions
// end up in the session's log (a member's page records each), so each is
// capped in bytes, each connection in count, and each session in total bytes
// (security review 2, item 2).
func TestSuggestionsAreBoundedInSizeCountAndRoomBytes(t *testing.T) {
	d := DefaultLimits()
	if d.MaxSuggestBytes != 70<<10 || d.MaxSuggestsPerSeat != 100 || d.MaxRoomSuggestBytes != 1<<20 {
		t.Fatalf("suggestion defaults = %d bytes, %d a connection, %d a room; want 70 KiB, 100, 1 MiB",
			d.MaxSuggestBytes, d.MaxSuggestsPerSeat, d.MaxRoomSuggestBytes)
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
	t.Run("size", func(t *testing.T) {
		h := newHarness(t, nil)
		_, hp, code := h.host(t, "tok-ada")
		g, gp := h.join(t, code, "Gus", "198.51.100.7")
		big := strings.Repeat("x", 70<<10)
		send(t, g, msg{"type": "suggest", "body": msg{"replacement": big}})
		if e := gp.last(t, "error"); e == nil || e["code"] != "suggestion-too-big" || delivered(hp) != 0 {
			t.Fatalf("a suggestion over 70 KiB: %v, delivered %d", e, delivered(hp))
		}
		send(t, g, msg{"type": "suggest", "body": msg{"replacement": big[:60<<10]}})
		if delivered(hp) != 1 {
			t.Fatal("a suggestion under the cap was not delivered")
		}
	})
	t.Run("count", func(t *testing.T) {
		h := newHarness(t, func(l *Limits) {
			l.SuggestPerSecond, l.SuggestBurst = 1<<20, 1<<20
			l.MaxSuggestsPerSeat = 3
		})
		_, hp, code := h.host(t, "tok-ada")
		g, gp := h.join(t, code, "Gus", "198.51.100.7")
		for i := 0; i < 4; i++ {
			send(t, g, msg{"type": "suggest", "body": msg{"replacement": "x"}})
		}
		if e := gp.last(t, "error"); e == nil || e["code"] != "too-many-suggestions" || delivered(hp) != 3 {
			t.Fatalf("the fourth suggestion: %v, delivered %d, want too-many-suggestions and 3", e, delivered(hp))
		}
	})
	t.Run("room", func(t *testing.T) {
		h := newHarness(t, func(l *Limits) {
			l.SuggestPerSecond, l.SuggestBurst = 1<<20, 1<<20
			l.MaxRoomSuggestBytes = 3000
		})
		_, hp, code := h.host(t, "tok-ada")
		g1, _ := h.join(t, code, "Gus", "198.51.100.7")
		g2, gp2 := h.join(t, code, "Ida", "198.51.100.8")
		body := msg{"replacement": strings.Repeat("y", 1400)}
		send(t, g1, msg{"type": "suggest", "body": body})
		send(t, g2, msg{"type": "suggest", "body": body})
		send(t, g2, msg{"type": "suggest", "body": body})
		if e := gp2.last(t, "error"); e == nil || e["code"] != "suggestions-full" || delivered(hp) != 2 {
			t.Fatalf("past the room's suggestion bytes: %v, delivered %d, want suggestions-full and 2", e, delivered(hp))
		}
	})
}

// TestRefusedMessagesAreBudgetedPerConnection: every control message, and
// every binary frame the relay refuses, spends a token of one bucket per
// connection; past it a message is dropped unanswered without taking the
// relay's lock, and a connection refused MaxRefusals times is closed
// (security review 2, item 7).
func TestRefusedMessagesAreBudgetedPerConnection(t *testing.T) {
	d := DefaultLimits()
	if d.MessagesPerSecond != 50 || d.MessageBurst != 100 || d.MaxRefusals != 1000 {
		t.Fatalf("message defaults = %d/s burst %d, close after %d; want 50/s, 100, 1000", d.MessagesPerSecond, d.MessageBurst, d.MaxRefusals)
	}
	h := newHarness(t, nil)
	_, _, code := h.host(t, "tok-ada")
	g, gp := h.join(t, code, "Gus", "198.51.100.7")
	gp.reset()
	errs := func() int {
		n := 0
		for _, m := range gp.texts(t) {
			if m["type"] == "error" {
				n++
			}
		}
		return n
	}
	// The join spent one token of the burst.
	for i := 0; i < d.MessageBurst+20; i++ {
		send(t, g, msg{"type": "nonsense"})
	}
	if n := errs(); n != d.MessageBurst-1 {
		t.Fatalf("%d refusals answered, want the burst's %d", n, d.MessageBurst-1)
	}
	// Past the budget the relay's lock is not taken.
	h.r.mu.Lock()
	done := make(chan struct{})
	go func() {
		send(t, g, msg{"type": "nonsense"})
		g.Receive(context.Background(), Frame{Binary: true, Data: []byte{9}})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		h.r.mu.Unlock()
		t.Fatal("a message past the budget waited for the relay's lock")
	}
	h.r.mu.Unlock()
	// A member's accepted updates spend nothing of it.
	h.clock.advance(time.Second)
	gp.reset()
	send(t, g, msg{"type": "nonsense"})
	if errs() != 1 {
		t.Fatal("a second later the refill was not spent on the next message")
	}
	// Sustained abuse closes the connection.
	for i := 0; i < d.MaxRefusals && !gp.closed; i++ {
		send(t, g, msg{"type": "nonsense"})
	}
	if !gp.closed || !gp.policy {
		t.Fatalf("after %d refusals the connection is open (closed=%v)", d.MaxRefusals, gp.closed)
	}
}

// TestAcceptedFramesSpendNoMessageBudget: a member typing fast is held only
// by the update rate, never by the message budget.
func TestAcceptedFramesSpendNoMessageBudget(t *testing.T) {
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	h.join(t, code, "Gus", "198.51.100.7")
	hp.reset()
	for i := 0; i < DefaultLimits().UpdateBurst; i++ {
		host.Receive(context.Background(), update(byte(i)))
	}
	if e := hp.last(t, "error"); e != nil {
		t.Fatalf("an update inside the update burst was refused: %v", e)
	}
}

// TestSessionsExpireAfterTheirLifetime: a session ends MaxSessionLife after
// it was opened, however busy, like a host's close, with closed
// {reason: "expired"}; so a rejoin token, which restores a member's role
// after GitHub has revoked it, lasts at most that long (security review 2,
// item 8).
func TestSessionsExpireAfterTheirLifetime(t *testing.T) {
	if d := DefaultLimits(); d.MaxSessionLife != 12*time.Hour {
		t.Fatalf("MaxSessionLife = %v, want 12h", d.MaxSessionLife)
	}
	h := newHarness(t, nil)
	host, hp, code := h.host(t, "tok-ada")
	tok := rejoinToken(t, hp, "hosted")
	_, gp := h.join(t, code, "Gus", "198.51.100.7")
	for i := 0; i < 11; i++ {
		h.clock.advance(time.Hour)
		host.Receive(context.Background(), update(byte(i)))
		h.r.Sweep()
	}
	if s := h.r.Stats(); s.Sessions != 1 {
		t.Fatal("a session was ended before its lifetime")
	}
	h.clock.advance(time.Hour)
	h.r.Sweep()
	for _, p := range []*fakePeer{hp, gp} {
		if c := p.last(t, "closed"); c == nil || c["reason"] != "expired" || !p.closed {
			t.Fatalf("a participant of an expired session: %v, closed=%v", p.last(t, "closed"), p.closed)
		}
	}
	if s := h.r.Stats(); s != (Stats{Connections: 2}) {
		t.Fatalf("after the session expired, the relay still holds %+v", s)
	}
	if _, p := h.rejoin(t, tok, "192.0.2.1"); p.last(t, "error")["code"] != "bad-rejoin" {
		t.Fatal("an expired session's rejoin token still works")
	}

	// Between sweeps, an expired session admits nobody.
	h2 := newHarness(t, nil)
	_, hp2, code2 := h2.host(t, "tok-ada")
	tok2 := rejoinToken(t, hp2, "hosted")
	h2.clock.advance(12 * time.Hour)
	if _, p := h2.join(t, code2, "Late", "198.51.100.9"); p.last(t, "joined") != nil {
		t.Fatal("an expired session admitted a guest before the sweep")
	}
	if c := hp2.last(t, "closed"); c == nil || c["reason"] != "expired" {
		t.Fatalf("the host of a session found expired was not told: %v", hp2.texts(t))
	}
	if _, p := h2.rejoin(t, tok2, "192.0.2.1"); p.last(t, "joined") != nil {
		t.Fatal("an expired session admitted a rejoin")
	}
}

// TestTokenExchangesAreLimitedPerAddress: the sign-in exchange is counted per
// address, keyed as wrong codes are, at most MaxExchangesPerAddr in
// ExchangeWindow (security review 2, item 10).
func TestTokenExchangesAreLimitedPerAddress(t *testing.T) {
	if d := DefaultLimits(); d.MaxExchangesPerAddr != 10 || d.ExchangeWindow != 10*time.Minute {
		t.Fatalf("exchange defaults = %d in %v, want 10 in 10m", d.MaxExchangesPerAddr, d.ExchangeWindow)
	}
	h := newHarness(t, nil)
	for i := 0; i < 10; i++ {
		if !h.r.AllowExchange("2001:db8:1:2::" + strconv.Itoa(i+1)) {
			t.Fatalf("exchange %d refused", i+1)
		}
	}
	// The same /48 is one address.
	if h.r.AllowExchange("2001:db8:1:ffff::1") {
		t.Fatal("an eleventh exchange from the same /48 was allowed")
	}
	if !h.r.AllowExchange("203.0.113.9") {
		t.Fatal("another address was held up")
	}
	h.clock.advance(10 * time.Minute)
	if !h.r.AllowExchange("2001:db8:1:2::1") {
		t.Fatal("after the window the address is still refused")
	}
}
