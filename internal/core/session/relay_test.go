package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	if len(got) != 2 || !bytes.Equal(got[0], []byte{TagUpdate, 1, 2, 3}) || !bytes.Equal(got[1], []byte{TagUpdate, 4, 5}) {
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
	if got := gp.binaries(); len(got) != 1 || !bytes.Equal(got[0], []byte{TagUpdate, 7}) {
		t.Fatalf("the guest received %v, want the member's update", got)
	}
	if len(hp.binaries()) != 0 {
		t.Fatal("the sender's own update was echoed back")
	}

	// Awareness is forwarded from anyone and never logged.
	guest.Receive(context.Background(), Frame{Binary: true, Data: []byte{TagAwareness, 1}})
	if got := hp.binaries(); len(got) != 1 || got[0][0] != TagAwareness {
		t.Fatalf("the host received %v, want the guest's awareness", got)
	}
	if s := h.r.Stats(); s.LogBytes != 2 {
		t.Fatalf("log bytes = %d, want 2 (the one update)", s.LogBytes)
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
	if got := hp.binaries(); len(got) != 1 || got[0][1] != 1 {
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
	// An IPv6 visitor is counted by its /64.
	for i := 0; i < 3; i++ {
		h.join(t, "ZZZZZZ", "Mal", "2001:db8:1:2::"+string(rune('a'+i)))
	}
	if _, p := h.join(t, code, "Mal", "2001:db8:1:2::ff"); p.last(t, "error")["code"] != "too-many-tries" {
		t.Fatal("rotating addresses inside one /64 escaped the limit")
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
		l.MaxLogBytes = 10
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
	if s := h.r.Stats(); s.LogBytes != 9 {
		t.Fatalf("log bytes = %d, want 9", s.LogBytes)
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
	if s := h.r.Stats(); s != (Stats{}) {
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
		"2001:db8:1:2::a":                         "2001:db8:1:2::/64",
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
	if !rateLimited(t, gp) {
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
	if !rateLimited(t, gp) {
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
	if !rateLimited(t, gp) {
		t.Fatalf("past the burst: %v, closed=%v; want rate-limited and still open", gp.last(t, "error"), gp.closed)
	}
	h.clock.advance(time.Second)
	send(t, g, msg{"type": "suggest", "body": msg{"text": "y"}})
	if n := count(); n != l.SuggestBurst+1 {
		t.Fatal("a second later the next suggestion was not delivered")
	}
	gp.reset()
	send(t, g, msg{"type": "suggest", "body": msg{"text": "z"}})
	if !rateLimited(t, gp) || count() != l.SuggestBurst+1 {
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
	if got := gp2.binaries(); len(got) != 1 || !bytes.Equal(got[0], []byte{TagUpdate, 1, 2}) {
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
	if s := h.r.Stats(); s != (Stats{}) {
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
	h := newHarness(t, nil)
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
// connection in the session, and each one's place is free again.
func TestClosingASessionGivesItsConnectionsBack(t *testing.T) {
	h := newHarness(t, nil)
	host, _, code := h.host(t, "tok-ada")
	h.join(t, code, "Gus", "198.51.100.7")
	send(t, host, msg{"type": "close"})
	if s := h.r.Stats(); s != (Stats{}) {
		t.Fatalf("after the close the relay holds %+v", s)
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
	if s := h.r.Stats(); s.Connections != 0 {
		t.Fatalf("connections = %d after the expiry, want 0", s.Connections)
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
