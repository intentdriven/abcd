// Package session is the brief session's relay, without a transport
// (itd-2610091016262335, spc-2610141709430146 step 1): the rooms a host opens,
// the join codes that reach them, who is a guest and who a team member, the
// log of opaque edit updates each room replays to a newcomer, a guest's
// suggestions routed to the members, the roster, and the limits that bound all
// of it.
//
// It never writes to stdout, opens no file and knows no transport: the front
// door (internal/surface/session) hands it one Frame at a time through a Conn
// and gives it a Peer to answer on. It holds everything in memory, and a room
// is dropped whole when its host closes it or when nobody has been connected
// for the idle limit (adr-2610032150581128, as amended 2026-10-14: no project
// content at rest).
//
// The edit updates and awareness messages are Yjs's, carried as opaque bytes
// behind a one-byte tag; the relay never parses them, nor the brief they
// carry. A GitHub token presented to verify is handed to the Verifier and not
// kept: only the login and the answer are.
package session

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// The binary frame tags: the first byte of every binary frame.
const (
	// TagUpdate is a Yjs document update: logged and forwarded, and accepted
	// from team members only.
	TagUpdate byte = 0
	// TagAwareness is a Yjs awareness message (cursors and selections):
	// forwarded from anyone, never logged.
	TagAwareness byte = 1
)

// Frame is one WebSocket message: a JSON control message in a text frame, or
// a tagged Yjs message in a binary frame.
type Frame struct {
	Binary bool
	Data   []byte
}

// Peer is the transport's handle on one connection.
type Peer interface {
	// Send queues f without blocking. False means the connection cannot take
	// it (its queue is full), and the relay drops the connection.
	Send(f Frame) bool
	// Close ends the connection; policy marks a refusal (WebSocket 1008)
	// rather than a normal end. It must not call back into the relay.
	Close(policy bool, reason string)
}

// Verifier asks GitHub who a token belongs to and whether that login may push
// to repo. The token is not kept by the relay after the call returns.
type Verifier interface {
	Verify(ctx context.Context, token, repo string) (login string, push bool, err error)
}

// Limits bound everything the relay holds.
type Limits struct {
	// MaxMessageBytes caps one frame, text or binary.
	MaxMessageBytes int
	// MaxLogBytes caps one session's update log.
	MaxLogBytes int
	// MaxParticipants caps the connections in one session.
	MaxParticipants int
	// MaxSessions caps the sessions one relay holds at once.
	MaxSessions int
	// IdleAfter is how long a session with nobody connected is kept.
	IdleAfter time.Duration
	// MaxFailedJoins wrong codes from one address within FailedJoinWindow
	// shut that address out of joining until the window passes.
	MaxFailedJoins   int
	FailedJoinWindow time.Duration
	// MaxTrackedAddrs caps the addresses the failed-join count remembers.
	MaxTrackedAddrs int
	// MaxVerifies caps the verify attempts of one connection, so the relay
	// cannot be used to hammer GitHub.
	MaxVerifies int
	// MaxCredits caps the Yjs client ids one session credits to members.
	MaxCredits int
}

// DefaultLimits are the relay's limits unless a caller sets others. Sixteen
// sessions of at most 8 MiB of log each keep a relay inside a free host's
// memory.
func DefaultLimits() Limits {
	return Limits{
		MaxMessageBytes:  256 << 10,
		MaxLogBytes:      8 << 20,
		MaxParticipants:  32,
		MaxSessions:      16,
		IdleAfter:        30 * time.Minute,
		MaxFailedJoins:   20,
		FailedJoinWindow: 10 * time.Minute,
		MaxTrackedAddrs:  10000,
		MaxVerifies:      5,
		MaxCredits:       512,
	}
}

// Options configure a Relay. A zero Now is time.Now and a nil Rand is
// crypto/rand; a zero limit takes its default.
type Options struct {
	Limits   Limits
	Verifier Verifier
	Now      func() time.Time
	Rand     io.Reader
}

// Stats is what the relay holds, in counts: after a session is purged, none of
// it counts towards any of these.
type Stats struct {
	Sessions     int
	Codes        int
	Participants int
	LogBytes     int
	Credits      int
}

// The two roles.
const (
	roleGuest  = "guest"
	roleMember = "member"
)

// Relay holds the sessions. One mutex guards all of it; Peer.Send never
// blocks, so nothing waits on a connection while holding it.
type Relay struct {
	mu       sync.Mutex
	limits   Limits
	verifier Verifier
	now      func() time.Time
	rand     io.Reader
	rooms    map[string]*room // by join code
	failed   *failedJoins
	nextConn int
}

