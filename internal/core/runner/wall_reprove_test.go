package runner

// wall_reprove_test.go holds what a drive does when the harness file its run proved is
// gone or changed: a new file signed by the same team is proved once, before any stage,
// and anything else stops the run with nothing run.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// reproveWorld is a drive that has passed its proof on one harness file, reached through
// a link, with a second version installed beside it.
type reproveWorld struct {
	f             fakeEnv
	c             *ClaudeCLI
	p             Profile
	link          string
	first, second string
	which         string
	// signed is each file's signing team, "" or absent for an unsigned one.
	signed map[string]string
}

// testTeam and otherTeam are signing teams a test's files carry.
const (
	testTeam  = "TEAMAAAAAA"
	otherTeam = "TEAMBBBBBB"
)

// useSigning has the drive read teams from signed.
func useSigning(t *testing.T, signed map[string]string) {
	t.Helper()
	was := Signing
	t.Cleanup(func() { Signing = was })
	Signing = HarnessSigning{
		Team: func(path string) string { return signed[path] },
		Verify: func(path, team string) error {
			if team == "" || signed[path] != team {
				return errors.New("not that team's")
			}
			return nil
		},
	}
}

// harnessScript is a stand-in harness: it logs its name, then runs the fake as version,
// with extra lines before that.
func harnessScript(t *testing.T, path, name, version, extra string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' " + name + " >> \"$" + fakeLogEnv + "/which\"\n" + extra +
		fakeVersionEnv + "=" + version + "; export " + fakeVersionEnv + "\nexec " + shellquote.Single(self) + " \"$@\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func newReproveWorld(t *testing.T, secondExtra string) *reproveWorld {
	t.Helper()
	f := newFake(t, "wall-ok", Claude)
	versions, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := &reproveWorld{f: f, link: filepath.Join(f.bin, Claude), first: filepath.Join(versions, "first"), second: filepath.Join(versions, "second"), which: filepath.Join(f.log, "which")}
	harnessScript(t, w.first, "first", "2.1.319", "")
	harnessScript(t, w.second, "second", "2.1.320", secondExtra)
	w.signed = map[string]string{w.first: testTeam, w.second: testTeam}
	useSigning(t, w.signed)
	w.repoint(t, w.first)
	w.p = wallProfile(t)
	w.c = newClaude("")
	w.c.profile = &w.p
	w.c.approver = wallBin
	if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if err := CheckLogin(context.Background(), w.p); err != nil {
		t.Fatal(err)
	}
	if err := ProveWall(context.Background(), w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.which); err != nil {
		t.Fatal(err)
	}
	return w
}

func (w *reproveWorld) repoint(t *testing.T, to string) {
	t.Helper()
	if err := os.Remove(w.link); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.Symlink(to, w.link); err != nil {
		t.Fatal(err)
	}
}

// update is the installer's act: the link moves to the second version, the first goes.
func (w *reproveWorld) update(t *testing.T) {
	t.Helper()
	w.repoint(t, w.second)
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
}

// ran is each launch since the last look, by name, and forgets them.
func (w *reproveWorld) ran(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(w.which)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.which); err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(raw))
}

