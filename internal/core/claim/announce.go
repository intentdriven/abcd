package claim

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	toolid "github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// An announcement (itd-2609151838312703, adr-2610150729333337) is a session's
// short fixed-form notice to every other participant: a kind from a fixed
// list, the record or pull request it concerns, and an expiry, with no free
// text. It is one empty-tree commit on the sender's own status branch,
// refs/heads/abcd-status/<username in lower case> on origin, pushed with one
// plain push, so it reuses the reservation's writer, fetch, push and reader
// rather than copying them (one-canonical-primitive).

const (
	// StatusRefPrefix is where the status branches sit on the remote, one per
	// person, and StatusTrackingPrefix where this checkout keeps what it last
	// fetched of them.
	StatusRefPrefix      = "refs/heads/abcd-status/"
	StatusTrackingPrefix = "refs/remotes/" + Remote + "/abcd-status/"

	// DefaultAnnouncementLife is how long an announcement stands when no
	// expiry is asked for, and MinAnnouncementLife and MaxAnnouncementLife the
	// bounds an expiry is held to (C3).
	DefaultAnnouncementLife = 30 * time.Minute
	MinAnnouncementLife     = time.Minute
	MaxAnnouncementLife     = 24 * time.Hour

	// maxAnnounceRetries is how many times a push that lost a race is rebuilt
	// on the tip that won and pushed again (C2).
	maxAnnounceRetries = 3
)

// The kinds an announcement may carry, in the order a refusal lists them.
var AnnouncementKinds = []string{"merging", "merged", "asking", "answered", "handed-over"}

// The trailer keys an announcement carries besides the copy, the expiry and
// the attribution, which it spells as a reservation does.
const (
	keyAnnouncement = "Abcd-Announcement"
	keyItem         = "Abcd-Item"
	keySender       = "Abcd-Sender"
	keySentAt       = "Abcd-Sent-At"
)

// announcementFamily is the announcement's: a kind and an item, sent by one
// person from one working copy for a while, validated by
// validateAnnouncement.
var announcementFamily = &family{
	name:    "announcement",
	keys:    []string{keyAnnouncement, keyItem, keySender, keyCopy, keySentAt, keyExpiresAt, keyAssistedBy},
	shorten: "an announcement carries no free text, so this is a defect in abcd",
}

// statusBranches is the status branches' namespace. The whole namespace is
// fetched, never one branch: a fetch naming a branch origin does not hold yet
// (a person's first announcement) fails.
var statusBranches = namespace{remote: StatusRefPrefix, tracking: StatusTrackingPrefix, stateDir: tierRelDir + "/status", noun: "the status branches"}

// noAnnouncement ends a refusal met reading or writing the remote.
const noAnnouncement = "no announcement was sent"

// The reasons an announcement is unreadable, beside the reservation's
// reasonCopy and reasonTime, which it shares. Each is fixed text: nothing a
// writer chose is echoed into one.
const (
	reasonAnnouncementKind   = "its kind is not one of merging, merged, asking, answered, handed-over"
	reasonAnnouncementItem   = "it does not name one issue or intent id or pull-request number as its item"
	reasonAnnouncementSender = "it does not name one GitHub username as its sender"
	reasonAnnouncementExpiry = "its expiry is not after the time it was sent and within 24 hours of it"
	reasonOutsideTrailers    = "it carries text outside its trailers, or a trailer not written exactly"
)

// prNumberRe is a pull request's number as an item names it, with or without
// its "#": a shell reads a word starting with "#" as a comment.
var prNumberRe = regexp.MustCompile(`^#?([1-9][0-9]{0,8})$`)

// EmbargoCheck asks the embargo store on this machine (`abcd embargo check`)
// whether an announcement may be made public: text is its whole message and
// records the record ids it names. held is true when the check refuses it;
// err when the check cannot run, which refuses it too. The front door wires
// it to internal/core/embargo, which this package cannot import (embargo
// writes its records through capture, which reaches this package).
type EmbargoCheck func(root, home, text string, records []string) (held bool, err error)

// AnnounceRequest is one announcement asked for.
type AnnounceRequest struct {
	// Kind is one of AnnouncementKinds.
	Kind string
	// Item is the record (iss-N or itd-N) or the pull request (#N, or N) it
	// concerns.
	Item string
	// For is how long it stands; zero means DefaultAnnouncementLife.
	For time.Duration
}

