package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/gittest"
)

// clearIdentityEnv removes every environment override git reads for either
// role, so a case sees only the config it sets.
func clearIdentityEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "")
	}
}

// localConfig reads one key from the repository's own .git/config only.
func localConfig(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "config", "--local", "--get", key)
	cmd.Env = gittest.Env(t)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// identityPrompter answers every confirm with one fixed answer and records what
// it was asked. AtTerminal makes it a TerminalPrompter, so a case chooses
// whether a person is at a terminal.
type identityPrompter struct {
	terminal bool
	answer   bool
	asked    []string
	onAsk    func(question string)
}

func (p *identityPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, q)
	if p.onAsk != nil {
		p.onAsk(q)
	}
	return p.answer
}
func (p *identityPrompter) Prompt(_ string, _ []string, def string) string { return def }
func (p *identityPrompter) AtTerminal() bool                               { return p.terminal }

func establishCtx(dir string, p Prompter, gaps ...string) *applyCtx {
	present := map[string]bool{}
	for _, g := range gaps {
		present[g] = true
	}
	return &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{ConfigChange: true},
		gapPresent: present,
		prompter:   p,
	}
}

const pinAlex = `{"name":"Alex Reppel","email":"alex@example.com"}`

// TestDetectGitIdentity_CommitterDiverges: the committer gap is raised beside
// the author statuses — here the author matches the pin, so it is the only gap.
func TestDetectGitIdentity_CommitterDiverges(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	idWritePin(t, dir, pinAlex)
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	gaps := detectGitIdentity(dir)
	if len(gaps) != 1 || gaps[0].ID != CommitterGapID || !gaps[0].Required || !gaps[0].Resolvable {
		t.Fatalf("want one required, resolvable %s gap, got %+v", CommitterGapID, gaps)
	}
	if !strings.Contains(gaps[0].Detail, "Test User") {
		t.Fatalf("the committer gap does not name the committer: %+v", gaps[0])
	}
}

// TestDetectGitIdentity_CommitterUnpinnedIsAdvisory: an un-pinned repo has not
// adopted the gate, so an author≠committer divergence is reported but never
// required work.
func TestDetectGitIdentity_CommitterUnpinnedIsAdvisory(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	gaps := detectGitIdentity(dir)
	var got *Gap
	for i := range gaps {
		if gaps[i].ID == CommitterGapID {
			got = &gaps[i]
		}
	}
	if got == nil || got.Required {
		t.Fatalf("want an advisory %s gap in an un-pinned repo, got %+v", CommitterGapID, gaps)
	}
}

// TestDetectGitIdentity_ToolIdentity is criterion 4: the routine's tool
// identity is detected and reported as a divergence, and the report points at
// the runner for establishing the human identity rather than writing it.
func TestDetectGitIdentity_ToolIdentity(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Claude", "noreply@anthropic.com")
	gaps := detectGitIdentity(dir)
	var got *Gap
	for i := range gaps {
		if gaps[i].ID == ToolIdentityGapID {
			got = &gaps[i]
		}
	}
	if got == nil || !got.Required {
		t.Fatalf("want a required %s gap for the harness default identity, got %+v", ToolIdentityGapID, gaps)
	}
	if !strings.Contains(got.Detail, "Claude <noreply@anthropic.com>") || !strings.Contains(got.FixHint, "iss-2608210932052003") {
		t.Fatalf("the tool-identity gap must name the identity and point at the runner: %+v", *got)
	}
}

// TestDetectGitIdentity_ForgeCommitterIsNotATool mirrors the attribution
// gate's role asymmetry: the forge's own committer stamp is not flagged.
func TestDetectGitIdentity_ForgeCommitterIsNotATool(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "GitHub")
	t.Setenv("GIT_COMMITTER_EMAIL", "noreply@github.com")
	for _, g := range detectGitIdentity(dir) {
		if g.ID == ToolIdentityGapID {
			t.Fatalf("the forge's committer stamp was flagged as a tool identity: %+v", g)
		}
	}
}

// TestStepGitIdentity_ProposesPinAndWritesOnlyOnConfirm is criterion 1: a
// repo-local override that differs from the pin is met with the pinned
// identity as the proposal, and repo-local config is written only after the
// person confirms — at the moment of asking, nothing has changed yet.
func TestStepGitIdentity_ProposesPinAndWritesOnlyOnConfirm(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, pinAlex)
	p := &identityPrompter{terminal: true, answer: true}
	p.onAsk = func(string) {
		if got := localConfig(t, dir, "user.name"); got != "Test User" {
			t.Fatalf("config was rewritten before the person confirmed: user.name=%q", got)
		}
	}
	a := establishCtx(dir, p, MismatchGapID)
	a.stepGitIdentity()
	if len(p.asked) != 1 || !strings.Contains(p.asked[0], "Alex Reppel <alex@example.com>") {
		t.Fatalf("want one question proposing the pinned identity, got %q", p.asked)
	}
	if n, e := localConfig(t, dir, "user.name"), localConfig(t, dir, "user.email"); n != "Alex Reppel" || e != "alex@example.com" {
		t.Fatalf("confirmed proposal not written: %q <%s>", n, e)
	}
	if res, err := identity.Check(dir); err != nil || res.Status != identity.StatusOK {
		t.Fatalf("after the confirmed write the gate should pass: %v %v", res.Status, err)
	}
	if len(a.writes) != 1 {
		t.Fatalf("the write must be reported, got %v", a.writes)
	}
}

