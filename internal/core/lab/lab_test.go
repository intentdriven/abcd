package lab

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMain points HOME at a throwaway directory before any test runs, so no
// test in this package can reach the operator's real lab store even if it
// forgets to set its own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "abcd-lab-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// fixture is a two-commit repository under a fresh test HOME, with the clock
// pinned so lab ids are predictable.
func fixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Commit("root")
	r.Write("a.txt", "a\n")
	r.Commit("second")
	prev := now
	now = func() time.Time { return time.Date(2026, 9, 25, 10, 11, 12, 0, time.UTC) }
	t.Cleanup(func() { now = prev })
	return r, home
}

// stubVintage makes every lab binary read as built at rev.
func stubVintage(t *testing.T, rev string, known bool) {
	t.Helper()
	prev := vintageOf
	vintageOf = func(io.ReaderAt) (vintage.Current, error) {
		return vintage.Current{Revision: rev, Known: known}, nil
	}
	t.Cleanup(func() { vintageOf = prev })
}

func mint(t *testing.T, r *gittest.Repo) Minted {
	t.Helper()
	m, err := Mint(r.Root(), "does the procedure transfer to design work?", "")
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	return m
}

// labDir is the absolute lab home under the test HOME.
func labDir(home string, m Minted) string {
	return filepath.Join(home, ".abcd", "lab", m.RootSHA, m.ID)
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Criterion 1: a lab home exists under the machine-scoped store with a registry
// entry, the snapshot pin and the lifecycle's sections, and nothing is written
// into the repository.
func TestMintLaysDownALabAndWritesNothingInTheRepo(t *testing.T) {
	r, home := fixture(t)
	before, _ := os.ReadDir(r.Root())
	head := r.Git("rev-parse", "HEAD")
	root := r.Git("rev-list", "--max-parents=0", "HEAD")

	m := mint(t, r)
	if m.ID != "lab-260925101112-"+head[:7] || m.Pin != head || m.RootSHA != root {
		t.Fatalf("minted %+v; want id at the pinned clock and HEAD %s, keyed on root %s", m.Entry, head, root)
	}
	dir := labDir(home, m)
	if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("lab home %s: %v (mode %v), want a 0700 directory", dir, err, fi)
	}
	for _, rel := range []string{"INTENTION.md", "findings.md", "corrections.md", "amendments.md", "home", "bin", "state/probes", "harvest", "snapshot/README.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("lab home lacks %s: %v", rel, err)
		}
	}
	intention := read(t, filepath.Join(dir, "INTENTION.md"))
	for _, want := range []string{"snapshot_pin: " + head, "does the procedure transfer", "## Hypothesis", "## STOP conditions", "## Lifecycle", "| DISCARD |"} {
		if !strings.Contains(intention, want) {
			t.Errorf("INTENTION.md lacks %q", want)
		}
	}
	idx := read(t, filepath.Join(home, ".abcd", "lab", root, "index.jsonl"))
	var e Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(idx)), &e); err != nil || e.ID != m.ID || e.Pin != head {
		t.Fatalf("registry = %q (%v), want one line for %s at the pin", idx, err, m.ID)
	}
	snap := filepath.Join(dir, "snapshot")
	if got := gitIn(t, snap, "rev-parse", "HEAD"); got != head {
		t.Errorf("snapshot HEAD = %s, want the pin %s", got, head)
	}
	if got := gitIn(t, snap, "remote"); got != "" {
		t.Errorf("snapshot remotes = %q, want none: the lab world has no path back to the checkout", got)
	}
	if strings.Contains(strings.Join(m.Written, " "), home) || strings.Contains(m.Home, home) {
		t.Errorf("result carries the home path: %+v", m)
	}

	if got := r.Git("status", "--porcelain", "--ignored"); got != "" {
		t.Errorf("mint wrote into the repository: %q", got)
	}
	after, _ := os.ReadDir(r.Root())
	if len(after) != len(before) {
		t.Errorf("repository top level changed: %d entries before, %d after", len(before), len(after))
	}
}

