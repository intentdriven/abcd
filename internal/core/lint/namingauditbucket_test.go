package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingAuditBucketCoversTheConsistencyPass (iss-2610262030494155): the
// register's closed bucket vocabulary once said `audit` compares reality
// against a recorded commitment, while the intent chapter's sub-verb table
// files `consistency`, which compares one recorded document against another
// with no delivered reality involved, under `audit`. The bucket row's audit
// clause must admit that comparison, and the table must still file the pass
// there, so the two ends agree.
func TestNamingAuditBucketCoversTheConsistencyPass(t *testing.T) {
	brief := filepath.Join(repoRootFromPackage, ".abcd", "development", "brief")
	naming, err := os.ReadFile(filepath.Join(brief, "02-constraints", "04-naming.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(naming), "\n") {
		if strings.HasPrefix(line, "| `bucket` ∈ `{lint, review, audit, gate}`") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("04-naming.md carries no `bucket` row")
	}
	at := strings.Index(row, "`audit` compares")
	if at < 0 {
		t.Fatalf("the bucket row defines no `audit` comparison: %q", row)
	}
	clause := row[at:]
	if end := strings.Index(clause, ";"); end >= 0 {
		clause = clause[:end]
	}
	for _, want := range []string{"reality against a recorded commitment", "one recorded commitment against another", "consistency pass"} {
		if !strings.Contains(clause, want) {
			t.Errorf("the bucket row's audit clause does not say %q: %q", want, clause)
		}
	}

	intent, err := os.ReadFile(filepath.Join(brief, "04-surfaces", "05-intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(intent), "| `consistency` | audit |") {
		t.Error("05-intent.md no longer files `consistency` under the audit bucket; the register's widening has lost its reason")
	}
}
