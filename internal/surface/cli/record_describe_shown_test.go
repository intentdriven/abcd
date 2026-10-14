package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDescribeAnIssueMasksAStoredRuneOnBothPayloads: `abcd iss-N` reads the
// issue through capture.List, which returns the text as stored, so the
// describe masks what it carries. A hand-edited body whose title line holds
// U+202E and a tag rune reaches neither the text answer nor `--json` raw.
func TestDescribeAnIssueMasksAStoredRuneOnBothPayloads(t *testing.T) {
	repo := captureLedgerRepo(t)
	runCLI(t, "capture", "the empty list panics", "--severity", "minor", "--remedy", "Print a dash.", "--json")
	matches, err := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", "*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the captured record: %v %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(data), "the empty list panics", "the empty‮ list\U000E0041 panics", 1)
	if rewritten == string(data) {
		t.Fatalf("the record carries the title line: %s", data)
	}
	if err := os.WriteFile(matches[0], []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}
	id := strings.SplitN(filepath.Base(matches[0]), "-", 3)
	issID := id[0] + "-" + id[1]
	for _, args := range [][]string{{issID, "--json"}, {issID}} {
		out := string(runCLI(t, args...))
		for _, r := range []string{"‮", "\U000E0041", `‮`} {
			if strings.Contains(out, r) {
				t.Errorf("%v: the stored rune %q reaches the answer raw: %s", args, r, out)
			}
		}
		if !strings.Contains(out, "the empty? list? panics") {
			t.Errorf("%v: the title is kept, masked: %s", args, out)
		}
	}
}
