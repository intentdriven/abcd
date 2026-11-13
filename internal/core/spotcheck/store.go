// Package spotcheck keeps the spot-check's record (itd-2610301750162657,
// spc-2610301925429250): the work abcd decided with no person involved,
// recorded as it happens, for a later draw to read.
//
// THE STORE. The record lives on one branch of the shared remote,
// BranchRef on origin, tracked at TrackingRef. It is an orphan whose tree
// holds folders of small JSON files, each written once and never edited:
// handbacks/ holds the drain's hand-backs and decisions/ the automated
// facilitator's decisions (the drain's judgements a model answered and the
// merge gate's in auto mode). Each file's name ends in a short hash of its
// content, so a name stands for one content: two records that differ in
// anything are two files, and two writers never write one file two ways.
//
// A write is committed on this checkout's local copy, LocalRef, at once,
// touching no HEAD, index or working tree, and reaches no network. LocalRef
// lives outside refs/heads/ and refs/remotes/, so no fetch moves it and every
// worktree of the checkout shares it: a lane's worktree removed after its
// landing takes none of its records with it. A record counts as kept only
// when the local copy holds a file of its name with its content; nothing read
// from origin's copy makes a record count as kept. One write commits all the
// records it is handed together, as one commit.
//
// push, which no verb calls yet, carries the local copy to origin: one fetch
// of the branch, the records origin lacks laid over its tip, and one plain
// push, never forced, so nothing another writer pushed is dropped; a push
// origin rejects is tried once more after a fresh fetch, and one that fails
// again leaves every record on the local copy for the next push.
//
// Everything read from the store was written by another session, account or
// machine, so it is read strictly: names are checked against the forms this
// package writes and against the hash of their content, each file is bounded
// and strictly decoded, the number of files a read takes is bounded, and a
// store that cannot be read is an error, never an empty store.
//
// FREE TEXT. A record's free text is cleaned by one rule that depends on
// nothing of the machine or account writing or reading it (clean): every
// "/" has an ASCII letter or digit right before it, and the text holds no "\"
// and no "~". A run of non-space characters that would break the rule is
// replaced whole by one fixed placeholder, so a repository-relative path such
// as internal/core/x.go stays, and an absolute path, a home-relative path,
// an address such as file:// or https://, and a path in quotes, brackets or
// backticks go. Control and invisible characters separate words and are
// dropped, and each text is bounded. The read side checks a stored text with
// the same function, so two readers of one store agree on what it holds.
//
// WHO A COMMIT NAMES. Every git command that makes an object or moves a ref
// runs through one helper (storeGit) whose environment names abcd's fixed
// identity as author and committer, whatever the environment held, with the
// dates in UTC. The record names its writer only by its holder id, the form
// the reservations use.
package spotcheck

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The store's refs.
const (
	// Branch is the store's branch on origin.
	Branch = claim.SpotcheckBranch
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
	// WriteTimeout bounds one write, all of its git commands together. Each
	// is local, so a write that takes longer is a git that hangs, and the
	// write is given up rather than hold its caller.
	WriteTimeout = 10 * time.Second
	// stopGrace is how long a git the write's deadline cut off has, once
	// told to stop (SIGTERM, on which git removes its lock files), before it
	// and every process it started are killed.
	stopGrace = 2 * time.Second
	// maxTextBytes bounds each free-text field a record carries.
	maxTextBytes = 2000
	// maxRecordBytes bounds one record file, on write and on read.
	maxRecordBytes = 16 << 10
	// maxListingBytes bounds one listing of the store's files.
	maxListingBytes = 8 << 20
	// maxTries bounds the moves of the local copy one write makes when
	// another writer of the checkout moved it first.
	maxTries = 3
	// hashLen is how many hex digits of its content's hash a file's name
	// carries.
	hashLen = 16
	// stamp is the form a record's time takes in its file name.
	stamp = "20060102T150405Z"
	// placeholder stands in for a run of text the cleaning rule removes.
	placeholder = "<removed>"
	// ellipsis ends a text cut at its bound.
	ellipsis = "…"
)

// maxReadFiles bounds the records one read takes: a store holding more is
// an error naming the count, never a read of some of them. A test lowers it.
var maxReadFiles = 4096

// The identity a store commit is written as: abcd's, set in the environment
// of every git command that makes an object (storeEnv). The record names its
// writer by holder id.
const (
	commitName  = "abcd spotcheck"
	commitEmail = "spotcheck@abcd.invalid"
)

// ErrNoOrigin is a checkout with no remote named origin: there is no shared
// remote to keep the record on, and nothing is written.
var ErrNoOrigin = errors.New("this checkout has no remote named " + claim.Remote + ", so the spot-check keeps no record here")