// TestAHarnessGoneIsProvedAgainAndTheNextStageRunsTheNewFile: nothing starts until the
// new file has passed; its one proof names both versions; a later check costs no launch.
func TestAHarnessGoneIsProvedAgainAndTheNextStageRunsTheNewFile(t *testing.T) {
	w := newReproveWorld(t, "")
	if HarnessOwed(w.p) {
		t.Fatal("the file the run proved is there")
	}
	if said, err := ReproveHarness(context.Background(), w.p, w.c.approver); said != "" || err != nil || len(w.ran(t)) != 0 {
		t.Fatalf("with the file there nothing is proved: %q %v", said, err)
	}
	w.update(t)
	if !HarnessOwed(w.p) {
		t.Fatal("the file the run proved is gone")
	}
	t.Setenv(fakeModeEnv, "ok")
	if _, _, err := w.c.Run(context.Background(), shellRequest(w.f)); err == nil {
		t.Fatal("no stage starts before the new file has passed")
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing started: %q", got)
	}
	t.Setenv(fakeModeEnv, "wall-ok")
	said, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(said, "\n") != 0 || !strings.Contains(said, "2.1.319") || !strings.Contains(said, "2.1.320") {
		t.Fatalf("one line naming both versions: %q", said)
	}
	if got := w.ran(t); !slices.Equal(got, []string{"second", "second"}) {
		t.Fatalf("the login check and the one proof, on the new file: %q", got)
	}
	if HarnessOwed(w.p) {
		t.Fatal("the new file has passed")
	}
	if err := CheckWallSettings(w.p, w.c.approver); err != nil {
		t.Fatalf("the check holds on the new file: %v", err)
	}
	if said, err := ReproveHarness(context.Background(), w.p, w.c.approver); said != "" || err != nil || len(w.ran(t)) != 0 {
		t.Fatalf("once: %q %v", said, err)
	}
	t.Setenv(fakeModeEnv, "ok")
	if _, _, err := w.c.Run(context.Background(), shellRequest(w.f)); err != nil {
		t.Fatal(err)
	}
	if got := w.ran(t); !slices.Equal(got, []string{"second"}) {
		t.Fatalf("the next stage runs the new file: %q", got)
	}
	if got := argWith(w.f.argv(t, Claude), "--mcp-config="); !strings.Contains(got, "--harness="+w.second) {
		t.Errorf("the approver is told the new file: %s", got)
	}
}

// TestAHarnessThatFailsItsProofStartsNothing: it has three tries, spaced minutes apart, the
// last refusal naming the check and the new version, and no stage starts on it.
func TestAHarnessThatFailsItsProofStartsNothing(t *testing.T) {
	w := newReproveWorld(t, fakeModeEnv+"=wall-tools; export "+fakeModeEnv+"\n")
	w.update(t)
	for try := 1; try <= 2; try++ {
		_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
		var none *NoVerdictError
		if !errors.As(err, &none) || none.RetryAfter < time.Minute || !strings.Contains(none.Detail, "2.1.320") {
			t.Fatalf("try %d waits, naming the version: %v", try, err)
		}
	}
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var we *WallError
	var none *NoVerdictError
	if errors.As(err, &none) || !errors.As(err, &we) || we.Wall != WallTools || !strings.Contains(err.Error(), "2.1.320") {
		t.Fatalf("the refusal names the check and the version: %v", err)
	}
	w.ran(t)
	if !HarnessOwed(w.p) {
		t.Fatal("the new file has not passed")
	}
	if err := CheckWallSettings(w.p, w.c.approver); err == nil {
		t.Fatal("the check does not hold on a file that failed")
	}
	t.Setenv(fakeModeEnv, "ok")
	if _, _, err := w.c.Run(context.Background(), shellRequest(w.f)); err == nil {
		t.Fatal("no stage starts")
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing started: %q", got)
	}
}

// TestAFailedProofCanPassOnALaterTry, and the tries are one file's: a new file has its own.
func TestAFailedProofCanPassOnALaterTry(t *testing.T) {
	w := newReproveWorld(t, "")
	third := filepath.Join(filepath.Dir(w.second), "third")
	failing := fakeModeEnv + "=wall-tools; export " + fakeModeEnv + "\n"
	harnessScript(t, w.second, "second", "2.1.320", failing)
	w.update(t)
	for try := 1; try <= 2; try++ {
		var none *NoVerdictError
		if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); !errors.As(err, &none) {
			t.Fatalf("try %d: %v", try, err)
		}
	}
	harnessScript(t, third, "third", "2.1.321", failing)
	w.signed[third] = testTeam
	w.repoint(t, third)
	if err := os.Remove(w.second); err != nil {
		t.Fatal(err)
	}
	var none *NoVerdictError
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); !errors.As(err, &none) {
		t.Fatalf("a new file starts its tries again: %v", err)
	}
	harnessScript(t, w.first, "first", "2.1.322", "")
	w.signed[w.first] = testTeam
	w.repoint(t, w.first)
	if err := os.Remove(third); err != nil {
		t.Fatal(err)
	}
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if HarnessOwed(w.p) {
		t.Fatal("it passed")
	}
}

