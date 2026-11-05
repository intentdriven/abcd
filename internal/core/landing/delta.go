package landing

// delta.go is the delta review (iss-2611012123291272): a review receipt for a
// later head that rests on the full pair an earlier head passed, so a wording fix
// after a full pair costs a pair over that fix rather than a pair over the branch.
// `abcd land --rests-on` records one, and the merge gate judges it again at
// landing.
//
// A delta stands only while all of these hold: the base head has a full pair's
// receipt, and a delta's receipt is not one; the head descends from the base;
// each return's first line states that it read base..head; and the
// branch's own changes since the base lie on the closed list, are plain text
// files of mode 100644, change wording alone, and are within the bound. Anything
// else needs a full pair (dcn-2611042205556221, dcn-2611042205556688).
//
// The branch's own changes are its single-parent commits, counted commit by
// commit. A merge counts for nothing when it takes a commit already on the
// default branch as its second parent and holds exactly the tree git makes of
// its two parents with no conflict; every other merge is refused, and so is any
// commit neither on the default branch nor on the first-parent line from the head
// to the base. The merge is recomputed with every merge driver git's built-in
// text merge, as the loop's conflict probe computes one, so it starts no program
// a repository names. History is read as stored throughout, whatever a replace ref
// or a graft says.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The bound on a delta: the most a branch's own commits may change since the
// base head and still rest on its full pair. It is drawn around what the records
// show a delta is for, a one-sentence fix to a record.
const (
	// MaxDeltaFiles is how many files a delta may touch, counted once each.
	MaxDeltaFiles = 3
	// MaxDeltaLines is how many lines a delta may change, added and removed
	// together, summed over the branch's own commits.
	MaxDeltaLines = 30
)

// deltaCovers is the closed list of the directories under which a delta review
// may change a file: a delta pair reviews a wording fix after a full pair. The
// list is in the build of abcd that runs land, its paths are compared exactly, and
// a path under none of them needs a full pair.
var deltaCovers = []string{
	"docs/explanation/",
	"docs/how-to/",
	"docs/tutorials/",
	".abcd/development/plans/",
	".abcd/development/roadmap/",
	".abcd/work/issues/resolved/",
	".abcd/work/issues/wontfix/",
}

// deltaCoversKind is the one kind of file the list admits: prose, not the
// scripts, data, attributes or pinned dependencies those directories also hold.
const deltaCoversKind = ".md"

// deltaLedger are the listed directories that hold ledger records, each of which
// carries frontmatter and may carry a grounds section.
var deltaLedger = []string{
	".abcd/work/issues/resolved/",
	".abcd/work/issues/wontfix/",
}

// hostInstructionPrefixes are how the names of hosts' instruction files begin,
// folded. A path part beginning with one is off the list, however it goes on.
var hostInstructionPrefixes = []string{
	"agent", "aider", "claude", "cline", "codex", "copilot", "crush", "cursor", "gemini",
	"instruction", "junie", "kiro", "qwen", "rules", "skill", "warp", "windsurf",
}

// covered reports whether a delta review may change p, a path as git names it.
// A path git would not write as it stands, with a .. in it, is not on the list
// whatever it starts with.
func covered(p string) bool {
	if !strings.HasSuffix(p, deltaCoversKind) || path.Clean(p) != p {
		return false
	}
	for _, dir := range deltaCovers {
		if rest, ok := strings.CutPrefix(p, dir); ok {
			return !hostLoads(rest)
		}
	}
	return false
}

// hostLoads reports whether a path below a listed directory has a part that is a
// dot-directory or a dot-file, or whose name begins as an instruction file's does,
// in any case.
func hostLoads(rest string) bool {
	for _, part := range strings.Split(rest, "/") {
		folded := strings.ToLower(part)
		if strings.HasPrefix(part, ".") || slices.ContainsFunc(hostInstructionPrefixes, func(prefix string) bool { return strings.HasPrefix(folded, prefix) }) {
			return true
		}
	}
	return false
}

// ledgerRecord reports whether p is a ledger record.
func ledgerRecord(p string) bool {
	return slices.ContainsFunc(deltaLedger, func(dir string) bool { return strings.HasPrefix(p, dir) })
}

// Modes in git's raw diff: absent on one side of a change, and the one mode a
// delta covers.
const (
	modeAbsent  = "000000"
	modeRegular = "100644"
)

// Bounds on what judging a delta reads.
const (
	maxReceiptBytes   = 1 << 20
	maxDeltaGitOutput = 1 << 20
	maxDeltaBlobBytes = 1 << 24
	// maxDeltaPatchBytes bounds a delta's patches, summed over the branch's own
	// commits with no context lines.
	maxDeltaPatchBytes = 1 << 16
)