func TestMintPinsAnEarlierCommit(t *testing.T) {
	r, home := fixture(t)
	first := r.Git("rev-parse", "HEAD~1")
	m, err := Mint(r.Root(), "what did the root commit hold?", "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Pin != first || !strings.HasSuffix(m.ID, "-"+first[:7]) {
		t.Fatalf("minted %+v, want the pin %s", m.Entry, first)
	}
	if got := gitIn(t, filepath.Join(labDir(home, m), "snapshot"), "rev-parse", "HEAD"); got != first {
		t.Errorf("snapshot HEAD = %s, want %s", got, first)
	}
}

func TestMintRefusesWhatIsNotAQuestionOrAPin(t *testing.T) {
	r, home := fixture(t)
	for _, c := range []struct{ q, pin string }{
		{"", ""},
		{"two\nlines", ""},
		{"a \x1b[31m coloured question", ""},
		{strings.Repeat("q", maxQuestionBytes+1), ""},
		{"fine question", "no-such-rev"},
		{"fine question", "--output=/tmp/x"},
	} {
		if _, err := Mint(r.Root(), c.q, c.pin); !errors.Is(err, ErrRefused) {
			t.Errorf("Mint(%q, %q) = %v, want a refusal", c.q, c.pin, err)
		}
	}
	if ents, _ := os.ReadDir(filepath.Join(home, ".abcd", "lab")); len(ents) > 1 {
		t.Errorf("a refused mint left lab homes behind: %v", ents)
	}
}

// Hand-run labs sit at the top of the store with their own index; the verb
// works in the root-commit lane and never writes them.
func TestMintNeverWritesTheHandRunTopLevelIndex(t *testing.T) {
	r, home := fixture(t)
	legacy := filepath.Join(home, ".abcd", "lab", "index.jsonl")
	write(t, legacy, `{"id": "lab-260831131349-976575f"}`+"\n")
	mint(t, r)
	if got := read(t, legacy); got != `{"id": "lab-260831131349-976575f"}`+"\n" {
		t.Errorf("the hand-run index changed: %q", got)
	}
}

func TestMintRefusesASymlinkedStore(t *testing.T) {
	r, home := fixture(t)
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(home, ".abcd", "lab")); err != nil {
		t.Fatal(err)
	}
	if _, err := Mint(r.Root(), "a question", ""); !errors.Is(err, ErrRefused) {
		t.Fatalf("Mint through a symlinked store = %v, want a refusal", err)
	}
	if ents, _ := os.ReadDir(elsewhere); len(ents) != 0 {
		t.Errorf("mint wrote through the symlink: %v", ents)
	}
}

func TestVerbsRefuseAnIDThatIsNotALab(t *testing.T) {
	r, _ := fixture(t)
	mint(t, r)
	for _, id := range []string{"../../etc", "lab-260925101112-zzzzzzz", "lab-999999999999-abcdef0"} {
		if _, err := Preflight(r.Root(), id); !errors.Is(err, ErrRefused) {
			t.Errorf("Preflight(%q) = %v, want a refusal", id, err)
		}
		if _, err := Record(r.Root(), id, "p1"); !errors.Is(err, ErrRefused) {
			t.Errorf("Record(%q) = %v, want a refusal", id, err)
		}
	}
}

