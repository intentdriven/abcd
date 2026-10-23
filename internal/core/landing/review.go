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

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ReviewReceiptsRelDir holds one review receipt per commit, named by its full
// object name, beside the preflight receipts (spc-2610161609350616 step 3).
// In auto mode the merge gate arms only a head a review receipt names.
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

// knownVerdicts are every verdict a validator writes.
var knownVerdicts = map[string]bool{"SHIP": true, "FIX FIRST": true, "APPROVE": true, "BLOCK": true, "NEEDS-INPUT": true}

var fullSHA = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)

// reviewedCommitPrefix is how much of the commit's name a return must carry:
// the nine characters the build loop's briefs abbreviate a head to.
const reviewedCommitPrefix = 9

// reviewVerdict reads the verdict a return states: the first non-blank line
// after its last `### Verdict` heading, emphasis and surrounding space
// removed, upper-cased. A return with no such section, or one stating a word
// no validator writes, is an error.
func reviewVerdict(text string) (string, error) {
	i := strings.LastIndex(text, "### Verdict")
	if i < 0 {
		return "", errors.New("it has no ### Verdict section")
	}
	for _, line := range strings.Split(text[i+len("### Verdict"):], "\n") {
		v := strings.ToUpper(strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "*_")))
		if v == "" {
			continue
		}
		if !knownVerdicts[v] {
			return "", fmt.Errorf("its verdict %q is not one a validator writes", clip(v))
		}
		return v, nil
	}
	return "", errors.New("its ### Verdict section states no verdict")
}

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
// rule: at least two returns, each naming the commit (its first nine
// characters) and each stating a passing verdict. Anything short of that is
// refused, naming the return, and nothing is written. sha must be the full
// object name of a commit the repository holds.
func RecordReview(repoRoot, sha string, returns []ReviewReturn) (string, error) {
	if !fullSHA.MatchString(sha) {
		return "", fmt.Errorf("%q is not a commit's full object name", clip(sha))
	}
	if _, err := gitutil.Run(repoRoot, "cat-file", "-e", sha+"^{commit}"); err != nil {
		return "", fmt.Errorf("%s is not a commit this repository holds", sha)
	}
	if len(returns) < minReviewers {
		return "", fmt.Errorf("a review receipt needs the review pair's returns, at least two; %d given", len(returns))
	}
	rec := reviewRecord{Commit: sha, RecordedAt: time.Now().UTC().Format(time.RFC3339)}
	for _, r := range returns {
		name := clip(r.Name)
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
		rec.Reviews = append(rec.Reviews, reviewRecordEntry{Name: name, Verdict: v, SHA256: hex.EncodeToString(sum[:])})
	}
	tier := filepath.Join(repoRoot, filepath.FromSlash(TierRelDir))
	if fi, err := os.Lstat(tier); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%s/ is not a directory in this checkout, so there is nowhere to keep the receipt", TierRelDir)
	}
	dir := filepath.Join(repoRoot, filepath.FromSlash(ReviewReceiptsRelDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, sha)
	if err := fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// reviewReceipt finds the review receipt for sha in any worktree git lists
// for the repository at repoRoot, as Receipt finds a preflight receipt, and
// returns its path, or "" when none names sha.
func reviewReceipt(repoRoot, sha string) (string, error) {
	return findReceipt(repoRoot, ReviewReceiptsRelDir, sha)
}

// clip bounds and quotes a value a refusal echoes.
func clip(s string) string {
	const n = 80
	if len(s) > n {
		s = s[:n] + "…"
	}
	return fmt.Sprintf("%q", s)
}
