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
// `### Verdict` heading.
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
// the word is off: a commit's name, and then the end of the line or a separator.
// Anything else makes the word the start of another statement.
var verdictEnd = regexp.MustCompile(`^(?:[ \t]+[0-9a-f]{7,64})?[*_]*[ \t]*(?:$|[\x{2014}\x{2013}(]|[:;,.](?:[ \t]|$)|-{1,2}[ \t])`)

// verdictHeading is a return's Verdict heading at the start of a line, at any
// depth, as the implement loop reads one; text indented as code, or a heading
// with more after it, is not one.
var verdictHeading = regexp.MustCompile(`(?im)^[ \t]{0,3}#{1,6}[ \t]+verdict[ \t]*#*[ \t]*$`)

// reviewedCommitPrefix is how much of the commit's name a return must carry:
// the nine characters the build loop's briefs abbreviate a head to.
const reviewedCommitPrefix = 9

// reviewVerdict reads the verdict a return states, by the implement loop's rule: the
// return has exactly one line opening as a Verdict (OneVerdictLine), counted wherever
// it stands, fenced or not; that line is the exact Verdict heading; and the first
// non-blank line under it begins with a verdict word (list marker and
// emphasis stripped, its words joined by a space, a hyphen or an underscore, in
// capitals) that ends the line or is followed by a separator (VerdictEnds). A return
// with no Verdict heading, more than one, or a first line beginning with no verdict
// word is an error.
func reviewVerdict(text string) (string, error) {
	if err := OneVerdictLine(text); err != nil {
		return "", err
	}
	heads := verdictHeading.FindAllStringIndex(text, -1)
	switch len(heads) {
	case 0:
		return "", errors.New("it has no ### Verdict section")
	case 1:
	default:
		return "", fmt.Errorf("it has %d Verdict sections; one states the verdict", len(heads))
	}
	line := ""
	for _, ln := range strings.Split(text[heads[0][1]:], "\n") {
		if line = strings.TrimSpace(ln); line != "" {
			break
		}
	}
	if line == "" {
		return "", errors.New("its ### Verdict section states no verdict")
	}
	t := strings.TrimLeft(strings.TrimLeft(line, "-*+ \t"), "*_`")
	norm := strings.NewReplacer("-", " ", "_", " ").Replace(t)
	for _, w := range knownVerdicts {
		ww := strings.NewReplacer("-", " ").Replace(w)
		if strings.HasPrefix(norm, ww) && VerdictEnds(t[len(ww):]) {
			return w, nil
		}
	}
	return "", fmt.Errorf("its verdict %s is not one a validator writes", clip(line))
}

// verdictOpening is any line whose first letter or digit starts the word Verdict, in
// any case, whatever leads it or follows it.
var verdictOpening = regexp.MustCompile(`(?mi)^[^\pL\pN\n]*verdict\b.*$`)

// OneVerdictLine reports a return with more than one line opening as a Verdict,
// naming a line to reword: the first that is not the exact heading. The implement
// loop's reader and this one both ask it, since only the exact heading is read for a
// verdict word, and any other such line may state another.
func OneVerdictLine(text string) error {
	lines := verdictOpening.FindAllString(text, -1)
	if len(lines) <= 1 {
		return nil
	}
	line := lines[1]
	for _, l := range lines {
		if !verdictHeading.MatchString(l) {
			line = l
			break
		}
	}
	return fmt.Errorf("it has %d Verdict sections; one states the verdict, so reword the line %s", len(lines), clip(strings.TrimSpace(line)))
}

// VerdictEnds reports whether what follows a verdict word on its line leaves it a
// verdict: the implement loop's reader and this one share the rule.
func VerdictEnds(rest string) bool {
	return verdictEnd.MatchString(strings.TrimLeft(rest, "*_`"))
}

// reviewRecord is the receipt's content.
type reviewRecord struct {
	Commit     string              `json:"commit"`
	RecordedAt string              `json:"recorded_at"`
	Reviews    []reviewRecordEntry `json:"reviews"`
	Delta      *deltaRecord        `json:"delta,omitempty"`
}

type reviewRecordEntry struct {
	Name    string `json:"name"`
	Verdict string `json:"verdict"`
	SHA256  string `json:"sha256"`
}

// RecordReview writes the review receipt for the commit sha in the local tier
// of the checkout at repoRoot, once the review pair's returns pass the stop
// rule: at least two returns, no two the same text, each stating the commit on
// its first line (statesHead) and each stating a passing verdict. Anything
// short of that is refused, naming the return, and nothing is written. sha
// must be the full object name of a commit the repository holds. A return that
// states a delta is refused: it is a delta's (RecordDeltaReview).
func RecordReview(repoRoot, sha string, returns []ReviewReturn) (string, error) {
	if err := holdsCommit(repoRoot, sha); err != nil {
		return "", err
	}
	entries, err := judgeReturns(returns, func(text string) error { return statesFullPair(text, sha) })
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

// judgeReturns applies the stop rule to a review pair's returns and gives each
// one's entry in the receipt. states is the rule binding a return to what it
// read; its refusal names the return.
func judgeReturns(returns []ReviewReturn, states func(text string) error) ([]reviewRecordEntry, error) {
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
		v, err := reviewVerdict(r.Text)
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
	return reviewRecord{Commit: sha, RecordedAt: time.Now().UTC().Format(time.RFC3339), Reviews: entries, Delta: delta}
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