// deltaRecord is what a delta's receipt adds to a full pair's: the base head it
// rests on and what the branch's own commits changed since it.
type deltaRecord struct {
	Base   string `json:"base"`
	Files  int    `json:"files"`
	Lines  int    `json:"lines"`
	Bytes  int    `json:"bytes"`
	Merges int    `json:"merges"`
	// Records marks the build loop's receipt for its own records commit on the
	// head its pair judged (loopreview.go), judged by judgeRecords.
	Records bool `json:"records,omitempty"`
}

const restsOnItself = "a head cannot rest on itself"

// reviewedDelta is a delta return's first line that is not blank, stating the delta
// it read, exactly: nothing before the words and nothing after the range.
var reviewedDelta = regexp.MustCompile(`\A(?:[ \t]*\r?\n)*Reviewed delta: ([0-9a-f]+)\.\.([0-9a-f]+)[ \t]*\r?(?:\n|\z)`)

// anyDeltaLine is any line that begins by stating a delta, in any case, whatever
// markup leads it or closes the words, by a colon or a range. A full pair's return
// holds no such line.
var anyDeltaLine = regexp.MustCompile(`(?mi)^[^\pL\pN\n]*(?:[0-9]+[.)][ \t]+)?reviewed[ \t]+delta\b[^\pL\pN\n]*?(?::|[0-9a-f]+\.\.)`)

// statesDelta reports why a return does not state that it read base..head: its
// first line reads Reviewed delta: <base>..<head>, naming each end by at least the
// nine characters a brief abbreviates a head to, no other line states a delta, and
// no line states a head.
func statesDelta(text, base, head string) error {
	if n := len(anyDeltaLine.FindAllString(text, -1)); n > 1 {
		return fmt.Errorf("it has %d Reviewed delta lines; its first line states what it read", n)
	}
	if anyHeadLine.MatchString(text) {
		return errors.New("it has a Reviewed head line, and a delta's return states on its first line the Reviewed delta it read")
	}
	m := reviewedDelta.FindStringSubmatch(text)
	if m == nil {
		return errors.New("its first line does not read Reviewed delta: <base>..<head>")
	}
	b, h := m[1], m[2]
	if len(b) < reviewedCommitPrefix || len(h) < reviewedCommitPrefix || !strings.HasPrefix(base, b) || !strings.HasPrefix(head, h) {
		return fmt.Errorf("it does not name the delta %s..%s, so it cannot be told it read this diff", base[:reviewedCommitPrefix], head[:reviewedCommitPrefix])
	}
	return nil
}

// reviewedHead is a full pair's return's first line that is not blank, stating
// the head it read, exactly: nothing before the words and nothing after the name.
var reviewedHead = regexp.MustCompile(`\A(?:[ \t]*\r?\n)*Reviewed head: ([0-9a-f]+)[ \t]*\r?(?:\n|\z)`)

// anyHeadLine is any line that begins by stating a head, in any case, whatever
// markup leads it.
var anyHeadLine = regexp.MustCompile(`(?mi)^[^\pL\pN\n]*reviewed[ \t]+head\b[^\pL\pN\n]*?(?::|[0-9a-f]{7,})`)

// statesHead reports why a full pair's return does not state that it read head:
// its first line reads Reviewed head: <name>, naming head by at least the nine
// characters a brief abbreviates it to, and no other line states a head.
func statesHead(text, head string) error {
	want := "Reviewed head: " + head[:reviewedCommitPrefix]
	m := reviewedHead.FindStringSubmatch(text)
	if m == nil {
		return fmt.Errorf("its first line does not read %s, so it cannot be told it reviewed this head", want)
	}
	if len(m[1]) < reviewedCommitPrefix || !strings.HasPrefix(head, m[1]) {
		return fmt.Errorf("its first line does not name the head; it should read %s", want)
	}
	if n := len(anyHeadLine.FindAllString(text, -1)); n > 1 {
		return fmt.Errorf("it has %d Reviewed head lines; its first line states what it read", n)
	}
	return nil
}

// statesFullPair reports why a return is not a full pair's for head: it states a
// delta, which a full pair's does not, since a full pair reads the whole head, or
// it does not state the head on its first line.
func statesFullPair(text, head string) error {
	if anyDeltaLine.MatchString(text) {
		return errors.New("it has a Reviewed delta line, so it read a delta and not the head; a full pair reads the whole head, so review the head again, or, where the delta stands, record it with --rests-on naming the base head")
	}
	return statesHead(text, head)
}