// TestAFirstProofHasOneTry: the tries are a re-proof's.
func TestAFirstProofHasOneTry(t *testing.T) {
	w := newReproveWorld(t, "")
	t.Setenv(fakeModeEnv, "wall-tools")
	err := ProveWall(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	var we *WallError
	if errors.As(err, &none) || !errors.As(err, &we) {
		t.Fatalf("a verdict: %v", err)
	}
}

// TestAHarnessChangedInPlaceStopsTheRun: a file's contents changing at the same path is no
// update, whatever it reports itself as, so nothing proves it or runs it.
func TestAHarnessChangedInPlaceStopsTheRun(t *testing.T) {
	for name, version := range map[string]string{"a new version": "2.1.321", "the same version": "2.1.319"} {
		t.Run(name, func(t *testing.T) {
			w := newReproveWorld(t, "")
			harnessScript(t, w.first, "in-place", version, "")
			if HarnessOwed(w.p) {
				t.Fatal("a file changed in place owes no proof")
			}
			if said, err := ReproveHarness(context.Background(), w.p, w.c.approver); said != "" || err != nil {
				t.Fatalf("nothing to prove: %q %v", said, err)
			}
			var we *WallError
			if err := CheckWallSettings(w.p, w.c.approver); !errors.As(err, &we) || !strings.Contains(err.Error(), "changed while the drive ran") {
				t.Fatalf("the check stops the run: %v", err)
			}
			if got := w.ran(t); len(got) != 0 {
				t.Fatalf("nothing ran: %q", got)
			}
		})
	}
}

// TestAHarnessChangedDuringItsProofIsNotPassed: the proof read a file that is no longer
// there, so it says nothing about what would run.
func TestAHarnessChangedDuringItsProofIsNotPassed(t *testing.T) {
	w := newReproveWorld(t, "case \"$*\" in *include-hook-events*) printf '# changed\\n' >> \"$0\";; esac\n")
	w.update(t)
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err == nil {
		t.Fatal("the file changed under its proof")
	}
	if !HarnessOwed(w.p) {
		t.Fatal("what is there has not passed")
	}
	w.ran(t)
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err == nil {
		t.Fatal("a file changed at its path is not proved a second time")
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing ran: %q", got)
	}
}

// TestANewHarnessWithNoIdentityStopsTheRun: a start that recorded no team, a new file of
// another team and an unsigned one each stop, say why and what to do, and run nothing.
func TestANewHarnessWithNoIdentityStopsTheRun(t *testing.T) {
	for name, set := range map[string]func(w *reproveWorld, t *testing.T){
		"the start recorded no team": func(w *reproveWorld, t *testing.T) {
			delete(w.signed, w.first)
			if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
				t.Fatal(err)
			}
		},
		"another team":     func(w *reproveWorld, _ *testing.T) { w.signed[w.second] = otherTeam },
		"an unsigned file": func(w *reproveWorld, _ *testing.T) { delete(w.signed, w.second) },
	} {
		t.Run(name, func(t *testing.T) {
			w := newReproveWorld(t, "")
			set(w, t)
			w.update(t)
			said, err := ReproveHarness(context.Background(), w.p, w.c.approver)
			var we *WallError
			if said != "" || !errors.As(err, &we) || we.Wall != WallSettingsFile || !strings.Contains(err.Error(), "signing team") || !strings.Contains(err.Error(), "start the drive again") {
				t.Fatalf("the stop says why and what to do: %q %v", said, err)
			}
			if got := w.ran(t); len(got) != 0 {
				t.Fatalf("nothing ran: %q", got)
			}
			if err := CheckWallSettings(w.p, w.c.approver); err == nil {
				t.Fatal("no stage starts")
			}
		})
	}
}

