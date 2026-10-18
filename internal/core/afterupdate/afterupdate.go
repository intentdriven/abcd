// Package afterupdate is the moment after an abcd update
// (spc-2610170635097509, step 6): the one line per release saying what
// changed, from the digest inside the binary, and the offer to run setup
// again where the update brings something this repository needs; and the
// record of what was shown and answered.
//
// The record lives in abcd's user-level home, never in the repository:
//
//	~/.abcd.noindex/after-update/<root-sha>/<tag>.json  this repository's answer for one release
//	~/.abcd.noindex/after-update/news-seen              the newest release whose news this computer was shown
//
// An answer is created exclusively, so of two sessions answering together
// the first wins; news-seen is replaced atomically under a lock and only ever
// raised. Only Record and MarkSeen write. Pending, Moment and Full read, so a
// session with no person to answer it (which runs neither) records nothing.
//
// Nothing here reaches the network, and no word of the repository's tracked
// files reaches a line: versions pass launch.ParseReleaseTag, counts are
// numbers, titles are core's, and change lines come from the digest alone.
// The package returns values and never prints.
package afterupdate

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/releasedigest"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The answers the offer takes.
const (
	AnswerYes    = "yes"
	AnswerNotNow = "not-now"
)

// FullCommand is the command that shows the full summary, named on every
// news line.
const FullCommand = "abcd update news --full"

// storeLeaf is the store's folder below abcd's home; newsSeenLeaf the file
// that holds the newest release whose news was shown; lockLeaf the lock its
// raise is taken under.
const (
	storeLeaf    = "after-update"
	newsSeenLeaf = "news-seen"
	lockLeaf     = "news-seen.lock"
)

// answerSchema is an answer file's schema version.
const answerSchema = 1

// maxStoreFileBytes caps every file of the store as it is read.
const maxStoreFileBytes = 4 << 10

// maxActionIDs bounds the record ids a news line names; the rest are counted.
const maxActionIDs = 5

// lockTimeout bounds the wait for news-seen's lock.
const lockTimeout = 5 * time.Second

// rootSHARe is a repository's root commit as a store key: a full SHA-1 or
// SHA-256 object name.
var rootSHARe = regexp.MustCompile(`\A(?:[0-9a-f]{40}|[0-9a-f]{64})\z`)

var (
	// ErrNotRelease is the refusal of a running binary that is not a release
	// (a dev build): it takes no part in the moment.
	ErrNotRelease = errors.New("this abcd is not a release build, so there is no update to record")
	// ErrNoOffer is Record's refusal where no offer applies: the repository's
	// setup version is not an older release than the running one, or the
	// repository has no root commit to key the answer on.
	ErrNoOffer = errors.New("no setup offer applies to this repository for this release")
	// ErrAlreadyAnswered is Record's refusal of a second answer for one
	// repository and release: the first stands.
	ErrAlreadyAnswered = errors.New("the offer for this release was already answered in this repository")
	// ErrBadAnswer is Record's refusal of an answer that is neither yes nor not-now.
	ErrBadAnswer = errors.New("the answer is yes or not-now")
)

// The seams the tests stand in for: the running version, the home, the
// repository's detection, the setup version it recorded, the binary record's
// previous release, the digest's range and what waits.
var (
	runningVersion = func() string { return core.Version }
	userHome       = os.UserHomeDir
	detect         = ahoy.Detect
	setupVersion   = ahoy.SetupVersion
	previousTag    = func(cwd string) (string, bool) {
		root, _ := ahoy.ResolvePluginRoot()
		return ahoy.PreviousReleaseTag(root, cwd)
	}
	digestRange = releasedigest.Range
	waitingIn   = ahoy.Waiting
)

// PendingState says what an after-update moment holds here: News, that this
// computer has not been shown what changed up to the running release; Offer,
// that setup should be offered in this repository for it.
type PendingState struct {
	News  bool `json:"news"`
	Offer bool `json:"offer"`
}

// Any reports whether the moment holds anything.
func (p PendingState) Any() bool { return p.News || p.Offer }

// Offer is the offer to run setup again: the release the repository was set
// up with, the one running, and what waits.
type Offer struct {
	From    string          `json:"from"`
	To      string          `json:"to"`
	Waiting ahoy.WaitingSet `json:"waiting"`
	Line    string          `json:"line"`
}

// MomentLines is what the agent relays in its first reply: a line per release
// whose news is unseen on this computer (and one naming any release the
// digest cannot describe), then the offer line, if one is due here.
type MomentLines struct {
	News    *releasedigest.Summary `json:"news,omitempty"`
	Lines   []string               `json:"lines"`
	Offer   *Offer                 `json:"offer,omitempty"`
	Partial bool                   `json:"partial"`
}