// RecordDeltaReview writes the review receipt for the commit head, resting on
// the full pair base has: a review pair as RecordReview takes one, each return
// also stating that it read base..head, over a delta that stands (judgeDelta).
// Anything short of that is refused and nothing is written. A delta never
// replaces a full pair already recorded for head.
func RecordDeltaReview(repoRoot, head, base string, returns []ReviewReturn) (string, error) {
	if err := holdsCommit(repoRoot, head); err != nil {
		return "", err
	}
	if !fullSHA.MatchString(base) {
		return "", fmt.Errorf("the base head %s is not a commit's full object name", clip(base))
	}
	if base == head {
		return "", errors.New(restsOnItself)
	}
	tip := gitutil.DefaultRef(repoRoot)
	if tip == "" {
		return "", errors.New("this repository has no default branch to read, so a delta cannot be judged")
	}
	reads, err := readReviewReceipts(repoRoot, head)
	if err != nil {
		return "", err
	}
	for _, r := range reads {
		if r.err == nil && isFullPair(r.rec, head) {
			return "", fmt.Errorf("%s already has a full review pair's receipt, which a delta would only weaken", short(head))
		}
	}
	entries, err := judgeReturns(returns, func(text string) error { return statesDelta(text, base, head) })
	if err != nil {
		return "", err
	}
	delta, err := judgeDelta(repoRoot, base, head, tip)
	if err != nil {
		if couldNotJudge(err) {
			return "", err
		}
		return "", fmt.Errorf("%w; the head needs a full review pair (abcd land --record-review)", err)
	}
	return writeReviewRecord(repoRoot, newReviewRecord(head, entries, &delta))
}

// judgeDelta answers whether head may rest on base's full pair, with the tip of
// the default branch, and what the branch's own commits changed. The error
// says what fails, and every failure means a full pair is needed.
func judgeDelta(repoRoot, base, head, tip string) (deltaRecord, error) {
	if base == head {
		return deltaRecord{}, errors.New(restsOnItself)
	}
	if err := hasFullPair(repoRoot, base); err != nil {
		return deltaRecord{}, err
	}
	on, err := storedAncestor(repoRoot, base, head)
	if err != nil {
		return deltaRecord{}, fmt.Errorf("git cannot say whether %s descends from %s: %w", short(head), short(base), err)
	}
	if !on {
		return deltaRecord{}, fmt.Errorf("%s does not descend from the base head %s", short(head), short(base))
	}
	return ownChanges(repoRoot, base, head, tip)
}

// storedBytes reads git's answer verbatim, with history as stored.
func storedBytes(repoRoot string, maxBytes int, args ...string) ([]byte, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(context.Background(), repoRoot, "", maxBytes, args...)
	if err != nil {
		return nil, notJudged{err}
	}
	return out, nil
}

// storedAncestor reports whether a commit is an ancestor of another, with history
// as stored.
func storedAncestor(repoRoot, ancestor, descendant string) (bool, error) {
	_, off, err := gitutil.RunCappedExitOneAsStored(repoRoot, maxDeltaGitOutput, "merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		return false, notJudged{err}
	}
	return !off, nil
}

// hasFullPair reports why base has no full pair's receipt: none in any
// worktree, one that cannot be read, a delta's, or one holding no passing pair.
func hasFullPair(repoRoot, base string) error {
	none := "the base head " + short(base) + " has no full review pair recorded"
	reads, err := readReviewReceipts(repoRoot, base)
	if err != nil {
		return fmt.Errorf("%s that can be read: %w", none, err)
	}
	if len(reads) == 0 {
		return errors.New(none)
	}
	for _, r := range reads {
		if r.err == nil && isFullPair(r.rec, base) {
			return nil
		}
	}
	for _, r := range reads {
		if couldNotJudge(r.err) {
			return fmt.Errorf("%s that can be read: %w", none, r.err)
		}
	}
	switch first := reads[0]; {
	case first.err != nil:
		return fmt.Errorf("%s that can be read: %w", none, first.err)
	case first.rec.Delta != nil:
		return fmt.Errorf("%s: its receipt rests on %s, and a delta rests on a full pair directly", none, short(first.rec.Delta.Base))
	}
	return fmt.Errorf("%s: its receipt holds no passing pair for it", none)
}

// notJudged marks a failure that leaves the gate unable to judge a receipt or a
// delta: a listing, a read or a git command that gave no answer. A valid receipt may
// lie behind it, so it is no refusal of the head. A receipt read and found wanting
// carries no such mark.
type notJudged struct{ error }

func (e notJudged) Unwrap() error { return e.error }

// couldNotJudge reports whether a failure carries the mark, however it was wrapped.
func couldNotJudge(err error) bool {
	var n notJudged
	return errors.As(err, &n)
}

// reviewReceiptRead is one worktree's review receipt as read, or why it cannot be.
type reviewReceiptRead struct {
	rec reviewRecord
	err error
}