// errMoved is a local copy another writer moved between this write's read
// of it and its move.
var errMoved = errors.New("the local copy of the spot-check's record moved while this write was made")

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
	// head, and the gate's answer as the gate gave it: whether it armed, the
	// method, its reason, and whether it rests on an answer the gate could
	// not read.
	Site   string `json:"site,omitempty"`
	PR     int    `json:"pr,omitempty"`
	Head   string `json:"head,omitempty"`
	Arm    bool   `json:"arm,omitempty"`
	Method string `json:"method,omitempty"`
	Why    string `json:"why,omitempty"`
	Unread bool   `json:"unread,omitempty"`
}

var (
	recordIDRe = regexp.MustCompile(`^(iss|itd)-[0-9]{1,20}$`)
	runIDRe    = regexp.MustCompile(`^run-[0-9]{1,20}$`)
	holderRe   = regexp.MustCompile(`^h_[0-9a-f]{32}$`)
	shaRe      = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	wordRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	// The names a folder holds: a time, what the record is of, and the
	// hash of its content.
	handBackNameRe = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-(iss|itd)-[0-9]{1,20}-[0-9a-f]{16}\.json$`)
	decisionNameRe = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-((iss|itd)-[0-9]{1,20}|pr[0-9]{1,10}-[0-9a-f]{12}-(armed|open))-[0-9a-f]{16}\.json$`)
)

// clean is free text as a record keeps it, by the one rule the package doc
// states: words are what lies between spaces and control or invisible
// characters, which are dropped, and the words are joined by single spaces;
// a word that holds "\" or "~", or a "/" with anything but an ASCII letter
// or digit right before it, is replaced whole by the placeholder; and the
// text is cut to maxTextBytes, the cut marked. It reads nothing of the
// machine, so a text it leaves unchanged is one it accepts on read
// (cleanText).
func clean(s string) string {
	words := strings.FieldsFunc(s, separates)
	for i, w := range words {
		if !pathFree(w) {
			words[i] = placeholder
		}
	}
	s = strings.Join(words, " ")
	if len(s) > maxTextBytes {
		cut := maxTextBytes - len(ellipsis)
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + ellipsis
	}
	return s
}

// separates reports whether r separates words of free text: a space, a
// control or invisible character (any rune termsafe.Sanitize masks), or a
// byte that is not UTF-8.
func separates(r rune) bool {
	return unicode.IsSpace(r) || r == utf8.RuneError || termsafe.Sanitize(string(r)) != string(r)
}

// pathFree reports whether a word keeps to the rule: no "\", no "~", and an
// ASCII letter or digit right before every "/".
func pathFree(w string) bool {
	if strings.ContainsAny(w, `\~`) {
		return false
	}
	for i := 0; i < len(w); i++ {
		if w[i] == '/' && (i == 0 || !asciiAlnum(w[i-1])) {
			return false
		}
	}
	return true
}

func asciiAlnum(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// cleanWord is a field the drain or the gate names from a fixed set (a kind,
// a route, an answer): kept when it reads as one, dropped otherwise.
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
		return HandBack{}, fmt.Errorf("the hand-back of %s carries no time", h.Issue)
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
		d.Site, d.PR, d.Head, d.Arm, d.Method, d.Why, d.Unread = "", 0, "", false, "", "", false
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
		d.Method, d.Why = clean(d.Method), clean(d.Why)
		d.Issue, d.Question, d.Answer, d.AnswerKind, d.Model, d.Rule, d.Reason = "", "", "", "", "", "", ""
	default:
		return Decision{}, fmt.Errorf("a decision of kind %s is not one the store keeps", termsafe.Sanitize(echo(d.Kind)))
	}
	return d, nil
}

// echo is s cut for a refusal to name.
func echo(s string) string {
	if len(s) > 64 {
		return strconv.Quote(s[:64]) + "…"
	}
	return strconv.Quote(s)
}

// gateKey is what names one merge-gate decision apart from its time and
// content: the pull request, its head and whether the gate armed.
func (d Decision) gateKey() string {
	answer := "open"
	if d.Arm {
		answer = "armed"
	}
	return "pr" + strconv.Itoa(d.PR) + "-" + d.Head[:12] + "-" + answer
}

// file is one record as the store keeps it: its folder, its name and its
// content. gateKey is set for a merge-gate decision.
type file struct {
	folder, name string
	data         []byte
	gateKey      string
	head         string
}

func (f file) path() string { return f.folder + "/" + f.name }

// named is the file name for a record of stem holding data: the stem, the
// first hashLen hex digits of the SHA-256 of data, and .json.
func named(stem string, data []byte) string {
	sum := sha256.Sum256(data)
	return stem + "-" + hex.EncodeToString(sum[:])[:hashLen] + ".json"
}

