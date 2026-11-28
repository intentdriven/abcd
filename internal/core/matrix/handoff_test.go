package matrix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A code task that names its hidden tests has them placed by the run as it
// starts and run by abcd itself (spc-2610161949559050 step 10): a host copy
// as it is handed back and a runner copy as its launch ends, each scored by
// abcd's run and its trial saying so.

// hiddenCheck is the hidden test the wiring's runs carry: it passes where the
// copy's tree holds README.md and the hidden test itself, and fails otherwise.
const hiddenCheck = "test -f README.md && test -f hidden_check.sh && echo ran the hidden check\n"

// hiddenDeclaration is goodDeclaration with one code task, feature-a, naming
// its hidden tests at folder, run by command.
func hiddenDeclaration(folder string, command []string) []byte {
	words := make([]string, len(command))
	for i, w := range command {
		words[i] = strconv.Quote(w)
	}
	d := strings.Replace(goodDeclaration,
		`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"},
    {"id": "doc-page", "kind": "writing", "source": ["docs/guide.md"], "criteria": "accurate against the source first"},
    {"id": "fidelity", "kind": "check", "material": ["docs/reference"]}`,
		`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": `+strconv.Quote(folder)+
			`, "test_command": [`+strings.Join(words, ", ")+`]}`, 1)
	return []byte(d)
}

// hiddenLab is a run of hiddenDeclaration started by the launcher: its hidden
// tests placed and each builder's one copy handed off.
func hiddenLab(t *testing.T, command []string) (*gittest.Repo, string, lab.Minted, RunRecord, *Launcher) {
	t.Helper()
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	folder := hiddenFolderWith(t, map[string]string{"hidden_check.sh": hiddenCheck})
	r, home, m, rec, _ := checkLab(t, func(in *runInputs) { in.Declaration = hiddenDeclaration(folder, command) })
	l := &Launcher{RepoRoot: r.Root(), Home: home}
	if err := l.Start(Plan{}, rec); err != nil {
		t.Fatalf("Start = %v", err)
	}
	return r, home, m, rec, l
}

// placedRefs is task's hidden record as the run's start wrote it.
func placedRefs(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, task string) ([]ContentRef, bool) {
	t.Helper()
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	refs, ok, err := readHiddenRecord(proc, rec.RunID, task)
	if err != nil {
		t.Fatal(err)
	}
	return refs, ok
}

// A consented run places its code task's hidden tests as it starts, before
// any copy: outside every clone, under their digest name, kept by content in
// the run's state; a run with no consent places nothing.
func TestAConsentedRunPlacesTheHiddenTests(t *testing.T) {
	r, home, m, rec, l := hiddenLab(t, []string{"/bin/sh", "hidden_check.sh"})
	refs, ok := placedRefs(t, r, m, rec, "feature-a")
	if !ok || len(refs) != 1 || refs[0] != contentOf("hidden_check.sh", []byte(hiddenCheck)) {
		t.Fatalf("the run's hidden record = %+v, %v; want the one test by its content", refs, ok)
	}
	store := cloneStore(home, rec.RootSHA)
	if _, err := readHiddenTests(store, rec.RunID, "feature-a", refs); err != nil {
		t.Fatalf("the placed tests do not read back as recorded: %v", err)
	}
	if len(l.Handed) != 6 {
		t.Fatalf("the run handed off %d copies; want three for each builder", len(l.Handed))
	}
	for _, h := range l.Handed {
		if h.State == HandOffRefused {
			t.Fatalf("copy %s was refused: %s", h.Key, h.Reason)
		}
		if err := hiddenOutsideClone(h.Clone, hiddenDir(store, rec.RunID, "feature-a"), refs); err != nil {
			t.Errorf("copy %s's clone reaches its hidden tests: %v", h.Key, err)
		}
	}
	// No consent, nothing started and nothing placed.
	r2, home2, m2, rec2, _ := checkLab(t, func(in *runInputs) {
		in.Declaration = hiddenDeclaration(hiddenFolderWith(t, map[string]string{"hidden_check.sh": hiddenCheck}), []string{"/bin/sh", "hidden_check.sh"})
	})
	if _, err := consent(r2.Root(), Plan{}, "", ConsentInputs{}, &Launcher{RepoRoot: r2.Root(), Home: home2}); !errors.Is(err, ErrNotConsented) {
		t.Fatalf("a run with no consent = %v; want ErrNotConsented", err)
	}
	if _, err := os.Stat(filepath.Join(cloneStore(home2, rec2.RootSHA), "hidden")); !os.IsNotExist(err) {
		t.Errorf("a run with no consent placed hidden tests: %v", err)
	}
	if _, ok := placedRefs(t, r2, m2, rec2, "feature-a"); ok {
		t.Error("a run with no consent recorded hidden tests")
	}
}

