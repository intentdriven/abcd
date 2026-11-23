package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAFailedRecordsRoundIsQuotedOnlyAsItsVerdictWasRecorded: the note on a
// round that failed the landing's records quotes its first finding from the
// return as the loop recorded the verdict from it; a return changed since
// quotes nothing, while the verdict is still named.
func TestAFailedRecordsRoundIsQuotedOnlyAsItsVerdictWasRecorded(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/.work.local/run/run-1/lane-1/validate/round-2/ruthless-reviewer/return.md"
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	recorded := "Reviewed head: x\n\n### Findings\n\n- the finding the reviewer wrote\n\n### Verdict\n\nFIX FIRST\n"
	if err := os.WriteFile(path, []byte(recorded), 0o600); err != nil {
		t.Fatal(err)
	}
	const records = "0123456789abcdef0123456789abcdef01234567"
	lane := Lane{ID: "lane-1", Landing: &Landing{Records: records}, Validation: []ValidationRound{{Round: 2, HeadSHA: records,
		Validators: []ValidatorRun{{Role: RoleRuthless, Return: rel, Verdict: "FIX FIRST", ReturnSHA256: answerDigest([]byte(recorded))}}}}}
	c := Context{RepoRoot: root}
	if got := failedRecordsRound(c, lane); !strings.Contains(got, "the finding the reviewer wrote") {
		t.Fatalf("the note quotes the recorded return's finding: %q", got)
	}
	changed := strings.Replace(recorded, "the finding the reviewer wrote", "words the reviewer never wrote", 1)
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	got := failedRecordsRound(c, lane)
	if strings.Contains(got, "never wrote") || !strings.Contains(got, "FIX FIRST") {
		t.Fatalf("a changed return is not quoted, and the verdict is still named: %q", got)
	}
}