// file is h as the store keeps it.
func (h HandBack) file() (file, error) {
	n, err := h.normalise()
	if err != nil {
		return file{}, err
	}
	data, err := encode(n)
	if err != nil {
		return file{}, err
	}
	return file{folder: HandBacksDir, name: named(n.At.Format(stamp)+"-"+n.Issue, data), data: data}, nil
}

// file is d as the store keeps it.
func (d Decision) file() (file, error) {
	n, err := d.normalise()
	if err != nil {
		return file{}, err
	}
	data, err := encode(n)
	if err != nil {
		return file{}, err
	}
	if n.Kind == KindMergeGate {
		return file{folder: DecisionsDir, name: named(n.At.Format(stamp)+"-"+n.gateKey(), data), data: data, gateKey: n.gateKey(), head: n.Head}, nil
	}
	return file{folder: DecisionsDir, name: named(n.At.Format(stamp)+"-"+n.Issue, data), data: data}, nil
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

// Batch is the records one write keeps, all in one commit.
type Batch struct {
	HandBacks []HandBack
	Decisions []Decision
}

// Written is what one write did with its batch.
type Written struct {
	// Wrote counts the records the write committed, and Kept those the local
	// copy already held.
	Wrote, Kept int
	// Failed counts the records the write tried to commit and could not, and
	// Err says why; Err is ErrNoOrigin, with nothing counted, for a checkout
	// with no remote named origin.
	Failed int
	Err    error
	// Unkeepable says, for each record the store cannot keep at all (no
	// time, an id outside the store's form), why.
	Unkeepable []string
}

// Write commits every record of b the local copy does not already hold, as
// one commit on the local copy, touching no HEAD, index or working tree and
// reaching no network. A record is already held when the local copy holds a
// file of its name with its content, and a merge-gate decision when the
// local copy holds one over the same pull request and head with the same
// answer, at any time, that origin's copy as last fetched does not also
// hold (gateHeld). A file of a record's name with other content is replaced
// by the record. The local copy is moved only from the value the write read
// (a compare-and-swap): another writer of the checkout that moved it first
// makes this one read it again, maxTries times in all. Every git command
// runs under parent and WriteTimeout.
func Write(parent context.Context, root string, b Batch) Written {
	var w Written
	if len(b.HandBacks)+len(b.Decisions) == 0 {
		return w
	}
	ctx, cancel := context.WithTimeout(parent, WriteTimeout)
	defer cancel()
	if err := originPresent(ctx, root); err != nil {
		w.Err = err
		if !errors.Is(err, ErrNoOrigin) {
			w.Failed = len(b.HandBacks) + len(b.Decisions)
		}
		return w
	}
	var files []file
	seen := map[string]bool{}
	add := func(f file, err error) {
		switch {
		case err != nil:
			w.Unkeepable = append(w.Unkeepable, err.Error())
		case !seen[f.path()]:
			seen[f.path()] = true
			files = append(files, f)
		}
	}
	for _, h := range b.HandBacks {
		add(h.file())
	}
	for _, d := range b.Decisions {
		add(d.file())
	}
	if len(files) == 0 {
		return w
	}
	for attempt := 1; ; attempt++ {
		wrote, kept, err := commitFiles(ctx, root, files, attempt)
		if err == nil {
			w.Wrote, w.Kept = wrote, kept
			return w
		}
		if errors.Is(err, errMoved) && attempt < maxTries {
			continue
		}
		w.Kept, w.Failed, w.Err = kept, len(files)-kept, err
		return w
	}
}

// beforeMove, when a test sets it, runs before each move of the local copy:
// the seam a test races another writer of the checkout through.
var beforeMove func(attempt int)

// commitFiles is one try of Write: the local copy read, the files it lacks
// committed over it, and the copy moved from the value read. kept is known
// once the copy is read.
func commitFiles(ctx context.Context, root string, files []file, attempt int) (wrote, kept int, err error) {
	local, err := storeRef(ctx, root, LocalRef)
	if err != nil {
		return 0, 0, err
	}
	held, err := listFiles(ctx, root, local)
	if err != nil {
		return 0, 0, err
	}
	var origin map[string]entry
	if slices.ContainsFunc(files, func(f file) bool { return f.gateKey != "" }) {
		// origin's copy is read only to keep its files from counting as this
		// copy's: a copy that cannot be read counts nothing as held.
		if tracking, terr := storeRef(ctx, root, TrackingRef); terr == nil {
			origin, _ = listFiles(ctx, root, tracking)
		}
	}
	var todo []file
	for _, f := range files {
		if e, ok := held[f.path()]; ok && e.typ == "blob" && e.sha == blobID(f.data, len(e.sha)) {
			kept++
			continue
		}
		if f.gateKey != "" {
			ok, err := gateHeld(ctx, root, held, origin, f)
			if err != nil {
				return 0, kept, err
			}
			if ok {
				kept++
				continue
			}
		}
		todo = append(todo, f)
	}
	if len(todo) == 0 {
		return 0, kept, nil
	}
	tree, err := treeWith(ctx, root, local, todo)
	if err != nil {
		return 0, kept, err
	}
	commit, err := commitTree(ctx, root, tree, local, fmt.Sprintf("spotcheck: %d record(s)", len(todo)))
	if err != nil {
		return 0, kept, err
	}
	if beforeMove != nil {
		beforeMove(attempt)
	}
	if err := moveRef(ctx, root, LocalRef, commit, local); err != nil {
		return 0, kept, err
	}
	return len(todo), kept, nil
}

// gateHeld reports whether the local copy already holds a merge-gate
// decision over f's pull request and head with f's answer: a file of that
// key the local copy holds and origin's copy as last fetched does not
// (origin nil counts every file as origin's), whose content reads as a
// record this package writes for the same head.
func gateHeld(ctx context.Context, root string, held, origin map[string]entry, f file) (bool, error) {
	if origin == nil {
		return false, nil
	}
	var cands []entry
	var names []string
	for p, e := range held {
		dir, name, _ := strings.Cut(p, "/")
		if dir != DecisionsDir || len(name) < len(stamp)+1 || !strings.HasPrefix(name[len(stamp)+1:], f.gateKey+"-") {
			continue
		}
		if _, theirs := origin[p]; theirs || e.typ != "blob" || e.size > maxRecordBytes {
			continue
		}
		cands, names = append(cands, e), append(names, name)
	}
	if len(cands) == 0 {
		return false, nil
	}
	blobs, err := readBlobs(ctx, root, cands)
	if err != nil {
		return false, err
	}
	for i, data := range blobs {
		var rec Records
		if readRecord(&rec, DecisionsDir, names[i], data) && len(rec.Decisions) == 1 && rec.Decisions[0].Head == f.head {
			return true, nil
		}
	}
	return false, nil
}

// blobID is the object name git gives a blob holding data, in the object
// format whose names are n hex digits long.
func blobID(data []byte, n int) string {
	header := "blob " + strconv.Itoa(len(data)) + "\x00"
	if n == 64 {
		sum := sha256.Sum256(append([]byte(header), data...))
		return hex.EncodeToString(sum[:])
	}
	sum := sha1.Sum(append([]byte(header), data...))
	return hex.EncodeToString(sum[:])
}

// entry is one file of the store as a listing names it.
type entry struct {
	mode, typ, sha string
	size           int
}

// storeRef is the commit ref names, "" when the checkout holds no such ref.
// A ref that names anything but a commit is an unreadable store, never an
// absent one.
func storeRef(ctx context.Context, root, ref string) (string, error) {
	out, err := gitutil.RunCappedContext(ctx, root, 64<<10, "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)", ref)
	if err != nil {
		return "", fmt.Errorf("reading the spot-check's record: %s", redact(err))
	}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) != 3 || f[0] != ref {
			continue
		}
		if f[2] != "commit" || !gitutil.IsFullSHA(f[1]) {
			return "", fmt.Errorf("%s names a %s, not a commit, so the spot-check's record cannot be read", ref, termsafe.Sanitize(f[2]))
		}
		return f[1], nil
	}
	return "", nil
}

