package matrix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
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

// A carried run whose records the secret screen refuses is refused when it is
// read, at its plan, so no re-run's record is ever written for it.
func TestARerunOfARunFolderTheScreenRefusesIsRefusedAsItIsRead(t *testing.T) {
	r, home, m, first := reportLab(t)
	dir := filepath.Join(t.TempDir(), "carried")
	if err := os.CopyFS(dir, os.DirFS(runDir(home, m, first.RunID))); err != nil {
		t.Fatal(err)
	}
	trial := filepath.Join(dir, TrialKey{"feature-a", "or-cheap", 2}.rel())
	data, err := os.ReadFile(trial)
	if err != nil {
		t.Fatal(err)
	}
	token := "ghp_" + testsecret.Synthetic(51, 36)
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["reason"] = "its hidden tests failed near " + token
	carried, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trial, carried, 0o600); err != nil {
		t.Fatal(err)
	}
	// PlanRerun and StartRerun each read the run through loadRerun first.
	if _, err := loadRerun(r.Root(), m.ID, dir); err == nil || !strings.Contains(err.Error(), "not copied") || strings.Contains(err.Error(), token) {
		t.Fatalf("a re-run whose carried trial holds a credential = %v; want it refused as it is read, the value never shown", err)
	}
}

// A re-run reads the first run's record as writeRun screened it: a four-part
// version that is a dotted number and nothing else, which reads as an
// address, is a version, so a run the lab recorded cleanly is repeated, from
// the lab and from its carried folder alike.
func TestARerunOfARunWithFourPartVersionsIsNotRefusedAsAnAddress(t *testing.T) {
	r, home, m, first := reportLab(t, func(in *runInputs) {
		in.Binary.Version = "1.10.2.3"
		for i := range in.Cells {
			in.Cells[i].HarnessVersion = "2.10.3.4"
		}
	})
	if _, err := loadRerun(r.Root(), m.ID, first.RunID); err != nil {
		t.Fatalf("a re-run of a recorded run with four-part versions = %v; want it read", err)
	}
	dir := filepath.Join(t.TempDir(), "carried")
	if err := os.CopyFS(dir, os.DirFS(runDir(home, m, first.RunID))); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRerun(r.Root(), m.ID, dir); err != nil {
		t.Fatalf("a re-run of a carried run with four-part versions = %v; want it read", err)
	}
}

// A trial is screened as a run's record is: a harness version that is a
// dotted number and nothing else is a version, so a four-part one is written,
// while an address in its free text, or a four-part version followed by a
// suffix or a port, still refuses it.
func TestATrialWithAFourPartHarnessVersionIsWrittenAndAnAddressIsNot(t *testing.T) {
	r, _, m, rec, _ := checkLab(t)
	tr := codeTrial(t, rec, "host-big", 1, StatusPassed)
	tr.HarnessVersion = "2.10.3.4"
	if err := writeTrial(r.Root(), m.ID, tr); err != nil {
		t.Fatalf("writeTrial with a four-part harness version = %v; want it written", err)
	}
	tr = codeTrial(t, rec, "host-big", 2, StatusFailed)
	tr.Reason = "its hidden tests could not reach 10.20.30.40"
	if err := writeTrial(r.Root(), m.ID, tr); err == nil || !strings.Contains(err.Error(), "net:ipv4") {
		t.Fatalf("writeTrial with an address in its reason = %v; want it refused", err)
	}
	// A harness version a host reports is withheld only when it is a dotted
	// number and nothing else: a MAC, one whose first pair is the number's last
	// part included, or a host name after the number, is still screened.
	for i, v := range []string{"3c:22:fb:4a:9e:01", "2.1-buildbox.local", "2.1.buildbox.local", "2.1a-kevin-laptop",
		"2.10:22:fb:4a:9e:01", "2.10-22-fb-4a-9e-01"} {
		tr = codeTrial(t, rec, "host-big", 3+i, StatusPassed)
		tr.HarnessVersion = v
		if err := writeTrial(r.Root(), m.ID, tr); err == nil || !strings.Contains(err.Error(), "carries a net:") {
			t.Fatalf("writeTrial with the harness version %q = %v; want it refused by the network screen", v, err)
		}
	}
	// The cost of that rule: a four-part version followed by a suffix or a
	// port is not a dotted number and nothing else, so it is read as written,
	// and its first four parts are refused as an address.
	for i, v := range []string{"2.10.3.4-beta", "10.20.30.40:8080"} {
		tr = codeTrial(t, rec, "host-big", 10+i, StatusPassed)
		tr.HarnessVersion = v
		if err := writeTrial(r.Root(), m.ID, tr); err == nil || !strings.Contains(err.Error(), "carries a net:ipv4") {
			t.Fatalf("writeTrial with the harness version %q = %v; want it refused as a net:ipv4", v, err)
		}
	}
	// A trial's unpinned names are screened as written.
	tr = codeTrial(t, rec, "host-big", 9, StatusPassed)
	tr.Unpinned = append(tr.Unpinned, oracle.Unpinned{Field: "buildbox.local", Reason: oracle.NotReported})
	if err := writeTrial(r.Root(), m.ID, tr); err == nil || !strings.Contains(err.Error(), "carries a net:") {
		t.Fatalf("writeTrial with a host name as an unpinned field = %v; want it refused by the network screen", err)
	}
}