// New builds a relay.
func New(o Options) *Relay {
	l := o.Limits
	d := DefaultLimits()
	fill := func(v *int, def int) {
		if *v <= 0 {
			*v = def
		}
	}
	fill(&l.MaxMessageBytes, d.MaxMessageBytes)
	fill(&l.MaxLogBytes, d.MaxLogBytes)
	fill(&l.MaxParticipants, d.MaxParticipants)
	fill(&l.MaxSessions, d.MaxSessions)
	fill(&l.MaxFailedJoins, d.MaxFailedJoins)
	fill(&l.MaxTrackedAddrs, d.MaxTrackedAddrs)
	fill(&l.MaxVerifies, d.MaxVerifies)
	fill(&l.MaxCredits, d.MaxCredits)
	if l.IdleAfter <= 0 {
		l.IdleAfter = d.IdleAfter
	}
	if l.FailedJoinWindow <= 0 {
		l.FailedJoinWindow = d.FailedJoinWindow
	}
	r := &Relay{limits: l, verifier: o.Verifier, now: o.Now, rand: o.Rand, rooms: map[string]*room{}}
	if r.now == nil {
		r.now = time.Now
	}
	if r.rand == nil {
		r.rand = rand.Reader
	}
	r.failed = newFailedJoins(l.MaxFailedJoins, l.FailedJoinWindow, l.MaxTrackedAddrs)
	return r
}

// Limits reports the limits in force, defaults filled in.
func (r *Relay) Limits() Limits { return r.limits }

// room is one session.
type room struct {
	repo       string // as the host named it
	repoKey    string // lower-cased, for comparison
	code       string
	hostLogin  string
	conns      []*Conn // in join order
	log        [][]byte
	logBytes   int
	credits    map[uint32]string // Yjs client id -> member login
	emptySince time.Time
	suggests   int
}

// Conn is one connection's state. Its methods are called by one goroutine per
// connection (the transport's read loop).
type Conn struct {
	r        *Relay
	id       string
	peer     Peer
	addr     string // the failed-join key: an IPv4 address or an IPv6 /64
	room     *room
	role     string
	name     string
	login    string // the verified GitHub login, if any
	memberOf string // the repository key the verified login may push to
	verifies int
	left     bool
}

// Connect admits a new connection from addr (the client's address as the
// transport judged it).
func (r *Relay) Connect(p Peer, addr string) *Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextConn++
	return &Conn{r: r, id: "p" + strconv.Itoa(r.nextConn), peer: p, addr: AddressKey(addr)}
}

// inbound is every field a control message may carry.
type inbound struct {
	Type   string          `json:"type"`
	Repo   string          `json:"repo"`
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Token  string          `json:"token"`
	Client json.Number     `json:"client"`
	Body   json.RawMessage `json:"body"`
}

// Receive handles one frame from the connection.
func (c *Conn) Receive(ctx context.Context, f Frame) {
	r := c.r
	if len(f.Data) > r.limits.MaxMessageBytes {
		r.mu.Lock()
		c.fail("too-big", fmt.Sprintf("a message is at most %d bytes", r.limits.MaxMessageBytes))
		r.mu.Unlock()
		return
	}
	if f.Binary {
		r.mu.Lock()
		defer r.mu.Unlock()
		c.binary(f.Data)
		return
	}
	var m inbound
	dec := json.NewDecoder(strings.NewReader(string(f.Data)))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil || dec.More() {
		r.mu.Lock()
		c.fail("bad-message", "a control message is one JSON object")
		r.mu.Unlock()
		return
	}
	if m.Type == "verify" {
		c.verify(ctx, m.Token, m.Repo)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.left {
		return
	}
	switch m.Type {
	case "host":
		c.host(m.Repo)
	case "join":
		c.join(m.Code, m.Name)
	case "suggest":
		c.suggest(m.Body)
	case "register":
		c.register(m.Client)
	case "rotate":
		c.rotate()
	case "close":
		c.closeRoom()
	default:
		c.fail("unknown-message", "the relay does not know this message")
	}
}

// Leave removes the connection; the transport calls it when the connection
// ends, and it may be called more than once.
func (c *Conn) Leave() {
	c.r.mu.Lock()
	defer c.r.mu.Unlock()
	c.leave()
}

// leave is Leave under the lock.
func (c *Conn) leave() {
	if c.left {
		return
	}
	c.left = true
	rm := c.room
	c.room = nil
	if rm == nil {
		return
	}
	for i, o := range rm.conns {
		if o == c {
			rm.conns = append(rm.conns[:i], rm.conns[i+1:]...)
			break
		}
	}
	if len(rm.conns) == 0 {
		rm.emptySince = c.r.now()
		return
	}
	c.r.broadcastRoster(rm)
}

// send queues one message on c; a peer that cannot take it is dropped.
func (c *Conn) send(f Frame) {
	if c.left {
		return
	}
	if !c.peer.Send(f) {
		c.leave()
		c.peer.Close(true, "too slow")
	}
}

func (c *Conn) sendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.send(Frame{Data: data})
}

