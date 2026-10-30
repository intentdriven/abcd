package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// R3-F6: the loop's stub forge client refuses a --json field the real one
// refuses (gh's own lists, internal/ghcontract), so a field gh has no such
// name for cannot pass the loop's tests and fail every real landing.
func TestTheStubForgeClientRefusesAFieldGhRefuses(t *testing.T) {
	gh := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(gh, []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		ok   bool
	}{
		{[]string{"pr", "view", "7", "--json", "mergeable,mergeStateStatus"}, true},
		{[]string{"pr", "list", "--head", "feat/x", "--state", "open", "--json", "number,url", "--limit", "2"}, true},
		{[]string{"pr", "view", "7", "--json", "mergeable,isInMergeQueue"}, false},
		{[]string{"pr", "list", "--json=number,isInMergeQueue"}, false},
	} {
		cmd := exec.Command(gh, c.args...)
		cmd.Dir = filepath.Dir(gh)
		out, err := cmd.CombinedOutput()
		if (err == nil) != c.ok {
			t.Fatalf("stub gh %v = %q, %v; want ok %v", c.args, out, err, c.ok)
		}
	}
}

// The stub answers the merge gate's REST reads only as a plain GET, as the
// real forge client does with no method or field: a call carrying either is
// a write the gate never makes.
func TestTheStubForgeClientAnswersOnlyAGet(t *testing.T) {
	gh := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(gh, []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"api", "-X", "POST", "repos/{owner}/{repo}/pulls/7/reviews?per_page=100"},
		{"api", "repos/{owner}/{repo}/pulls/7/reviews?per_page=100", "-f", "event=APPROVE"},
	} {
		cmd := exec.Command(gh, args...)
		cmd.Dir = filepath.Dir(gh)
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Fatalf("stub %v answered a non-GET: %s", args, out)
		}
	}
}