// FullSummary is the full summary: every release in the range, each entry needing
// action first with the record ids to read more, the releases the digest
// cannot describe, and what waits here.
type FullSummary struct {
	Summary releasedigest.Summary `json:"summary"`
	Waiting ahoy.WaitingSet       `json:"waiting"`
	Lines   []string              `json:"lines"`
	Partial bool                  `json:"partial"`
}

// Answer is one repository's recorded answer for one release.
type Answer struct {
	SchemaVersion int    `json:"schema_version"`
	Release       string `json:"release"`
	Answer        string `json:"answer"`
}

// Pending reports, reading only, what the moment holds for the repository
// at cwd, whose detection the caller already holds (the session start runs
// one): Pending runs none of its own. A dev build holds nothing.
func Pending(cwd string, det ahoy.DetectionResult) (PendingState, error) {
	running, ok := release(runningVersion())
	if !ok {
		return PendingState{}, nil
	}
	var st PendingState
	news, err := unseenNews(cwd, running)
	if err != nil {
		return PendingState{}, err
	}
	st.News = news != nil
	offer, err := dueOffer(cwd, running, &det)
	if err != nil {
		return PendingState{}, err
	}
	st.Offer = offer != nil
	return st, nil
}

// Moment composes the moment for the repository at cwd, reading only.
func Moment(cwd string) (MomentLines, error) {
	m := MomentLines{Lines: []string{}}
	running, ok := release(runningVersion())
	if !ok {
		return m, nil
	}
	news, err := unseenNews(cwd, running)
	if err != nil {
		return MomentLines{}, err
	}
	if news != nil {
		m.News = news
		m.Lines = append(m.Lines, newsLines(*news)...)
		m.Partial = news.Partial
	}
	offer, err := dueOffer(cwd, running, nil)
	if err != nil {
		return MomentLines{}, err
	}
	if offer != nil {
		m.Offer = offer
		m.Lines = append(m.Lines, offer.Line)
	}
	return m, nil
}

// Full composes the full summary for the repository at cwd, reading
// only, so it can be shown again at any time. from is the release the range
// starts after; "" starts it at the binary record's previous release, else
// the repository's setup version, else the running release alone. What
// waits is listed where the repository's setup version is an older release.
func Full(cwd, from string) (FullSummary, error) {
	running, ok := release(runningVersion())
	if !ok {
		return FullSummary{}, ErrNotRelease
	}
	if from == "" {
		from = firstRelease(func() (string, bool) { return previousTag(cwd) }, func() (string, bool) { return setupVersion(cwd) })
	} else if _, err := launch.ParseReleaseTag(from); err != nil {
		return FullSummary{}, releasedigest.ErrNotReleaseTag
	}
	sum, err := digestRange(from, running)
	if err != nil {
		return FullSummary{}, err
	}
	f := FullSummary{Summary: sum, Waiting: ahoy.WaitingSet{Questions: []ahoy.WaitingItem{}, Defaults: []ahoy.WaitingItem{}}, Partial: sum.Partial}
	if set, ok := setupVersion(cwd); ok && older(set, running) {
		det, err := detect(cwd)
		if err != nil {
			return FullSummary{}, err
		}
		if f.Waiting, err = waitingIn(cwd, det, set, running); err != nil {
			return FullSummary{}, err
		}
	}
	f.Lines = fullLines(sum, f.Waiting)
	return f, nil
}

// Record records the person's answer to the offer for the repository at cwd
// and the running release, and raises news-seen to that release, since the
// answer was given to the moment that showed it. The answer is created
// exclusively: a second answer for the same repository and release is
// ErrAlreadyAnswered and changes nothing.
func Record(cwd, answer string) (Answer, error) {
	if answer != AnswerYes && answer != AnswerNotNow {
		return Answer{}, ErrBadAnswer
	}
	running, ok := release(runningVersion())
	if !ok {
		return Answer{}, ErrNotRelease
	}
	set, ok := setupVersion(cwd)
	if !ok || !older(set, running) {
		return Answer{}, ErrNoOffer
	}
	det, err := detect(cwd)
	if err != nil {
		return Answer{}, err
	}
	if !rootSHARe.MatchString(det.RootSHA) {
		return Answer{}, ErrNoOffer
	}
	home, err := homeDir()
	if err != nil {
		return Answer{}, err
	}
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(storeLeaf, det.RootSHA), abcdhome.DirMode)
	if err != nil {
		return Answer{}, err
	}
	defer root.Close()
	a := Answer{SchemaVersion: answerSchema, Release: running, Answer: answer}
	body := fmt.Sprintf("{\n  \"schema_version\": %d,\n  \"release\": %q,\n  \"answer\": %q\n}\n", a.SchemaVersion, a.Release, a.Answer)
	if err := fsutil.CreateExclusiveIn(root, running+".json", []byte(body), abcdhome.FileMode); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Answer{}, ErrAlreadyAnswered
		}
		return Answer{}, err
	}
	if err := raiseNewsSeen(home, running); err != nil {
		return a, err
	}
	return a, nil
}