// listFiles is every file of the store at commit, at any depth, by path;
// empty for no commit. A line that does not read as a file, and a folder of
// the store's that is not a folder, are an unreadable store.
func listFiles(ctx context.Context, root, commit string) (map[string]entry, error) {
	files := map[string]entry{}
	if commit == "" {
		return files, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxListingBytes, "ls-tree", "-r", "-l", "-z", "--full-tree", commit)
	if err != nil {
		return nil, fmt.Errorf("reading the spot-check's record: %s", redact(err))
	}
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, path, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 4 || !gitutil.IsFullSHA(f[2]) || !storePath(path) {
			return nil, errors.New("the spot-check's record holds a file whose listing does not read as one")
		}
		size := -1
		if f[1] == "blob" {
			if size, err = strconv.Atoi(f[3]); err != nil || size < 0 {
				return nil, errors.New("the spot-check's record holds a file whose size does not read as one")
			}
		}
		if path == HandBacksDir || path == DecisionsDir {
			return nil, fmt.Errorf("the spot-check's record holds %s as a %s, not a folder", path, termsafe.Sanitize(f[1]))
		}
		files[path] = entry{mode: f[0], typ: f[1], sha: f[2], size: size}
	}
	return files, nil
}

// storePath reports whether p reads as a path of the store's tree: segments
// of printable characters, none empty, "." or "..".
func storePath(p string) bool {
	if p == "" || strings.ContainsAny(p, "\n\r\\") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// readBlobs is the content of each entry, in order, read through one `git
// cat-file --batch`. Each entry is a blob within maxRecordBytes, as its
// listing gave its size, so the whole answer is bounded before it is asked.
func readBlobs(ctx context.Context, root string, es []entry) ([][]byte, error) {
	var in strings.Builder
	limit := 0
	for _, e := range es {
		in.WriteString(e.sha + "\n")
		limit += e.size + len(e.sha) + 32
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, in.String(), limit, "cat-file", "--batch")
	if err != nil {
		return nil, fmt.Errorf("reading the spot-check's record: %s", redact(err))
	}
	res := make([][]byte, 0, len(es))
	for _, e := range es {
		nl := bytes.IndexByte(out, '\n')
		if nl < 0 {
			return nil, errors.New("reading the spot-check's record: git answered no header for a file")
		}
		header := strings.Fields(string(out[:nl]))
		out = out[nl+1:]
		if len(header) != 3 || header[0] != e.sha || header[1] != "blob" || header[2] != strconv.Itoa(e.size) || len(out) < e.size+1 || out[e.size] != '\n' {
			return nil, errors.New("reading the spot-check's record: git's answer for a file is not the blob its listing named")
		}
		res = append(res, out[:e.size:e.size])
		out = out[e.size+1:]
	}
	return res, nil
}

// treeWith is the root tree of base ("" for none) with each of files at its
// path, made in an index of its own outside the repository, so the
// checkout's index and working tree are never touched.
func treeWith(ctx context.Context, root, base string, files []file) (string, error) {
	tmp, err := os.MkdirTemp("", "abcd-spotcheck-")
	if err != nil {
		return "", fmt.Errorf("writing the spot-check's record: %s", redact(err))
	}
	defer os.RemoveAll(tmp)
	var paths strings.Builder
	for i, f := range files {
		p := filepath.Join(tmp, strconv.Itoa(i)+".json")
		if err := os.WriteFile(p, f.data, 0o600); err != nil {
			return "", fmt.Errorf("writing the spot-check's record: %s", redact(err))
		}
		paths.WriteString(p + "\n")
	}
	out, err := storeGit(ctx, root, "", paths.String(), "hash-object", "-w", "--no-filters", "--stdin-paths")
	if err != nil {
		return "", err
	}
	shas := strings.Fields(out)
	if len(shas) != len(files) {
		return "", errors.New("writing the spot-check's record: git named another number of blobs than it was given")
	}
	index := filepath.Join(tmp, "index")
	if base == "" {
		_, err = storeGit(ctx, root, index, "", "read-tree", "--empty")
	} else {
		_, err = storeGit(ctx, root, index, "", "read-tree", base+"^{tree}")
	}
	if err != nil {
		return "", err
	}
	var info strings.Builder
	for i, f := range files {
		if !gitutil.IsFullSHA(shas[i]) {
			return "", errors.New("writing the spot-check's record: git named a blob that is not an object name")
		}
		info.WriteString("100644 " + shas[i] + "\t" + f.path() + "\n")
	}
	if _, err := storeGit(ctx, root, index, info.String(), "update-index", "--add", "--index-info"); err != nil {
		return "", err
	}
	return storeGit(ctx, root, index, "", "write-tree")
}

// commitTree writes a commit of tree over parent ("" for the first).
func commitTree(ctx context.Context, root, tree, parent, msg string) (string, error) {
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	return storeGit(ctx, root, "", msg+"\n", append(args, "-F", "-")...)
}

// moveRef moves ref to commit from old, the value read ("" for none): git
// refuses the move when the ref holds anything else by then, which is
// errMoved.
func moveRef(ctx context.Context, root, ref, commit, old string) error {
	from := old
	if from == "" {
		from = strings.Repeat("0", len(commit))
	}
	_, err := storeGit(ctx, root, "", "", "update-ref", "-m", "spotcheck", ref, commit, from)
	if err == nil {
		return nil
	}
	if now, rerr := storeRef(ctx, root, ref); rerr == nil && now != old {
		return fmt.Errorf("%w: %s", errMoved, err)
	}
	return err
}

// storeGit runs one git command of the store's that makes an object or moves
// a ref, bound to ctx, and answers its trimmed output. It is the one way
// this package writes: abcd's identity and UTC dates in the environment
// (storeEnv), index the index the command uses ("" for the checkout's own),
// and the program-starting settings pinned off (gitutil.ExecPins). When ctx
// ends, git and every process it started are told to stop with SIGTERM, on
// which git removes its lock files, and are killed stopGrace later.
func storeGit(ctx context.Context, root, index, stdin string, args ...string) (string, error) {
	what := "git " + args[0]
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("writing the spot-check's record: %s: %w", what, err)
	}
	full := append(append(gitutil.ExecPins(), "-c", "core.quotePath=false", "-c", "core.sparseCheckout=false", "-C", root), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = storeEnv(time.Now(), index)
	procgroup.Lead(cmd)
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = stopGrace
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	stdout := &boundedBuffer{max: 1 << 20}
	stderr := &boundedBuffer{max: 4096}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		procgroup.KillLeft(cmd)
		return "", fmt.Errorf("writing the spot-check's record: %s: %w", what, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("writing the spot-check's record: %s: %v (%s)", what, err, termsafe.Sanitize(fsutil.RedactHome(strings.TrimSpace(stderr.b.String()))))
	}
	if stdout.over {
		return "", fmt.Errorf("writing the spot-check's record: %s wrote more than it may", what)
	}
	return strings.TrimSpace(stdout.b.String()), nil
}

// storeEnv is the environment a store command runs with: the isolated one
// (gitutil.IsolatedEnv), less any identity, date or index the parent's
// environment held, with abcd's identity as author and committer, both
// dates now in UTC, and index as GIT_INDEX_FILE when one is given.
func storeEnv(now time.Time, index string) []string {
	base := gitutil.IsolatedEnv()
	env := make([]string, 0, len(base)+8)
	for _, kv := range base {
		switch k, _, _ := strings.Cut(kv, "="); k {
		case "GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_AUTHOR_DATE",
			"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "GIT_COMMITTER_DATE",
			"GIT_INDEX_FILE", "EMAIL":
			continue
		}
		env = append(env, kv)
	}
	date := strconv.FormatInt(now.Unix(), 10) + " +0000"
	env = append(env,
		"GIT_AUTHOR_NAME="+commitName, "GIT_AUTHOR_EMAIL="+commitEmail, "GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_NAME="+commitName, "GIT_COMMITTER_EMAIL="+commitEmail, "GIT_COMMITTER_DATE="+date)
	if index != "" {
		env = append(env, "GIT_INDEX_FILE="+index)
	}
	return env
}

// boundedBuffer keeps at most max bytes and marks what it dropped.
type boundedBuffer struct {
	b    bytes.Buffer
	max  int
	over bool
}

func (w *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if room := w.max - w.b.Len(); n > room {
		n, w.over = max(room, 0), true
	}
	w.b.Write(p[:n])
	return len(p), nil
}

// redact is err's text as a note shows it: the home directory replaced and
// control characters masked.
func redact(err error) string {
	return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
}

// originPresent is ErrNoOrigin for a checkout with no remote named origin.
func originPresent(ctx context.Context, root string) error {
	out, err := gitutil.RunCappedContext(ctx, root, 64<<10, "remote")
	if err != nil {
		return fmt.Errorf("listing this checkout's remotes: %s", redact(err))
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

// Records is what the store holds at one ref, read strictly: every file
// whose name and content are one this package writes, and Unreadable naming
// each file that is not. A ref that cannot be read is an error.
type Records struct {
	HandBacks  []HandBack
	Decisions  []Decision
	Unreadable []string
}

// read reads the store at ref (LocalRef or TrackingRef), all its files'
// content through one batch. A ref the checkout does not hold reads as no
// records, with ok false. A store holding more than maxReadFiles records is
// an error naming the count, and nothing of it is read.
func read(ctx context.Context, root, ref string) (rec Records, ok bool, err error) {
	if ref != LocalRef && ref != TrackingRef {
		return Records{}, false, fmt.Errorf("%s is not one of the store's refs", termsafe.Sanitize(echo(ref)))
	}
	commit, err := storeRef(ctx, root, ref)
	if err != nil || commit == "" {
		return Records{}, false, err
	}
	files, err := listFiles(ctx, root, commit)
	if err != nil {
		return Records{}, false, err
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	var take []entry
	var names []string
	for _, p := range paths {
		e := files[p]
		dir, name, _ := strings.Cut(p, "/")
		if dir != HandBacksDir && dir != DecisionsDir {
			continue
		}
		if strings.Contains(name, "/") || e.typ != "blob" || e.mode != "100644" || e.size > maxRecordBytes {
			rec.Unreadable = append(rec.Unreadable, termsafe.Sanitize(p))
			continue
		}
		take, names = append(take, e), append(names, p)
	}
	if len(take) > maxReadFiles {
		return Records{}, false, fmt.Errorf("the spot-check's record at %s holds %d records, over the %d one read takes, so none was read", ref, len(take), maxReadFiles)
	}
	blobs, err := readBlobs(ctx, root, take)
	if err != nil {
		return Records{}, false, err
	}
	for i, data := range blobs {
		dir, name, _ := strings.Cut(names[i], "/")
		if !readRecord(&rec, dir, name, data) {
			rec.Unreadable = append(rec.Unreadable, termsafe.Sanitize(names[i]))
		}
	}
	return rec, true, nil
}

// readRecord decodes one file into rec, strictly: false when it does not
// parse, is of another version, reads back other than as written (its
// fields as normalise leaves them, every text as clean leaves it) or is kept
// under a name its content does not give, its hash included.
func readRecord(rec *Records, folder, name string, data []byte) bool {
	switch folder {
	case HandBacksDir:
		var h HandBack
		if !handBackNameRe.MatchString(name) || jsonstrict.Decode(data, &h) != nil || h.SchemaVersion != SchemaVersion {
			return false
		}
		if f, err := h.file(); err != nil || f.name != name || !bytes.Equal(f.data, data) {
			return false
		}
		rec.HandBacks = append(rec.HandBacks, h)
	case DecisionsDir:
		var d Decision
		if !decisionNameRe.MatchString(name) || jsonstrict.Decode(data, &d) != nil || d.SchemaVersion != SchemaVersion {
			return false
		}
		if f, err := d.file(); err != nil || f.name != name || !bytes.Equal(f.data, data) {
			return false
		}
		rec.Decisions = append(rec.Decisions, d)
	default:
		return false
	}
	return true
}

// nameMatches reports whether a file named name holds data by its name: the
// hash its name ends in is data's.
func nameMatches(name string, data []byte) bool {
	stem, ok := strings.CutSuffix(name, ".json")
	if !ok || len(stem) < hashLen+1 || stem[len(stem)-hashLen-1] != '-' {
		return false
	}
	return named(stem[:len(stem)-hashLen-1], data) == name
}

// PushResult is what one push carried to origin.
type PushResult struct {
	// Commit is origin's copy after the push, "" when nothing was pushed.
	Commit string
	// Pushed counts the records the push carried that origin lacked.
	Pushed int
	// Unreadable names each file origin or the local copy held under a
	// record's name whose content its name does not give: the copy whose
	// content it does give is kept, and origin's when neither does.
	Unreadable []string
}

// beforePush, when a test sets it, runs before each push attempt: the seam a
// test races another writer through.
var beforePush func(attempt int)

// push carries the local copy's records to origin: a fetch of the branch,
// the records origin lacks laid over its tip when another writer moved it,
// and one plain push (claim.PushBranch), never forced. A push origin rejects
// (a writer that pushed between the fetch and the push), or a local copy
// another writer of the checkout moved during the lay-over, is tried once
// more after a fresh fetch; a second failure is an error, and every record
// stays on the local copy for the next push. A branch origin no longer holds
// is no copy at all: its tracking ref is removed and the local copy pushed
// whole. Nothing origin holds is removed, and only a file its name does not
// give is replaced. A checkout whose local copy holds nothing origin lacks
// pushes nothing.
func push(ctx context.Context, root string) (PushResult, error) {
	if err := originPresent(ctx, root); err != nil {
		return PushResult{}, err
	}
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return PushResult{}, fmt.Errorf("the spot-check's record was not pushed: %w", err)
		}
		local, err := storeRef(ctx, root, LocalRef)
		if err != nil {
			return PushResult{}, err
		}
		if local == "" {
			return PushResult{}, nil
		}
		found, err := claim.FetchBranch(ctx, root, BranchRef, TrackingRef)
		if err != nil {
			return PushResult{}, fmt.Errorf("the spot-check's record was not pushed: %w", err)
		}
		tracking := ""
		if found {
			if tracking, err = storeRef(ctx, root, TrackingRef); err != nil {
				return PushResult{}, err
			}
		} else if old, err := storeRef(ctx, root, TrackingRef); err == nil && old != "" {
			if _, err := storeGit(ctx, root, "", "", "update-ref", "-d", TrackingRef, old); err != nil {
				return PushResult{}, err
			}
		}
		res, err := layOver(ctx, root, local, tracking)
		if errors.Is(err, errMoved) {
			last = err
			continue
		}
		if err != nil {
			return PushResult{}, err
		}
		if res.Commit == "" {
			return res, nil
		}
		if beforePush != nil {
			beforePush(attempt)
		}
		if err := claim.PushBranch(ctx, root, res.Commit, BranchRef); err != nil {
			last = err
			continue
		}
		// The tracking copy follows what origin took, so the next push lays
		// on it; a move refused here is put right by the next fetch.
		_, _ = storeGit(ctx, root, "", "", "update-ref", TrackingRef, res.Commit)
		return res, nil
	}
	return PushResult{}, fmt.Errorf("the spot-check's record was not pushed after two tries, so every record stays on this copy for the next push: %w", last)
}

// layOver is the commit to push for the local copy and origin's (tracking,
// "" for none) as just fetched: the local copy when it already holds
// origin's, a commit over origin's copy holding every file the local copy
// has and it lacks, at any depth, when the two have parted, and "" when
// origin holds every file already. A path both hold with other content is
// settled by its name (nameMatches): the copy its name gives is kept, and
// the other named in Unreadable. The local copy is moved to that commit,
// from the value read, so the records stay on it whatever the push does.
func layOver(ctx context.Context, root, local, tracking string) (PushResult, error) {
	mine, err := listFiles(ctx, root, local)
	if err != nil {
		return PushResult{}, err
	}
	theirs, err := listFiles(ctx, root, tracking)
	if err != nil {
		return PushResult{}, err
	}
	var res PushResult
	var adds []placed
	var clash []string
	for p, e := range mine {
		t, held := theirs[p]
		switch {
		case !held:
			res.Pushed++
			adds = append(adds, placed{path: p, entry: e})
		case t.sha != e.sha:
			clash = append(clash, p)
		}
	}
	slices.Sort(clash)
	for _, p := range clash {
		mineGood, theirsGood := false, false
		e, t := mine[p], theirs[p]
		if e.typ == "blob" && t.typ == "blob" && e.size <= maxRecordBytes && t.size <= maxRecordBytes {
			blobs, err := readBlobs(ctx, root, []entry{e, t})
			if err != nil {
				return PushResult{}, err
			}
			name := p[strings.LastIndexByte(p, '/')+1:]
			mineGood, theirsGood = nameMatches(name, blobs[0]), nameMatches(name, blobs[1])
		}
		res.Unreadable = append(res.Unreadable, termsafe.Sanitize(p))
		if mineGood && !theirsGood {
			res.Pushed++
			adds = append(adds, placed{path: p, entry: e})
		}
	}
	if tracking == "" {
		if res.Pushed > 0 {
			res.Commit = local
		}
		return res, nil
	}
	if res.Pushed == 0 {
		// Origin holds everything: the local copy follows it.
		if local != tracking {
			if err := moveRef(ctx, root, LocalRef, tracking, local); err != nil {
				return PushResult{}, err
			}
		}
		return res, nil
	}
	ahead, err := isAncestor(ctx, root, tracking, local)
	if err != nil {
		return PushResult{}, err
	}
	if ahead {
		res.Commit = local
		return res, nil
	}
	tree, err := treeOver(ctx, root, tracking, adds)
	if err != nil {
		return PushResult{}, err
	}
	commit, err := commitTree(ctx, root, tree, tracking, fmt.Sprintf("spotcheck: %d record(s) laid over %s's copy", res.Pushed, claim.Remote))
	if err != nil {
		return PushResult{}, err
	}
	if err := moveRef(ctx, root, LocalRef, commit, local); err != nil {
		return PushResult{}, err
	}
	res.Commit = commit
	return res, nil
}

// placed is one file of the local copy at its path, for a lay-over.
type placed struct {
	path string
	entry
}

// treeOver is the root tree of base with each of adds, already in the object
// store, at its path, made in an index of its own.
func treeOver(ctx context.Context, root, base string, adds []placed) (string, error) {
	tmp, err := os.MkdirTemp("", "abcd-spotcheck-")
	if err != nil {
		return "", fmt.Errorf("writing the spot-check's record: %s", redact(err))
	}
	defer os.RemoveAll(tmp)
	index := filepath.Join(tmp, "index")
	if _, err := storeGit(ctx, root, index, "", "read-tree", base+"^{tree}"); err != nil {
		return "", err
	}
	var info strings.Builder
	for _, a := range adds {
		info.WriteString(a.mode + " " + a.sha + "\t" + a.path + "\n")
	}
	if _, err := storeGit(ctx, root, index, info.String(), "update-index", "--add", "--index-info"); err != nil {
		return "", err
	}
	return storeGit(ctx, root, index, "", "write-tree")
}

// isAncestor reports whether a is an ancestor of b.
func isAncestor(ctx context.Context, root, a, b string) (bool, error) {
	_, one, err := gitutil.RunCappedExitOneContext(ctx, root, 64, "merge-base", "--is-ancestor", a, b)
	if err != nil {
		return false, fmt.Errorf("reading the spot-check's record: %w", err)
	}
	return !one, nil
}
