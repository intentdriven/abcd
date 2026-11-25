package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// startedMatrixRun is matrixRunLab's run, consented and started from the CLI.
func startedMatrixRun(t *testing.T) (home, id, run string) {
	t.Helper()
	home, id, decl := matrixRunLab(t)
	_, out, _ := runLab(t, "lab", "--json", "matrix", id, decl)
	var plan struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("the plan: %s (%v)", out, err)
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", id, decl, "--approve", plan.Digest)
	var started struct {
		RunID string `json:"run_id"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil || started.RunID == "" {
		t.Fatalf("approve (exit %d): %s%s", code, out, errOut)
	}
	return home, id, started.RunID
}

// A runner cell's copy is launched from the CLI: its cell's runner (a fake
// opencode on PATH, no network) makes it in its clone and its trial is
// written, and the output carries no home path; a host cell's copy, a key the
// run did not hand off and a copy launched already are refused at exit 2.
func TestLabMatrixLaunchMakesARunnerCellsCopyFromTheCLI(t *testing.T) {
	// opencode 1.18.31's run mode, as the runners' fixture records it, read
	// before the lab's checkout becomes the working directory.
	raw, err := os.ReadFile(filepath.Join("..", "..", "core", "runner", "testdata", "opencode-run-1.18.31.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	home, id, run := startedMatrixRun(t)
	bin := t.TempDir()
	stream := filepath.Join(bin, "stream.jsonl")
	if err := os.WriteFile(stream, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"case \"$1\" in --version|-v) echo 1.18.31; exit 0;; esac\n" +
		"r=''\nfor a in \"$@\"; do case \"$a\" in *'Receipt: '*) r=$(printf '%s\\n' \"$a\" | sed -n 's/^Receipt: //p');; esac; done\n" +
		"printf '# README\\n\\nIt is the root.\\n' > \"$r\"\n" +
		"cat '" + stream + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	for _, v := range []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "OPENCODE_CONFIG_DIR"} {
		t.Setenv(v, "")
	}

	for key, want := range map[string]string{"feature-a/host-big/1": "host", "feature-a/or-cheap/9": "no copy", "nope": "key"} {
		if code, _, errOut := runLab(t, "lab", "matrix", "launch", id, run, key); code != 2 || !strings.Contains(errOut, want) {
			t.Errorf("launching %s: exit %d, %s; want 2 naming %q", key, code, errOut, want)
		}
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", "launch", id, run, "doc-a/or-cheap/1")
	var res struct {
		Run    string `json:"run"`
		Copies []struct {
			State, Status, Reason string
		} `json:"copies"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &res) != nil || len(res.Copies) != 1 {
		t.Fatalf("launch (exit %d): %s%s", code, out, errOut)
	}
	if c := res.Copies[0]; c.State != "launched" || c.Status != "passed" {
		t.Fatalf("the writing copy is launched and its trial written as made: %+v", c)
	}
	if _, err := os.Stat(labState(t, home, id, run, "trials", "doc-a", "or-cheap", "1.json")); err != nil {
		t.Fatalf("the copy's trial is not written: %v", err)
	}
	if strings.Contains(out+errOut, home) {
		t.Errorf("the launch's output carries the home path:\n%s%s", out, errOut)
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "launch", id, run, "doc-a/or-cheap/1"); code != 2 || !strings.Contains(errOut, "already launched") {
		t.Errorf("a second launch of the copy: exit %d, %s; want 2", code, errOut)
	}
	if code, out, _ := runLab(t, "lab", "matrix", "copy", id, run, "doc-a/or-cheap/1"); code != 0 || !strings.Contains(out, "launched") {
		t.Errorf("the copy's hand-off shows it launched: exit %d, %s", code, out)
	}
}

// A second launch of a run, while one holds the run's launch lock, is refused
// at exit 3 and launches nothing.
func TestLabMatrixLaunchWhileARunIsHeldExitsThree(t *testing.T) {
	home, id, run := startedMatrixRun(t)
	// The lab's store is keyed on the repository's root commit, which names
	// the run's folder in the hand-off store too.
	rootSHA := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(labState(t, home, id)))))
	dir := abcdhome.Path(home, "matrix-handoff", rootSHA, run)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WithFileLock(filepath.Join(dir, "launch.lock"), time.Second, func() error {
		code, _, errOut := runLab(t, "lab", "matrix", "launch", id, run)
		if code != 3 || !strings.Contains(errOut, "one launch at a time") || !strings.Contains(errOut, "nothing was launched") {
			t.Errorf("a launch while the run is held: exit %d, %s; want 3", code, errOut)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Before the person says yes to a plan, it tells them the runners and each
// copy's code run as their account and are not sandboxed: in the text above
// the line that starts the run, and in the JSON.
func TestTheMatrixPlanSaysTheCopiesAreNotSandboxedBeforeItsApproval(t *testing.T) {
	_, id, decl := matrixRunLab(t)
	code, out, errOut := runLab(t, "lab", "matrix", id, decl)
	if code != 0 {
		t.Fatalf("the plan: exit %d, %s%s", code, out, errOut)
	}
	said, approve := strings.Index(out, "are not sandboxed"), strings.Index(out, "to start it:")
	if said < 0 || approve < 0 || said > approve || !strings.Contains(out, "run as this account") {
		t.Fatalf("the plan says the copies are not sandboxed before its approval line:\n%s", out)
	}
	code, out, _ = runLab(t, "lab", "--json", "matrix", id, decl)
	var plan struct {
		NotSandboxed string `json:"not_sandboxed"`
		ApproveWith  string `json:"approve_with"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &plan) != nil || !strings.Contains(plan.NotSandboxed, "not sandboxed") || plan.ApproveWith == "" {
		t.Fatalf("the JSON plan carries not_sandboxed: %s", out)
	}
}
