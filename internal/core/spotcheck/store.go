// Package spotcheck keeps the spot-check's record (itd-2610301750162657,
// spc-2610301925429250): the work abcd decided with no person involved,
// recorded as it happens, for a later draw to read.
//
// THE STORE. The record lives on one branch of the shared remote,
// BranchRef on origin, tracked at TrackingRef. It is an orphan whose tree
// holds folders of small JSON files, each written once and never edited:
// handbacks/ holds the drain's hand-backs and decisions/ the automated
// facilitator's decisions (the drain's judgements a model answered and the
// merge gate's in auto mode). Every write is a new file whose name its
// content fixes, so two writers never edit one file.
//
// A write is committed on this checkout's local copy, LocalRef, at once,
// touching no HEAD, index or working tree, and reaches no network. LocalRef
// lives outside refs/heads/ and refs/remotes/, so no fetch moves it and every
// worktree of the checkout shares it: a lane's worktree removed after its
// landing takes none of its records with it. Push carries the local copy to
// origin: one fetch of the branch, the records origin lacks laid over its tip,
// and one plain push, never forced, so nothing another writer pushed is
// dropped; a push origin rejects is tried once more after a fresh fetch, and
// one that fails again leaves every record on the local copy for the next
// push. A record whose name origin already holds is kept as origin holds it,
// and the note says so.
//
// Everything read from the store was written by another session, account or
// machine, so it is read strictly: names are checked against the forms this
// package writes, each file is bounded and strictly decoded, and a store that
// cannot be read is an error, never an empty store. What a record carries is
// cleaned when it is written: control characters out, the home directory and
// any absolute path replaced, each text bounded, and the writer named only by
// its holder id, the form the reservations use.
package spotcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The store's refs.
const (
	// Branch is the store's branch on origin.
	Branch = "abcd-spotcheck"
	// BranchRef is the branch as a full ref on origin.
	BranchRef = "refs/heads/" + Branch
	// TrackingRef is where a fetch keeps origin's copy.
	TrackingRef = "refs/remotes/" + claim.Remote + "/" + Branch
	// LocalRef is this checkout's local copy, every record written here and
	// not yet pushed included.
	LocalRef = "refs/abcd/spotcheck"
)

// The store's folders.
const (
	HandBacksDir = "handbacks"
	DecisionsDir = "decisions"
)

// SchemaVersion is the version every record this package writes carries; a
// record of any other version is not read.
const SchemaVersion = 1

// The kinds of decision a decisions/ file holds.
const (
	KindDrainJudgement = "drain-judgement"
	KindMergeGate      = "merge-gate"
)

// The sites a merge-gate decision is recorded from.
const (
	SiteLand = "land"
	SiteLane = "lane"
)

// Bounds.
const (
	// WriteTimeout bounds one record's write, all of its git commands
	// together. Each is local, so a write that takes longer is a git that
	// hangs, and the write is given up rather than hold its caller.
	WriteTimeout = 10 * time.Second
	// maxTextBytes bounds each free-text field a record carries.
	maxTextBytes = 2000
	// maxRecordBytes bounds one record file, on write and on read.
	maxRecordBytes = 16 << 10
	// maxListingBytes bounds one folder's listing.
	maxListingBytes = 8 << 20
	// stamp is the form a record's time takes in its file name.
	stamp = "20060102T150405Z"
)

// The identity a store commit is written as: abcd's, so no commit on the
// branch carries a person's name or address. The record names its writer by
// holder id.
const (
	commitName  = "abcd spotcheck"
	commitEmail = "spotcheck@abcd.invalid"
)

// ErrNoOrigin is a checkout with no remote named origin: there is no shared
// remote to keep the record on, and nothing is written.
var ErrNoOrigin = errors.New("this checkout has no remote named " + claim.Remote + ", so the spot-check keeps no record here")

