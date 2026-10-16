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