// readReviewReceipts reads every worktree's review receipt naming sha.
func readReviewReceipts(repoRoot, sha string) ([]reviewReceiptRead, error) {
	found, err := findReceipts(repoRoot, ReviewReceiptsRelDir, sha)
	if err != nil {
		return nil, notJudged{err}
	}
	reads := make([]reviewReceiptRead, 0, len(found))
	for _, f := range found {
		if f.err != nil {
			// git said it tracks the receipt, or could not be asked.
			var refused *localtier.Refused
			if errors.As(f.err, &refused) && refused.Cause != nil {
				f.err = notJudged{f.err}
			}
			reads = append(reads, reviewReceiptRead{err: f.err})
			continue
		}
		rec, err := readReviewRecord(f.path)
		reads = append(reads, reviewReceiptRead{rec, err})
	}
	return reads, nil
}

// isFullPair reports whether a receipt is a full pair's for sha.
func isFullPair(rec reviewRecord, sha string) bool {
	return rec.Delta == nil && rec.Commit == sha && holdsPair(rec)
}

// holdsPair reports whether a receipt lists a review pair that passed.
func holdsPair(rec reviewRecord) bool {
	if len(rec.Reviews) < minReviewers {
		return false
	}
	for _, r := range rec.Reviews {
		if !passingVerdicts[r.Verdict] {
			return false
		}
	}
	return true
}

func readReviewRecord(path string) (reviewRecord, error) {
	data, err := fsutil.ReadGuarded(path, maxReceiptBytes)
	if err != nil {
		if errors.Is(err, fsutil.ErrTooBig) || errors.Is(err, fsutil.ErrNotRegular) {
			return reviewRecord{}, err
		}
		return reviewRecord{}, notJudged{err}
	}
	var rec reviewRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return reviewRecord{}, errors.New("the receipt does not parse")
	}
	return rec, nil
}

// reviewProof is what the merge gate reads of the review receipts for a head.
type reviewProof struct {
	reviewed bool
	// note says what a standing delta rests on, "" for a full pair.
	note string
	// deltaWhy says why a delta's receipt for the head does not stand.
	deltaWhy string
	// unreadable says why the receipts could not be read.
	unreadable string
	// unread says the gate could not judge them, so no answer rests on them.
	unread bool
}

// readReviewProof reads every worktree's review receipt naming head. A full pair
// in any of them is the answer; failing that, a delta that stands against the
// default branch's tip; failing that, the first the gate could not judge, or what is
// wrong with the first.
func readReviewProof(repoRoot, head, tip string) reviewProof {
	reads, err := readReviewReceipts(repoRoot, head)
	if err != nil {
		return reviewProof{unreadable: termsafe.Sanitize(fsutil.RedactHome(err.Error())), unread: true}
	}
	for _, r := range reads {
		if r.err == nil && isFullPair(r.rec, head) {
			return reviewProof{reviewed: true}
		}
	}
	var first reviewProof
	for i, r := range reads {
		proof := receiptProof(repoRoot, head, tip, r)
		if proof.reviewed {
			return proof
		}
		// A receipt the gate could not judge outweighs one it found wanting: what
		// it could not read may be the one that stands.
		if i == 0 || (proof.unread && !first.unread) {
			first = proof
		}
	}
	return first
}

// receiptProof judges one receipt for head. A receipt that does not parse, or
// holds no passing pair for head, proves nothing.
func receiptProof(repoRoot, head, tip string, r reviewReceiptRead) reviewProof {
	if r.err != nil {
		return reviewProof{unreadable: termsafe.Sanitize(fsutil.RedactHome(r.err.Error())), unread: couldNotJudge(r.err)}
	}
	rec := r.rec
	if rec.Delta == nil {
		if !isFullPair(rec, head) {
			return reviewProof{unreadable: "the receipt holds no passing pair for " + short(head)}
		}
		return reviewProof{reviewed: true}
	}
	why := func(err error) reviewProof {
		return reviewProof{deltaWhy: termsafe.Sanitize(fsutil.RedactHome(err.Error())), unread: couldNotJudge(err)}
	}
	if rec.Commit != head || !holdsPair(rec) {
		return why(fmt.Errorf("its receipt holds no passing pair for %s", short(head)))
	}
	if !fullSHA.MatchString(rec.Delta.Base) {
		return why(fmt.Errorf("its receipt names %s as its base, which is not a commit's full object name", clip(rec.Delta.Base)))
	}
	if rec.Delta.Records {
		if _, err := judgeRecords(repoRoot, rec.Delta.Base, head); err != nil {
			return why(err)
		}
		return reviewProof{reviewed: true, note: "the landing's records on the full pair at " + short(rec.Delta.Base)}
	}
	if _, err := judgeDelta(repoRoot, rec.Delta.Base, head, tip); err != nil {
		return why(err)
	}
	return reviewProof{reviewed: true, note: "a delta review resting on the full pair at " + short(rec.Delta.Base)}
}