// Criteria 2 and 5: a failed check halts the lab naming it, and the refusal is
// recorded as a finding rather than adapted around.
func TestPreflightHaltsNamingTheFailedCheckAndRecordsIt(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)

	res, err := Preflight(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) {
		t.Fatalf("Preflight with no work binary = %v, want a halt", err)
	}
	if !strings.Contains(err.Error(), "binary.work") {
		t.Errorf("the halt does not name the failed check: %v", err)
	}
	failed := map[string]bool{}
	for _, c := range res.Checks {
		if !c.OK {
			failed[c.ID] = true
		}
	}
	if !failed["binary.work"] || failed["isolation.home"] || failed["isolation.snapshot"] || failed["isolation.remotes"] || failed["isolation.hooks"] {
		t.Errorf("failed checks = %v, want binary.work and no isolation failure on a fresh mint", failed)
	}
	art := read(t, filepath.Join(dir, "state", "preflight.md"))
	if !strings.Contains(art, "| binary.work | dual-binary | FAIL |") || !strings.Contains(art, "| isolation.hooks | harness-isolation | pass |") {
		t.Errorf("preflight artefact does not record both check groups:\n%s", art)
	}
	fs := parseFindings(read(t, filepath.Join(dir, "findings.md")))
	if len(fs) != 1 || fs[0].Kind != KindGate || fs[0].ID != res.Finding || !strings.Contains(fs[0].Gate, "binary.work") {
		t.Fatalf("findings = %+v, want one gate finding naming binary.work", fs)
	}

	// The same refusal again is the same finding, not a second one.
	if _, err := Preflight(r.Root(), m.ID); !errors.Is(err, ErrHalted) {
		t.Fatal(err)
	}
	if n := len(parseFindings(read(t, filepath.Join(dir, "findings.md")))); n != 1 {
		t.Errorf("a repeated refusal logged %d findings, want 1", n)
	}

	// Halted, the lab records no probe: the refusal is not adapted around.
	if _, err := Record(r.Root(), m.ID, "p1"); !errors.Is(err, ErrHalted) {
		t.Errorf("Record on a halted lab = %v, want a halt refusal", err)
	}
	ls, err := List(r.Root())
	if err != nil || len(ls.Labs) != 1 || strings.Join(ls.Labs[0].Halted, ",") != "preflight" {
		t.Errorf("List = %+v (%v), want the lab shown halted by its preflight", ls, err)
	}
}

func TestPreflightPassesLiftsTheHaltAndPinsTheWorkBinary(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	if _, err := Preflight(r.Root(), m.ID); !errors.Is(err, ErrHalted) {
		t.Fatal(err)
	}
	stubVintage(t, m.Pin, true)
	write(t, filepath.Join(dir, "bin", "abcd"), "work binary v1")
	res, err := Preflight(r.Root(), m.ID)
	if err != nil || !res.Passed {
		t.Fatalf("Preflight = %+v, %v; want a pass", res, err)
	}
	if res.Lifted != "F-1" {
		t.Errorf("Lifted = %q, want the halt's finding F-1", res.Lifted)
	}
	if _, err := os.Stat(filepath.Join(dir, "state", "halt-preflight.json")); !os.IsNotExist(err) {
		t.Errorf("the halt still stands after a pass: %v", err)
	}
	if n := len(parseFindings(read(t, filepath.Join(dir, "findings.md")))); n != 1 {
		t.Errorf("the finding must stay recorded after the halt lifts; have %d", n)
	}
	if _, err := Record(r.Root(), m.ID, "p1"); err != nil {
		t.Errorf("Record after the halt lifted: %v", err)
	}

	// A mid-lab rebuild of the work binary is a provenance event: refused.
	write(t, filepath.Join(dir, "bin", "abcd"), "work binary v2")
	res, err = Preflight(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) || failedIDs(res) != "binary.pinned" {
		t.Errorf("Preflight after a rebuild = %v, failed %q; want binary.pinned", err, failedIDs(res))
	}
}

func failedIDs(res Preflighted) string {
	var out []string
	for _, c := range res.Checks {
		if !c.OK {
			out = append(out, c.ID)
		}
	}
	return strings.Join(out, ",")
}

