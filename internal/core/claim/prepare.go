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
	// Embargo is the embargo check an announcement is held to (C5); nil
	// refuses every announcement, since an unchecked one is never let through.
	// A reservation, renewal or take-over consults it as well, and with nil
	// the check the front door registered (RegisterEmbargoCheck); with
	// neither it is refused.
	Embargo EmbargoCheck
	// ForgeUser answers whose GitHub account the forge token gh resolves from
	// the session's environment belongs to (login and numeric id): the user endpoint, asked only by an
	// explicit act when git's identity names no holder. nil means the real
	// call, `gh api user`, asked once per process for each checkout and forge
	// credential; tests set it so none touches the network, and an answer it
	// gives is never cached.
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
	// Record is the issue or intent id, as the person typed it, or a release
	// subject, rel-vX.Y.Z.
	Record string
	// Footprint is the repository-relative files the work will change; a
	// reservation of a record requires at least one, and a release subject
	// and a release (the tombstone) take none.
	Footprint []string
	// Lease is how long the reservation holds; zero means lease.Default, or
	// lease.Max for a release subject.
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
	// Narrow, with Renew, is the footprint the renewal writes in place of the
	// tip's, when every path of it lies within the tip's footprint (equal to
	// one of its paths or beneath one): a hold that narrows never widens
	// (spc-2610161044461448 step 12). A tip whose footprint does not cover
	// it keeps its own, and Result.Narrowed is false; so does a narrowing
	// naming a path no reservation may carry (ValidFootprintPath), or one
	// too long to write: the renewal is made all the same.
	Narrow []string
	// HandBack, with Release, publishes a hand-back: the tombstone names this
	// kind, one of HandBackKinds, and the blob id of the record on origin's
	// default branch (Result.HandBackUnpublished says why it names none). A
	// release subject, which has no record, is refused one.
	HandBack string
	// HandBackBlob, with HandBack, is the blob id of the record the hand-back
	// was judged over: a release finding origin's default branch holding the
	// record at another blob publishes no hand-back, and says so
	// (Result.HandBackUnpublished). Empty names the blob as last fetched.
	HandBackBlob string
	// OverAbsent, with HandBack, HandBackBlob and no Tip, publishes the hand-back over a
	// branch origin no longer holds (the sweep deletes a released branch at
	// its next run) as the branch's first commit. The push is plain, so origin
	// takes it only while the branch is still absent: a branch created
	// meanwhile, or found when the release fetches, holding a reservation is
	// left as it stands with a *NotHeld, NotHeldMoved.
	OverAbsent bool
	// Drain marks a reservation a drain makes: it backs off a hand-back
	// tombstone whose record is unchanged on origin's default branch, where a
	// person's own reservation goes ahead and is told (Result.HandedBack).
	Drain bool
	// WarnOwnOverlap lets a person's own act go ahead over a footprint path
	// that their live reservation from another copy holds, naming each such
	// path (Result.OwnOverlap), where every other act backs off it. Only the
	// reserve verb sets it: a build's, a drain's and the land fence's acts are
	// automated, and two automated lanes must never edit the same files
	// (spc-2610161044461448 step 11).
	WarnOwnOverlap bool
	// Live, with Renew, refuses one's own reservation that has lapsed too
	// (NotHeldLapsed): the land fence renews only a reservation that holds.
	Live bool
	// Stage, with Renew, is the stage of the drain lane renewing, one of
	// LaneStages, which the renewal carries as its Abcd-Stage trailer; "" for
	// every other act, and for a renewal that names no stage (a person's),
	// which then carries none. With Drain, and no TakeOver, a drain's
	// reservation may name one too: it is written when the act renews this
	// copy's own live reservation, and the act is never refused for it when
	// it reserves afresh (no tip, a released or lapsed one, a take-over),
	// which writes none, since only a renewal carries a stage. A stage on any
	// other act (a release, a take-over asked for, a person's reservation or
	// a build's start), or one outside LaneStages, is refused before anything
	// is pushed.
	Stage string
	// Admit, on a release subject, is the number of a pull request the
	// release reservation withdrew, which the cutting copy rules in: it is
	// re-armed in its recorded form, pinned to its withdrawn head, and a
	// renewal records it admitted. 0 for every other act.
	Admit int
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
	// home is where the machine layer of the configuration is read, for the
	// cap (maxlive.go).
	home string
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
	if len(req.Narrow) > 0 && !req.Renew {
		return nil, refuse("only a renewal narrows its reservation's footprint; %s", nothingPushed)
	}
	// A narrowing naming a path no reservation may carry is dropped, and the
	// renewal keeps the tip's footprint, as one over the message cap does
	// below: a renewal never fails over its narrowing, which would let the
	// reservation lapse while its work goes on.
	for _, p := range req.Narrow {
		if !validFootprint(p) {
			req.Narrow = nil
			break
		}
	}
	req.Narrow = uniquePaths(req.Narrow)
	if req.Live && !req.Renew {
		return nil, refuse("only a renewal can require a live reservation; %s", nothingPushed)
	}
	if req.HandBack != "" && !req.Release {
		return nil, refuse("only a release publishes a hand-back; %s", nothingPushed)
	}
	if req.OverAbsent && (req.HandBack == "" || req.Tip != "") {
		return nil, refuse("only a release that publishes a hand-back, naming no reservation commit, publishes it over an absent branch; %s", nothingPushed)
	}
	if req.OverAbsent && !gitutil.IsFullSHA(req.HandBackBlob) {
		// With no branch left to fence on, the judged blob is all that ties
		// the hand-back to the record a person was asked about.
		return nil, refuse("a hand-back published over an absent branch names the blob it was judged over; %s", nothingPushed)
	}
	if req.HandBack != "" && isRelease(id) {
		// A release subject has no record on origin's default branch, so a
		// hand-back over it could name no blob and would hold nothing back.
		return nil, refuse("a release subject has no record to hand back, so its release publishes no hand-back; %s", nothingPushed)
	}
	if req.HandBack != "" && !slices.Contains(HandBackKinds, req.HandBack) {
		return nil, refuse("hand-back kind %s is not one of %s; %s", echo(req.HandBack), strings.Join(HandBackKinds, ", "), nothingPushed)
	}
	if req.Stage != "" && !req.Renew && (!req.Drain || req.TakeOver) {
		return nil, refuse("only a renewal, or a drain's reservation, names a lane stage; %s", nothingPushed)
	}
	if req.Stage != "" && !ValidLaneStage(req.Stage) {
		return nil, refuse("lane stage %s is not one of %s; %s", echo(req.Stage), strings.Join(LaneStages, ", "), nothingPushed)
	}
	if req.HandBackBlob != "" && req.HandBack == "" {
		return nil, refuse("only a hand-back names the record blob it was judged over; %s", nothingPushed)
	}
	if req.HandBackBlob != "" && !validBlob(req.HandBackBlob) {
		return nil, refuse("record blob %s is not an object name; %s", echo(req.HandBackBlob), nothingPushed)
	}
	if req.Drain && (req.Release || req.Renew) {
		return nil, refuse("only a reservation is a drain's to back off a hand-back, never a release or a renewal; %s", nothingPushed)
	}
	release := isRelease(id)
	if req.Admit != 0 {
		switch {
		case !release:
			return nil, refuse("only a release reservation admits a pull request; %s", nothingPushed)
		case !validPRNumber(req.Admit):
			return nil, refuse("%d is not a pull request number; %s", req.Admit, nothingPushed)
		case req.Release || req.Renew || req.TakeOver || len(req.Footprint) > 0 || req.Lease != 0 || req.Drain || req.Stage != "":
			return nil, refuse("an admission takes no footprint, lease, take-over, renewal or release; %s", nothingPushed)
		}
	}
	if release && len(req.Footprint) > 0 {
		return nil, refuse("a release reservation holds every merge and takes no footprint; %s", nothingPushed)
	}
	if !release && !req.Release && !req.Renew && len(req.Footprint) == 0 {
		return nil, refuse("a reservation names the files the work will change: pass --footprint <path> once per path, a renewal of one you hold included; %s", nothingPushed)
	}
	for _, p := range req.Footprint {
		if !validFootprint(p) {
			return nil, refuse("footprint %s is not a repository-relative path; %s", echo(p), nothingPushed)
		}
	}
	// A path named twice is one path, kept where it was first named, so a
	// renewal naming the held footprint again beside its new paths never
	// grows the message with repeats until it passes the cap
	// (iss-2610290507514841). A valid path is already clean, so equal text
	// is the same path.
	req.Footprint = uniquePaths(req.Footprint)
	term := req.Lease
	switch {
	case term == 0 && release && !req.Release:
		// A release holds for a day unless a lease is named: a cut waits on
		// checks and an approval, and must outlast them (release.go).
		term = lease.Max
	case term == 0:
		term = lease.Default
	}
	if term < lease.Min || term > lease.Max {
		return nil, refuse("lease %s is outside %s to %s; %s", term, lease.Min, lease.Max, nothingPushed)
	}
	p := &prepared{root: root, id: id, req: req, lease: term, now: o.now(), label: o.assistedBy(), home: o.home()}
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
	if len(req.Narrow) > 0 {
		// A narrowed renewal may name more paths than the tip it narrows (a
		// directory's files), so it is measured as a reservation is, and one
		// the writer may not write is not narrowed: the renewal keeps the
		// tip's footprint (Result.Narrowed false) rather than refusing.
		narrowed := *p
		narrowed.req.Footprint = req.Narrow
		for _, text := range []string{narrowed.reservation().text(), narrowed.longestTombstone(req.Narrow).text()} {
			if checkMessage(root, text) != nil {
				p.req.Narrow = nil
				break
			}
		}
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

// uniquePaths is paths with each repeat dropped, in the order first named.
func uniquePaths(paths []string) []string {
	if len(paths) < 2 {
		return paths
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
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