// ownChanges walks the first-parent line from head back to base, adding up each
// single-parent commit and proving each merge adds nothing of its own, and
// refuses what leaves that line.
func ownChanges(repoRoot, base, head, tip string) (deltaRecord, error) {
	out, err := storedBytes(repoRoot, maxDeltaGitOutput, "rev-list", "--parents", head, "^"+base, "^"+tip)
	if err != nil {
		return deltaRecord{}, fmt.Errorf("git cannot list the branch's own commits: %w", err)
	}
	parents := map[string][]string{}
	for _, ln := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f := strings.Fields(ln); len(f) > 0 {
			parents[f[0]] = f[1:]
		}
	}
	d := deltaRecord{Base: base}
	files := map[string]bool{}
	walked := 0
	for cur := head; cur != base; {
		ps, own := parents[cur]
		if !own {
			return deltaRecord{}, fmt.Errorf("the first-parent line from %s to the base head %s leaves the branch at %s", short(head), short(base), short(cur))
		}
		walked++
		switch len(ps) {
		case 1:
			if err := addOwnCommit(repoRoot, ps[0], cur, files, &d); err != nil {
				return deltaRecord{}, err
			}
		case 2:
			if err := provesCleanMerge(repoRoot, cur, ps[0], ps[1], tip); err != nil {
				return deltaRecord{}, err
			}
			d.Merges++
		default:
			return deltaRecord{}, fmt.Errorf("the commit %s has %d parents, and only a merge of one default-branch commit counts for nothing", short(cur), len(ps))
		}
		cur = ps[0]
	}
	if walked != len(parents) {
		return deltaRecord{}, fmt.Errorf("%d of the branch's commits since %s are off its first-parent line, on no default branch, so no merge accounts for them", len(parents)-walked, short(base))
	}
	return d, nil
}

// addOwnCommit adds one single-parent commit to the delta, refusing a path off
// the closed list, an added or a removed file, any mode but 100644, anything but
// plain text on either side of a change, anything past the bound, and any changed
// line that is not wording or falls on its side inside a code fence or a ledger
// record's grounds. Every file keeps its frontmatter byte for byte.
func addOwnCommit(repoRoot, parent, commit string, files map[string]bool, d *deltaRecord) error {
	read := func(format ...string) ([]byte, error) {
		args := append([]string{"diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none"}, format...)
		out, err := storedBytes(repoRoot, maxDeltaGitOutput, append(args, parent, commit)...)
		if err != nil {
			return nil, fmt.Errorf("git cannot read what %s changes: %w", short(commit), err)
		}
		return out, nil
	}
	diff := func(format string) ([]string, error) {
		out, err := read(format, "-z")
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			return nil, nil
		}
		return strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00"), nil
	}
	raw, err := diff("--raw")
	if err != nil {
		return err
	}
	if len(raw)%2 != 0 {
		return fmt.Errorf("git's list of what %s changes does not parse", short(commit))
	}
	type sideOfChange struct {
		blob, path string
		file, side int
	}
	var blobs []sideOfChange
	var paths []string
	for i := 0; i < len(raw); i += 2 {
		f, path := strings.Fields(raw[i]), raw[i+1]
		if len(f) < 4 {
			return fmt.Errorf("git's list of what %s changes does not parse", short(commit))
		}
		if !covered(path) {
			return fmt.Errorf("%s changes %s, outside what a delta review covers (%s)", short(commit), clip(path), strings.Join(deltaCovers, ", "))
		}
		for side, mode := range []string{strings.TrimPrefix(f[0], ":"), f[1]} {
			switch mode {
			case modeAbsent:
				return fmt.Errorf("%s adds or removes %s, and a delta review changes the wording of a file already there", short(commit), clip(path))
			case modeRegular:
				blobs = append(blobs, sideOfChange{f[2+side], path, len(paths), side})
			default:
				return fmt.Errorf("%s changes %s at mode %s, and a delta review covers mode %s alone", short(commit), clip(path), mode, modeRegular)
			}
		}
		files[path] = true
		paths = append(paths, path)
	}
	d.Files = len(files)
	if d.Files > MaxDeltaFiles {
		return pastTheBound(d)
	}
	heads := make([][2]string, len(paths))
	fenced := make([][2][]bool, len(paths))
	grounded := make([][2][]bool, len(paths))
	for _, side := range blobs {
		data, err := storedBytes(repoRoot, maxDeltaBlobBytes, "cat-file", "blob", side.blob)
		if err != nil {
			return fmt.Errorf("%s changes %s, which cannot be read whole: %w", short(commit), clip(side.path), err)
		}
		if why := notPlainText(data); why != "" {
			return fmt.Errorf("%s changes %s, which holds %s, and a delta review covers plain text alone", short(commit), clip(side.path), why)
		}
		fenced[side.file][side.side] = fencedLines(string(data))
		head := frontmatterBlock(string(data))
		if ledgerRecord(side.path) {
			if head == "" {
				return fmt.Errorf("%s changes %s, which holds no frontmatter on one side, and a delta review changes a ledger record's body alone", short(commit), clip(side.path))
			}
			grounded[side.file][side.side] = groundsLines(string(data))
		}
		heads[side.file][side.side] = head
	}
	for i, head := range heads {
		if head[0] != head[1] {
			return fmt.Errorf("%s changes %s's frontmatter, and a delta review keeps every file's frontmatter", short(commit), clip(paths[i]))
		}
	}
	stat, err := diff("--numstat")
	if err != nil {
		return err
	}
	counted, err := countLines(stat, paths)
	if err != nil {
		return fmt.Errorf("%s %w", short(commit), err)
	}
	d.Lines += counted
	if d.Lines > MaxDeltaLines {
		return pastTheBound(d)
	}
	patch, err := read("-p", "-U0")
	if err != nil {
		return err
	}
	d.Bytes += len(patch)
	if d.Bytes > maxDeltaPatchBytes {
		return fmt.Errorf("the branch's own commits since %s change at least %d bytes, past the bound of %d bytes", short(d.Base), d.Bytes, maxDeltaPatchBytes)
	}
	changed, ok := changedLines(patch, len(paths))
	if !ok {
		return fmt.Errorf("git's diff of %s cannot be read line by line, so it cannot be told to change wording alone", short(commit))
	}
	roots, err := rootFiles(repoRoot, commit)
	if err != nil {
		return err
	}
	for _, line := range changed {
		in := fenced[line.file][line.side]
		if line.number < 1 || line.number > len(in) || in[line.number-1] {
			return fmt.Errorf("%s changes a line inside a code fence, and a delta review covers a change of wording alone", short(commit))
		}
		if g := grounded[line.file][line.side]; line.number <= len(g) && g[line.number-1] {
			return fmt.Errorf("%s changes a line under %s's %s heading, and a delta review leaves a ledger record's grounds as they are", short(commit), clip(paths[line.file]), grounds.Heading)
		}
		if why := notWording(line.text, roots); why != "" {
			return fmt.Errorf("%s changes a line holding %s, and a delta review covers a change of wording alone", short(commit), why)
		}
	}
	return nil
}

