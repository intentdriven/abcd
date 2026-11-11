package landing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ReviewReceiptsRelDir holds one review receipt per commit, named by its full
// object name, beside the preflight receipts (spc-2610161609350616 step 3).
// The merge gate reads one in auto mode.
const ReviewReceiptsRelDir = TierRelDir + "/review-receipts"

// MaxReviewReturnBytes bounds one review return read for a receipt.
const MaxReviewReturnBytes = 1 << 20

// minReviewers is the review pair: a receipt needs at least two returns.
const minReviewers = 2

// ReviewReturn is one reviewer's return: the name it is recorded under and its
// text, which states what it read and states its verdict under one
// `### Verdict` heading, naming the head it judges.
type ReviewReturn struct {
	Name string
	Text string
}

// passingVerdicts are the verdicts that pass the stop rule (no critical or
// major finding): a ruthless reviewer's SHIP and a security reviewer's
// APPROVE. FIX FIRST, BLOCK and NEEDS-INPUT do not.
var passingVerdicts = map[string]bool{"SHIP": true, "APPROVE": true}

// knownVerdicts are every verdict a validator writes, a word that begins
// another after it.
var knownVerdicts = []string{"NEEDS-INPUT", "FIX FIRST", "APPROVE", "BLOCK", "SHIP"}

var fullSHA = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)

// verdictEnd is what may follow a verdict word on its line, once the emphasis closing
// the word is off: what it judges by name, and then the end of the line or a
// separator. Anything else makes the word the start of another statement.
var verdictEnd = regexp.MustCompile("^(?:[ \\t]+[*_`]*(?:sha256:)?[0-9a-f]{7,64}[*_`]*)?[*_`]*[ \\t]*(?:$|[\\x{2014}\\x{2013}(]|[:;,.](?:[ \\t]|$)|-{1,2}[ \\t])")

// verdictHeading is the one Verdict heading a return's verdict is read under, at any
// depth, starting its line. A line indented or quoted, or a heading with more after
// it, is not it.
var verdictHeading = regexp.MustCompile(`(?i)^#{1,6}[ \t]+verdict[ \t]*#*[ \t]*$`)

// reviewedCommitPrefix is how much of the commit's name a return's first line must
// carry: the nine characters the build loop's briefs abbreviate a head to.
const reviewedCommitPrefix = 9

// lineBreaks are the ways a text can end a line. A verdict is read line by line, so
// each ends one.
var lineBreaks = strings.NewReplacer("\r\n", "\n", "\r", "\n", "\f", "\n", "\v", "\n", "\u0085", "\n", "\u2028", "\n", "\u2029", "\n")

// NormalizeLineBreaks ends every line of text with a newline, whichever way it ended,
// so a text is read for a verdict by the lines ReadVerdict reads.
func NormalizeLineBreaks(text string) string { return lineBreaks.Replace(text) }

// ReadVerdict reads the verdict a return states of subject, the full name of what it
// judges: the head of a commit, or a draft's digest. It is the one rule every reader
// of a reviewer's return asks, and it holds a return to this:
//
//   - exactly one line opens a verdict (verdictLines), counted wherever it stands,
//     fenced or not, and that line is the exact Verdict heading;
//   - the first non-blank line under it begins with a verdict word (list marker and
//     emphasis stripped, its words joined by a space, a hyphen or an underscore, in
//     capitals) that ends the line or is followed by a separator;
//   - subject is named in full, on that line or on a `Head: <name>` line directly
//     under the heading, and no other name stands on either: a run of hex holding a
//     digit is another name, unless it is the number of a record id, all digits
//     and no more than twenty of them (recordid.IDNumberAt), as the digits of
//     iss-2611060835372141 are.
//
// subject is named in full because a head's name is a hash over the text it reviews,
// which therefore cannot hold it, and only the full name is. It returns the word as
// knownVerdicts spells it, and an error for anything short of the above.
func ReadVerdict(text, subject string) (string, error) {
	lines := strings.Split(NormalizeLineBreaks(text), "\n")
	heading, err := oneVerdictHeading(lines)
	if err != nil {
		return "", err
	}
	below := make([]string, 0, 2)
	for _, ln := range lines[heading+1:] {
		if t := strings.TrimSpace(ln); t != "" {
			if below = append(below, t); len(below) == 2 {
				break
			}
		}
	}
	if len(below) == 0 {
		return "", errors.New("its ### Verdict section states no verdict")
	}
	named := false
	line := below[0]
	if m := verdictNameLine.FindStringSubmatch(line); m != nil {
		if m[1] != subject {
			return "", namesOther(m[1], subject)
		}
		named = true
		if len(below) < 2 {
			return "", errors.New("its ### Verdict section names the head and states no verdict")
		}
		line = below[1]
	}
	word, rest, ok := verdictWordOf(line)
	if !ok {
		return "", fmt.Errorf("its verdict %s is not one a validator writes", clip(line))
	}
	abbreviated := false
	for _, at := range verdictNames.FindAllStringIndex(rest, -1) {
		switch name := rest[at[0]:at[1]]; {
		case name == subject:
			named = true
		case strings.HasPrefix(subject, name):
			abbreviated = true
		case recordid.IDNumberAt(rest, at[0], at[1]):
		case strings.ContainsAny(name, "0123456789"):
			return "", namesOther(name, subject)
		}
	}
	switch {
	case named:
		return word, nil
	case abbreviated:
		return "", fmt.Errorf("its verdict %s abbreviates what it judges; the line names it in full, %s", clip(line), subject)
	}
	return "", fmt.Errorf("its verdict %s names no head; the line under the ### Verdict heading opens with the verdict and names what it judges in full, %s", clip(line), subject)
}