// A hidden tests' folder the run cannot read where it lies apart refuses the
// start with nothing placed and no copy handed off, and a frozen commit that
// already holds a hidden test refuses its copies before they start.
func TestAStartRefusesHiddenTestsItCannotKeepHidden(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	r, home, m, rec, _ := checkLab(t, func(in *runInputs) {
		in.Declaration = hiddenDeclaration(filepath.Join(realPath(t, t.TempDir()), "no-such-folder"), []string{"/bin/sh", "x"})
	})
	l := &Launcher{RepoRoot: r.Root(), Home: home}
	if err := l.Start(Plan{}, rec); err == nil || len(l.Handed) != 0 {
		t.Fatalf("a start with an unreadable hidden tests' folder = %v, %d handed off; want it refused before any copy", err, len(l.Handed))
	}
	if _, err := os.Stat(filepath.Join(cloneStore(home, rec.RootSHA), "hidden")); !os.IsNotExist(err) {
		t.Errorf("a refused start placed hidden tests: %v", err)
	}
	if _, ok := placedRefs(t, r, m, rec, "feature-a"); ok {
		t.Error("a refused start recorded hidden tests")
	}

	// The frozen commit holds the hidden test: every copy is refused.
	folder := hiddenFolderWith(t, map[string]string{"hidden_check.sh": hiddenCheck})
	r2, home2, _, rec2, _ := checkLabWith(t, map[string]string{"leaked/hidden_check.sh": hiddenCheck},
		func(in *runInputs) {
			in.Declaration = hiddenDeclaration(folder, []string{"/bin/sh", "hidden_check.sh"})
		})
	l2 := &Launcher{RepoRoot: r2.Root(), Home: home2}
	if err := l2.Start(Plan{}, rec2); err != nil {
		t.Fatalf("Start = %v", err)
	}
	for _, h := range l2.Handed {
		if h.State != HandOffRefused || !strings.Contains(h.Reason, "hidden test") || h.Clone != "" {
			t.Errorf("copy %s of a frozen commit holding its hidden test = %s (%s); want it refused, its clone removed", h.Key, h.State, h.Reason)
		}
	}
}

// A host copy handed back made, with no score, is scored by abcd's own run of
// its hidden tests: the trial says abcd placed and ran them, names the tree
// the run read, and is never marked the host's report; what the tests printed
// is kept beside it. A score the host hands back is refused, since the tests
// are abcd's to run.
func TestAHostCopyHandedBackIsScoredByAbcdsRun(t *testing.T) {
	r, home, m, rec, l := hiddenLab(t, []string{"/bin/sh", "hidden_check.sh"})
	key := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 1}
	var clone string
	for _, h := range l.Handed {
		if h.Key == key {
			clone = h.Clone
		}
	}
	transcript := []byte("edited feature.go in " + clone + "\n")
	scored := hostReceipt(t, map[string]any{"score": map[string]int{"passed": 4, "total": 4}})
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: scored, Transcript: transcript}); err == nil ||
		!strings.Contains(err.Error(), "abcd's to run") {
		t.Fatalf("a host copy handed back with its own score = %v; want it refused", err)
	}
	got, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: hostReceipt(t, nil), Transcript: transcript})
	if err != nil {
		t.Fatalf("CollectCopy = %v", err)
	}
	digest, _, err := copyTreeDigest(context.Background(), clone)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusPassed || got.Score == nil || got.Score.Detail != AbcdScoreDetail || !got.AbcdRanHiddenTests() ||
		got.HostReportedScore() || got.TreeDigest != digest {
		t.Fatalf("the host copy's trial = %+v; want passed on abcd's run, naming the tree it read, never the host's report", got)
	}
	data := trialAt(t, home, m, rec, key)
	if !strings.Contains(data, `"hidden_tests":"`+HiddenTestsByAbcd+`"`) || strings.Contains(data, `"trial.score"`) {
		t.Errorf("the written trial does not say abcd placed and ran the tests:\n%s", data)
	}
	out, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "hidden-output", key.Task, key.Cell, "1.txt"))
	if err != nil || !strings.Contains(string(out), "ran the hidden check") {
		t.Errorf("what the hidden tests printed is not kept: %q, %v", out, err)
	}
}

