package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gittest"
)

// rerunOf records a re-run of first in its lab, as consent would: the run's
// record from the same declaration on a machine of cores cores, the first
// run's records copied under it by the re-run's starter, and the re-run's
// trials, each edit made to them first.
func rerunOf(t *testing.T, r *gittest.Repo, m lab.Minted, first RunRecord, from string, cores int, edit func([]TrialRecord)) RunRecord {
	t.Helper()
	rr, err := loadRerun(r.Root(), m.ID, from)
	if err != nil {
		t.Fatalf("loadRerun(%s): %v", from, err)
	}
	in := inputs(t)
	in.RunID, in.Lab, in.RootSHA, in.FrozenCommit, in.Declaration = "mx-261024012949-cd34", m.ID, m.RootSHA, m.Pin, rr.Declaration
	in.Machine.Cores = cores
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(r.Root(), rec, in.Declaration); err != nil {
		t.Fatal(err)
	}
	inner := &countingStarter{}
	if err := rr.starter(r.Root(), m.ID, inner).Start(Plan{}, rec); err != nil {
		t.Fatalf("the re-run's starter: %v", err)
	}
	if inner.starts != 1 {
		t.Fatalf("the re-run's starter started the run %d times; want once, after copying the first run", inner.starts)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	trials, err := readTrials(proc, first.RunID, first.RunID+"/trials")
	proc.Close()
	if err != nil {
		t.Fatal(err)
	}
	for i := range trials {
		trials[i].Run = rec.RunID
	}
	edit(trials)
	for _, tr := range trials {
		if err := writeTrial(r.Root(), m.ID, tr); err != nil {
			t.Fatal(err)
		}
	}
	return rec
}

// A re-run starts from a recorded run (its declaration, at the frozen commit
// it pinned) and its report shows, field by field, where it drifted from that
// run: a field of the run's record, and a field of one copy's trial, each
// with both values; a field that stayed the same is not listed, and the
// identifiers and clocks that differ by construction are not drift.
func TestARerunReportsDriftFieldByField(t *testing.T) {
	r, _, m, first := reportLab(t)
	rec := rerunOf(t, r, m, first, first.RunID, 8, func(ts []TrialRecord) {
		for i := range ts {
			if ts[i].Key == (TrialKey{"feature-a", "or-cheap", 2}) {
				ts[i].Status, ts[i].Score = StatusPassed, &Score{Passed: 4, Total: 4}
			}
		}
	})
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Drift == nil || rep.Drift.Of != first.RunID {
		t.Fatalf("the re-run's report = %+v; want drift from %s", rep.Drift, first.RunID)
	}
	got := map[string]DriftField{}
	for _, f := range rep.Drift.Fields {
		got[f.Field] = f
	}
	cores, ok := got["machine.cores"]
	if !ok || cores.First != "12" || cores.Now != "8" {
		t.Errorf("the machine's drift = %+v; want 12 then 8", cores)
	}
	status, ok := got["trials.feature-a/or-cheap/2.status"]
	if !ok || status.First != `"failed"` || status.Now != `"passed"` {
		t.Errorf("the copy's drift = %+v; want failed then passed", status)
	}
	if _, ok := got["trials.feature-a/or-cheap/2.score.passed"]; !ok {
		t.Errorf("the copy's score did not drift: %+v", rep.Drift.Fields)
	}
	for f := range got {
		if f == "run_id" || strings.HasSuffix(f, ".run") || strings.HasSuffix(f, "started_at") || strings.Contains(f, "host-big/1") {
			t.Errorf("%s is listed as drift: %+v", f, got[f])
		}
	}
	if rep.Drift.Same == 0 {
		t.Error("the drift counts no field that stayed the same")
	}
	if first := mustReport(t, r, m, first.RunID); first.Drift != nil {
		t.Errorf("the first run's report shows drift: %+v", first.Drift)
	}
}

func mustReport(t *testing.T, r *gittest.Repo, m lab.Minted, run string) Report {
	t.Helper()
	rep, err := BuildReport(r.Root(), m.ID, run)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// A re-run may start from a run's folder carried from elsewhere, and is
// refused when the lab's pin is not the commit that run froze: the re-run
// would start from other code.
func TestARerunStartsFromARunFolderAtItsOwnFrozenCommitOnly(t *testing.T) {
	r, home, m, first := reportLab(t)
	dir := filepath.Join(t.TempDir(), "carried")
	if err := os.CopyFS(dir, os.DirFS(runDir(home, m, first.RunID))); err != nil {
		t.Fatal(err)
	}
	rerunOf(t, r, m, first, dir, 12, func([]TrialRecord) {})

	run := filepath.Join(dir, "run.json")
	data, err := os.ReadFile(run)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(run, []byte(strings.ReplaceAll(string(data), first.FrozenCommit, strings.Repeat("f", 40))), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRerun(r.Root(), m.ID, dir); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("a re-run of another commit's run = %v; want a refusal naming the frozen commit", err)
	}
	if _, err := loadRerun(r.Root(), m.ID, "mx-261023012949-ffff"); err == nil {
		t.Fatal("a re-run of a run the lab never recorded was admitted")
	}
}