// verdictNames is a name on a verdict line. A run of hex letters with no digit in it
// is a word unless it is the name itself, and a record id's number is that id's.
var verdictNames = regexp.MustCompile(`\b(?:sha256:)?[0-9a-f]{7,64}\b`)

// verdictNameLine is a labelled line naming what a verdict judges and nothing else,
// directly under the heading and above the verdict.
var verdictNameLine = regexp.MustCompile("^(?:[-*+][ \\t]+)?[*_`]*(?i:head)[*_`]*:[*_`]*[ \\t]*[*_`]*((?:sha256:)?[0-9a-f]{7,64})[*_`]*$")

// namesOther is the refusal of a verdict naming something other than subject.
func namesOther(name, subject string) error {
	if strings.HasPrefix(subject, name) {
		return fmt.Errorf("its verdict abbreviates what it judges to %s; it names it in full, %s", clip(name), subject)
	}
	return fmt.Errorf("its verdict names %s, and this review judges %s", clip(name), subject)
}

// verdictWordOf reads the verdict word line begins with, and what follows it.
func verdictWordOf(line string) (word, rest string, ok bool) {
	t := strings.TrimLeft(strings.TrimLeft(line, "-*+ \t"), "*_`")
	norm := strings.NewReplacer("-", " ", "_", " ").Replace(t)
	for _, w := range knownVerdicts {
		ww := strings.NewReplacer("-", " ").Replace(w)
		if strings.HasPrefix(norm, ww) && verdictEnds(t[len(ww):]) {
			return w, t[len(ww):], true
		}
	}
	return "", "", false
}

// verdictEnds reports whether what follows a verdict word on its line leaves it a
// verdict.
func verdictEnds(rest string) bool {
	return verdictEnd.MatchString(strings.TrimLeft(rest, "*_`"))
}

// oneVerdictHeading finds the exact Verdict heading in a return's lines, refusing a
// return with more than one line opening a verdict, naming a line to reword: the
// first that is not the exact heading.
func oneVerdictHeading(lines []string) (int, error) {
	opening := verdictLines(lines)
	switch len(opening) {
	case 0:
		return 0, errors.New("it has no ### Verdict section")
	case 1:
		if at := opening[0]; !verdictHeading.MatchString(lines[at]) {
			return 0, fmt.Errorf("it has no ### Verdict section, and the line %s stands in for one; a verdict is read under the exact heading", clip(strings.TrimSpace(lines[at])))
		}
		return opening[0], nil
	}
	line := lines[opening[1]]
	for _, at := range opening {
		if !verdictHeading.MatchString(lines[at]) {
			line = lines[at]
			break
		}
	}
	return 0, fmt.Errorf("it has %d Verdict sections; one states the verdict, so reword the line %s", len(opening), clip(strings.TrimSpace(line)))
}

// verdictInLine is the word Verdict within a line's first words: what leads the line,
// the words before it, and what follows it.
var verdictInLine = regexp.MustCompile(`(?i)^([^\pL\pN]*)((?:[\pL\pN][\pL\pN'\x{2019}_-]*[^\pL\pN]+){0,4}?)verdicts?([^\pL\pN].*)?$`)

// numberedItem is a numbered list marker, which is no word of the line it leads.
var numberedItem = regexp.MustCompile(`^([^\pL\pN]*)[0-9]+[.)][ \t]+`)

// lineLead is what leads a line and says nothing of it: a quote or a list marker.
var lineLead = regexp.MustCompile(`^[ \t>]*(?:[-*+][ \t]+)?`)

// beforeEmphasisCloses is what may stand between the word Verdict and the emphasis
// closing it: a bracketed note and a separator.
var beforeEmphasisCloses = regexp.MustCompile(`^(?:[ \t]*\([^)]*\))?[ \t]*[:=.]?[ \t]*`)