// fenceLead is what may stand before a code fence's mark on its line: indent,
// block quote marks and list markers.
var fenceLead = regexp.MustCompile(`^[ \t>]*(?:(?:[-*+]|[0-9]+[.)])[ \t]+)*[ \t>]*`)

// fencedLines says, line by line as git numbers them, whether any reading places a
// line in a code fence or on its mark: mdrecord's readings of the file as it
// stands, and its top-level reading with what leads each line taken off. A fence
// never closed runs to the end of the file.
func fencedLines(text string) []bool {
	lines := strings.Split(text, "\n")
	bare := make([]string, len(lines))
	for i, line := range lines {
		bare[i] = fenceLead.ReplaceAllString(line, "")
	}
	in := make([]bool, len(lines))
	for _, read := range []mdrecord.Reading{
		mdrecord.Read(lines, mdrecord.TopLevel),
		mdrecord.Read(lines, mdrecord.ListNested),
		mdrecord.Read(bare, mdrecord.TopLevel),
	} {
		for i, m := range read.Mask {
			in[i] = in[i] || m&mdrecord.MaskFence != 0
		}
	}
	return in
}

// groundsLines says, line by line as git numbers them, whether a line lies in a
// ledger record's grounds section, as the section's reader finds it in the body.
func groundsLines(text string) []bool {
	head, body := frontmatter.Split(text)
	in := make([]bool, strings.Count(text, "\n")+1)
	start, end, ok := grounds.SectionLines(body)
	if !ok {
		return in
	}
	offset := strings.Count(head, "\n")
	for i := start; i < end && offset+i < len(in); i++ {
		in[offset+i] = true
	}
	return in
}

// hunkHeader is a line opening a hunk of a patch, with where each side starts and
// how many lines it holds.
var hunkHeader = regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

// changedLine is one line a patch adds or removes: the file by its place in the
// diff, its side (0 removed, 1 added), its number on that side and its text.
type changedLine struct {
	file, side, number int
	text               string
}

