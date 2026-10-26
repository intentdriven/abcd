package issuemention

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// prCorpus is every shape RS004 decides a pull request on: a mention declared
// in the body, in the title, or not at all; both spellings and the near misses
// the vocabulary closes; id lists, a mixed-family list, word boundaries, CRLF,
// and a fenced mention, which is a mention all the same.
var prCorpus = map[string]struct{ title, body string }{
	"no mention":               {"fix the walk", "Summary\n"},
	"declared resolves":        {"fix the walk", "Fixes it.\n\nResolves: iss-12\n"},
	"declared refs":            {"touch iss-12", "Refs: iss-12\n"},
	"undeclared in body":       {"fix", "See iss-12.\n"},
	"undeclared in title":      {"fix iss-12", "Summary\n"},
	"declared by the title":    {"Refs: iss-12", "See iss-12.\n"},
	"id list":                  {"fix", "iss-1 and iss-2\n\nRefs: iss-1, iss-2\n"},
	"id list missing one":      {"fix", "iss-1 and iss-2 and iss-3\n\nRefs: iss-1, iss-2\n"},
	"mixed family":             {"fix", "iss-9 and itd-146\n\nRefs: itd-146, iss-9\n"},
	"near miss ref":            {"fix", "Ref: iss-12\n"},
	"near miss see":            {"fix", "See: iss-12\n"},
	"lower-case keyword":       {"fix", "refs: iss-12\n"},
	"keyword not alone":        {"fix", "Refs: iss-12 and more\n"},
	"trailing space":           {"fix", "Refs: iss-12  \n"},
	"tab after colon":          {"fix", "Refs:\tiss-12\n"},
	"no space after colon":     {"fix", "Refs:iss-12\n"},
	"indented declaration":     {"fix", "  Refs: iss-12\n"},
	"word boundary":            {"fix", "xiss-12 and iss-12x\n"},
	"longer id is not shorter": {"fix", "iss-123\n\nRefs: iss-12\n"},
	"crlf":                     {"fix", "See iss-12.\r\n\r\nRefs: iss-12\r\n"},
	"fenced mention":           {"fix", "```\niss-12\n```\n"},
	"fenced declaration":       {"fix", "```\nRefs: iss-12\n```\n"},
	"resolves and refs":        {"fix", "iss-1 iss-2\n\nResolves: iss-1\nRefs: iss-2\n"},
	"empty":                    {"", ""},
	"declaration on two lines": {"fix", "Refs:\niss-12\n"},
	"after punctuation":        {"fix", "(iss-12), [iss-13]\n\nRefs: iss-12\n"},
}

// TestUndeclaredMatchesTheGate ties Undeclared to RS004's pull-request arm in
// scripts/check-issue-resolution.sh, the one place the rule is decided: the gate
// runs in CI without Go, so the two share a test rather than code. Every pull
// request in the corpus goes through the gate itself, and Undeclared must find
// an undeclared id in exactly the ones the gate refuses.
func TestUndeclaredMatchesTheGate(t *testing.T) {
	gate, err := filepath.Abs(filepath.Join("..", "..", "..", "scripts", "check-issue-resolution.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// The gate refuses to run outside a full git repository; the pull-request
	// arm reads only the two files, so an empty one is enough.
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for name, pr := range prCorpus {
		t.Run(name, func(t *testing.T) {
			title, body := filepath.Join(repo, "title.txt"), filepath.Join(repo, "body.md")
			if err := os.WriteFile(title, []byte(pr.title), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(body, []byte(pr.body), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", gate, "pr", title, body)
			cmd.Dir = repo
			out, err := cmd.CombinedOutput()
			gateAccepts := err == nil
			if ee, ok := err.(*exec.ExitError); err != nil && (!ok || ee.ExitCode() != 1) {
				t.Fatalf("the gate could not judge the pull request: %v\n%s", err, out)
			}
			undeclared := Undeclared(pr.title, pr.body)
			if got := len(undeclared) == 0; got != gateAccepts {
				t.Fatalf("Undeclared(%q, %q) = %v, but the gate %s it:\n%s", pr.title, pr.body, undeclared,
					map[bool]string{true: "accepts", false: "refuses"}[gateAccepts], out)
			}
		})
	}
}

func TestUndeclaredNamesEachIdOnceInOrder(t *testing.T) {
	got := Undeclared("fix iss-3 and iss-1", "iss-1, iss-2, iss-3 again\n\nRefs: iss-2\n")
	if want := []string{"iss-3", "iss-1"}; !slices.Equal(got, want) {
		t.Fatalf("Undeclared = %v, want %v", got, want)
	}
}

func TestDeclaredResolvesOutranksRefs(t *testing.T) {
	got := Declared("Refs: iss-1\nResolves: iss-1, iss-2\nRefs: iss-2\nRefs: iss-3\n")
	want := map[string]string{"iss-1": "Resolves", "iss-2": "Resolves", "iss-3": "Refs"}
	if len(got) != len(want) {
		t.Fatalf("Declared = %v, want %v", got, want)
	}
	for id, kw := range want {
		if got[id] != kw {
			t.Errorf("Declared[%s] = %q, want %q", id, got[id], kw)
		}
	}
}