// setextUnderline is the line that makes the one above it a heading.
var setextUnderline = regexp.MustCompile(`^[ \t]*(?:=+|-{2,})[ \t]*$`)

// verdictStatement is what follows the word Verdict in a plain line, once its
// emphasis is off, that makes the line state one: a separator, or a verdict word,
// directly or after is. A bracketed note may stand before either.
var verdictStatement = regexp.MustCompile("^(?:\\([^)]*\\)[ \\t]*)?(?:[:=|\\x{2014}\\x{2013}]|-{1,2}[ \\t]|(?:(?:is|was|remains|stands)[ \\t:]+[*_`]*)?(?:SHIP|APPROVE|BLOCK|FIX[ _-]FIRST|NEEDS[ _-]INPUT)\\b)")

// verdictQualifier is a word that, before Verdict alone on a line, makes the line name
// one rather than end a sentence.
var verdictQualifier = regexp.MustCompile(`(?i)^(?:final|overall|my|our|the|last|review|reviewer'?s?|reviewers'?|security|ruthless|code|actual|formal)\b`)

// verdictLines is the index of every line opening a verdict, in the shapes a reviewer
// writes one. A line that begins with the word does, whatever follows it. With words
// before it (`Final`, `My`, `The overall`), a heading does, and so does a line whose
// emphasis wraps the word (emphasisWrapsWord), and any other line does when it goes on
// to state one (verdictStatement) or ends there after a qualifier: the word in a
// sentence opens nothing.
func verdictLines(lines []string) []int {
	var at []int
	for i, line := range lines {
		next := ""
		if i+1 < len(lines) {
			next = lines[i+1]
		}
		if opensVerdict(line, next) {
			at = append(at, i)
		}
	}
	return at
}

func opensVerdict(line, next string) bool {
	m := verdictInLine.FindStringSubmatch(numberedItem.ReplaceAllString(line, "$1"))
	if m == nil {
		return false
	}
	lead, qualifiers, rest := m[1], m[2], m[3]
	if qualifiers == "" {
		return true
	}
	marks := lineLead.ReplaceAllString(lead, "")
	if strings.Contains(marks, "#") || setextUnderline.MatchString(next) || emphasisWrapsWord(marks, qualifiers, rest) {
		return true
	}
	rest = strings.TrimLeft(rest, "*_` \t")
	if rest == "" {
		return verdictQualifier.MatchString(qualifiers)
	}
	return verdictStatement.MatchString(rest)
}

// emphasisWrapsWord reports whether emphasis opened in what leads a line stays open
// up to the word Verdict and closes directly after it, a bracketed note or a separator
// aside: `**Final verdict:**`. Emphasis closed before the word, as a finding's label
// is (`*Failure scenario:* a verdict ...`), or still running past it, wraps a sentence.
func emphasisWrapsWord(marks, qualifiers, rest string) bool {
	after := beforeEmphasisCloses.ReplaceAllString(rest, "")
	for _, mark := range []string{"*", "_"} {
		if strings.Contains(marks, mark) && !strings.Contains(qualifiers, mark) && strings.HasPrefix(after, mark) {
			return true
		}
	}
	return false
}

// verdictRule is the reading of a verdict a receipt records itself under: 2 holds a
// return's verdict to naming the head it judges (ReadVerdict), and a receipt with no
// rule was read before it did. A receipt keeps its returns' digests and not their
// text, so its verdicts cannot be read again: the merge gate and a delta's base count
// only a receipt of this rule, whichever build of abcd wrote the file, and the head
// records its pair again. A receipt binds one head, in one machine's local tier, so
// that costs only the heads in flight as this rule arrives.
const verdictRule = 2

// reviewRecord is the receipt's content.
type reviewRecord struct {
	Commit      string              `json:"commit"`
	VerdictRule int                 `json:"verdict_rule"`
	RecordedAt  string              `json:"recorded_at"`
	Reviews     []reviewRecordEntry `json:"reviews"`
	Delta       *deltaRecord        `json:"delta,omitempty"`
}

// readUnderThisRule reports whether a receipt's verdicts were read as ReadVerdict
// reads one.
func readUnderThisRule(rec reviewRecord) bool { return rec.VerdictRule == verdictRule }

// recordAgain is what the gate and a delta say of a receipt read under an earlier
// rule.
const recordAgain = "its receipt was recorded before a verdict named the head it judges, so record the pair again (abcd land --record-review), each return's verdict naming the head"

type reviewRecordEntry struct {
	Name    string `json:"name"`
	Verdict string `json:"verdict"`
	SHA256  string `json:"sha256"`
}