// AnnounceResult is an announcement that reached origin.
type AnnounceResult struct {
	Kind string
	// Item is in its canonical form: a record's canonical id, or "#N".
	Item string
	// Sender is the person's GitHub username as resolved, and Branch their
	// status branch, abcd-status/<Sender in lower case>.
	Sender    string
	Branch    string
	SentAt    time.Time
	ExpiresAt time.Time
	Commit    string
	// Attempts is how many pushes it took: 1, or more after a lost race.
	Attempts int
}

// NotSentError is an announcement origin did not take: the race was lost on
// every retry, or the push was refused for another reason. Reason says why,
// home-redacted. It is an ErrRefused refusal.
type NotSentError struct {
	Reason string
}

func (e *NotSentError) Error() string        { return "not sent: " + e.Reason }
func (e *NotSentError) Is(target error) bool { return target == ErrRefused }

// AnnounceError is an announcement that failed for a reason abcd does not
// class as a refusal (a defect, or a failure no check names), so its message
// does not say what became of the push. Pushes is how many pushes were tried
// before it: 0 means nothing was pushed; more means each was refused, but a
// push git reports as refused can still have reached origin, so whether one
// did is not known. Branch is the status branch they went to. Announce returns
// every error that is not an ErrRefused refusal as one.
type AnnounceError struct {
	Err    error
	Pushes int
	Branch string
}

func (e *AnnounceError) Error() string { return e.Err.Error() }
func (e *AnnounceError) Unwrap() error { return e.Err }

// announceFailure is err as Announce returns it: unchanged when it is nil or
// an ErrRefused refusal, which says what became of the push itself, and
// otherwise an *AnnounceError counting the pushes tried before it.
func announceFailure(err error, pushes int, branch string) error {
	if err == nil || errors.Is(err, ErrRefused) {
		return err
	}
	return &AnnounceError{Err: err, Pushes: pushes, Branch: branch}
}

// announcement is one announcement commit's content.
type announcement struct {
	Kind       string
	Item       string
	Sender     string
	Copy       string
	SentAt     time.Time
	ExpiresAt  time.Time
	AssistedBy string
}

// messageFamily makes an announcement a commitMessage.
func (announcement) messageFamily() *family { return announcementFamily }

// subject is the subject line an announcement's kind and item give; the
// validator holds the message's lead to it.
func (a announcement) subject() string { return "abcd announcement: " + a.Kind + " " + a.Item }

// text is the commit message: the subject, a blank line, then one trailer per
// line and nothing else. Every value is written here, on standard input to
// commit-tree, and never becomes a git argument.
func (a announcement) text() string {
	var b strings.Builder
	b.WriteString(a.subject() + "\n\n")
	line := func(k, v string) { b.WriteString(k + ": " + v + "\n") }
	line(keyAnnouncement, a.Kind)
	line(keyItem, a.Item)
	line(keySender, a.Sender)
	line(keyCopy, a.Copy)
	line(keySentAt, stamp(a.SentAt))
	line(keyExpiresAt, stamp(a.ExpiresAt))
	if a.AssistedBy != "" {
		line(keyAssistedBy, a.AssistedBy)
	}
	return b.String()
}

// parseAnnouncement reads an announcement message: its trailers through the
// one collector, judged by the announcement's validator. It returns the reason
// the message is unreadable, "" when it reads.
func parseAnnouncement(text string) (announcement, string) {
	return validateAnnouncement(collectTrailers(text, announcementFamily.keys))
}

