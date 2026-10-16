package loop

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The ledger batch's bound (spc-2610161044461448 step 1 follow-up): the open
// records a move reads from origin go through one `git cat-file --batch`, so
// what that batch may answer is bounded by the distinct objects it reads,
// each once, and refused past a fixed ceiling.

// gitStdin runs git in dir with stdin, under the hermetic test environment.
func gitStdin(t *testing.T, dir, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// ledgerCommit is a commit in dir whose .abcd/work/issues/open/ holds one
// entry per blob given, named iss-<n>-r.md, and nothing else.
func ledgerCommit(t *testing.T, dir string, blobs []string) originView {
	t.Helper()
	var open strings.Builder
	for i, b := range blobs {
		fmt.Fprintf(&open, "100644 blob %s\tiss-26093000%08d-r.md\n", b, i+1)
	}
	tree := gitStdin(t, dir, open.String(), "mktree")
	for _, name := range []string{"open", "issues", "work", ".abcd"} {
		tree = gitStdin(t, dir, "040000 tree "+tree+"\t"+name+"\n", "mktree")
	}
	commit := gitStdin(t, dir, "", "commit-tree", tree, "-m", "a ledger")
	return originView{ref: "refs/remotes/origin/main", commit: commit}
}

// recordBlob writes a record just under the read limit, told apart by n.
func recordBlob(t *testing.T, dir string, n int) (sha, text string) {
	t.Helper()
	head := fmt.Sprintf("---\nid: iss-%d\n---\n", n)
	text = head + strings.Repeat("x", int(issueschema.RecordReadLimit)-len(head)-1)
	return gitStdin(t, dir, text, "hash-object", "-w", "--stdin"), text
}

// TestTheLedgerBatchReadsABlobManyRecordsShareOnce: seventy open records
// naming one blob just under the read limit would answer past the batch's
// ceiling were each read on its own; the blob is read once, and every record
// carries its text.
func TestTheLedgerBatchReadsABlobManyRecordsShareOnce(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	sha, text := recordBlob(t, repo.Root(), 1)
	blobs := make([]string, 70)
	for i := range blobs {
		blobs[i] = sha
	}
	if int64(len(blobs))*issueschema.RecordReadLimit <= maxLedgerBatch {
		t.Fatalf("the fixture does not pass the ceiling read entry by entry")
	}
	l, err := treeLedger(repo.Root(), ledgerCommit(t, repo.Root(), blobs))
	if err != nil {
		t.Fatalf("records sharing one blob are read once, under the ceiling: %v", err)
	}
	if len(l.Files) != len(blobs) {
		t.Fatalf("the ledger lists %d files; want %d", len(l.Files), len(blobs))
	}
	for _, f := range l.Files {
		if f.Unread != "" || f.Text != strings.TrimRight(text, "\n") {
			t.Fatalf("%s: unread %q, %d bytes of text; want the shared record", f.Path, f.Unread, len(f.Text))
		}
	}
}

// TestTheLedgerBatchRefusesDistinctRecordsPastItsCeiling: distinct open
// records whose sizes together pass the ceiling refuse the read, naming the
// ceiling, rather than buffering them.
func TestTheLedgerBatchRefusesDistinctRecordsPastItsCeiling(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	n := int(maxLedgerBatch/issueschema.RecordReadLimit) + 1
	blobs := make([]string, n)
	for i := range blobs {
		blobs[i], _ = recordBlob(t, repo.Root(), i)
	}
	_, err := treeLedger(repo.Root(), ledgerCommit(t, repo.Root(), blobs))
	if err == nil || !strings.Contains(err.Error(), "past the 64 MiB") {
		t.Fatalf("distinct records past the ceiling: %v; want a refusal naming it", err)
	}
}
