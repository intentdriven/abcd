package loop

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// awaitingLane drives a fresh run to its implement step's await and returns
// the run id, the lane as the state holds it, and the lane's directory.
func awaitingLane(t *testing.T) (*gittest.Repo, string, Lane, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StepImplement)
	res, err := Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("the implement step awaits an implementer: %+v", res)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	wantReceipt := RunRelDir + "/" + start.RunID + "/lane-1/" + ReceiptFileName
	if l.Awaiting.Receipt != wantReceipt || l.Awaiting.Brief != l.Brief {
		t.Fatalf("the await names the lane's brief and receipt: %+v", l.Awaiting)
	}
	return repo, start.RunID, l, filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), start.RunID, "lane-1")
}

// laneCommit commits one file in the lane's worktree and returns its sha.
func laneCommit(t *testing.T, repo *gittest.Repo, l Lane, name string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(l.Worktree, name), []byte(name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", l.Worktree, "add", "--", name)
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "lane: "+name)
	return strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
}

func zero() *int { z := 0; return &z }

// goodReceipt is a receipt that verifies, with the report and the definition of
// done's output written beside it.
func goodReceipt(t *testing.T, runID string, l Lane, dir string, commits ...string) LaneReceipt {
	t.Helper()
	for name, body := range map[string]string{ReportFileName: "built it\n", DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: runID, Lane: l.ID, Branch: l.Branch, Commits: commits,
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: DoDFileName}, Report: ReportFileName, Model: "a-model"}
}