// validateAnnouncement is the announcement family's validator, and it admits
// nothing but its own trailers (C3): the final paragraph holds only its keys,
// each written exactly `Key: value` at the start of its line, in its
// canonical spelling; the message before it is exactly the subject the kind
// and item give; the kind is in the list, the item an issue or intent id or a
// pull-request number, the sender a GitHub username and the copy a key, each
// once; the times are RFC 3339 in UTC to the second, the expiry after the time
// sent and within 24 hours of it; and the attribution appears at most once.
func validateAnnouncement(t trailers) (announcement, string) {
	// The lead is the subject alone: one line, no paragraph after it.
	if len(t.other) > 0 || len(t.inexact) > 0 || len(t.vals[keyAssistedBy]) > 1 || t.lead == "" || strings.Contains(t.lead, "\n") {
		return announcement{}, reasonOutsideTrailers
	}
	var a announcement
	var ok bool
	if a.Kind, ok = t.one(keyAnnouncement); !ok || !slices.Contains(AnnouncementKinds, a.Kind) {
		return announcement{}, reasonAnnouncementKind
	}
	item, ok := t.one(keyItem)
	if !ok {
		return announcement{}, reasonAnnouncementItem
	}
	if canon, _, err := announcementItem(item); err != nil || canon != item {
		return announcement{}, reasonAnnouncementItem
	}
	a.Item = item
	if t.lead != a.subject() {
		return announcement{}, reasonOutsideTrailers
	}
	if a.Sender, ok = t.one(keySender); !ok || !validUsername(a.Sender) {
		return announcement{}, reasonAnnouncementSender
	}
	if a.Copy, ok = t.one(keyCopy); !ok || !validCopyKey(a.Copy) {
		return announcement{}, reasonCopy
	}
	for _, f := range []struct {
		key  string
		into *time.Time
	}{{keySentAt, &a.SentAt}, {keyExpiresAt, &a.ExpiresAt}} {
		v, ok := t.one(f.key)
		if !ok {
			return announcement{}, reasonTime
		}
		at, err := time.Parse(time.RFC3339, v)
		if err != nil || stamp(at) != v {
			return announcement{}, reasonTime
		}
		*f.into = at.UTC()
	}
	if life := a.ExpiresAt.Sub(a.SentAt); life <= 0 || life > MaxAnnouncementLife {
		return announcement{}, reasonAnnouncementExpiry
	}
	if v, ok := t.one(keyAssistedBy); ok {
		a.AssistedBy = v
	}
	return a, ""
}

// announcementItem is an item in its canonical form, and whether it is a
// record (iss-N or itd-N) rather than a pull request ("#N"), or the refusal.
func announcementItem(arg string) (canon string, record bool, err error) {
	if m := prNumberRe.FindStringSubmatch(arg); m != nil {
		return "#" + m[1], false, nil
	}
	if recordid.CitedIDRe.MatchString(arg) {
		if id := recordid.CanonCitedID(arg); id != "" && (id[:4] == "iss-" || id[:4] == "itd-") {
			return id, true, nil
		}
	}
	return "", false, refuse("item %s is not an issue or intent id (iss-N, itd-N) or a pull-request number (#N); %s", echo(arg), nothingPushed)
}

// preparedAnnouncement is a request checked and composed from the checkout,
// before the remote is asked anything.
type preparedAnnouncement struct {
	root   string
	home   string
	now    time.Time
	who    identity
	msg    announcement
	record bool
	branch string
}

// prepareAnnouncement checks a request in the order the refusals promise: the
// kind, the item and the expiry before any git runs (C3), then the person's
// git identity, refused when it is a tool's before anything is written, then
// the holder and the working copy, then the composed message, read back
// through the validator and measured against the cap and the outbound policy.
func prepareAnnouncement(root string, req AnnounceRequest, o Options) (*preparedAnnouncement, error) {
	if !slices.Contains(AnnouncementKinds, req.Kind) {
		return nil, refuse("kind %s is not one of %s; %s", echo(req.Kind), strings.Join(AnnouncementKinds, ", "), nothingPushed)
	}
	item, record, err := announcementItem(req.Item)
	if err != nil {
		return nil, err
	}
	life := req.For
	if life == 0 {
		life = DefaultAnnouncementLife
	}
	if life < MinAnnouncementLife || life > MaxAnnouncementLife {
		return nil, refuse("expiry %s is outside %s to %s; %s", life, MinAnnouncementLife, MaxAnnouncementLife, nothingPushed)
	}
	p := &preparedAnnouncement{root: root, home: o.home(), now: o.now(), record: record}
	if p.who, err = gitIdentity(root); err != nil {
		return nil, err
	}
	// A hosted container's default identity is a tool's. An announcement is
	// authored as the person who sends it, so it is refused here, before a
	// copy key, an object or a ref is written, and the refusal never repeats
	// the identity it rejected.
	if toolid.IsToolIdentity(toolid.RoleAuthor, p.who.Name, p.who.Email) {
		return nil, refuse("could not work out whose GitHub account this session uses: git's user.name and user.email here name a tool, "+
			"not a person, and an announcement is authored as the person who sends it; set git config user.email to your GitHub "+
			"no-reply address (<id>+<username>@users.noreply.github.com) and git config user.name to your own name, then retry; %s", nothingPushed)
	}
	sender, err := resolveHolder(root, p.home, p.who.Email)
	if err != nil {
		return nil, err
	}
	key, err := copyKey(root)
	if err != nil {
		return nil, err
	}
	p.msg = announcement{Kind: req.Kind, Item: item, Sender: sender, Copy: key, SentAt: p.now, ExpiresAt: p.now.Add(life),
		AssistedBy: o.assistedBy()}
	p.branch = strings.ToLower(sender)
	text := p.msg.text()
	if back, why := parseAnnouncement(text); why != "" || back != p.msg {
		return nil, refuse("the announcement abcd composed does not read back (%s), which is a defect in abcd; %s", why, nothingPushed)
	}
	if err := checkText(root, announcementFamily, text); err != nil {
		return nil, err
	}
	return p, nil
}