// TestAStartReadsATeamOnlyWhereItsSignatureHolds: what a signature claims is not recorded
// until the signature is verified.
func TestAStartReadsATeamOnlyWhereItsSignatureHolds(t *testing.T) {
	w := newReproveWorld(t, "")
	Signing.Verify = func(string, string) error { return errors.New("does not hold") }
	if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	Signing.Verify = func(string, string) error { return nil }
	w.update(t)
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err == nil || !strings.Contains(err.Error(), "signing team") {
		t.Fatalf("no team was recorded: %v", err)
	}
}

// TestTheNewFileIsVerifiedBeforeAnythingRunsIt: against the team the start read, and with
// no launch made yet.
func TestTheNewFileIsVerifiedBeforeAnythingRunsIt(t *testing.T) {
	w := newReproveWorld(t, "")
	w.update(t)
	verified := 0
	Signing.Verify = func(path, team string) error {
		verified++
		if path != w.second || team != testTeam {
			t.Errorf("verified %s against %s", path, team)
		}
		if got := w.ran(t); len(got) != 0 {
			t.Errorf("ran before it was verified: %q", got)
		}
		return nil
	}
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if verified != 1 {
		t.Fatalf("verified %d times", verified)
	}
}

// TestAnOlderSignedHarnessIsAccepted: the team is what is held to, not the version, and
// the line says what was checked.
func TestAnOlderSignedHarnessIsAccepted(t *testing.T) {
	w := newReproveWorld(t, "")
	harnessScript(t, w.second, "second", "2.1.300", "")
	w.update(t)
	said, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2.1.319", "2.1.300", "signed by the same team", "checks a launch can show"} {
		if !strings.Contains(said, want) {
			t.Errorf("%q lacks %q", said, want)
		}
	}
	if strings.Contains(said, "full check") {
		t.Errorf("%q claims more than was checked", said)
	}
}

// TestAHarnessInTheCheckoutIsNeverProved: a new file under the repository, a lane's
// worktree, the stage's own files or the profile is refused before it is verified or run.
func TestAHarnessInTheCheckoutIsNeverProved(t *testing.T) {
	for name, where := range map[string]func(w *reproveWorld) string{
		"the repository": func(w *reproveWorld) string { return filepath.Join(w.p.Repo, "bin") },
		"a lane's worktree": func(w *reproveWorld) string {
			return abcdhome.Path(w.p.PersonHome, "worktrees", "a-root-commit", "a-run-a-lane")
		},
		"the stage's files": func(w *reproveWorld) string { return filepath.Join(w.p.StageRoot, "some-dir") },
		"the profile":       func(w *reproveWorld) string { return filepath.Join(w.p.Dir, "some-dir") },
	} {
		t.Run(name, func(t *testing.T) {
			w := newReproveWorld(t, "")
			w.p.Repo = t.TempDir()
			dir := where(w)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			inside := filepath.Join(dir, "third")
			harnessScript(t, inside, "third", "2.1.321", "")
			w.signed[inside] = testTeam
			w.repoint(t, inside)
			if err := os.Remove(w.first); err != nil {
				t.Fatal(err)
			}
			_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
			var we *WallError
			var none *NoVerdictError
			if errors.As(err, &none) || !errors.As(err, &we) || we.Wall != WallLaunch {
				t.Fatalf("refused: %v", err)
			}
			if got := w.ran(t); len(got) != 0 {
				t.Fatalf("nothing ran: %q", got)
			}
		})
	}
}