func writeReceipt(t *testing.T, dir string, rc any) string {
	t.Helper()
	var data []byte
	switch v := rc.(type) {
	case string:
		data = []byte(v)
	default:
		var err error
		if data, err = json.Marshal(v); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, ReceiptFileName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestAVerifiedReceiptAdvancesTheLane is criterion 4's pass: a receipt naming
// the lane's commits, a passing definition of done's output and the report
// completes the implement step, and the lane's head moves to its branch's tip.
func TestAVerifiedReceiptAdvancesTheLane(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c1 := laneCommit(t, repo, l, "one.txt")
	c2 := laneCommit(t, repo, l, "two.txt")
	path := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c1, c2))

	res, err := Receipt(repo.Root(), runID, path, DefaultSteps(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Performed != StepImplement || res.Step != StepValidate {
		t.Fatalf("a verified receipt completes the implement step: %+v", res)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Lanes[0]; got.HeadSHA != c2 || got.BaseSHA != l.BaseSHA || got.Receipt != l.Awaiting.Receipt || got.Awaiting != nil {
		t.Fatalf("the lane's head is its branch's tip: %+v", got)
	}
}

// TestAReceiptShortOfItsLaneIsRefusedNamingWhatIsMissing is criterion 4's
// refusal: a receipt without commits on the branch, without the definition of
// done's output or without the report is refused naming what is missing, and
// the lane is not advanced. A receipt that is not what the brief names — a
// verdict field, a second document, a symlink, a path out of the lane's
// directory — is refused the same way.
func TestAReceiptShortOfItsLaneIsRefusedNamingWhatIsMissing(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(t *testing.T, repo *gittest.Repo, l Lane, dir string, rc *LaneReceipt) any
		wants []string
	}{
		{"no commits", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits = nil
			return rc
		}, []string{"commits on the lane's branch (it names none)"}},
		{"a commit on the default branch", func(t *testing.T, repo *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits = append(rc.Commits, strings.TrimSpace(repo.Git("rev-parse", "main")))
			return rc
		}, []string{"already on the default branch"}},
		{"a commit on another branch", func(t *testing.T, repo *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			repo.Git("checkout", "-q", "-b", "side")
			repo.Write("side.txt", "side\n")
			repo.Commit("side")
			rc.Commits = []string{strings.TrimSpace(repo.Git("rev-parse", "HEAD"))}
			return rc
		}, []string{"not on build/"}},
		{"a commit that does not exist", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits = []string{strings.Repeat("ab", 20)}
			return rc
		}, []string{"no such commit"}},
		{"a short name", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits = []string{"--all"}
			return rc
		}, []string{"not a full object name"}},
		{"no definition of done", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.DefinitionOfDone = nil
			return rc
		}, []string{"the definition of done's output (no definition_of_done)"}},
		{"a failing definition of done", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			one := 1
			rc.DefinitionOfDone.ExitCode = &one
			return rc
		}, []string{"it exited 1"}},
		{"no definition of done output", func(t *testing.T, _ *gittest.Repo, _ Lane, dir string, rc *LaneReceipt) any {
			if err := os.Remove(filepath.Join(dir, DoDFileName)); err != nil {
				t.Fatal(err)
			}
			return rc
		}, []string{"the definition of done's output (dod.log does not exist)"}},
		{"no report", func(t *testing.T, _ *gittest.Repo, _ Lane, dir string, rc *LaneReceipt) any {
			if err := os.Remove(filepath.Join(dir, ReportFileName)); err != nil {
				t.Fatal(err)
			}
			return rc
		}, []string{"the report (report.md does not exist)"}},
		{"everything missing at once", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits, rc.DefinitionOfDone, rc.Report = nil, nil, ""
			return rc
		}, []string{"it names none", "no definition_of_done", "the report (none named)"}},
		{"a report out of the lane's directory", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Report = "../../../../AGENTS.md"
			return rc
		}, []string{"is not a path inside the lane's directory"}},
		{"a symlinked report", func(t *testing.T, repo *gittest.Repo, _ Lane, dir string, rc *LaneReceipt) any {
			if err := os.Remove(filepath.Join(dir, ReportFileName)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(repo.Root(), "AGENTS.md"), filepath.Join(dir, ReportFileName)); err != nil {
				t.Fatal(err)
			}
			return rc
		}, []string{"report.md is not a regular file"}},
		{"another lane's receipt", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Lane, rc.Branch = "lane-2", "build/other"
			return rc
		}, []string{"the run and lane", "the lane's branch"}},
		{"a verdict the loop did not record", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			b, _ := json.Marshal(rc)
			return strings.Replace(string(b), `"schema_version":1`, `"schema_version":1,"verdict":"SHIP"`, 1)
		}, []string{"unknown field \"verdict\""}},
		{"a second document", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			b, _ := json.Marshal(rc)
			return string(b) + "\n{}\n"
		}, []string{"more than one JSON document"}},
		{"an oversize receipt", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			return strings.Repeat(" ", maxReceiptBytes+1)
		}, []string{"cannot be read as a receipt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, l, dir := awaitingLane(t)
			c1 := laneCommit(t, repo, l, "one.txt")
			rc := goodReceipt(t, runID, l, dir, c1)
			path := writeReceipt(t, dir, tc.edit(t, repo, l, dir, &rc))
			before := stateBytes(t, repo.Root(), runID)

			_, err := Receipt(repo.Root(), runID, path, DefaultSteps(), Options{})
			r := mustRefusal(t, err)
			if r.Step != "receipt" || r.Lane != "lane-1" {
				t.Fatalf("want the receipt refused for the lane: %+v", r)
			}
			for _, want := range tc.wants {
				if !strings.Contains(r.Reason, want) {
					t.Fatalf("the refusal names what is missing (%q): %q", want, r.Reason)
				}
			}
			if !bytes.Equal(before, stateBytes(t, repo.Root(), runID)) {
				t.Fatal("a refused receipt must not advance the lane")
			}
		})
	}
}

// TestASymlinkedReceiptIsRefused: the receipt is read through the guarded
// reader, so a symlink standing in for it is refused, not followed.
func TestASymlinkedReceiptIsRefused(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c1 := laneCommit(t, repo, l, "one.txt")
	elsewhere := writeReceipt(t, t.TempDir(), goodReceipt(t, runID, l, dir, c1))
	path := filepath.Join(dir, ReceiptFileName)
	if err := os.Symlink(elsewhere, path); err != nil {
		t.Fatal(err)
	}
	_, err := Receipt(repo.Root(), runID, path, DefaultSteps(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "cannot be read as a receipt") {
		t.Fatalf("want the symlinked receipt refused: %+v", r)
	}
}

// TestNoReceiptYetIsRefused: handing back a receipt before it exists refuses
// and leaves the lane awaiting it.
func TestNoReceiptYetIsRefused(t *testing.T) {
	repo, runID, _, dir := awaitingLane(t)
	_, err := Receipt(repo.Root(), runID, filepath.Join(dir, ReceiptFileName), DefaultSteps(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "no receipt at") {
		t.Fatalf("want the absent receipt named: %+v", r)
	}
}