// fail answers c with an error message. Its text is fixed per code and never
// echoes what the client sent.
func (c *Conn) fail(code, text string) {
	c.sendJSON(map[string]string{"type": "error", "code": code, "message": text})
}

// binary handles a tagged Yjs frame, under the lock.
func (c *Conn) binary(data []byte) {
	if c.left {
		return
	}
	rm := c.room
	switch {
	case len(data) == 0:
		c.fail("empty-frame", "a binary frame starts with its tag")
	case rm == nil:
		c.fail("not-in-session", "join or host a session first")
	case data[0] == TagUpdate:
		if c.role != roleMember {
			c.fail("read-only", "a guest suggests changes; only a team member edits")
			return
		}
		if rm.logBytes+len(data) > c.r.limits.MaxLogBytes {
			c.fail("log-full", "this session's log is full; the host can close it and open a new one")
			return
		}
		entry := append([]byte(nil), data...)
		rm.log = append(rm.log, entry)
		rm.logBytes += len(entry)
		c.r.forward(rm, c, entry)
	case data[0] == TagAwareness:
		c.r.forward(rm, c, append([]byte(nil), data...))
	default:
		c.fail("unknown-tag", "the relay knows update (0) and awareness (1) frames")
	}
}

// forward sends a binary frame to everyone in rm but from.
func (r *Relay) forward(rm *room, from *Conn, data []byte) {
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o != from {
			o.send(Frame{Binary: true, Data: data})
		}
	}
}

// repoRe is a GitHub owner/name: an owner of letters, digits and hyphens, a
// name of letters, digits, '.', '_' and '-'.
var repoRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

// ValidRepo reports whether s is an owner/name a URL path may carry.
func ValidRepo(s string) bool {
	if !repoRe.MatchString(s) {
		return false
	}
	name := s[strings.IndexByte(s, '/')+1:]
	return name != "." && name != ".."
}

// verify asks the Verifier about a token, outside the lock.
func (c *Conn) verify(ctx context.Context, token, repo string) {
	r := c.r
	r.mu.Lock()
	if c.left {
		r.mu.Unlock()
		return
	}
	if c.role == roleMember {
		c.fail("already-verified", "this connection is a team member already")
		r.mu.Unlock()
		return
	}
	if c.verifies >= r.limits.MaxVerifies {
		c.fail("too-many-verifies", "too many sign-in attempts on this connection")
		r.mu.Unlock()
		return
	}
	if c.room != nil {
		if repo != "" && !strings.EqualFold(repo, c.room.repo) {
			c.fail("bad-repo", "a session verifies against its own repository")
			r.mu.Unlock()
			return
		}
		repo = c.room.repo
	}
	if !ValidRepo(repo) {
		c.fail("bad-repo", "name the repository as owner/name")
		r.mu.Unlock()
		return
	}
	if token == "" || r.verifier == nil {
		c.fail("verify-failed", "GitHub did not accept the sign-in")
		r.mu.Unlock()
		return
	}
	c.verifies++
	r.mu.Unlock()

	login, push, err := r.verifier.Verify(ctx, token, repo)
	token = "" // dropped straight after the check
	_ = token

	r.mu.Lock()
	defer r.mu.Unlock()
	if c.left {
		return
	}
	if err != nil || login == "" {
		c.fail("verify-failed", "GitHub did not accept the sign-in")
		return
	}
	c.login = login
	if push {
		c.memberOf = strings.ToLower(repo)
	}
	member := false
	if rm := c.room; rm != nil && push && c.memberOf == rm.repoKey {
		c.role, c.name, member = roleMember, login, true
	} else if c.room == nil {
		member = push
	}
	c.sendJSON(map[string]any{"type": "verified", "login": login, "member": member})
	if c.room != nil {
		r.broadcastRoster(c.room)
	}
}

