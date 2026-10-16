package claim

import (
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Options carries what a caller may set about the environment an act runs in.
type Options struct {
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Home is the home directory the machine layer of the configuration is
	// read under; "" means the process's own.
	Home string
	// Version is the binary's version the Assisted-by label names; "" means
	// the running binary's (core.Version).
	Version string
	// ForgeUser answers whose GitHub account the forge token gh resolves from
	// the session's environment belongs to (login and numeric id): the user endpoint, asked only by an
	// explicit act when git's identity names no holder. nil means the real
	// call, `gh api user`; tests set it so none touches the network.
	ForgeUser func(root string) (login string, id int64, err error)
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

func (o Options) home() string {
	if o.Home != "" {
		return o.Home
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func (o Options) assistedBy() string {
	v := o.Version
	if v == "" {
		v = core.Version
	}
	return assistedby.ComposedValue(v)
}

// Request is one reservation act asked for.
type Request struct {
	// Record is the issue or intent id, as the person typed it.
	Record string
	// Footprint is the repository-relative files the work will change; a
	// reservation requires at least one, a release takes none.
	Footprint []string
	// Lease is how long the reservation holds; zero means lease.Default.
	Lease time.Duration
	// TakeOver moves this person's own live reservation from another copy to
	// this one. It never overrides another person's.
	TakeOver bool
	// Release pushes the tombstone instead.
	Release bool
	// Tip, with Release, names the one reservation commit the release may
	// tombstone: a tip that has moved since (renewed or reserved again by
	// hand, or replaced) is refused with a *NotHeld, NotHeldMoved, and nothing
	// is pushed. Empty releases whatever reservation of this person's from
	// this copy the tip holds.
	Tip string
	// Renew renews this person's own reservation from this copy, live or
	// lapsed, and nothing else: an absent, unreadable or released tip, and one
	// another person or another copy holds, is refused with a *NotHeld naming
	// why, and nothing is pushed. A run's renewal therefore never reserves a
	// record anew and never takes one over. It keeps the tip's footprint, so
	// Footprint must be empty.
	Renew bool
	// HandBack, with Release, publishes a hand-back: the tombstone names this
	// kind, one of HandBackKinds, and the blob id of the record on origin's
	// default branch (Result.HandBackUnpublished says why it names none).
	HandBack string
	// Drain marks a reservation a drain makes: it backs off a hand-back
	// tombstone whose record is unchanged on origin's default branch, where a
	// person's own reservation goes ahead and is told (Result.HandedBack).
	Drain bool
	// Live, with Renew, refuses one's own reservation that has lapsed too
	// (NotHeldLapsed): the land fence renews only a reservation that holds.
	Live bool
}

// prepared is a request checked and filled in from the checkout, before the
// remote is asked anything.
type prepared struct {
	root   string
	id     string
	req    Request
	lease  time.Duration
	now    time.Time
	who    identity
	holder string
	copy   string
	label  string
	// derivedGap is why the act's fetch could not read origin's default
	// branch, "" when it read it: the derived release is then not applied.
	derivedGap string
}

// prepare checks a request in the order the refusals promise: the id before
// any ref name is formed, then the footprint and the lease before any git
// runs, then the working copy and the caller's local checks (local, such as
// Reserve's origin check), then the person's identity and holder. Every check
// that can refuse on the checkout's own facts runs before the holder, so a
// checkout refused locally never asks GitHub whose account the token is. A
// reservation's message, and the tombstone that would release it,
// are composed and measured against the cap and the outbound policy here too,
// so a message that may not be written is refused before anything reaches the
// remote, and a reservation accepted is one that can be released.
func prepare(root string, req Request, o Options, local ...func(root string) error) (*prepared, error) {
	id, err := reservableID(req.Record)
	if err != nil {
		return nil, err
	}
	if req.Release && (len(req.Footprint) > 0 || req.TakeOver || req.Lease != 0) {
		return nil, refuse("a release takes no footprint, take-over or lease; %s", nothingPushed)
	}
	if req.Tip != "" && (!req.Release || !gitutil.IsFullSHA(req.Tip)) {
		return nil, refuse("only a release names the reservation commit it releases, by its full object name; %s", nothingPushed)
	}
	if req.Renew && (req.Release || req.TakeOver || len(req.Footprint) > 0) {
		return nil, refuse("a renewal keeps its reservation's footprint, and takes no footprint, take-over or release; %s", nothingPushed)
	}
	if req.Live && !req.Renew {
		return nil, refuse("only a renewal can require a live reservation; %s", nothingPushed)
	}
	if req.HandBack != "" && !req.Release {
		return nil, refuse("only a release publishes a hand-back; %s", nothingPushed)
	}
	if req.HandBack != "" && !slices.Contains(HandBackKinds, req.HandBack) {
		return nil, refuse("hand-back kind %s is not one of %s; %s", echo(req.HandBack), strings.Join(HandBackKinds, ", "), nothingPushed)
	}
	if req.Drain && (req.Release || req.Renew) {
		return nil, refuse("only a reservation is a drain's to back off a hand-back, never a release or a renewal; %s", nothingPushed)
	}
	if !req.Release && !req.Renew && len(req.Footprint) == 0 {
		return nil, refuse("a reservation names the files the work will change: pass --footprint <path> once per path; %s", nothingPushed)
	}
	for _, p := range req.Footprint {
		if !validFootprint(p) {
			return nil, refuse("footprint %s is not a repository-relative path; %s", echo(p), nothingPushed)
		}
	}
	term := req.Lease
	if term == 0 {
		term = lease.Default
	}
	if term < lease.Min || term > lease.Max {
		return nil, refuse("lease %s is outside %s to %s; %s", term, lease.Min, lease.Max, nothingPushed)
	}
	p := &prepared{root: root, id: id, req: req, lease: term, now: o.now(), label: o.assistedBy()}
	if p.copy, err = copyKey(root); err != nil {
		return nil, err
	}
	for _, check := range local {
		if err := check(root); err != nil {
			return nil, err
		}
	}
	if p.who, err = gitIdentity(root); err != nil {
		return nil, err
	}
	h, err := resolveHolder(root, p.who, o, true)
	if err != nil {
		return nil, err
	}
	p.holder = h.Login
	if !p.who.person() {
		// Git's identity is unset or a machine's: the token's account authors
		// the reservation, so that identity never reaches a commit.
		p.who = identity{Name: h.Login, Email: h.NoReply}
	}
	if !req.Release && !req.Renew {
		// The reservation is measured by the longer of its own message and the
		// tombstone that will one day release it (a longer subject, an
		// Abcd-Released-At line, and the longest hand-back with the longer of
		// git's object names), so a reservation the writer accepts can always
		// be released, handed back or not.
		for _, text := range []string{p.reservation().text(), p.longestTombstone(req.Footprint).text()} {
			if err := checkMessage(root, text); err != nil {
				return nil, err
			}
		}
	}
	return p, nil
}

// reservation is the message a reserve, renewal or take-over pushes.
func (p *prepared) reservation() message {
	return message{
		Record: p.id, Holder: p.holder, Copy: p.copy, Footprint: p.req.Footprint,
		ClaimedAt: p.now, ExpiresAt: p.now.Add(p.lease), AssistedBy: p.label,
	}
}

// tombstone is the message a release pushes over a tip whose footprint it
// keeps.
func (p *prepared) tombstone(footprint []string) message {
	return message{
		Record: p.id, Holder: p.holder, Copy: p.copy, Footprint: footprint,
		ClaimedAt: p.now, ExpiresAt: p.now, ReleasedAt: p.now, AssistedBy: p.label,
	}
}

// longestTombstone is the longest tombstone that may one day release a
// reservation with footprint: one naming the longest hand-back kind and the
// longer of git's object names.
func (p *prepared) longestTombstone(footprint []string) message {
	m := p.tombstone(footprint)
	m.HandedBack, m.RecordBlob = longestHandBack(), longestBlob
	return m
}

// checkMessage refuses a reservation message the writer may not write
// (checkText, for the reservation family).
func checkMessage(root, text string) error { return checkText(root, reservationFamily, text) }

// checkText refuses a message of family f the writer may not write: one over
// the size cap, or one the outbound policy finds a session URL or a tool footer
// in, since every message this package writes is public the moment it is
// pushed. Each refusal names the family's message and, at the cap, its remedy.
func checkText(root string, f *family, text string) error {
	if n := len(text); n > MaxMessageBytes {
		return refuse("the %s message would be %d bytes, over the %d-byte cap: %s; %s",
			f.name, n, MaxMessageBytes, f.shorten, nothingPushed)
	}
	findings, err := scanner.CheckOutbound(root, text, f.name+" message")
	if len(findings) > 0 {
		return refuse("the %s message carries %s, which public text may not carry; %s", f.name, findings[0].Kind, nothingPushed)
	}
	if err != nil {
		return refuse("the %s message could not be judged against the outbound policy: %s; %s",
			f.name, redacted(err), nothingPushed)
	}
	return nil
}
