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

func idGitRepo(t *testing.T, name, email string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	idMustGit(t, dir, "init")
	if name != "" {
		idMustGit(t, dir, "config", "user.name", name)
	}
	if email != "" {
		idMustGit(t, dir, "config", "user.email", email)
	}
	return dir
}

func idMustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// idWritePin records body as the person's own machine-wide identity record,
// under the test's HOME (gittest.Env pins it to a per-test temp dir). Nothing
// is written into the repository: it names no one (itd-2610240918547728).
func idWritePin(t *testing.T, _ string, body string) {
	t.Helper()
	gittest.Env(t)
	p := filepath.Join(os.Getenv("HOME"), filepath.FromSlash(identity.MachineRecordRef().Rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// idCommitPin writes body as the identity pin a repository committed before
// itd-2610240918547728.
func idCommitPin(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "identity.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectGitIdentity_Mismatch(t *testing.T) {
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	gaps := detectGitIdentity(dir)
	if len(gaps) != 1 || gaps[0].ID != "git_identity.mismatch" || !gaps[0].Required {
		t.Fatalf("want one required git_identity.mismatch gap, got %+v", gaps)
	}
	// Criterion 3: the refusal names the person's own record and how to change it.
	if !strings.Contains(gaps[0].FixHint, identity.MachineRecordRef().Display) {
		t.Fatalf("the mismatch gap does not say how to change the record: %+v", gaps[0])
	}
}

func TestDetectGitIdentity_Match(t *testing.T) {
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	idWritePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	if gaps := detectGitIdentity(dir); len(gaps) != 0 {
		t.Fatalf("want no gap on match, got %+v", gaps)
	}
}

func TestDetectGitIdentity_Unrecorded(t *testing.T) {
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	gaps := detectGitIdentity(dir)
	if len(gaps) != 1 || gaps[0].ID != UnrecordedGapID || gaps[0].Required || !gaps[0].Resolvable || gaps[0].Category != MachineIdentity {
		t.Fatalf("want one advisory, resolvable %s gap, got %+v", UnrecordedGapID, gaps)
	}
}

// Criterion 2: a contributor in a repository that still commits the
// maintainer's identity is shown nothing that names the maintainer, is not
// offered the move, and is checked against their own record.
func TestDetectGitIdentity_ContributorIsNeverShownTheMaintainer(t *testing.T) {
	dir := idGitRepo(t, "Second Contributor", "second@example.com")
	idCommitPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	gaps := detectGitIdentity(dir)
	for _, g := range gaps {
		if strings.Contains(g.Title+g.Detail+g.FixHint, "Maintainer") || strings.Contains(g.Title+g.Detail+g.FixHint, "maintainer@") {
			t.Fatalf("a gap names the maintainer to a contributor: %+v", g)
		}
		if g.Required {
			t.Fatalf("a contributor's own human identity raised required work: %+v", g)
		}
	}
	pin := gapByID(gaps, LegacyPinGapID)
	if pin == nil || pin.Resolvable {
		t.Fatalf("want a non-resolvable %s note on a contributor's machine, got %+v", LegacyPinGapID, gaps)
	}
	if gapByID(gaps, UnrecordedGapID) == nil {
		t.Fatalf("the contributor is not offered a record of their own: %+v", gaps)
	}
}

func TestDetectGitIdentity_OwnerIsOfferedTheMove(t *testing.T) {
	dir := idGitRepo(t, "A Maintainer", "maintainer@example.com")
	idCommitPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	pin := gapByID(detectGitIdentity(dir), LegacyPinGapID)
	if pin == nil || !pin.Resolvable || pin.Required {
		t.Fatalf("want an advisory, resolvable %s gap on the owner's machine, got %+v", LegacyPinGapID, pin)
	}
}

// Criterion 1: at a terminal, a person with no record is offered their usual
// git identity as its own question, and it is recorded on their machine only
// on a yes.
func TestStepIdentityRecord_RecordsOnYesAtATerminal(t *testing.T) {
	dir := idGitRepo(t, "Alex Reppel", "alex@example.com")
	p := &identityPrompter{terminal: true, answer: true}
	a := &applyCtx{cwd: dir, approved: map[GapCategory]bool{}, gapPresent: map[string]bool{UnrecordedGapID: true}, prompter: p}
	a.stepIdentityRecord()
	if len(p.asked) != 1 || !strings.Contains(p.asked[0], "Alex Reppel <alex@example.com>") {
		t.Fatalf("want one question proposing the usual identity, got %q", p.asked)
	}
	got, _, ok, err := identity.LoadRecord(os.Getenv("HOME"), "")
	if err != nil || !ok || got.Name != "Alex Reppel" || got.Email != "alex@example.com" {
		t.Fatalf("record not written: %+v ok=%v err=%v", got, ok, err)
	}
	if len(a.writes) != 1 || a.writes[0] != identity.MachineRecordRef().Display {
		t.Fatalf("expected one noted write of the record, got %v", a.writes)
	}
}

func TestStepIdentityRecord_WritesNothingOffATerminalUnderYesOrOnNo(t *testing.T) {
	for name, a := range map[string]func(dir string, p *identityPrompter) *applyCtx{
		"off a terminal": func(dir string, p *identityPrompter) *applyCtx {
			p.terminal = false
			return &applyCtx{cwd: dir, gapPresent: map[string]bool{UnrecordedGapID: true}, prompter: p}
		},
		"under --yes": func(dir string, p *identityPrompter) *applyCtx {
			return &applyCtx{cwd: dir, gapPresent: map[string]bool{UnrecordedGapID: true}, prompter: p, autoYes: true}
		},
		"declined": func(dir string, p *identityPrompter) *applyCtx {
			p.answer = false
			return &applyCtx{cwd: dir, gapPresent: map[string]bool{UnrecordedGapID: true}, prompter: p}
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := idGitRepo(t, "Test User", "test@example.com")
			p := &identityPrompter{terminal: true, answer: true}
			ctx := a(dir, p)
			ctx.stepIdentityRecord()
			if _, _, ok, _ := identity.LoadRecord(os.Getenv("HOME"), ""); ok {
				t.Fatal("a record was written without a yes at a terminal")
			}
			if name != "declined" && len(p.asked) != 0 {
				t.Fatalf("asked %q", p.asked)
			}
		})
	}
}

// Criterion 4 at the install: on the owner's machine, at a terminal, a yes
// moves the pin to their machine and out of the repository, and says so.
func TestStepIdentityMigrate_MovesThePinOnYes(t *testing.T) {
	dir := idGitRepo(t, "A Maintainer", "maintainer@example.com")
	idCommitPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com","production_mode":"scribe-transcribed"}`)
	p := &identityPrompter{terminal: true, answer: true}
	a := &applyCtx{cwd: dir, gapPresent: map[string]bool{LegacyPinGapID: true}, prompter: p}
	a.stepIdentityMigrate()
	if len(p.asked) != 1 {
		t.Fatalf("want the move asked once, got %q", p.asked)
	}
	if _, err := os.Stat(filepath.Join(dir, identity.LegacyPinRelPath)); !os.IsNotExist(err) {
		t.Fatalf("the pin is still in the repository: %v", err)
	}
	if res, err := identity.Check(dir); err != nil || res.Status != identity.StatusOK {
		t.Fatalf("after the move the owner's record does not decide: %v %v", res.Status, err)
	}
	if len(a.writes) != 3 {
		t.Fatalf("want the record, the production mode and the removal said, got %v", a.writes)
	}
}

func TestStepIdentityMigrate_OffATerminalMovesNothing(t *testing.T) {
	dir := idGitRepo(t, "A Maintainer", "maintainer@example.com")
	idCommitPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	p := &identityPrompter{terminal: false, answer: true}
	a := &applyCtx{cwd: dir, gapPresent: map[string]bool{LegacyPinGapID: true}, prompter: p}
	a.stepIdentityMigrate()
	if len(p.asked) != 0 {
		t.Fatalf("asked off a terminal: %q", p.asked)
	}
	if _, err := os.Stat(filepath.Join(dir, identity.LegacyPinRelPath)); err != nil {
		t.Fatalf("the pin was moved off a terminal: %v", err)
	}
}

// A committed pin the guarded reader refuses (here a symlink to a pin inside
// the working tree) is reported on the present-but-unreadable path,
// git_identity.uncheckable, and nothing is read through it.
func TestDetectGitIdentity_SymlinkedPinIsUncheckable(t *testing.T) {
	dir := idGitRepo(t, "A Maintainer", "maintainer@example.com")
	if err := os.WriteFile(filepath.Join(dir, "elsewhere.json"), []byte(`{"name":"A Maintainer","email":"maintainer@example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "..", "elsewhere.json"), filepath.Join(dir, ".abcd", "config", "identity.json")); err != nil {
		t.Fatal(err)
	}
	gaps := detectGitIdentity(dir)
	if g := gapByID(gaps, UncheckableGapID); g == nil || g.Resolvable {
		t.Fatalf("want a non-resolvable %s gap for a symlinked pin, got %+v", UncheckableGapID, gaps)
	}
	if gapByID(gaps, LegacyPinGapID) != nil {
		t.Fatalf("a symlinked pin was read and offered for the move: %+v", gaps)
	}
}
