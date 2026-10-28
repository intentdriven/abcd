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
// Nothing reads a review receipt yet: the merge gate reading one in auto
// mode is the spec's next step.
const ReviewReceiptsRelDir = TierRelDir + "/review-receipts"

// MaxReviewReturnBytes bounds one review return read for a receipt.
const MaxReviewReturnBytes = 1 << 20

// minReviewers is the review pair: a receipt needs at least two returns.
const minReviewers = 2

// ReviewReturn is one reviewer's return: the name it is recorded under and its
// text, which names the commit it read and ends in a `### Verdict` section.
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

// verdictHeading is a return's Verdict heading at the start of a line, at any
// depth, as the implement loop reads one; text indented as code, or a heading
// with more after it, is not one.
var verdictHeading = regexp.MustCompile(`(?im)^[ \t]{0,3}#{1,6}[ \t]+verdict[ \t]*#*[ \t]*$`)

// reviewedCommitPrefix is how much of the commit's name a return must carry:
// the nine characters the build loop's briefs abbreviate a head to.
const reviewedCommitPrefix = 9

// reviewVerdict reads the verdict a return states, by the implement loop's
// rule: the return has exactly one Verdict heading, and the first non-blank
// line under it begins with a verdict word (list marker and emphasis
// stripped, its words joined by a space, a hyphen or an underscore) followed
// by anything but a letter. A return with no Verdict heading, more than one,
// or a first line beginning with no verdict word is an error.
func reviewVerdict(text string) (string, error) {
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
	norm := strings.ToUpper(strings.NewReplacer("-", " ", "_", " ").Replace(t))
	for _, w := range knownVerdicts {
		ww := strings.NewReplacer("-", " ").Replace(w)
		if rest, ok := strings.CutPrefix(norm, ww); ok && (rest == "" || !isLetter(rest[0])) {
			return w, nil
		}
	}
	return "", fmt.Errorf("its verdict %s is not one a validator writes", clip(line))
}

func isLetter(b byte) bool { return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }

// reviewRecord is the receipt's content.
type reviewRecord struct {
	Commit     string              `json:"commit"`
	RecordedAt string              `json:"recorded_at"`
	Reviews    []reviewRecordEntry `json:"reviews"`
}

type reviewRecordEntry struct {
	Name    string `json:"name"`
	Verdict string `json:"verdict"`
	SHA256  string `json:"sha256"`
}

// RecordReview writes the review receipt for the commit sha in the local tier
// of the checkout at repoRoot, once the review pair's returns pass the stop
// rule: at least two returns, no two the same text, each naming the commit
// (its first nine characters) and each stating a passing verdict. Anything
// short of that is refused, naming the return, and nothing is written. sha
// must be the full object name of a commit the repository holds.
func RecordReview(repoRoot, sha string, returns []ReviewReturn) (string, error) {
	if !fullSHA.MatchString(sha) {
		return "", fmt.Errorf("%s is not a commit's full object name", clip(sha))
	}
	if _, err := gitutil.Run(repoRoot, "cat-file", "-e", sha+"^{commit}"); err != nil {
		return "", fmt.Errorf("%s is not a commit this repository holds", sha)
	}
	if len(returns) < minReviewers {
		return "", fmt.Errorf("a review receipt needs the review pair's returns, at least two; %d given", len(returns))
	}
	rec := reviewRecord{Commit: sha, RecordedAt: time.Now().UTC().Format(time.RFC3339)}
	seen := map[string]string{}
	for _, r := range returns {
		name := clip(r.Name) // as a refusal echoes it
		if len(r.Text) > MaxReviewReturnBytes {
			return "", fmt.Errorf("the return %s is larger than %d bytes", name, MaxReviewReturnBytes)
		}
		if !strings.Contains(r.Text, sha[:reviewedCommitPrefix]) {
			return "", fmt.Errorf("the return %s does not name the commit %s, so it cannot be told it reviewed this head", name, sha[:reviewedCommitPrefix])
		}
		v, err := reviewVerdict(r.Text)
		if err != nil {
			return "", fmt.Errorf("the return %s: %v", name, err)
		}
		if !passingVerdicts[v] {
			return "", fmt.Errorf("the return %s says %s, which does not pass the stop rule (no critical or major finding)", name, v)
		}
		sum := sha256.Sum256([]byte(r.Text))
		digest := hex.EncodeToString(sum[:])
		if first, ok := seen[digest]; ok {
			return "", fmt.Errorf("the return %s is the same text as %s, so it is one review given twice, not a second reviewer's", name, first)
		}
		seen[digest] = name
		rec.Reviews = append(rec.Reviews, reviewRecordEntry{Name: r.Name, Verdict: v, SHA256: digest})
	}
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
	rel := ReviewReceiptsRelDir + "/" + sha
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