// changedLines gives the lines a patch with no context lines adds and removes,
// walking each hunk by its own count, so a line's text is never taken for a
// header. It is not ok when a hunk does not hold what it says it holds, or the
// patch does not hold the number of files git's raw list names.
func changedLines(patch []byte, files int) (changed []changedLine, ok bool) {
	count := func(s string) int {
		if s == "" {
			return 1
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return -1
		}
		return n
	}
	lines := strings.Split(string(patch), "\n")
	file := -1
	for i := 0; i < len(lines); {
		if strings.HasPrefix(lines[i], "diff --git ") {
			file++
		}
		m := hunkHeader.FindStringSubmatch(lines[i])
		i++
		if m == nil {
			continue
		}
		removed, added := count(m[2]), count(m[4])
		from, to := count(m[1]), count(m[3])
		if removed < 0 || added < 0 || from < 0 || to < 0 || file < 0 || file >= files {
			return nil, false
		}
		for left := removed + added; left > 0; {
			if i >= len(lines) || lines[i] == "" {
				return nil, false
			}
			line := lines[i]
			i++
			switch line[0] {
			case '-':
				changed = append(changed, changedLine{file, 0, from, strings.TrimSuffix(line[1:], "\r")})
				from++
				left--
			case '+':
				changed = append(changed, changedLine{file, 1, to, strings.TrimSuffix(line[1:], "\r")})
				to++
				left--
			case '\\':
			default:
				return nil, false
			}
		}
	}
	return changed, file == files-1
}

// rootFiles names the files at the top of a commit's tree, whose names need no
// directory or dot to be paths.
func rootFiles(repoRoot, commit string) (map[string]bool, error) {
	out, err := storedBytes(repoRoot, maxDeltaGitOutput, "ls-tree", "-z", commit)
	if err != nil {
		return nil, fmt.Errorf("git cannot list what %s holds: %w", short(commit), err)
	}
	names := map[string]bool{}
	for _, entry := range strings.Split(string(out), "\x00") {
		if info, name, ok := strings.Cut(entry, "\t"); ok && strings.Contains(info, " blob ") {
			names[name] = true
		}
	}
	return names, nil
}

// idShape is any token in a record id's shape, in any case: a family, a hyphen and
// a number, a receipt's twelve hex digits or a frozen log line's L-number, with
// whatever slug follows; or a run of hex digits as long as a commit's short name
// or a bare ADR number. The allocator takes any lower-case family and consults no
// list (recordid.Minter), so this names no family, and one added later is in it.
var idShape = regexp.MustCompile(`(?i)[a-z]+-(?:[0-9]+|[0-9a-f]{12}|l[0-9]+)(?:[^\pL\pN]|$)|(?:^|[^\pL\pN])[0-9a-f]{7,64}(?:[^\pL\pN]|$)`)

// mentionsID reports whether text holds anything in a record id's shape.
func mentionsID(text string) bool { return idShape.MatchString(text) }

// markerShape is a gate's line marker: a tool's name and a colon, or any colon
// joined to what follows it, which a sentence's colon is not.
var markerShape = regexp.MustCompile(`(?i)(?:abcd-[a-z]*|[a-z]+-lint|gitleaks)[ \t]*:|[^ \t]:[^ \t]`)

// fileShape is a file's name with an extension, or a name that starts with a dot.
var fileShape = regexp.MustCompile(`[\pL\pN_-]\.[\pL\pN_-]{2}|(?:^|[\s(])\.[\pL\pN]`)

// wordingPunctuation is the punctuation of a sentence, and all a line of wording
// holds besides letters, digits and spaces.
const wordingPunctuation = " \t.,;:!?'\"\u2019\u2018\u201c\u201d()-\u2013\u2014%&*_\u2026"

// notWording says what keeps a changed line from being wording, "" when nothing
// does. Wording is letters, digits, spaces and a sentence's punctuation, naming no
// record and no file; anything else is not classed as wording.
func notWording(line string, roots map[string]bool) string {
	switch {
	case strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"):
		return "code by its indent"
	case strings.Contains(strings.ToLower(line), "(receipt"):
		return "the words (receipt"
	case mentionsID(line):
		return "a record id"
	case markerShape.MatchString(line):
		return "a marker"
	case fileShape.MatchString(line):
		return "a file's name"
	}
	for _, r := range line {
		if invisibleRune(r) {
			return fmt.Sprintf("the invisible character %U", r)
		}
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r) && !strings.ContainsRune(wordingPunctuation, r) {
			return fmt.Sprintf("the character %q, which is not wording's", r)
		}
	}
	for _, word := range strings.FieldsFunc(line, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if roots[word] {
			return "a file's name"
		}
	}
	return ""
}

// invisibleRune reports a rune that draws nothing, whatever its class: a variation
// selector, a default-ignorable code point, or what termsafe hides.
func invisibleRune(r rune) bool {
	return unicode.Is(unicode.Variation_Selector, r) || unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) ||
		unicode.Is(unicode.Cf, r) || termsafe.IsHidden(r)
}