func TestPreflightRefusesAWorkBinaryThatIsNotThePristinePin(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	write(t, filepath.Join(dir, "bin", "abcd"), "bytes")
	for _, c := range []struct {
		rev   string
		known bool
		want  string
	}{
		{"0123456789abcdef0123456789abcdef01234567", true, "is not the pin"},
		{m.Pin, false, "modified tree"},
		{"", false, "no vcs stamp"},
	} {
		stubVintage(t, c.rev, c.known)
		res, err := Preflight(r.Root(), m.ID)
		if !errors.Is(err, ErrHalted) {
			t.Fatalf("rev %q: %v, want a halt", c.rev, err)
		}
		for _, ch := range res.Checks {
			if ch.ID == "binary.work" && (ch.OK || !strings.Contains(ch.Detail, c.want)) {
				t.Errorf("rev %q known %v: binary.work = %+v, want a failure saying %q", c.rev, c.known, ch, c.want)
			}
		}
	}

	// An operator-level installation linked in is never the work binary.
	if err := os.Remove(filepath.Join(dir, "bin", "abcd")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "abcd")
	write(t, outside, "operator install")
	if err := os.Symlink(outside, filepath.Join(dir, "bin", "abcd")); err != nil {
		t.Fatal(err)
	}
	stubVintage(t, m.Pin, true)
	res, _ := Preflight(r.Root(), m.ID)
	if !strings.Contains(failedIDs(res), "binary.work") {
		t.Errorf("a linked work binary passed: %+v", res.Checks)
	}
}

// The harness-isolation half: an operator-level hooks path, a remote back to a
// checkout, and a link out of the lab's HOME each fail by name.
func TestPreflightRefusesEachIsolationBreach(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	stubVintage(t, m.Pin, true)
	write(t, filepath.Join(dir, "bin", "abcd"), "work")

	operatorHooks := filepath.Join(t.TempDir(), "hooks")
	write(t, filepath.Join(home, ".gitconfig"), "[core]\n\thooksPath = "+operatorHooks+"\n")
	gitIn(t, filepath.Join(dir, "snapshot"), "remote", "add", "origin", r.Root())
	if err := os.Symlink(filepath.Join(home, ".abcd"), filepath.Join(dir, "home", ".abcd")); err != nil {
		t.Fatal(err)
	}

	res, err := Preflight(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) {
		t.Fatalf("Preflight = %v, want a halt", err)
	}
	if got := failedIDs(res); got != "isolation.home,isolation.remotes,isolation.hooks" {
		t.Errorf("failed = %q, want the three isolation breaches", got)
	}
	if strings.Contains(read(t, filepath.Join(dir, "state", "preflight.md")), home) {
		t.Error("the artefact carries the home path")
	}

	// Another account's home is never inside the lab.
	write(t, filepath.Join(home, ".gitconfig"), "[core]\n\thooksPath = ~root/hooks\n")
	res, _ = Preflight(r.Root(), m.ID)
	if !strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("a ~user hooks path passed: %+v", res.Checks)
	}

	// A hooks path inside the lab is the lab's own.
	write(t, filepath.Join(home, ".gitconfig"), "[core]\n\thooksPath = .githooks\n")
	res, _ = Preflight(r.Root(), m.ID)
	if strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("a snapshot-relative hooks path failed: %+v", res.Checks)
	}
}