// TestAProofRunsNoHarnessFromTheCheckout: the first proof and the login check judge the
// repository as the re-proof does.
func TestAProofRunsNoHarnessFromTheCheckout(t *testing.T) {
	w := newReproveWorld(t, "")
	w.p.Repo = t.TempDir()
	inside := filepath.Join(w.p.Repo, "third")
	harnessScript(t, inside, "third", "2.1.319", "")
	w.repoint(t, inside)
	if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if err := CheckLogin(context.Background(), w.p); err == nil {
		t.Error("the login check ran it")
	}
	if err := ProveWall(context.Background(), w.p, w.c.approver); err == nil {
		t.Error("the proof ran it")
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing ran: %q", got)
	}
}

// TestAReproofStartsFromAnEmptyStageHome: what a stage left there does not reach the proof.
func TestAReproofStartsFromAnEmptyStageHome(t *testing.T) {
	w := newReproveWorld(t, "")
	left := filepath.Join(w.p.ShellHome, "left-by-a-stage")
	if err := os.WriteFile(left, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.update(t)
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(left); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the home was not emptied: %v", err)
	}
}

// TestTheSigningTeamIsReadWithTheSystemTool: its description for the team, its verifier
// for the requirement, and no team of another shape put into one.
func TestTheSigningTeamIsReadWithTheSystemTool(t *testing.T) {
	was := codesignRun
	t.Cleanup(func() { codesignRun = was })
	var calls [][]string
	output := "Executable=/a/file\nTeamIdentifier=" + testTeam + "\n"
	codesignRun = func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return []byte(strings.ReplaceAll(output, "\\n", "\n")), nil
	}
	if got := codesignTeam("/a/file"); got != testTeam {
		t.Fatalf("the team: %q", got)
	}
	if err := codesignVerify("/a/file", testTeam); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-dv", "/a/file"}; !slices.Equal(calls[0], want) {
		t.Errorf("read with %q", calls[0])
	}
	if got := calls[1]; len(got) != 3 || got[0] != "--verify" || !strings.Contains(got[1], `certificate leaf[subject.OU] = "`+testTeam+`"`) || !strings.HasPrefix(got[1], "-R=anchor apple generic") || got[2] != "/a/file" {
		t.Errorf("verified with %q", got)
	}
	for _, team := range []string{"", "not set", `A" or "B`, "SHORT"} {
		output = "TeamIdentifier=" + team + "\n"
		if got := codesignTeam("/a/file"); got != "" {
			t.Errorf("%q read as %q", team, got)
		}
		calls = nil
		if err := codesignVerify("/a/file", team); err == nil || len(calls) != 0 {
			t.Errorf("%q was put to the tool: %v %q", team, err, calls)
		}
	}
	codesignRun = func(...string) ([]byte, error) { return []byte("TeamIdentifier=" + testTeam), errors.New("not signed") }
	if got := codesignTeam("/a/file"); got != "" {
		t.Errorf("a failed reading names a team: %q", got)
	}
	if codesignTool != "/usr/bin/codesign" {
		t.Errorf("the tool is named by absolute path: %s", codesignTool)
	}
}

// TestAnUnsignedFileHasNoTeam, with the tool itself where the system has one.
func TestAnUnsignedFileHasNoTeam(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(plain, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := verifiedTeam(plain); got != "" {
		t.Fatalf("an unsigned file reads as %q", got)
	}
}

// TestNoHarnessLeftWaits: the file gone and nothing on PATH yet, as an upgrade leaves it for
// a moment, says so and waits; the file that arrives is still held to the team.
func TestNoHarnessLeftWaits(t *testing.T) {
	w := newReproveWorld(t, "")
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.link); err != nil {
		t.Fatal(err)
	}
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	if !errors.As(err, &none) || !strings.Contains(none.Detail, "no harness") {
		t.Fatalf("it waits, saying why: %v", err)
	}
	if !HarnessOwed(w.p) {
		t.Fatal("it still owes")
	}
	w.signed[w.second] = otherTeam
	w.repoint(t, w.second)
	var we *WallError
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); errors.As(err, &none) || !errors.As(err, &we) || !strings.Contains(err.Error(), "signing team") {
		t.Fatalf("the new file meets the team: %v", err)
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing ran: %q", got)
	}
}