// Announce sends one announcement: it checks and composes it, asks the
// embargo check (C5), fetches the status branches and the default branch in
// one fetch, refuses a record the default branch does not hold (C4), writes
// one empty-tree commit on the sender's status branch as the person, and makes
// one plain push. A push that lost a race to another of the person's sessions
// is rebuilt on the tip that won and pushed again, up to three times (C2);
// after that, or on a push refused for any other reason, it is a
// *NotSentError. Each refused push is followed by one fetch, the last
// included, so the refusal says which of the two it was. Nothing is ever
// forced. A refusal pushes nothing. Any other error is an *AnnounceError
// counting the pushes tried before it.
func Announce(root string, req AnnounceRequest, o Options) (_ AnnounceResult, err error) {
	pushes, branch := 0, ""
	defer func() { err = announceFailure(err, pushes, branch) }()
	p, err := prepareAnnouncement(root, req, o)
	if err != nil {
		return AnnounceResult{}, err
	}
	var records []string
	if p.record {
		records = []string{p.msg.Item}
	}
	if o.Embargo == nil {
		return AnnounceResult{}, refuse("the embargo store on this machine could not be checked: no embargo check was given; %s", nothingPushed)
	}
	held, err := o.Embargo(root, p.home, p.msg.text(), records)
	if err != nil {
		return AnnounceResult{}, refuse("an announcement about %s is refused: the embargo store on this machine could not be checked: %s; %s",
			p.msg.Item, redacted(err), nothingPushed)
	}
	if held {
		return AnnounceResult{}, refuse("an announcement about %s is refused: the embargo store on this machine holds it, or text the announcement carries, under embargo; %s",
			p.msg.Item, nothingPushed)
	}
	if err := originPresent(root, "announcements"); err != nil {
		return AnnounceResult{}, err
	}
	tree, err := emptyTree(root)
	if err != nil {
		return AnnounceResult{}, refuse("%s; %s", redacted(err), nothingPushed)
	}
	gap, err := fetch(root, p.now, noAnnouncement, statusBranches)
	if err != nil {
		return AnnounceResult{}, err
	}
	if p.record {
		if err := publicOnDefault(root, p.msg.Item, gap); err != nil {
			return AnnounceResult{}, err
		}
	}
	parent, err := statusTip(root, p.branch)
	if err != nil {
		return AnnounceResult{}, err
	}
	ref := statusBranches.ref(p.branch)
	branch = "abcd-status/" + p.branch
	sent := func(sha string, attempts int) AnnounceResult {
		return AnnounceResult{Kind: p.msg.Kind, Item: p.msg.Item, Sender: p.msg.Sender, Branch: branch,
			SentAt: p.msg.SentAt, ExpiresAt: p.msg.ExpiresAt, Commit: sha, Attempts: attempts}
	}
	// refused is what origin said of the last push it refused, "" before any.
	refused := ""
	for attempt := 1; ; attempt++ {
		sha, err := writeEmptyCommit(root, p.who, p.now, p.msg, tree, parent)
		if err != nil {
			if pushes == 0 {
				return AnnounceResult{}, err
			}
			// A push was made, so this is not a refusal that pushed nothing: it
			// says what origin said of the last push and why the retry was not
			// written, and the front door adds that a push was made.
			return AnnounceResult{}, fmt.Errorf("%s; %s moved, and the announcement rebuilt on its new tip could not be written: %s",
				refused, branch, withoutOutcome(err))
		}
		if beforePush != nil {
			beforePush(root)
		}
		perr := push(root, pushTo(sha, ref))
		pushes++
		if perr == nil {
			return sent(sha, attempt), nil
		}
		refused = Remote + " refused the push: " + redacted(perr)
		// The tip check runs first, on every attempt, the last included: a lost
		// race shows as a tip that moved, and a push refused while the tip
		// stands was refused for another reason, so it is not retried and is
		// not blamed on the race, however many races came before it. When the
		// branch cannot be read back, whether the push landed is not known, so
		// it is not called "not sent": it is an *AnnounceError counting the
		// push.
		if _, ferr := fetch(root, p.now, noAnnouncement, statusBranches); ferr != nil {
			return AnnounceResult{}, fmt.Errorf("%s; %s could not be fetched afterwards to see whether the push landed: %s",
				refused, branch, withoutOutcome(ferr))
		}
		next, terr := statusTip(root, p.branch)
		if terr != nil {
			return AnnounceResult{}, fmt.Errorf("%s; %s could not be fetched afterwards to see whether the push landed: %s",
				refused, branch, withoutOutcome(terr))
		}
		// A push git reports as refused can still have reached origin, and
		// another session may have pushed on top of it since. Either way the
		// branch now holds this commit, so it was sent, and is not sent again.
		landed := next == sha
		if !landed && next != "" {
			if landed, err = statusHolds(root, sha, next); err != nil {
				return AnnounceResult{}, fmt.Errorf("%s; whether the push landed on %s could not be read: %s", refused, branch, redacted(err))
			}
		}
		if landed {
			return sent(sha, attempt), nil
		}
		if next == parent {
			if attempt == 1 {
				return AnnounceResult{}, &NotSentError{Reason: refused}
			}
			return AnnounceResult{}, &NotSentError{Reason: fmt.Sprintf("%s; abcd-status/%s had not moved, so this push lost no race (the %d before it each did)",
				refused, p.branch, attempt-1)}
		}
		if attempt > maxAnnounceRetries {
			return AnnounceResult{}, &NotSentError{Reason: fmt.Sprintf("abcd-status/%s moved under this push and each of %d retries; %s",
				p.branch, maxAnnounceRetries, refused)}
		}
		parent = next
	}
}