// MarkSeen records that this computer was shown the news up to the running
// release, raising news-seen and never lowering it.
func MarkSeen() error {
	running, ok := release(runningVersion())
	if !ok {
		return ErrNotRelease
	}
	home, err := homeDir()
	if err != nil {
		return err
	}
	return raiseNewsSeen(home, running)
}

// unseenNews is the range of releases whose news this computer has not been
// shown, up to running, or nil where there is none. The range starts after
// news-seen, else the binary record's previous release, else the
// repository's setup version, else it is the running release alone.
func unseenNews(cwd, running string) (*releasedigest.Summary, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	seen, err := readNewsSeen(home)
	if err != nil {
		return nil, err
	}
	from := firstRelease(
		func() (string, bool) { return seen, seen != "" },
		func() (string, bool) { return previousTag(cwd) },
		func() (string, bool) { return setupVersion(cwd) },
	)
	sum, err := digestRange(from, running)
	if err != nil {
		return nil, err
	}
	if len(sum.Releases) == 0 && sum.Gap == nil {
		return nil, nil
	}
	return &sum, nil
}

// dueOffer is the offer due in the repository at cwd for running, or nil:
// the setup version is an older release, the repository has a root commit,
// no answer is recorded for it and running, and something waits. held is the
// caller's detection of the repository; nil runs one, and only where the
// setup version is an older release.
func dueOffer(cwd, running string, held *ahoy.DetectionResult) (*Offer, error) {
	set, ok := setupVersion(cwd)
	if !ok || !older(set, running) {
		return nil, nil
	}
	var det ahoy.DetectionResult
	if held != nil {
		det = *held
	} else {
		var err error
		if det, err = detect(cwd); err != nil {
			return nil, err
		}
	}
	if !rootSHARe.MatchString(det.RootSHA) {
		return nil, nil
	}
	answered, err := hasAnswer(det.RootSHA, running)
	if err != nil || answered {
		return nil, err
	}
	w, err := waitingIn(cwd, det, set, running)
	if err != nil {
		return nil, err
	}
	if len(w.Questions)+len(w.Defaults) == 0 {
		return nil, nil
	}
	return &Offer{From: set, To: running, Waiting: w, Line: offerLine(set, running, len(w.Questions), len(w.Defaults))}, nil
}

// hasAnswer reports whether an answer is recorded for the repository whose
// root commit is sha and the release tag: anything at its name counts.
func hasAnswer(sha, tag string) (bool, error) {
	home, err := homeDir()
	if err != nil {
		return false, err
	}
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel(storeLeaf, sha))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer root.Close()
	if _, err := root.Lstat(tag + ".json"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// readNewsSeen returns the release news-seen holds, or "" where it holds
// none: absent, or not exactly a release tag (a raise replaces it).
func readNewsSeen(home string) (string, error) {
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel(storeLeaf))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer root.Close()
	return newsSeenIn(root)
}

func newsSeenIn(root *os.Root) (string, error) {
	data, err := fsutil.ReadGuardedInRoot(root, newsSeenLeaf, maxStoreFileBytes)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	tag, ok := release(strings.TrimSpace(string(data)))
	if !ok {
		return "", nil
	}
	return tag, nil
}

// raiseNewsSeen sets news-seen to tag unless it already holds tag or a newer
// release, atomically and under the store's lock, so two raises together
// never lower it.
func raiseNewsSeen(home, tag string) error {
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(storeLeaf), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WithFileLockIn(root, lockLeaf, lockTimeout, func() error {
		cur, err := newsSeenIn(root)
		if err != nil {
			return err
		}
		if cur != "" && !older(cur, tag) {
			return nil
		}
		return fsutil.WriteFileAtomicInRoot(root, newsSeenLeaf, []byte(tag+"\n"), abcdhome.FileMode)
	})
}

// homeDir is the person's home directory, or an error naming that none
// resolves.
func homeDir() (string, error) {
	home, err := userHome()
	if err != nil || home == "" {
		return "", errors.New("no home directory resolves, so the after-update record has nowhere to live")
	}
	return home, nil
}

// release returns v as a release tag when it is exactly one.
func release(v string) (string, bool) {
	sv, err := launch.ParseReleaseTag(v)
	if err != nil {
		return "", false
	}
	return sv.Tag(), true
}