// frontmatterBlock is a file's frontmatter with all that stands before it, by the
// readers' rule: the first line that is not blank or an HTML comment opens it, and
// the next delimiter closes it. It is "" for a file with no block.
func frontmatterBlock(text string) string {
	lines := strings.Split(text, "\n")
	open, _ := mdrecord.FirstContent(lines)
	if open >= len(lines) || !frontmatter.IsDelimiter(frontmatter.TrimBOM(lines[open])) {
		return ""
	}
	end := frontmatter.CloseAfter(lines, open)
	if end < 0 {
		return ""
	}
	return strings.Join(lines[:end+1], "\n")
}

// notPlainText says what keeps a file from being plain text, "" when nothing does:
// a NUL byte, bytes that are not UTF-8, a control character, or a line break other
// than LF and CRLF.
func notPlainText(data []byte) string {
	if bytes.IndexByte(data, 0) >= 0 {
		return "a NUL byte, so it is a binary file"
	}
	if !utf8.Valid(data) {
		return "bytes that are not UTF-8"
	}
	for i, r := range string(data) {
		switch {
		case r == '\r':
			if i+1 == len(data) || data[i+1] != '\n' {
				return "a CR with no LF after it"
			}
		case r == '\t' || r == '\n':
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			return fmt.Sprintf("the control character %U", r)
		case r == 0x2028 || r == 0x2029:
			return fmt.Sprintf("the line break %U", r)
		}
	}
	return ""
}

// countLines adds up git's count of one commit's changes, lines added and
// removed, and refuses a count that does not speak for exactly the paths the raw
// list names, in its order: one missing, one more or another, one git calls
// binary or one that does not parse. Its error reads on from the commit's name.
func countLines(stat, paths []string) (int, error) {
	const unparsed = "has a count of what it changes that does not parse"
	if len(stat) != len(paths) {
		return 0, fmt.Errorf("changes %d paths, and git counts %d", len(paths), len(stat))
	}
	lines := 0
	for i, e := range stat {
		f := strings.SplitN(e, "\t", 3)
		if len(f) != 3 {
			return 0, errors.New(unparsed)
		}
		if f[2] != paths[i] {
			return 0, fmt.Errorf("changes %s, and git counts %s in its place", clip(paths[i]), clip(f[2]))
		}
		if f[0] == "-" || f[1] == "-" {
			return 0, fmt.Errorf("changes %s, which git counts as a binary file", clip(f[2]))
		}
		added, aerr := strconv.ParseUint(f[0], 10, 31)
		removed, rerr := strconv.ParseUint(f[1], 10, 31)
		if aerr != nil || rerr != nil {
			return 0, errors.New(unparsed)
		}
		lines += int(added) + int(removed)
	}
	return lines, nil
}

func pastTheBound(d *deltaRecord) error {
	return fmt.Errorf("the branch's own commits since %s change at least %d lines in %d files, past the bound of %d lines in %d files", short(d.Base), d.Lines, d.Files, MaxDeltaLines, MaxDeltaFiles)
}

// provesCleanMerge reports why the merge commit is not exactly the merge git
// makes of a default-branch commit: its second parent lies off the default
// branch, git's merge of its parents conflicts, or its tree is another tree.
func provesCleanMerge(repoRoot, commit, first, second, tip string) error {
	on, err := storedAncestor(repoRoot, second, tip)
	if err != nil {
		return fmt.Errorf("git cannot say where the merge %s's second parent lies: %w", short(commit), err)
	}
	if !on {
		return fmt.Errorf("the merge %s takes %s, which is not on the default branch", short(commit), short(second))
	}
	overrides, err := gitutil.MergeDriverOverrides(repoRoot)
	if err != nil {
		return fmt.Errorf("the merge %s cannot be recomputed: %w", short(commit), notJudged{err})
	}
	args := append(overrides, "-c", "merge.union.driver="+gitutil.BuiltinMergeDriver(), "-c", "merge.renormalize=false",
		"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", first, second)
	out, conflicted, err := gitutil.RunCappedExitOneAsStored(repoRoot, maxDeltaGitOutput, args...)
	if err != nil {
		return fmt.Errorf("the merge %s cannot be recomputed: %w", short(commit), notJudged{err})
	}
	made, _, _ := strings.Cut(string(out), "\x00")
	if !gitutil.IsFullSHA(made) {
		return fmt.Errorf("the merge %s cannot be recomputed: git named no merged tree", short(commit))
	}
	if conflicted {
		return fmt.Errorf("the merge %s conflicts when git makes it, so its resolution is the author's own change", short(commit))
	}
	has, err := gitutil.RunCappedAsStored(repoRoot, maxDeltaGitOutput, "rev-parse", "--verify", commit+"^{tree}")
	if err != nil {
		return fmt.Errorf("the merge %s's tree cannot be read: %w", short(commit), notJudged{err})
	}
	if has != made {
		return fmt.Errorf("the merge %s holds a tree no clean merge makes of its parents", short(commit))
	}
	return nil
}