// statusHolds reports whether commit is on the status branch that ends in
// tip: an ancestor of it, read as stored (storedObjects). git answers in the
// exit status, 1 meaning no.
func statusHolds(root, commit, tip string) (bool, error) {
	trace("merge-base", "--is-ancestor", commit, tip)
	_, no, err := gitutil.RunCappedExitOne(root, 4096, storedObjects, "merge-base", "--is-ancestor", commit, tip)
	if err != nil {
		return false, err
	}
	return !no, nil
}

// withoutOutcome is a refusal's text without the closing words that say what
// was not done, for quoting inside a message that says it instead.
func withoutOutcome(err error) string {
	msg := redacted(err)
	for _, end := range []string{nothingPushed, noAnnouncement} {
		msg = strings.TrimSuffix(msg, "; "+end)
	}
	return msg
}

// statusTip is the commit the person's status branch ends in as just fetched,
// "" when origin holds none yet.
func statusTip(root, branch string) (string, error) {
	ref := statusBranches.track(branch)
	name, typ, ok, err := resolveRef(root, ref)
	if err != nil {
		return "", refuse("could not read %s: %s; %s", gitutil.ShortRef(ref), redacted(err), noAnnouncement)
	}
	if !ok {
		return "", nil
	}
	if typ != "commit" {
		return "", refuse("abcd-status/%s on %s does not end in a commit, so nothing can be added to it; %s", branch, Remote, noAnnouncement)
	}
	return name, nil
}

// publicOnDefault refuses a record absent from origin's default branch as
// just fetched (C4), read through peers.FoldersAt, the reader the derived
// release reads by; and refuses any record when no default branch was
// fetched, or the fetch could not read it (gap), since a stale copy cannot say
// what is public now.
func publicOnDefault(root, id, gap string) error {
	ref := originDefault(root)
	if ref == "" {
		return refuse("no default branch of %s was fetched here (this clone has no %s/HEAD), so whether %s is public cannot be told; "+
			"run git remote set-head %s --auto, then retry; %s", Remote, Remote, id, Remote, noAnnouncement)
	}
	if gap != "" {
		return refuse("%s, so whether %s is public cannot be told; %s", gap, id, noAnnouncement)
	}
	folders, err := peers.FoldersAt(root, ref)
	if err != nil {
		return refuse("could not read whether %s is on %s's default branch: %s; %s", id, Remote, redacted(err), noAnnouncement)
	}
	if len(folders[id]) == 0 {
		return refuse("%s is not on %s's default branch as last fetched (%s), so it is not public yet: an announcement names only a public record; %s",
			id, Remote, strings.TrimPrefix(ref, "refs/remotes/"+Remote+"/"), noAnnouncement)
	}
	return nil
}