// HandBack is one of the drain's hand-backs, as handbacks/ holds it.
type HandBack struct {
	SchemaVersion int       `json:"schema_version"`
	At            time.Time `json:"at"`
	Issue         string    `json:"issue"`
	// From, Kind and Route are the drain's own words for where the hand-back
	// came from, what kind it is and where it went.
	From  string `json:"from"`
	Kind  string `json:"kind"`
	Route string `json:"route"`
	// Rule is the rule a field hand-back names; DrainRule the decision record
	// the drain read its rule from.
	Rule      string `json:"rule,omitempty"`
	DrainRule string `json:"drain_rule,omitempty"`
	Reason    string `json:"reason"`
	// PR is the lane's pull request, when the lane opened one; RunID the
	// lane's run.
	PR    int    `json:"pr,omitempty"`
	RunID string `json:"run_id,omitempty"`
	// Blob is the issue record's blob on origin's default branch when the
	// hand-back was routed.
	Blob   string `json:"blob,omitempty"`
	Holder string `json:"holder,omitempty"`
}

// Decision is one of the automated facilitator's decisions, as decisions/
// holds it: the drain's judgement a model answered (KindDrainJudgement) or
// the merge gate's in auto mode over a maintainer's pull request
// (KindMergeGate). Each kind sets its own fields and leaves the other's empty.
type Decision struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	At            time.Time `json:"at"`
	Holder        string    `json:"holder,omitempty"`

	// A drain judgement: the issue, the question put, the answer and the
	// hand-back kind a yes named, the model that answered, the drain's rule
	// record and the reason given.
	Issue      string `json:"issue,omitempty"`
	Question   string `json:"question,omitempty"`
	Answer     string `json:"answer,omitempty"`
	AnswerKind string `json:"answer_kind,omitempty"`
	Model      string `json:"model,omitempty"`
	Rule       string `json:"rule,omitempty"`
	Reason     string `json:"reason,omitempty"`

	// A merge-gate decision: where it was asked, the pull request and its
	// head, the mode and the layer that declared it, the author's role,
	// whether a review receipt named the head, and the gate's answer.
	Site      string `json:"site,omitempty"`
	PR        int    `json:"pr,omitempty"`
	Head      string `json:"head,omitempty"`
	Mode      string `json:"mode,omitempty"`
	ModeLayer string `json:"mode_layer,omitempty"`
	Role      string `json:"role,omitempty"`
	Reviewed  bool   `json:"reviewed,omitempty"`
	Arm       bool   `json:"arm,omitempty"`
	Method    string `json:"method,omitempty"`
	Why       string `json:"why,omitempty"`
}

