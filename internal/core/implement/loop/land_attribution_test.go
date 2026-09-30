package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The records commit's text carries prose a model composed (the receipt's
// resolution note and grounds, the audit's verdict), so it discloses that
// model and is made with the repository's hooks running; these tests hold the
// landing to both (iss-2609301046433372's finding, the review's ll1).

// TestTheRecordsCommitTrailerIsTheReceiptsModel: the trailer is the model the
// lane's receipt reported, in the vendor form the attribution gate takes.
func TestTheRecordsCommitTrailerIsTheReceiptsModel(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.model = "claude-test-7[1m]"
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !slices.Contains(strings.Split(msg, "\n"), "Assisted-by: Claude:claude-test-7[1m]") || strings.Contains(msg, "Assisted-by: None") {
		t.Fatalf("the records commit discloses the receipt's model, never None:\n%s", msg)
	}
}

// TestALandingWhoseReceiptReportsNoModelIsRefused: with no model to disclose,
// the records commit is not made, and nothing is written in the lane's
// worktree; the refusal names the missing value.
func TestALandingWhoseReceiptReportsNoModelIsRefused(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.model = ""
	l := f.validated(t)
	head := l.HeadSHA
	f.step(t) // prepare
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Stage != string(StageLand) || !strings.Contains(r.Reason, "model") || !strings.Contains(r.Remedy, "model") {
		t.Fatalf("a landing with no reported model is refused naming the model: %+v", r)
	}
	l = currentLane(t, f.repo, f.runID)
	if l.HeadSHA != head {
		t.Fatalf("no records commit is made without a model: head %s, was %s", l.HeadSHA, head)
	}
	if dirty := f.repo.Git("-C", l.Worktree, "status", "--porcelain"); dirty != "" {
		t.Fatalf("the refusal comes before any record is written:\n%s", dirty)
	}
}

// TestAssistedByTrailers: a bare Claude model id takes the vendor prefix, a
// vendor-qualified id is kept, and a missing or unrecognised one is refused.
func TestAssistedByTrailers(t *testing.T) {
	for _, tc := range []struct {
		models []string
		want   string
		gap    bool
	}{
		{[]string{"claude-opus-5-5"}, "Assisted-by: Claude:claude-opus-5-5", false},
		{[]string{"claude-opus-5[1m]"}, "Assisted-by: Claude:claude-opus-5[1m]", false},
		{[]string{"Claude:claude-opus-5-5"}, "Assisted-by: Claude:claude-opus-5-5", false},
		{[]string{"claude-a", "claude-b", "claude-a"}, "Assisted-by: Claude:claude-a\nAssisted-by: Claude:claude-b", false},
		{[]string{"ExampleVendor:model-1"}, "Assisted-by: ExampleVendor:model-1", false},
		{nil, "", true},
		{[]string{""}, "", true},
		{[]string{"claude-a", ""}, "", true},
		{[]string{"a-model"}, "", true},
		{[]string{"claude-a\nAssisted-by: None"}, "", true},
		{[]string{"None"}, "", true},
	} {
		var rs []ReceiptRecord
		for _, m := range tc.models {
			rs = append(rs, ReceiptRecord{Role: RoleImplementer, Receipt: "r.json", Model: m})
		}
		got, gap := assistedByTrailers(rs)
		if (gap != "") != tc.gap || strings.Join(got, "\n") != tc.want {
			t.Errorf("%q: got %q (gap %q), want %q (gap %v)", tc.models, got, gap, tc.want, tc.gap)
		}
	}
}

// installHook writes an executable hook of the repository's, in its common git
// directory, where every worktree's commit runs it.
func installHook(t *testing.T, f *landFixture, name, body string) string {
	t.Helper()
	common := strings.TrimSpace(f.repo.Git("rev-parse", "--path-format=absolute", "--git-common-dir"))
	dir := filepath.Join(common, "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestARefusingCommitMsgHookStopsTheLanding: the records commit runs the
// repository's commit-msg hook (the outbound gate); a hook that refuses stops
// the landing loudly with no commit made, and the landing resumes once the
// hook is satisfied.
func TestARefusingCommitMsgHookStopsTheLanding(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	head := l.HeadSHA
	seen := filepath.Join(t.TempDir(), "seen")
	hook := installHook(t, f, "commit-msg", "#!/bin/sh\ncat \"$1\" > '"+seen+"'\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
	f.step(t) // prepare
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Stage != string(StageLand) || !strings.Contains(r.Reason, "refused by the test") {
		t.Fatalf("a refusing commit-msg hook stops the landing, naming what it said: %+v", r)
	}
	if b, err := os.ReadFile(seen); err != nil || !strings.Contains(string(b), "Assisted-by: Claude:claude-test-5") {
		t.Fatalf("the hook judged the records commit's message: %q %v", b, err)
	}
	if got := currentLane(t, f.repo, f.runID); got.HeadSHA != head || got.Landing.RecordsDone {
		t.Fatalf("no records commit is made over a refusing hook: %+v", got.Landing)
	}

	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.HeadSHA == head || !l.Landing.RecordsDone {
		t.Fatalf("the landing resumes once the hook passes: %+v", l.Landing)
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	for _, want := range []string{"Delivers: itd-10", "Resolves: " + f.issue, "Assisted-by: Claude:claude-test-5"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the resumed records commit carries %q:\n%s", want, msg)
		}
	}
}

// TestTheLandedRangePassesTheAttributionGate runs the repository's own
// attribution gate over the range the landing added to the lane.
func TestTheLandedRangePassesTheAttributionGate(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("the attribution gate is a bash script")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// The gate's session-URL half runs `abcd lint outbound`; the checker is
	// built from this checkout, before the fixture moves HOME (and the build
	// cache with it).
	bin := filepath.Join(t.TempDir(), "abcd")
	build := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	build.Dir = root
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the outbound checker: %v\n%s", err, b)
	}
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	implHead := l.HeadSHA
	f.step(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.HeadSHA == implHead {
		t.Fatal("the landing made no records commit")
	}
	cmd := exec.Command("bash", filepath.Join(root, "scripts", "check-attribution.sh"), "commits", implHead, l.HeadSHA)
	cmd.Dir = l.Worktree
	cmd.Env = append(os.Environ(), "ABCD_OUTBOUND_BIN="+bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the attribution gate passes the landed range %s..%s: %v\n%s", shortSHA(implHead), shortSHA(l.HeadSHA), err, out)
	}
}
