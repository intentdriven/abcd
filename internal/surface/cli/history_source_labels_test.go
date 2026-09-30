package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// historySourceRepo is a one-commit repository with HOME pointed at a temp dir,
// so the transcript store the CLI resolves is the test's own.
func historySourceRepo(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitCmd(t, repo, "init")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", ".")
	gitCommit(t, repo, "commit", "-m", "init")
	t.Chdir(repo)
}

// TestHistoryCaptureNamesToolAndRoute is ruling J13 at the front door: the
// caller names the route with --kind and the producing tool with --tool, the
// JSON envelope carries both labels, and the text renders of capture, list and
// show name both.
func TestHistoryCaptureNamesToolAndRoute(t *testing.T) {
	historySourceRepo(t)

	out := runCLIStdin(t, "user: exported elsewhere\n",
		"history", "capture", "--session", "sess-imp", "--kind", "import", "--tool", "other-harness", "--json")
	var res struct {
		Record struct {
			SourceKind string `json:"source_kind"`
			SourceTool string `json:"source_tool"`
		} `json:"record"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if res.Record.SourceKind != "import" || res.Record.SourceTool != "other-harness" {
		t.Errorf("envelope labels = %q / %q, want import / other-harness", res.Record.SourceKind, res.Record.SourceTool)
	}

	list := string(runCLI(t, "history", "list"))
	if !strings.Contains(list, "import (other-harness)") {
		t.Errorf("list does not name the route and the tool:\n%s", list)
	}
	show := string(runCLI(t, "history", "show", "sess-imp"))
	if !strings.Contains(show, "source:     import (other-harness)") {
		t.Errorf("show does not name the route and the tool:\n%s", show)
	}
}

// TestHistoryCaptureRefusesAForgedLabel: a tool name offered as the route, or a
// route word offered as the tool, is refused at the CLI as it is in the store.
func TestHistoryCaptureRefusesAForgedLabel(t *testing.T) {
	historySourceRepo(t)

	for _, args := range [][]string{
		{"--kind", "other-harness"},
		{"--kind", "import", "--tool", "native"},
		{"--kind", "import"},
	} {
		full := append([]string{"history", "capture", "--session", "sess-forged"}, args...)
		if out, err := runCLIStdinErr(t, "user: hi\n", full...); err == nil {
			t.Errorf("%v was accepted:\n%s", args, out)
		}
	}
}

// TestHistoryCaptureRefusesAFusedKindWithAnotherTool: the legacy fused --kind
// already names its tool, so a --tool naming a different one is refused at the
// CLI, naming both values and the two-label spelling; the same tool is accepted.
func TestHistoryCaptureRefusesAFusedKindWithAnotherTool(t *testing.T) {
	historySourceRepo(t)

	out, err := runCLIStdinErr(t, "user: hi\n",
		"history", "capture", "--session", "sess-conflict", "--kind", "specstory-import", "--tool", "cursor")
	if err == nil {
		t.Fatalf("--kind specstory-import --tool cursor was accepted:\n%s", out)
	}
	msg := err.Error() + string(out)
	for _, want := range []string{"specstory-import", "cursor", "kind import with tool cursor"} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal does not name %q:\n%s", want, msg)
		}
	}
	runCLIStdin(t, "user: hi\n",
		"history", "capture", "--session", "sess-agree", "--kind", "specstory-import", "--tool", "specstory")
	if list := string(runCLI(t, "history", "list")); !strings.Contains(list, "import (specstory)") {
		t.Errorf("the fused kind with its own tool did not store as import (specstory):\n%s", list)
	}
}
