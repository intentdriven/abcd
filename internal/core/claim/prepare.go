package claim

import (
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
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
// runs, then the person's identity, holder and working copy from the
// checkout. A reservation's message, and the tombstone that would release it,
// are composed and measured against the cap and the outbound policy here too,
// so a message that may not be written is refused before anything reaches the
// remote, and a reservation accepted is one that can be released.
func prepare(root string, req Request, o Options) (*prepared, error) {
	id, err := reservableID(req.Record)
	if err != nil {
		return nil, err
	}
	if req.Release && (len(req.Footprint) > 0 || req.TakeOver || req.Lease != 0) {
		return nil, refuse("a release takes no footprint, take-over or lease; %s", nothingPushed)
	}
	if !req.Release && len(req.Footprint) == 0 {
		return nil, refuse("a reservation names the files the work will change: pass --footprint <path> once per path; %s", nothingPushed)
	}
	for _, p := range req.Footprint {
		if !fsutil.ValidRelPath(p) {
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
	if p.who, err = gitIdentity(root); err != nil {
		return nil, err
	}
	if p.holder, err = resolveHolder(root, o.home(), p.who.Email); err != nil {
		return nil, err
	}
	if p.copy, err = copyKey(root); err != nil {
		return nil, err
	}
	if !req.Release {
		// The reservation is measured by the longer of its own message and the
		// tombstone that will one day release it (a longer subject and an
		// Abcd-Released-At line), so a reservation the writer accepts can
		// always be released.
		for _, text := range []string{p.reservation().text(), p.tombstone(req.Footprint).text()} {
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

// checkMessage refuses a message the writer may not write: one over the size
// cap, or one the outbound policy finds a session URL or a tool footer in,
// since a reservation is public the moment it is pushed.
func checkMessage(root, text string) error {
	if n := len(text); n > MaxMessageBytes {
		return refuse("the reservation message would be %d bytes, over the %d-byte cap: name fewer or shorter footprint paths; %s",
			n, MaxMessageBytes, nothingPushed)
	}
	findings, err := scanner.CheckOutbound(root, text, "reservation message")
	if len(findings) > 0 {
		return refuse("the reservation message carries %s, which public text may not carry; %s", findings[0].Kind, nothingPushed)
	}
	if err != nil {
		return refuse("the reservation message could not be judged against the outbound policy: %s; %s",
			redacted(err), nothingPushed)
	}
	return nil
}