// The hooks path is judged as git expands it (iss-2609261004261615): a
// %(prefix)/ value names git's install prefix, never a path in the snapshot,
// and a ~/ value is inside the lab only when git's own expansion lands there.
func TestPreflightJudgesTheHooksPathGitExpands(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	snap := filepath.Join(dir, "snapshot")
	stubVintage(t, m.Pin, true)
	write(t, filepath.Join(dir, "bin", "abcd"), "work")
	hooks := func(v string) Preflighted {
		t.Helper()
		write(t, filepath.Join(home, ".gitconfig"), "[core]\n\thooksPath = "+v+"\n")
		res, _ := Preflight(r.Root(), m.ID)
		return res
	}

	const prefixed = "%(prefix)/share/evil-hooks"
	write(t, filepath.Join(home, ".gitconfig"), "[core]\n\thooksPath = "+prefixed+"\n")
	// The operator's global configuration in force, as the check reads it.
	probe := exec.Command("git", "-C", snap, "config", "--type=path", "--get", "core.hooksPath")
	probe.Env = os.Environ()
	expanded, err := probe.Output()
	if err != nil {
		t.Fatalf("git config --type=path: %v", err)
	}
	if strings.HasPrefix(string(expanded), "%(prefix)") {
		t.Skip("this git does not expand %(prefix)/, so a session would read the path as snapshot-relative too")
	}
	if res := hooks(prefixed); !strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("a %%(prefix)/ hooks path passed: %+v", res.Checks)
	}
	if res := hooks("~/ghooks"); !strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("a home hooks path outside the lab passed: %+v", res.Checks)
	}
	inLab, err := filepath.Rel(home, filepath.Join(snap, ".githooks"))
	if err != nil {
		t.Fatal(err)
	}
	if res := hooks("~/" + filepath.ToSlash(inLab)); strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("a ~/ hooks path git expands into the lab failed: %+v", res.Checks)
	}
	// An empty value makes git look for hooks at the filesystem root.
	if res := hooks(""); !strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("an empty hooks path passed: %+v", res.Checks)
	}
	// A value git cannot expand is refused, never guessed at.
	if res := hooks("~no-such-account-zz/hooks"); !strings.Contains(failedIDs(res), "isolation.hooks") {
		t.Errorf("an unexpandable hooks path passed: %+v", res.Checks)
	}
}

func TestPreflightRefusesALinkedWorktreeSnapshot(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	snap := filepath.Join(labDir(home, m), "snapshot")
	if err := os.RemoveAll(snap); err != nil {
		t.Fatal(err)
	}
	r.Git("worktree", "add", "--detach", snap, m.Pin)
	res, _ := Preflight(r.Root(), m.ID)
	if !strings.Contains(failedIDs(res), "isolation.snapshot") {
		t.Errorf("a linked worktree passed as the snapshot: %+v", res.Checks)
	}
}

// Probe-record scaffolding.
func TestRecordScaffoldsAProbeRecordOnce(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	p, err := Record(r.Root(), m.ID, "guard-exit")
	if err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(labDir(home, m), "state", "probes", "guard-exit")
	for _, f := range []string{"input", "argv", "exit", "stdout", "stderr", "record.md"} {
		if _, err := os.Stat(filepath.Join(pdir, f)); err != nil {
			t.Errorf("probe lacks %s: %v", f, err)
		}
	}
	if !strings.Contains(read(t, filepath.Join(pdir, "record.md")), "artefact: none: bin/abcd is absent") || p.Artefact == "" {
		t.Errorf("record.md does not name the artefact observed: %+v", p)
	}
	if _, err := Record(r.Root(), m.ID, "guard-exit"); !errors.Is(err, ErrRefused) {
		t.Errorf("a second record of one probe = %v, want a refusal", err)
	}
	for _, bad := range []string{"../escape", "Upper", "a/b", ""} {
		if _, err := Record(r.Root(), m.ID, bad); !errors.Is(err, ErrRefused) {
			t.Errorf("Record(%q) = %v, want a refusal", bad, err)
		}
	}
}