// host opens a session for repo, which c must be a verified member of.
func (c *Conn) host(repo string) {
	r := c.r
	switch {
	case c.room != nil:
		c.fail("already-in-session", "this connection is in a session already")
		return
	case !ValidRepo(repo):
		c.fail("bad-repo", "name the repository as owner/name")
		return
	case c.login == "":
		c.fail("not-verified", "sign in with GitHub before hosting")
		return
	case c.memberOf != strings.ToLower(repo):
		c.fail("not-a-member", "only someone who can change the repository hosts its session")
		return
	case len(r.rooms) >= r.limits.MaxSessions:
		c.fail("too-many-sessions", "this relay holds as many sessions as it may")
		return
	}
	code, err := r.newCode()
	if err != nil {
		c.fail("busy", "no join code could be drawn")
		return
	}
	rm := &room{repo: repo, repoKey: strings.ToLower(repo), code: code, hostLogin: c.login, credits: map[uint32]string{}}
	r.rooms[code] = rm
	c.room, c.role, c.name = rm, roleMember, c.login
	rm.conns = append(rm.conns, c)
	c.sendJSON(map[string]any{"type": "hosted", "id": c.id, "code": code, "repo": repo})
	c.sendJSON(map[string]string{"type": "synced"})
	r.broadcastRoster(rm)
}

// join admits c to the session code names.
func (c *Conn) join(code, name string) {
	r := c.r
	if c.room != nil {
		c.fail("already-in-session", "this connection is in a session already")
		return
	}
	now := r.now()
	if r.failed.blocked(c.addr, now) {
		c.fail("too-many-tries", "too many wrong codes from this address; try again later")
		c.leave()
		c.peer.Close(true, "too many wrong codes")
		return
	}
	rm := r.rooms[NormalizeCode(code)]
	if rm == nil {
		c.fail("bad-code", "no session has that code")
		if r.failed.note(c.addr, now) >= r.limits.MaxFailedJoins {
			c.leave()
			c.peer.Close(true, "too many wrong codes")
		}
		return
	}
	if len(rm.conns) >= r.limits.MaxParticipants {
		c.fail("full", "this session has as many participants as it may")
		return
	}
	c.room = rm
	if c.login != "" && c.memberOf == rm.repoKey {
		c.role, c.name = roleMember, c.login
	} else {
		c.role, c.name = roleGuest, CleanName(name)
	}
	rm.conns = append(rm.conns, c)
	c.sendJSON(map[string]any{"type": "joined", "id": c.id, "role": c.role, "name": c.name, "repo": rm.repo, "host": c.isHost()})
	for _, u := range rm.log {
		c.send(Frame{Binary: true, Data: u})
	}
	c.sendJSON(map[string]string{"type": "synced"})
	r.broadcastRoster(rm)
}

// isHost reports whether c may rotate and close: a member signed in as the
// login that opened the session, so a host whose page reloads keeps the role.
func (c *Conn) isHost() bool {
	return c.room != nil && c.role == roleMember && c.login == c.room.hostLogin
}

// suggest routes a guest's proposed change to the team members.
func (c *Conn) suggest(body json.RawMessage) {
	rm := c.room
	switch {
	case rm == nil:
		c.fail("not-in-session", "join or host a session first")
		return
	case c.role != roleGuest:
		c.fail("not-a-guest", "a team member edits directly")
		return
	}
	trimmed := strings.TrimSpace(string(body))
	var obj map[string]json.RawMessage
	if !strings.HasPrefix(trimmed, "{") || json.Unmarshal(body, &obj) != nil {
		c.fail("bad-suggestion", "a suggestion's body is one JSON object")
		return
	}
	rm.suggests++
	id := "s" + strconv.Itoa(rm.suggests)
	out := map[string]any{"type": "suggest", "id": id, "from": c.name, "from_id": c.id, "body": json.RawMessage(trimmed)}
	delivered := 0
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o.role == roleMember {
			o.sendJSON(out)
			delivered++
		}
	}
	c.sendJSON(map[string]any{"type": "suggested", "id": id, "delivered": delivered})
}

// register credits a Yjs client id to the member c is signed in as.
func (c *Conn) register(raw json.Number) {
	rm := c.room
	if rm == nil {
		c.fail("not-in-session", "join or host a session first")
		return
	}
	if c.role != roleMember {
		c.fail("read-only", "only a team member's edits are credited")
		return
	}
	id, err := strconv.ParseUint(string(raw), 10, 32)
	if err != nil {
		c.fail("bad-client", "a client id is a whole number from 0 to 4294967295")
		return
	}
	cid := uint32(id)
	if owner, ok := rm.credits[cid]; ok {
		if owner != c.login {
			c.fail("client-taken", "that client id is credited to someone else")
		}
		return
	}
	if len(rm.credits) >= c.r.limits.MaxCredits {
		c.fail("too-many-clients", "this session credits as many client ids as it may")
		return
	}
	rm.credits[cid] = c.login
	c.r.broadcastRoster(rm)
}