// RecordReview writes the review receipt for the commit sha in the local tier
// of the checkout at repoRoot, once the review pair's returns pass the stop
// rule: at least two returns, no two the same text, each stating the commit on
// its first line (statesHead) and each stating a passing verdict that names the
// commit (ReadVerdict). Anything short of that is refused, naming the return, and
// nothing is written. sha must be the full object name of a commit the repository
// holds. A return that states a delta is refused: it is a delta's
// (RecordDeltaReview).
func RecordReview(repoRoot, sha string, returns []ReviewReturn) (string, error) {
	if err := holdsCommit(repoRoot, sha); err != nil {
		return "", err
	}
	entries, err := judgeReturns(returns, sha, func(text string) error { return statesFullPair(text, sha) })
	if err != nil {
		return "", err
	}
	return writeReviewRecord(repoRoot, newReviewRecord(sha, entries, nil))
}

// holdsCommit reports why sha is not the full object name of a commit the
// repository holds.
func holdsCommit(repoRoot, sha string) error {
	if !fullSHA.MatchString(sha) {
		return fmt.Errorf("%s is not a commit's full object name", clip(sha))
	}
	if _, err := gitutil.Run(repoRoot, "cat-file", "-e", sha+"^{commit}"); err != nil {
		return fmt.Errorf("%s is not a commit this repository holds", sha)
	}
	return nil
}

// judgeReturns applies the stop rule to a review pair's returns on head and gives
// each one's entry in the receipt. states is the rule binding a return to what it
// read; its refusal names the return.
func judgeReturns(returns []ReviewReturn, head string, states func(text string) error) ([]reviewRecordEntry, error) {
	if len(returns) < minReviewers {
		return nil, fmt.Errorf("a review receipt needs the review pair's returns, at least two; %d given", len(returns))
	}
	var entries []reviewRecordEntry
	seen := map[string]string{}
	for _, r := range returns {
		name := clip(r.Name) // as a refusal echoes it
		if len(r.Text) > MaxReviewReturnBytes {
			return nil, fmt.Errorf("the return %s is larger than %d bytes", name, MaxReviewReturnBytes)
		}
		if err := states(r.Text); err != nil {
			return nil, fmt.Errorf("the return %s: %v", name, err)
		}
		v, err := ReadVerdict(r.Text, head)
		if err != nil {
			return nil, fmt.Errorf("the return %s: %v", name, err)
		}
		if !passingVerdicts[v] {
			return nil, fmt.Errorf("the return %s says %s, which does not pass the stop rule (no critical or major finding)", name, v)
		}
		sum := sha256.Sum256([]byte(r.Text))
		digest := hex.EncodeToString(sum[:])
		if first, ok := seen[digest]; ok {
			return nil, fmt.Errorf("the return %s is the same text as %s, so it is one review given twice, not a second reviewer's", name, first)
		}
		seen[digest] = name
		entries = append(entries, reviewRecordEntry{Name: r.Name, Verdict: v, SHA256: digest})
	}
	return entries, nil
}

func newReviewRecord(sha string, entries []reviewRecordEntry, delta *deltaRecord) reviewRecord {
	return reviewRecord{Commit: sha, VerdictRule: verdictRule, RecordedAt: time.Now().UTC().Format(time.RFC3339), Reviews: entries, Delta: delta}
}

// writeReviewRecord writes the receipt in the checkout's local tier.
func writeReviewRecord(repoRoot string, rec reviewRecord) (string, error) {
	tier := filepath.Join(repoRoot, filepath.FromSlash(TierRelDir))
	if fi, err := os.Lstat(tier); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%s/ is not a directory in this checkout, so there is nowhere to keep the receipt", TierRelDir)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", err
	}
	// The receipt is written through the checkout's root, so a symlink a
	// branch commits on its path cannot carry the write out of the checkout.
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.MkdirAll(ReviewReceiptsRelDir, 0o700); err != nil {
		return "", fmt.Errorf("the review receipts directory cannot be made inside this checkout: %w", err)
	}
	rel := ReviewReceiptsRelDir + "/" + rec.Commit
	if err := fsutil.WriteFileAtomicInRoot(root, rel, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("the review receipt cannot be written inside this checkout: %w", err)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(rel)), nil
}

// reviewReceipt finds the review receipt for sha in any worktree git lists
// for the repository at repoRoot, as Receipt finds a preflight receipt, and
// returns its path, or "" when none names sha.
func reviewReceipt(repoRoot, sha string) (string, error) {
	return findReceipt(repoRoot, ReviewReceiptsRelDir, sha)
}

// clip bounds and quotes a value a refusal echoes, cut on a character
// boundary.
func clip(s string) string {
	const n = 80
	if len(s) > n {
		i := n
		for i > 0 && !utf8.RuneStart(s[i]) {
			i--
		}
		s = s[:i] + "…"
	}
	return fmt.Sprintf("%q", s)
}