// Criterion 3: every instance of a retracted pattern is listed, and an
// unapplied correction fails the sweep.
func TestSweepListsEveryInstanceAndFailsOnAnUnappliedCorrection(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	write(t, filepath.Join(dir, "corrections.md"), read(t, filepath.Join(dir, "corrections.md"))+
		"\n- retract: `the guard returned 500` it was a lowercase tool id\n- retract: `offline suites prove loading`\n")
	write(t, filepath.Join(dir, "findings.md"), read(t, filepath.Join(dir, "findings.md"))+
		"\n## F-1 Guard misfire\n\n- kind: product\n- probes: p5\n\nWe saw the guard returned 500 on the call.\n")
	write(t, filepath.Join(dir, "review-1-triage.md"), "line one\nand the guard returned 500 again\n")
	// Instrument output and the world under study are not claims.
	write(t, filepath.Join(dir, "state", "probes", "p5", "stdout"), "the guard returned 500\n")
	write(t, filepath.Join(dir, "snapshot", "notes.txt"), "the guard returned 500\n")

	res, err := Sweep(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) || res.Passed {
		t.Fatalf("Sweep = %+v, %v; want a halt", res, err)
	}
	if len(res.Corrections) != 2 {
		t.Fatalf("corrections = %+v, want two", res.Corrections)
	}
	c := res.Corrections[0]
	var got []string
	for _, in := range c.Instances {
		got = append(got, in.File)
	}
	if c.Applied || strings.Join(got, ",") != "findings.md,review-1-triage.md" {
		t.Errorf("correction 1 instances = %v, want findings.md and review-1-triage.md only", got)
	}
	if !res.Corrections[1].Applied {
		t.Errorf("correction 2 has no instance and must read applied: %+v", res.Corrections[1])
	}
	fdoc := read(t, filepath.Join(dir, "findings.md"))
	fs := parseFindings(fdoc)
	if len(fs) != 2 || fs[1].Kind != KindGate || fs[1].Gate != "sweep/correction-1" {
		t.Fatalf("findings = %+v, want the sweep's gate finding", fs)
	}
	if strings.Count(fdoc, "the guard returned 500") != 1 {
		t.Error("the halt finding quoted the retracted pattern, making itself an instance")
	}

	// Applied everywhere, the sweep passes and the halt lifts.
	write(t, filepath.Join(dir, "findings.md"), strings.Replace(fdoc, "We saw the guard returned 500 on the call.", "The tool id was lower-case.", 1))
	write(t, filepath.Join(dir, "review-1-triage.md"), "line one\n")
	res, err = Sweep(r.Root(), m.ID)
	if err != nil || !res.Passed || res.Lifted == "" {
		t.Fatalf("Sweep after applying = %+v, %v; want a pass that lifts the halt", res, err)
	}
	if !strings.Contains(read(t, filepath.Join(dir, "state", "sweep.md")), "absent, applied") {
		t.Error("the sweep artefact does not record the corrections as applied")
	}
}

// The sweep is fail-closed (iss-2609261004260522): a probe's record.md is prose
// the harvest cites and is swept; only the five capture files are instrument
// output.
func TestSweepReadsAProbeRecordButNotItsCaptureFiles(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	if _, err := Record(r.Root(), m.ID, "p5"); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "state", "probes", "p5")
	write(t, filepath.Join(dir, "corrections.md"), "- retract: `the guard returned 500`\n")
	write(t, filepath.Join(pdir, "record.md"), read(t, filepath.Join(pdir, "record.md"))+"\nthe guard returned 500 here\n")
	write(t, filepath.Join(pdir, "stdout"), "the guard returned 500\n")
	res, err := Sweep(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) || len(res.Corrections) != 1 {
		t.Fatalf("Sweep = %+v, %v; want a halt on the probe record", res, err)
	}
	if is := res.Corrections[0].Instances; len(is) != 1 || is[0].File != "state/probes/p5/record.md" {
		t.Errorf("instances = %+v, want the probe's record.md alone", is)
	}
}