// older reports whether release tag a is older than release tag b; both have
// passed release.
func older(a, b string) bool {
	av, err := launch.ParseReleaseTag(a)
	if err != nil {
		return false
	}
	bv, err := launch.ParseReleaseTag(b)
	if err != nil {
		return false
	}
	return launch.CoreGreater(bv, av)
}

// firstRelease returns the first candidate that gives a release tag, or "".
func firstRelease(candidates ...func() (string, bool)) string {
	for _, c := range candidates {
		if v, ok := c(); ok {
			if tag, ok := release(v); ok {
				return tag
			}
		}
	}
	return ""
}

// newsLines is one line per release in sum, newest first, then a line naming
// what the digest cannot describe.
func newsLines(sum releasedigest.Summary) []string {
	var out []string
	for _, r := range sum.Releases {
		n := len(r.Entries)
		line := fmt.Sprintf("abcd %s: %s", r.Tag, plural(n, "change", "changes"))
		if k := r.NeedsAction(); k > 0 {
			verb := "need"
			if k == 1 {
				verb = "needs"
			}
			line += fmt.Sprintf(", %d %s you to act (%s)", k, verb, actionIDs(r))
		}
		out = append(out, line+". Ask for details: "+FullCommand)
	}
	if sum.Gap != nil {
		out = append(out, gapLine(*sum.Gap))
	}
	return out
}

// gapLine names what the digest cannot describe and where its notes are.
func gapLine(g releasedigest.Gap) string {
	what := "abcd " + g.Through
	if g.After != "" {
		what = "the abcd releases after " + g.After + " through " + g.Through
	}
	return "abcd cannot describe " + what + "; its notes are at " + g.NotesURL +
		" and every release's at " + g.ReleasesURL + ", so this summary is partial."
}

// actionIDs names the breaking records a release's entries cite, the first
// maxActionIDs of them and a count of the rest.
func actionIDs(r releasedigest.Release) string {
	var ids []string
	seen := map[string]bool{}
	for _, e := range r.Entries {
		for _, rec := range e.Records {
			if rec.Impact == releasedigest.ImpactBreaking && !seen[rec.ID] {
				seen[rec.ID] = true
				ids = append(ids, rec.ID)
			}
		}
	}
	if len(ids) > maxActionIDs {
		return strings.Join(ids[:maxActionIDs], ", ") + fmt.Sprintf(" and %d more", len(ids)-maxActionIDs)
	}
	return strings.Join(ids, ", ")
}

// offerLine is the offer: the two versions and the count of what waits.
func offerLine(from, to string, questions, defaults int) string {
	var what string
	switch {
	case questions > 0 && defaults > 0:
		what = plural(questions, "setup question", "setup questions") + " and " +
			plural(defaults, "changed default", "changed defaults") + " wait here"
	case questions > 0:
		what = plural(questions, "setup question", "setup questions") + waits(questions)
	default:
		what = plural(defaults, "changed default", "changed defaults") + waits(defaults)
	}
	return "abcd was updated from " + from + " to " + to + " since this repository was set up; " +
		what + ". Run setup now? (yes / not now)"
}

func waits(n int) string {
	if n == 1 {
		return " waits here"
	}
	return " wait here"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// fullLines renders the full summary: each release, its entries needing
// action first, each with the records it cites; the releases not described;
// then what waits here.
func fullLines(sum releasedigest.Summary, w ahoy.WaitingSet) []string {
	out := []string{}
	if len(sum.Releases) == 0 && sum.Gap == nil {
		out = append(out, "abcd "+sum.To+": no release to describe after "+sum.From+".")
	}
	for _, r := range sum.Releases {
		head := "abcd " + r.Tag
		if r.BaseTag != "" {
			head += " (after " + r.BaseTag + ")"
		}
		out = append(out, head+": "+plural(len(r.Entries), "change", "changes"))
		for _, e := range r.Entries {
			ids := make([]string, len(e.Records))
			for i, rec := range e.Records {
				ids[i] = rec.ID
			}
			mark := "  - "
			if e.NeedsAction() {
				mark = "  - needs you to act: "
			}
			out = append(out, mark+e.Line+" ("+strings.Join(ids, ", ")+")")
		}
	}
	if sum.Gap != nil {
		out = append(out, gapLine(*sum.Gap))
	}
	if len(w.Questions)+len(w.Defaults) > 0 {
		out = append(out, "Waiting in this repository:")
		for _, q := range w.Questions {
			out = append(out, "  - setup question: "+q.Title)
		}
		for _, d := range w.Defaults {
			out = append(out, "  - changed default: "+d.Title)
		}
	}
	return out
}