var (
	recordIDRe = regexp.MustCompile(`^(iss|itd)-[0-9]{1,20}$`)
	runIDRe    = regexp.MustCompile(`^run-[0-9]{1,20}$`)
	holderRe   = regexp.MustCompile(`^h_[0-9a-f]{32}$`)
	shaRe      = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	wordRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9._:/-]{0,127}$`)
	// The names a folder holds: each file's name is fixed by its content.
	handBackNameRe = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-(iss|itd)-[0-9]{1,20}\.json$`)
	decisionNameRe = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-((iss|itd)-[0-9]{1,20}|pr[0-9]{1,10}-[0-9a-f]{12}-(armed|open))\.json$`)
	// absPathRe finds an absolute path in free text: a slash after the start
	// or a separator, then at least one segment ending in a slash.
	absPathRe = regexp.MustCompile(`(^|[\s"'(=,;:\[])/(?:[^\s/"'()\[\],;]+/)+[^\s/"'()\[\],;]*`)
)

// Name is the file a hand-back is kept in, under HandBacksDir.
func (h HandBack) Name() string {
	return h.At.UTC().Format(stamp) + "-" + h.Issue + ".json"
}

// Name is the file a decision is kept in, under DecisionsDir: a drain
// judgement by its issue, a merge-gate decision by its pull request, its
// head and whether it armed, so the gate asked again over the same head with
// the same answer keeps the one record (MergeGateKey).
func (d Decision) Name() string {
	if d.Kind == KindMergeGate {
		return d.At.UTC().Format(stamp) + "-" + d.MergeGateKey() + ".json"
	}
	return d.At.UTC().Format(stamp) + "-" + d.Issue + ".json"
}

// MergeGateKey is what names one merge-gate decision apart from its time.
func (d Decision) MergeGateKey() string {
	head := d.Head
	if len(head) > 12 {
		head = head[:12]
	}
	answer := "open"
	if d.Arm {
		answer = "armed"
	}
	return "pr" + strconv.Itoa(d.PR) + "-" + head + "-" + answer
}

// clean is free text as a record keeps it: the home directory and every
// absolute path replaced, control characters out, at most maxTextBytes.
func clean(s string) string {
	s = fsutil.RedactHome(s)
	s = absPathRe.ReplaceAllString(s, "${1}<path>")
	s = termsafe.Sanitize(s)
	if len(s) > maxTextBytes {
		cut := maxTextBytes
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "…"
	}
	return s
}

// cleanWord is a field the drain or the gate names from a fixed set (a kind,
// a route, a mode): kept when it reads as one, dropped otherwise.
func cleanWord(s string) string {
	if wordRe.MatchString(s) {
		return s
	}
	return ""
}

// normalise is h as it is written, every field checked or cleaned.
func (h HandBack) normalise() (HandBack, error) {
	h.SchemaVersion = SchemaVersion
	h.At = h.At.UTC().Truncate(time.Second)
	if !recordIDRe.MatchString(h.Issue) {
		return HandBack{}, fmt.Errorf("a hand-back names no record id the store keeps (%s)", termsafe.Sanitize(echo(h.Issue)))
	}
	if h.At.IsZero() {
		return HandBack{}, errors.New("a hand-back carries no time")
	}
	h.From, h.Kind, h.Route = cleanWord(h.From), cleanWord(h.Kind), cleanWord(h.Route)
	h.Rule, h.DrainRule, h.Reason = clean(h.Rule), clean(h.DrainRule), clean(h.Reason)
	if h.PR < 0 {
		h.PR = 0
	}
	if !runIDRe.MatchString(h.RunID) {
		h.RunID = ""
	}
	if !shaRe.MatchString(h.Blob) {
		h.Blob = ""
	}
	if !holderRe.MatchString(h.Holder) {
		h.Holder = ""
	}
	return h, nil
}

// normalise is d as it is written, every field checked or cleaned.
func (d Decision) normalise() (Decision, error) {
	d.SchemaVersion = SchemaVersion
	d.At = d.At.UTC().Truncate(time.Second)
	if d.At.IsZero() {
		return Decision{}, errors.New("a decision carries no time")
	}
	if !holderRe.MatchString(d.Holder) {
		d.Holder = ""
	}
	switch d.Kind {
	case KindDrainJudgement:
		if !recordIDRe.MatchString(d.Issue) {
			return Decision{}, fmt.Errorf("a judgement names no record id the store keeps (%s)", termsafe.Sanitize(echo(d.Issue)))
		}
		d.Answer, d.AnswerKind, d.Model = cleanWord(d.Answer), cleanWord(d.AnswerKind), clean(d.Model)
		d.Question, d.Rule, d.Reason = clean(d.Question), clean(d.Rule), clean(d.Reason)
		d.Site, d.PR, d.Head, d.Mode, d.ModeLayer, d.Role, d.Reviewed, d.Arm, d.Method, d.Why = "", 0, "", "", "", "", false, false, "", ""
	case KindMergeGate:
		if d.PR <= 0 {
			return Decision{}, errors.New("a merge-gate decision names no pull request")
		}
		if !shaRe.MatchString(d.Head) {
			return Decision{}, errors.New("a merge-gate decision names no head commit")
		}
		if d.Site != SiteLand && d.Site != SiteLane {
			return Decision{}, fmt.Errorf("a merge-gate decision names no site the store keeps (%s)", termsafe.Sanitize(echo(d.Site)))
		}
		d.Mode, d.ModeLayer, d.Role, d.Method = cleanWord(d.Mode), cleanWord(d.ModeLayer), cleanWord(d.Role), cleanWord(d.Method)
		d.Why = clean(d.Why)
		d.Issue, d.Question, d.Answer, d.AnswerKind, d.Model, d.Rule, d.Reason = "", "", "", "", "", "", ""
	default:
		return Decision{}, fmt.Errorf("a decision of kind %s is not one the store keeps", termsafe.Sanitize(echo(d.Kind)))
	}
	return d, nil
}

// echo is s cut for a refusal to name.
func echo(s string) string {
	if len(s) > 64 {
		return s[:64] + "…"
	}
	return strconv.Quote(s)
}

// WriteHandBack commits h on the local copy, under HandBacksDir: wrote is
// false when the copy already holds its file, which is then left as it is.
// The record is cleaned first (normalise). It reaches no network.
func WriteHandBack(ctx context.Context, root string, h HandBack) (wrote bool, err error) {
	h, err = h.normalise()
	if err != nil {
		return false, err
	}
	data, err := encode(h)
	if err != nil {
		return false, err
	}
	name := h.Name()
	return appendFile(ctx, root, HandBacksDir, name, data, func(names []string) bool { return slices.Contains(names, name) })
}

// WriteDecision commits d on the local copy, under DecisionsDir: wrote is
// false when the copy already holds it, a merge-gate decision over the same
// pull request and head with the same answer at any time included. It reaches
// no network.
func WriteDecision(ctx context.Context, root string, d Decision) (wrote bool, err error) {
	d, err = d.normalise()
	if err != nil {
		return false, err
	}
	data, err := encode(d)
	if err != nil {
		return false, err
	}
	name := d.Name()
	held := func(names []string) bool { return slices.Contains(names, name) }
	if d.Kind == KindMergeGate {
		suffix := "-" + d.MergeGateKey() + ".json"
		held = func(names []string) bool {
			return slices.ContainsFunc(names, func(n string) bool { return strings.HasSuffix(n, suffix) })
		}
	}
	return appendFile(ctx, root, DecisionsDir, name, data, held)
}

// encode is v as a record file holds it: indented JSON and a newline, within
// maxRecordBytes.
func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if b.Len() > maxRecordBytes {
		return nil, fmt.Errorf("the record is %d bytes, over the store's %d", b.Len(), maxRecordBytes)
	}
	return b.Bytes(), nil
}

// treeEntry is one line of a tree as git lists it.
type treeEntry struct {
	mode, typ, sha, name string
}

func (e treeEntry) line() string { return e.mode + " " + e.typ + " " + e.sha + "\t" + e.name + "\x00" }

// appendFile commits name in folder, holding data, on the local copy, unless
// held says the folder already holds it. The copy is moved only from the
// value this write read (a compare-and-swap), so a writer in another worktree
// of the checkout that moved it first makes this one read again; three
// tries, then the write is given up.
func appendFile(parent context.Context, root, folder, name string, data []byte, held func([]string) bool) (bool, error) {
	ctx, cancel := context.WithTimeout(parent, WriteTimeout)
	defer cancel()
	if err := originPresent(ctx, root); err != nil {
		return false, err
	}
	var last error
	for range 3 {
		refs, err := readRefs(ctx, root)
		if err != nil {
			return false, err
		}
		base, err := refs.base(ctx, root)
		if err != nil {
			return false, err
		}
		top, entries, err := folderEntries(ctx, root, base, folder)
		if err != nil {
			return false, err
		}
		if held(entryNames(entries)) {
			return false, nil
		}
		blob, err := git(ctx, root, string(data), "hash-object", "-w", "--stdin")
		if err != nil {
			return false, err
		}
		entries = append(entries, treeEntry{mode: "100644", typ: "blob", sha: blob, name: name})
		tree, err := rootWith(ctx, root, top, folder, entries)
		if err != nil {
			return false, err
		}
		commit, err := commitTree(ctx, root, tree, base, "spotcheck: "+folder+"/"+name)
		if err != nil {
			return false, err
		}
		if err := moveLocal(ctx, root, commit, refs.local); err != nil {
			last = err
			continue
		}
		return true, nil
	}
	return false, fmt.Errorf("the local copy of the spot-check's record kept moving while this write was made: %w", last)
}

// refs are the store's two refs as this checkout holds them, "" for one it
// does not.
type refs struct {
	local, tracking string
}

// readRefs reads LocalRef and TrackingRef. A ref that names anything but a
// commit is an unreadable store, never an absent one.
func readRefs(ctx context.Context, root string) (refs, error) {
	out, err := git(ctx, root, "", "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)", LocalRef, TrackingRef)
	if err != nil {
		return refs{}, err
	}
	var r refs
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) != 3 || (f[0] != LocalRef && f[0] != TrackingRef) {
			continue
		}
		if f[2] != "commit" || !gitutil.IsFullSHA(f[1]) {
			return refs{}, fmt.Errorf("%s names a %s, not a commit, so the spot-check's record cannot be read", f[0], termsafe.Sanitize(f[2]))
		}
		if f[0] == LocalRef {
			r.local = f[1]
		} else {
			r.tracking = f[1]
		}
	}
	return r, nil
}

// base is the commit a new record is laid on: the local copy, or origin's
// copy as last fetched when the local copy holds nothing it lacks; "" for a
// store neither holds yet.
func (r refs) base(ctx context.Context, root string) (string, error) {
	switch {
	case r.local == "":
		return r.tracking, nil
	case r.tracking == "" || r.tracking == r.local:
		return r.local, nil
	}
	behind, err := isAncestor(ctx, root, r.local, r.tracking)
	if err != nil {
		return "", err
	}
	if behind {
		return r.tracking, nil
	}
	return r.local, nil
}

// isAncestor reports whether a is an ancestor of b.
func isAncestor(ctx context.Context, root, a, b string) (bool, error) {
	_, one, err := gitutil.RunCappedExitOneContext(ctx, root, 64, "merge-base", "--is-ancestor", a, b)
	if err != nil {
		return false, fmt.Errorf("reading the spot-check's record: %w", err)
	}
	return !one, nil
}

// folderEntries are the root tree's entries at commit and folder's own; both
// empty for no commit.
func folderEntries(ctx context.Context, root, commit, folder string) (top, entries []treeEntry, err error) {
	if commit == "" {
		return nil, nil, nil
	}
	top, err = listTree(ctx, root, commit+"^{tree}")
	if err != nil {
		return nil, nil, err
	}
	for _, e := range top {
		if e.name != folder {
			continue
		}
		if e.typ != "tree" {
			return nil, nil, fmt.Errorf("the spot-check's record holds %s as a %s, not a folder", folder, termsafe.Sanitize(e.typ))
		}
		entries, err = listTree(ctx, root, e.sha)
		return top, entries, err
	}
	return top, nil, nil
}

// listTree lists one tree, strictly: a line that does not read as an entry
// is an unreadable store.
func listTree(ctx context.Context, root, tree string) ([]treeEntry, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxListingBytes, "ls-tree", "-z", tree)
	if err != nil {
		return nil, fmt.Errorf("reading the spot-check's record: %w", err)
	}
	var entries []treeEntry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, name, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 || name == "" || strings.ContainsAny(name, "/\n") || !gitutil.IsFullSHA(f[2]) {
			return nil, errors.New("the spot-check's record holds a tree entry that does not read as one")
		}
		entries = append(entries, treeEntry{mode: f[0], typ: f[1], sha: f[2], name: name})
	}
	return entries, nil
}

func entryNames(entries []treeEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.name)
	}
	return names
}

// rootWith is the root tree top with folder holding entries.
func rootWith(ctx context.Context, root string, top []treeEntry, folder string, entries []treeEntry) (string, error) {
	sub, err := mktree(ctx, root, entries)
	if err != nil {
		return "", err
	}
	next := slices.DeleteFunc(slices.Clone(top), func(e treeEntry) bool { return e.name == folder })
	next = append(next, treeEntry{mode: "040000", typ: "tree", sha: sub, name: folder})
	return mktree(ctx, root, next)
}

// mktree writes one tree of entries.
func mktree(ctx context.Context, root string, entries []treeEntry) (string, error) {
	var in strings.Builder
	for _, e := range entries {
		in.WriteString(e.line())
	}
	return git(ctx, root, in.String(), "mktree", "-z")
}

// commitTree writes a commit of tree over parent ("" for the first), as
// abcd's identity.
func commitTree(ctx context.Context, root, tree, parent, msg string) (string, error) {
	args := []string{"-c", "user.name=" + commitName, "-c", "user.email=" + commitEmail, "commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	return git(ctx, root, msg+"\n", append(args, "-F", "-")...)
}

// moveLocal moves LocalRef to commit from old, the value read ("" for none):
// git refuses the move when the ref holds anything else by then.
func moveLocal(ctx context.Context, root, commit, old string) error {
	if old == "" {
		old = strings.Repeat("0", len(commit))
	}
	_, err := git(ctx, root, "", "update-ref", "-m", "spotcheck", LocalRef, commit, old)
	return err
}

// git runs one local git command, isolated and bound to ctx, and answers its
// trimmed output.
func git(ctx context.Context, root, stdin string, args ...string) (string, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, stdin, 64<<10, args...)
	if err != nil {
		return "", fmt.Errorf("writing the spot-check's record: %s", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	return strings.TrimSpace(string(out)), nil
}

// originPresent is ErrNoOrigin for a checkout with no remote named origin.
func originPresent(ctx context.Context, root string) error {
	out, err := gitutil.RunCappedContext(ctx, root, 64<<10, "remote")
	if err != nil {
		return fmt.Errorf("listing this checkout's remotes: %s", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	if !slices.Contains(strings.Fields(out), claim.Remote) {
		return ErrNoOrigin
	}
	return nil
}

// Holder is this account's holder id, the form a reservation names its
// holder by (claim.KeptHolderID: read, never minted), or "" when none can be
// read: a record then names no writer.
func Holder() string {
	if id := claim.KeptHolderID(); holderRe.MatchString(id) {
		return id
	}
	return ""
}

// Kept is the name of every record file the local copy or origin's copy as
// last fetched holds, as folder/name: what a writer that rebuilds its records
// from its own state checks first, so it writes only what neither holds. A
// store that cannot be read is an error.
func Kept(ctx context.Context, root string) (map[string]bool, error) {
	r, err := readRefs(ctx, root)
	if err != nil {
		return nil, err
	}
	kept := map[string]bool{}
	for _, c := range []string{r.local, r.tracking} {
		files, err := allFiles(ctx, root, c)
		if err != nil {
			return nil, err
		}
		for folder, entries := range files {
			for name := range entries {
				kept[folder+"/"+name] = true
			}
		}
	}
	return kept, nil
}

// Records is what the store holds at one ref, read strictly: every file
// whose name and content are one this package writes, and Unreadable naming
// each file that is not. A ref that cannot be read is an error.
type Records struct {
	HandBacks  []HandBack
	Decisions  []Decision
	Unreadable []string
}

// Read reads the store at ref (LocalRef or TrackingRef). A ref the checkout
// does not hold reads as no records, with ok false.
func Read(ctx context.Context, root, ref string) (rec Records, ok bool, err error) {
	r, err := readRefs(ctx, root)
	if err != nil {
		return Records{}, false, err
	}
	commit := r.local
	if ref == TrackingRef {
		commit = r.tracking
	} else if ref != LocalRef {
		return Records{}, false, fmt.Errorf("%s is not one of the store's refs", termsafe.Sanitize(echo(ref)))
	}
	if commit == "" {
		return Records{}, false, nil
	}
	for _, folder := range []string{HandBacksDir, DecisionsDir} {
		_, entries, err := folderEntries(ctx, root, commit, folder)
		if err != nil {
			return Records{}, false, err
		}
		for _, e := range entries {
			path := folder + "/" + e.name
			data, err := readBlob(ctx, root, e)
			if err != nil {
				return Records{}, false, err
			}
			if data == nil || !readRecord(&rec, folder, e.name, data) {
				rec.Unreadable = append(rec.Unreadable, termsafe.Sanitize(path))
			}
		}
	}
	return rec, true, nil
}

// readBlob is e's content, nil when e is not a blob within maxRecordBytes.
// Its size is asked first, so one past the bound is never read.
func readBlob(ctx context.Context, root string, e treeEntry) ([]byte, error) {
	if e.typ != "blob" || e.mode != "100644" {
		return nil, nil
	}
	size, err := git(ctx, root, "", "cat-file", "-s", e.sha)
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(size)
	if err != nil || n < 0 || n > maxRecordBytes {
		return nil, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxRecordBytes, "cat-file", "blob", e.sha)
	if err != nil {
		return nil, fmt.Errorf("reading the spot-check's record: %s", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	return out, nil
}

// readRecord decodes one file into rec, strictly: false when it does not
// parse, is of another version, reads back other than as written (its
// fields as normalise leaves them) or is kept under a name its content does
// not give.
func readRecord(rec *Records, folder, name string, data []byte) bool {
	switch folder {
	case HandBacksDir:
		var h HandBack
		if !handBackNameRe.MatchString(name) || jsonstrict.Decode(data, &h) != nil || h.SchemaVersion != SchemaVersion {
			return false
		}
		if n, err := h.normalise(); err != nil || !sameAsWritten(n, data) || h.Name() != name {
			return false
		}
		rec.HandBacks = append(rec.HandBacks, h)
	case DecisionsDir:
		var d Decision
		if !decisionNameRe.MatchString(name) || jsonstrict.Decode(data, &d) != nil || d.SchemaVersion != SchemaVersion {
			return false
		}
		if n, err := d.normalise(); err != nil || !sameAsWritten(n, data) || d.Name() != name {
			return false
		}
		rec.Decisions = append(rec.Decisions, d)
	default:
		return false
	}
	return true
}

// sameAsWritten reports whether data is exactly the file this package writes
// for v: a record whose fields read back other than as written was not
// written by abcd.
func sameAsWritten(v any, data []byte) bool {
	want, err := encode(v)
	return err == nil && bytes.Equal(want, data)
}

// PushResult is what one Push carried to origin.
type PushResult struct {
	// Commit is origin's copy after the push, "" when nothing was pushed.
	Commit string
	// Pushed counts the records the push carried that origin lacked.
	Pushed int
	// Dropped names each record of this copy's whose name origin already
	// held with other content: origin's is kept, as the earlier one.
	Dropped []string
}

// beforePush, when a test sets it, runs before each push attempt: the seam a
// test races another writer through.
var beforePush func(attempt int)

// Push carries the local copy's records to origin: a fetch of the branch,
// the records origin lacks laid over its tip when another writer moved it,
// and one plain push (claim.PushBranch), never forced. A push origin rejects
// (a writer that pushed between the fetch and the push) is tried once more
// after a fresh fetch; a second rejection is an error, and every record stays
// on the local copy for the next push. Nothing origin holds is removed or
// rewritten. A checkout whose local copy holds nothing origin lacks pushes
// nothing.
func Push(ctx context.Context, root string) (PushResult, error) {
	if err := originPresent(ctx, root); err != nil {
		return PushResult{}, err
	}
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return PushResult{}, fmt.Errorf("the spot-check's record was not pushed: %w", err)
		}
		r, err := readRefs(ctx, root)
		if err != nil {
			return PushResult{}, err
		}
		if r.local == "" {
			return PushResult{}, nil
		}
		if _, err := claim.FetchBranch(root, BranchRef, TrackingRef); err != nil {
			return PushResult{}, fmt.Errorf("the spot-check's record was not pushed: %w", err)
		}
		if r, err = readRefs(ctx, root); err != nil {
			return PushResult{}, err
		}
		res, err := layOver(ctx, root, r)
		if err != nil {
			return PushResult{}, err
		}
		if res.Commit == "" {
			return res, nil
		}
		if beforePush != nil {
			beforePush(attempt)
		}
		if err := claim.PushBranch(root, res.Commit, BranchRef); err != nil {
			last = err
			continue
		}
		// The tracking copy follows what origin took, so the next write lays
		// on it; a move refused here is put right by the next fetch.
		_, _ = git(ctx, root, "", "update-ref", TrackingRef, res.Commit)
		return res, nil
	}
	return PushResult{}, fmt.Errorf("%s refused the spot-check's record twice, so every record stays on this copy for the next push: %w", claim.Remote, last)
}

// layOver is the commit to push for refs as just fetched: the local copy
// when it already holds origin's, a commit over origin's copy holding every
// record the local copy has and it lacks when the two have parted, and ""
// when origin holds every record already. The local copy is moved to that
// commit, so the records stay on it whatever the push does.
func layOver(ctx context.Context, root string, r refs) (PushResult, error) {
	mine, err := allFiles(ctx, root, r.local)
	if err != nil {
		return PushResult{}, err
	}
	theirs, err := allFiles(ctx, root, r.tracking)
	if err != nil {
		return PushResult{}, err
	}
	var res PushResult
	var adds = map[string][]treeEntry{}
	for folder, entries := range mine {
		for name, e := range entries {
			switch t, held := theirs[folder][name]; {
			case !held:
				res.Pushed++
				adds[folder] = append(adds[folder], e)
			case t.sha != e.sha:
				res.Dropped = append(res.Dropped, termsafe.Sanitize(folder+"/"+name))
			}
		}
	}
	slices.Sort(res.Dropped)
	if r.tracking == "" {
		if res.Pushed > 0 {
			res.Commit = r.local
		}
		return res, nil
	}
	if res.Pushed == 0 {
		// Origin holds everything: the local copy follows it.
		if r.local != r.tracking {
			if err := moveLocal(ctx, root, r.tracking, r.local); err != nil {
				return PushResult{}, err
			}
		}
		return res, nil
	}
	ahead, err := isAncestor(ctx, root, r.tracking, r.local)
	if err != nil {
		return PushResult{}, err
	}
	if ahead {
		res.Commit = r.local
		return res, nil
	}
	top, err := listTree(ctx, root, r.tracking+"^{tree}")
	if err != nil {
		return PushResult{}, err
	}
	folders := make([]string, 0, len(adds))
	for f := range adds {
		folders = append(folders, f)
	}
	slices.Sort(folders)
	for _, folder := range folders {
		_, entries, err := folderEntries(ctx, root, r.tracking, folder)
		if err != nil {
			return PushResult{}, err
		}
		tree, err := rootWith(ctx, root, top, folder, append(entries, adds[folder]...))
		if err != nil {
			return PushResult{}, err
		}
		if top, err = listTree(ctx, root, tree); err != nil {
			return PushResult{}, err
		}
	}
	tree, err := mktree(ctx, root, top)
	if err != nil {
		return PushResult{}, err
	}
	commit, err := commitTree(ctx, root, tree, r.tracking, fmt.Sprintf("spotcheck: %d record(s) laid over %s's copy", res.Pushed, claim.Remote))
	if err != nil {
		return PushResult{}, err
	}
	if err := moveLocal(ctx, root, commit, r.local); err != nil {
		return PushResult{}, err
	}
	res.Commit = commit
	return res, nil
}

// allFiles is every file of the store at commit, by folder and name; empty
// for no commit.
func allFiles(ctx context.Context, root, commit string) (map[string]map[string]treeEntry, error) {
	files := map[string]map[string]treeEntry{}
	if commit == "" {
		return files, nil
	}
	top, err := listTree(ctx, root, commit+"^{tree}")
	if err != nil {
		return nil, err
	}
	for _, f := range top {
		if f.typ != "tree" {
			continue
		}
		entries, err := listTree(ctx, root, f.sha)
		if err != nil {
			return nil, err
		}
		files[f.name] = map[string]treeEntry{}
		for _, e := range entries {
			files[f.name][e.name] = e
		}
	}
	return files, nil
}