// A document the sweep cannot read fails it while a correction stands to be
// checked, listed by path: over the read cap, a NUL in its head, or a link.
// With nothing retracted there is nothing to miss, and the same tree passes.
func TestSweepFailsOnADocumentItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		plant func(t *testing.T, dir string)
	}{
		{"over-cap.md", func(t *testing.T, dir string) {
			write(t, filepath.Join(dir, "over-cap.md"), strings.Repeat("x", maxDocBytes)+"\nthe guard returned 500\n")
		}},
		{"nul-head.md", func(t *testing.T, dir string) {
			write(t, filepath.Join(dir, "nul-head.md"), "\x00the guard returned 500\n")
		}},
		{"linked.md", func(t *testing.T, dir string) {
			elsewhere := filepath.Join(t.TempDir(), "elsewhere.md")
			write(t, elsewhere, "the guard returned 500\n")
			if err := os.Symlink(elsewhere, filepath.Join(dir, "linked.md")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, home := fixture(t)
			m := mint(t, r)
			dir := labDir(home, m)
			tc.plant(t, dir)
			write(t, filepath.Join(dir, "corrections.md"), "- retract: `the guard returned 500`\n")
			res, err := Sweep(r.Root(), m.ID)
			if !errors.Is(err, ErrHalted) || res.Passed {
				t.Fatalf("Sweep = %+v, %v; want a halt on the document not swept", res, err)
			}
			if len(res.NotSwept) != 1 || !strings.HasPrefix(res.NotSwept[0], tc.name+" (") {
				t.Errorf("not swept = %q, want %s listed by path", res.NotSwept, tc.name)
			}
			fs := parseFindings(read(t, filepath.Join(dir, "findings.md")))
			if len(fs) != 1 || fs[0].Gate != "sweep/unswept" {
				t.Errorf("findings = %+v, want the sweep's unswept gate finding", fs)
			}
			if !strings.Contains(read(t, filepath.Join(dir, "state", "sweep.md")), "- "+tc.name+" (") {
				t.Errorf("the artefact does not list %s as not swept", tc.name)
			}

			write(t, filepath.Join(dir, "corrections.md"), "")
			if res, err := Sweep(r.Root(), m.ID); err != nil || !res.Passed || res.Lifted == "" {
				t.Errorf("Sweep with nothing retracted = %+v, %v; want a pass that lifts the halt", res, err)
			}
		})
	}
}

func TestSweepRefusesAPatternTooShortToMean(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	write(t, filepath.Join(dir, "corrections.md"), "- retract: `ab`\n")
	res, err := Sweep(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) || res.Corrections[0].Invalid == "" {
		t.Errorf("Sweep = %+v, %v; want the short pattern refused", res, err)
	}
}

func TestSweepPassesWithNothingRetracted(t *testing.T) {
	r, _ := fixture(t)
	m := mint(t, r)
	res, err := Sweep(r.Root(), m.ID)
	if err != nil || !res.Passed || len(res.Corrections) != 0 {
		t.Errorf("Sweep on a fresh lab = %+v, %v; want a pass", res, err)
	}
}

func fillProbe(t *testing.T, dir, name string) {
	t.Helper()
	pdir := filepath.Join(dir, "state", "probes", name)
	write(t, filepath.Join(pdir, "argv"), "abcd guard check 'rm -rf /'\n")
	write(t, filepath.Join(pdir, "exit"), "2\n")
}