// rotate replaces the session's join code.
func (c *Conn) rotate() {
	if !c.isHost() {
		c.fail("not-host", "only the host does that")
		return
	}
	rm := c.room
	code, err := c.r.newCode()
	if err != nil {
		c.fail("busy", "no join code could be drawn")
		return
	}
	delete(c.r.rooms, rm.code)
	rm.code = code
	c.r.rooms[code] = rm
	for _, o := range append([]*Conn(nil), rm.conns...) {
		if o.isHost() {
			o.sendJSON(map[string]string{"type": "code", "code": code})
		}
	}
}

// closeRoom ends the session for everyone and drops all of it.
func (c *Conn) closeRoom() {
	if !c.isHost() {
		c.fail("not-host", "only the host does that")
		return
	}
	rm := c.room
	conns := append([]*Conn(nil), rm.conns...)
	for _, o := range conns {
		o.sendJSON(map[string]string{"type": "closed", "reason": "host"})
	}
	for _, o := range conns {
		o.left, o.room = true, nil
		o.peer.Close(false, "session closed")
	}
	rm.conns = nil
	c.r.purge(rm)
}

// purge drops a room and everything it holds.
func (r *Relay) purge(rm *room) {
	if r.rooms[rm.code] == rm {
		delete(r.rooms, rm.code)
	}
	rm.log, rm.logBytes, rm.credits = nil, 0, nil
}

// rosterEntry is one participant as the roster shows them.
type rosterEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Login string `json:"login,omitempty"`
	Role  string `json:"role"`
	Host  bool   `json:"host,omitempty"`
}

// broadcastRoster sends rm's roster to everyone in it: the participants with
// their verified identities, and the Yjs client ids credited to each member.
func (r *Relay) broadcastRoster(rm *room) {
	parts := make([]rosterEntry, 0, len(rm.conns))
	for _, o := range rm.conns {
		parts = append(parts, rosterEntry{ID: o.id, Name: o.name, Login: o.login, Role: o.role, Host: o.isHost()})
	}
	clients := make(map[string]string, len(rm.credits))
	for id, login := range rm.credits {
		clients[strconv.FormatUint(uint64(id), 10)] = login
	}
	data, err := json.Marshal(map[string]any{"type": "roster", "participants": parts, "clients": clients})
	if err != nil {
		return
	}
	for _, o := range append([]*Conn(nil), rm.conns...) {
		o.send(Frame{Data: data})
	}
}

// Sweep purges every session nobody has been connected to for the idle
// limit, and forgets addresses whose wrong codes have aged out. The transport
// calls it on a timer.
func (r *Relay) Sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for _, rm := range r.rooms {
		if len(rm.conns) == 0 && now.Sub(rm.emptySince) >= r.limits.IdleAfter {
			r.purge(rm)
		}
	}
	r.failed.sweep(now)
}

// Stats reports what the relay holds.
func (r *Relay) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	var s Stats
	for code, rm := range r.rooms {
		s.Sessions++
		if code != "" {
			s.Codes++
		}
		s.Participants += len(rm.conns)
		s.LogBytes += rm.logBytes
		s.Credits += len(rm.credits)
	}
	return s
}

// debugState renders every string the relay holds about its sessions and
// connections, for a test to prove what is not among them.
func (r *Relay) debugState() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	for code, rm := range r.rooms {
		fmt.Fprintf(&b, "%s %s %s %v\n", code, rm.repo, rm.hostLogin, rm.credits)
		for _, o := range rm.conns {
			fmt.Fprintf(&b, "%+v\n", *o)
		}
	}
	return b.String()
}

// CleanName makes a guest's display name safe to show and to stamp: control
// and format characters (bidirectional overrides among them) removed, spaces
// collapsed, at most 40 characters, and "Guest" when nothing is left.
func CleanName(s string) string {
	var b strings.Builder
	n := 0
	space := false
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			if n >= 39 {
				break
			}
			b.WriteRune(' ')
			n++
		}
		space = false
		if n >= 40 {
			break
		}
		b.WriteRune(r)
		n++
	}
	if b.Len() == 0 {
		return "Guest"
	}
	return b.String()
}