// A runner copy is scored by abcd's run of its hidden tests as its launch
// ends; where that run reaches no outcome (its command cannot start), the
// copy waits, launched, and a hand-back with a made receipt and no score runs
// them again, against the tree its launch read.
func TestALaunchedCopyIsScoredByAbcdsRun(t *testing.T) {
	bin := filepath.Join(realPath(t, t.TempDir()), "bin")
	runnerPath := filepath.Join(bin, "run-hidden")
	r, home, m, rec, l := hiddenLab(t, []string{runnerPath})
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	var clone string
	for _, h := range l.Handed {
		if h.Key == key {
			clone = h.Clone
		}
	}
	f := &fakeCell{answer: madeReceipt, transcript: inClone(clone)}
	// The test command is not there yet: abcd's run of it reaches no
	// outcome, which is the launch's error and never the copy's score.
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err == nil {
		t.Fatal("a launch whose hidden tests could not run reports no error")
	}
	if _, ok := trialOf(t, r, m, rec, key); ok {
		t.Fatal("a copy whose hidden tests did not run has a trial")
	}
	h := handOffNow(t, r, m, rec, key)
	if h.State != HandOffLaunched || h.Trial == nil || !h.Trial.AbcdRanHiddenTests() || h.Trial.TreeDigest == "" {
		t.Fatalf("the copy waits, launched, its tree read: %+v", h)
	}
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runnerPath, []byte("#!/bin/sh\nexec /bin/sh hidden_check.sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: []byte(`{"schema_version":1,"outcome":"made","score":{"passed":1,"total":1}}`)}); err == nil {
		t.Fatal("a launched copy handed back with the host's score is taken")
	}
	got, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: []byte(`{"schema_version":1,"outcome":"made"}`)})
	if err != nil {
		t.Fatalf("CollectCopy = %v", err)
	}
	if got.Status != StatusPassed || !got.AbcdRanHiddenTests() || got.HostReportedScore() || got.Score.Detail != AbcdScoreDetail ||
		got.TreeDigest != h.Trial.TreeDigest {
		t.Fatalf("the launched copy's trial = %+v; want passed on abcd's run of the tree its launch read", got)
	}

	// A second runner copy, its command in place, is scored as its launch
	// ends, with no hand-back.
	key2 := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	var clone2 string
	for _, h := range l.Handed {
		if h.Key == key2 {
			clone2 = h.Clone
		}
	}
	f2 := &fakeCell{answer: madeReceipt, transcript: inClone(clone2)}
	res, err := launchOne(t, testLauncher(r, home, m, f2.dispatch), rec, key2)
	if err != nil {
		t.Fatalf("launch = %v", err)
	}
	if len(res.Copies) != 1 || res.Copies[0].Status != StatusPassed {
		t.Fatalf("the launch = %+v; want the copy passed on abcd's run", res.Copies)
	}
	got2, ok := trialOf(t, r, m, rec, key2)
	if !ok || !got2.AbcdRanHiddenTests() || got2.HostReportedScore() || got2.TreeDigest == "" {
		t.Fatalf("the launched copy's trial = %+v, %v; want abcd's run, naming its tree", got2, ok)
	}
}