// Criterion 4: the harvest is assembled in the lifeboat's section shape with
// the probe records cited, and product findings are listed as capture
// candidates.
func TestHarvestAssemblesTheLifeboatShapeCitingProbes(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	for _, p := range []string{"p1", "p2"} {
		if _, err := Record(r.Root(), m.ID, p); err != nil {
			t.Fatal(err)
		}
		fillProbe(t, dir, p)
	}
	write(t, filepath.Join(dir, "findings.md"), read(t, filepath.Join(dir, "findings.md"))+`
## F-1 A fragment pattern refuses a cited author's name

- kind: product
- status: open
- probes: p1
- refutation: the banlist's own entry, shape only

The name-guard refused the merge three times.

## F-2 Real-session smoke is a mandatory stage

- kind: procedure
- status: worked
- probes: p1, p2

## F-3 The procedure transferred

- kind: result
- status: worked
- probes: p2
`)
	res, err := Harvest(r.Root(), m.ID)
	if err != nil || !res.Written {
		t.Fatalf("Harvest = %+v, %v", res, err)
	}
	doc := read(t, filepath.Join(dir, "harvest", "harvest.md"))
	last := -1
	for i, s := range []string{"## 1. Intention", "## 2. Method", "## 3. Findings — what worked", "## 4. Findings — open", "## 5. Candidates", "## 6. Coverage"} {
		at := strings.Index(doc, s)
		if at <= last {
			t.Fatalf("section %d %q missing or out of order in:\n%s", i+1, s, doc)
		}
		last = at
	}
	for _, want := range []string{"`state/probes/p1/`", "`state/probes/p2/` (exit 2)", "does the procedure transfer"} {
		if !strings.Contains(doc, want) {
			t.Errorf("harvest lacks %q", want)
		}
	}
	if len(res.Candidates) != 1 || res.Candidates[0].Finding != "F-1" ||
		res.Candidates[0].Command != "abcd capture 'A fragment pattern refuses a cited author'\\''s name' --found-during '"+m.ID+"'" {
		t.Errorf("candidates = %+v, want F-1 with its capture line", res.Candidates)
	}
	if !strings.Contains(doc, res.Candidates[0].Command) {
		t.Error("the harvest does not list the capture candidate's line")
	}
	if len(res.Amendments) != 1 || !strings.HasPrefix(res.Amendments[0], "F-2") {
		t.Errorf("amendments = %v, want F-2", res.Amendments)
	}

	// Rerunning replaces the assembled harvest; a hand-written one is refused.
	if _, err := Harvest(r.Root(), m.ID); err != nil {
		t.Errorf("rerun: %v", err)
	}
	write(t, filepath.Join(dir, "harvest", "harvest.md"), "# my own harvest\n")
	if _, err := Harvest(r.Root(), m.ID); !errors.Is(err, ErrRefused) {
		t.Errorf("Harvest over a hand-written harvest = %v, want a refusal", err)
	}
}

// No claim outlives its input.
func TestHarvestRefusesAFindingItsRecordsCannotBackAndWritesNothing(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	if _, err := Record(r.Root(), m.ID, "p1"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "findings.md"), read(t, filepath.Join(dir, "findings.md"))+
		"\n## F-1 Cites an unfilled probe\n\n- kind: product\n- probes: p1\n"+
		"\n## F-2 Cites a probe never made\n\n- kind: product\n- probes: ghost\n"+
		"\n## F-3 Cites nothing\n\n- kind: result\n")
	res, err := Harvest(r.Root(), m.ID)
	if !errors.Is(err, ErrHalted) || res.Written {
		t.Fatalf("Harvest = %+v, %v; want a refusal", res, err)
	}
	var ids []string
	for _, g := range res.Gaps {
		ids = append(ids, g.Finding)
	}
	if strings.Join(ids, ",") != "F-1,F-2,F-3" {
		t.Errorf("gaps = %+v, want F-1, F-2 and F-3", res.Gaps)
	}
	if _, err := os.Stat(filepath.Join(dir, "harvest", "harvest.md")); !os.IsNotExist(err) {
		t.Errorf("a refused harvest wrote a file: %v", err)
	}
}

// A halted lab is still harvested, its gate finding leading.
func TestHarvestOfAHaltedLabLeadsWithItsGateFinding(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	dir := labDir(home, m)
	if _, err := Preflight(r.Root(), m.ID); !errors.Is(err, ErrHalted) {
		t.Fatal(err)
	}
	res, err := Harvest(r.Root(), m.ID)
	if err != nil || strings.Join(res.Halted, ",") != "preflight" {
		t.Fatalf("Harvest = %+v, %v", res, err)
	}
	doc := read(t, filepath.Join(dir, "harvest", "harvest.md"))
	if !strings.Contains(doc, "**F-1 Halted: the preflight refused") || !strings.Contains(doc, "`state/preflight.md`") {
		t.Errorf("the harvest does not lead with the gate finding:\n%s", doc)
	}
}

func TestListWritesNothing(t *testing.T) {
	r, home := fixture(t)
	ls, err := List(r.Root())
	if err != nil || ls.StoreSeen || len(ls.Labs) != 0 {
		t.Fatalf("List = %+v, %v", ls, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !os.IsNotExist(err) {
		t.Errorf("List created the store: %v", err)
	}
}