// carriedRun is the report lab's first run carried to a folder of its own,
// with edit applied to its record, which is written back as abcd encodes it.
func carriedRun(t *testing.T, edit func(dir string, rec *RunRecord)) (*gittest.Repo, lab.Minted, string) {
	t.Helper()
	r, home, m, first := reportLab(t)
	dir := filepath.Join(t.TempDir(), "carried")
	if err := os.CopyFS(dir, os.DirFS(runDir(home, m, first.RunID))); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec RunRecord
	if err := strictRun(raw, &rec); err != nil {
		t.Fatal(err)
	}
	edit(dir, &rec)
	js, err := encodeRun(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.json"), js, 0o600); err != nil {
		t.Fatal(err)
	}
	return r, m, dir
}

// A carried run's record is screened for network kinds past what has a
// version's, a digest's or abcd's own field name's shape alone: a MAC as a
// version, a host name as a digest and an unpinned name abcd did not build
// each refuse it.
func TestARerunOfACarriedRunWithAnAddressInAVersionFieldIsRefused(t *testing.T) {
	for name, edit := range map[string]func(*RunRecord){
		"version": func(rec *RunRecord) { rec.Binary.Version = "3c:22:fb:4a:9e:01" },
		// A carried record's version is held to no token shape, so a number
		// running into a phrase still reads as one.
		"version phrase": func(rec *RunRecord) { rec.Binary.Version = "2.1the kevin-laptop" },
		"digest":         func(rec *RunRecord) { rec.Binary.SHA256 = "buildbox.local" },
		"field": func(rec *RunRecord) {
			rec.Unpinned = append(rec.Unpinned, oracle.Unpinned{Field: "buildbox.local", Reason: oracle.NotReported})
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, m, dir := carriedRun(t, func(_ string, rec *RunRecord) { edit(rec) })
			if _, err := loadRerun(r.Root(), m.ID, dir); err == nil || !strings.Contains(err.Error(), "carries a net:") {
				t.Fatalf("a re-run of a carried run with an address in its %s = %v; want it refused by the network screen", name, err)
			}
		})
	}
}

// A carried run's record and declaration are screened as the text they
// decode to: the secret scanner reads JSON escapes, so a credential spelled
// with every character escaped is refused.
func TestARerunOfACarriedRunWithAnEscapedCredentialIsRefused(t *testing.T) {
	token := "ghp_" + testsecret.Synthetic(52, 36)
	escaped := escapeAll(token)
	t.Run("record", func(t *testing.T) {
		r, m, dir := carriedRun(t, func(string, *RunRecord) {})
		raw, err := os.ReadFile(filepath.Join(dir, "run.json"))
		if err != nil {
			t.Fatal(err)
		}
		first, _, ok := strings.Cut(string(raw), `"version": "`)
		if !ok {
			t.Fatal("fixture: the record names no version")
		}
		_, rest, _ := strings.Cut(string(raw)[len(first)+len(`"version": "`):], `"`)
		edited := first + `"version": "` + escaped + `"` + rest
		if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(edited), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadRerun(r.Root(), m.ID, dir); err == nil || !strings.Contains(err.Error(), "not copied") || strings.Contains(err.Error(), token) {
			t.Fatalf("a re-run of a carried record with an escaped credential = %v; want it refused, the value never shown", err)
		}
	})
	t.Run("declaration", func(t *testing.T) {
		r, m, dir := carriedRun(t, func(dir string, rec *RunRecord) {
			decl, err := os.ReadFile(filepath.Join(dir, "declaration.json"))
			if err != nil {
				t.Fatal(err)
			}
			var d struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(decl, &d); err != nil || d.Name == "" {
				t.Fatalf("fixture: the declaration names nothing: %v", err)
			}
			edited := strings.Replace(string(decl), `"`+d.Name+`"`, `"`+escaped+`"`, 1)
			if err := os.WriteFile(filepath.Join(dir, "declaration.json"), []byte(edited), 0o600); err != nil {
				t.Fatal(err)
			}
			rec.Declaration = contentOf("declaration.json", []byte(edited))
		})
		if _, err := loadRerun(r.Root(), m.ID, dir); err == nil || !strings.Contains(err.Error(), "not copied") || strings.Contains(err.Error(), token) {
			t.Fatalf("a re-run of a carried declaration with an escaped credential = %v; want it refused, the value never shown", err)
		}
	})
}

// escapeAll is v with every character written as a JSON escape.
func escapeAll(v string) string {
	var b strings.Builder
	for _, r := range v {
		fmt.Fprintf(&b, `\u%04x`, r)
	}
	return b.String()
}