// TestAHarnessGoneDuringItsProofWaits: gone is no verdict on the file that follows, which
// has its own check; new bytes at the path are told apart, and stop the run.
func TestAHarnessGoneDuringItsProofWaits(t *testing.T) {
	w := newReproveWorld(t, "case \"$*\" in *include-hook-events*) /bin/rm -f \"$0\";; esac\n")
	w.update(t)
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	if !errors.As(err, &none) {
		t.Fatalf("it waits: %v", err)
	}
	if !HarnessOwed(w.p) {
		t.Fatal("nothing has passed")
	}
	harnessScript(t, w.first, "first", "2.1.321", "")
	w.repoint(t, w.first)
	if _, err := ReproveHarness(context.Background(), w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
}

// TestAFileThatComesBackUnchangedOwesNothing: the file that went returning as it was needs
// no proof.
func TestAFileThatComesBackUnchangedOwesNothing(t *testing.T) {
	w := newReproveWorld(t, "")
	raw, err := os.ReadFile(w.first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	if !HarnessOwed(w.p) {
		t.Fatal("a file that went owes a proof")
	}
	if err := os.WriteFile(w.first, raw, 0o755); err != nil {
		t.Fatal(err)
	}
	if line, err := ReproveHarness(context.Background(), w.p, w.c.approver); err != nil || line != "" {
		t.Fatalf("nothing is owed: %q %v", line, err)
	}
	if got := w.ran(t); len(got) != 0 {
		t.Fatalf("nothing ran: %q", got)
	}
}

// TestAHarnessGoneBeforeItsProofWaits: the file going after the login check spends no try.
func TestAHarnessGoneBeforeItsProofWaits(t *testing.T) {
	w := newReproveWorld(t, "case \"$*\" in *auth*status*) /bin/rm -f \"$0\";; esac\n")
	w.update(t)
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	if !errors.As(err, &none) || none.RetryAfter != 0 {
		t.Fatalf("it waits as an outage does: %v", err)
	}
}

// TestAFirstProofWithNoHarnessIsRefused: a start's proof is of the file it resolved, so
// waiting would bring nothing.
func TestAFirstProofWithNoHarnessIsRefused(t *testing.T) {
	w := newReproveWorld(t, "")
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	err := ProveWall(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	var we *WallError
	if errors.As(err, &none) || !errors.As(err, &we) {
		t.Fatalf("a refusal: %v", err)
	}
}

// TestALoginCheckOnAFileThatWentWaits: the pinned file gone is no answer about the login,
// so it does not tell a person to log in.
func TestALoginCheckOnAFileThatWentWaits(t *testing.T) {
	w := newReproveWorld(t, "")
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	err := CheckLogin(context.Background(), w.p)
	var none *NoVerdictError
	if !errors.As(err, &none) || strings.Contains(err.Error(), "log in") {
		t.Fatalf("no verdict: %v", err)
	}
}

// TestOnlyAnAnswerOfLoggedOutIsAVerdictOnTheLogin: a failure to answer, or an answer that
// cannot be read, says nothing of the login.
func TestOnlyAnAnswerOfLoggedOutIsAVerdictOnTheLogin(t *testing.T) {
	for _, c := range []struct {
		name, reply string
		verdict     bool
	}{
		{"logged out exit 0", `printf '{"loggedIn":false}'; exit 0`, true},
		{"logged out exit 1", `printf '{"loggedIn":false}'; exit 1`, true},
		{"no answer", `exit 1`, false},
		{"not json", `printf 'garbage'; exit 0`, false},
		{"no such field", `printf '{}'; exit 0`, false},
		{"a failure with garbage", `printf 'garbage'; exit 7`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newReproveWorld(t, "")
			harnessScript(t, w.first, "first", "2.1.319", "case \"$*\" in *auth*status*) "+c.reply+";; esac\n")
			if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
				t.Fatal(err)
			}
			err := CheckLogin(context.Background(), w.p)
			var none *NoVerdictError
			var we *WallError
			switch {
			case c.verdict && (errors.As(err, &none) || !errors.As(err, &we) || we.Wall != WallLogin || !strings.Contains(err.Error(), "log in")):
				t.Fatalf("a verdict: %v", err)
			case !c.verdict && (!errors.As(err, &none) || strings.Contains(err.Error(), "log in")):
				t.Fatalf("no verdict, and no telling a person to log in: %v", err)
			}
		})
	}
}

// TestAWaitLineNamesTheFileWhoseLaunchFailed: a launch that stops before it shows anything,
// on a flag a new harness does not take, is named by its file.
func TestAWaitLineNamesTheFileWhoseLaunchFailed(t *testing.T) {
	w := newReproveWorld(t, "case \"$*\" in *include-hook-events*) echo \"error: unknown option\" >&2; exit 1;; esac\n")
	w.update(t)
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var none *NoVerdictError
	if !errors.As(err, &none) || !strings.Contains(none.Detail, "harness file second") {
		t.Fatalf("the file is named: %v", err)
	}
}

// TestTheCheckBesideTheHarnessLetsOnlyAGoneFileThrough: the check a stage's turn and a wait
// make leaves a file that went to the proof; new bytes at the path stop the run there too.
func TestTheCheckBesideTheHarnessLetsOnlyAGoneFileThrough(t *testing.T) {
	w := newReproveWorld(t, "")
	harnessScript(t, w.first, "in-place", "2.1.319", "")
	if err := CheckWallSettingsBesideHarness(w.p, w.c.approver); err == nil {
		t.Fatal("new bytes at the path stop the run")
	}
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettingsBesideHarness(w.p, w.c.approver); err != nil {
		t.Fatalf("a file that went waits for the proof: %v", err)
	}
	if err := CheckWallSettings(w.p, w.c.approver); err == nil {
		t.Fatal("the launch's own check still refuses")
	}
}

// TestALimitAtTheProofIsNoVerdict: the same error the first proof's wait reads, the new
// file still owing its proof, and the proof that follows says the version the run proved.
func TestALimitAtTheProofIsNoVerdict(t *testing.T) {
	w := newReproveWorld(t, "")
	w.update(t)
	t.Setenv(fakeModeEnv, "wall-limit-exit")
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var limit *LimitError
	if !errors.As(err, &limit) {
		t.Fatalf("a limit: %v", err)
	}
	if !HarnessOwed(w.p) {
		t.Fatal("a limit proves nothing")
	}
	t.Setenv(fakeModeEnv, "wall-ok")
	said, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(said, "2.1.319") || !strings.Contains(said, "2.1.320") {
		t.Fatalf("both versions: %q", said)
	}
}

// TestNothingIsProvedWhileAStageIsInItsCall: the file going during a stage's call leaves
// the call alone; the proof waits for the next check.
func TestNothingIsProvedWhileAStageIsInItsCall(t *testing.T) {
	w := newReproveWorld(t, "")
	harnessScript(t, w.first, "first", "2.1.319", "/bin/rm -f \"$0\"\n")
	// The start this drive made read the file before this change: read it again.
	if _, err := WriteWallSettings(w.p, w.c.approver); err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeModeEnv, "ok")
	if _, _, err := w.c.Run(context.Background(), shellRequest(w.f)); err != nil {
		t.Fatal(err)
	}
	if got := w.ran(t); !slices.Equal(got, []string{"first"}) {
		t.Fatalf("the one call, and no proof inside it: %q", got)
	}
	if !HarnessOwed(w.p) {
		t.Fatal("the file went during the call")
	}
}