// TestStepGitIdentity_DeclineWritesNothing is criterion 5's declined half.
func TestStepGitIdentity_DeclineWritesNothing(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, pinAlex)
	before, _ := os.ReadFile(filepath.Join(dir, ".git", "config"))
	p := &identityPrompter{terminal: true, answer: false}
	a := establishCtx(dir, p, MismatchGapID)
	a.stepGitIdentity()
	after, _ := os.ReadFile(filepath.Join(dir, ".git", "config"))
	if string(before) != string(after) || len(a.writes) != 0 {
		t.Fatalf("a declined proposal changed git config or reported a write:\n%s", after)
	}
	if len(p.asked) != 1 {
		t.Fatalf("want the proposal asked once, got %q", p.asked)
	}
}

// TestStepGitIdentity_NoTerminalFailsClosed is criterion 3: with no terminal
// the step never asks (so it can never block on an unanswerable prompt), writes
// nothing — even when the piped answer would have been yes — and says why.
func TestStepGitIdentity_NoTerminalFailsClosed(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, pinAlex)
	before, _ := os.ReadFile(filepath.Join(dir, ".git", "config"))
	for _, p := range []Prompter{&identityPrompter{terminal: false, answer: true}, RefusingPrompter{}} {
		a := establishCtx(dir, p, MismatchGapID)
		a.stepGitIdentity()
		if sp, ok := p.(*identityPrompter); ok && len(sp.asked) != 0 {
			t.Fatalf("a run with no terminal was asked %q", sp.asked)
		}
		after, _ := os.ReadFile(filepath.Join(dir, ".git", "config"))
		if string(before) != string(after) || len(a.writes) != 0 {
			t.Fatalf("a run with no terminal changed git config:\n%s", after)
		}
		if !strings.Contains(strings.Join(a.notes, "\n"), "no terminal") {
			t.Fatalf("the fail-closed refusal was not said: %q", a.notes)
		}
	}
}

// TestStepGitIdentity_NeverUnderYes: --yes approves categories, never a change
// of who commits.
func TestStepGitIdentity_NeverUnderYes(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, pinAlex)
	p := &identityPrompter{terminal: true, answer: true}
	a := establishCtx(dir, p, MismatchGapID)
	a.autoYes = true
	a.stepGitIdentity()
	if len(p.asked) != 0 || localConfig(t, dir, "user.name") != "Test User" {
		t.Fatalf("--yes asked or wrote: asked=%q user.name=%q", p.asked, localConfig(t, dir, "user.name"))
	}
}

// TestStepGitIdentity_ProposesGlobalWhenUnpinned: with no pin the proposal is
// the global git identity, read from disk (adr-38: no network lookup).
func TestStepGitIdentity_ProposesGlobalWhenUnpinned(t *testing.T) {
	clearIdentityEnv(t)
	dir := t.TempDir()
	global, _ := gittest.SplitIdentity(t, dir)
	idMustGit(t, dir, "config", "--local", "user.name", "Claude")
	idMustGit(t, dir, "config", "--local", "user.email", "noreply@anthropic.com")
	p := &identityPrompter{terminal: true, answer: true}
	a := establishCtx(dir, p, ToolIdentityGapID)
	a.stepGitIdentity()
	want := global.Name + " <" + global.Email + ">"
	if len(p.asked) != 1 || !strings.Contains(p.asked[0], want) {
		t.Fatalf("want the global identity %s proposed, got %q", want, p.asked)
	}
	if localConfig(t, dir, "user.name") != global.Name {
		t.Fatalf("confirmed global proposal not written: %q", localConfig(t, dir, "user.name"))
	}
}

// TestStepGitIdentity_EnvOverrideIsNotRewritten: an environment override
// outranks any config, so writing repo-local config would change nothing. The
// step says so and neither asks nor writes.
func TestStepGitIdentity_EnvOverrideIsNotRewritten(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	idWritePin(t, dir, pinAlex)
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	p := &identityPrompter{terminal: true, answer: true}
	a := establishCtx(dir, p, CommitterGapID)
	a.stepGitIdentity()
	if len(p.asked) != 0 || len(a.writes) != 0 {
		t.Fatalf("an env override cannot be fixed by config, yet the step asked %q / wrote %v", p.asked, a.writes)
	}
	if !strings.Contains(strings.Join(a.notes, "\n"), "GIT_COMMITTER_NAME") {
		t.Fatalf("the note must name the overriding variable: %q", a.notes)
	}
}

// TestStepIdentityPin_RefusesAToolIdentity: adopting the gate must never
// canonicalise the harness's own identity as the repo's human.
func TestStepIdentityPin_RefusesAToolIdentity(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Claude", "noreply@anthropic.com")
	a := establishCtx(dir, &identityPrompter{terminal: true, answer: true}, OptionalPinGapID)
	a.stepIdentityPin()
	if _, ok, _ := identity.LoadPin(dir); ok {
		t.Fatal("a tool identity was pinned as the repo's human")
	}
	if len(a.notes) == 0 {
		t.Fatal("the refusal to pin a tool identity was not said")
	}
}
