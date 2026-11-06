package runner

// wall_reprove_test.go holds what a drive does when the harness file its run proved is
// gone or changed: it proves what is there now, once, before any stage.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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

// TestAHarnessThatFailsItsProofStartsNothing: the refusal names the check and the new
// version, and no stage starts on it afterwards.
func TestAHarnessThatFailsItsProofStartsNothing(t *testing.T) {
	w := newReproveWorld(t, fakeModeEnv+"=wall-tools; export "+fakeModeEnv+"\n")
	w.update(t)
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var we *WallError
	if !errors.As(err, &we) || we.Wall != WallTools || !strings.Contains(err.Error(), "2.1.320") {
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

// TestAHarnessChangedInPlaceIsProvedAsWell: a file's contents changing at the same path is
// told from one gone by what the check reads, and gets the same proof; the same version
// says its file changed.
func TestAHarnessChangedInPlaceIsProvedAsWell(t *testing.T) {
	for _, c := range []struct {
		name, version string
		want          []string
	}{
		{"a new version", "2.1.321", []string{"2.1.319", "2.1.321"}},
		{"the same version", "2.1.319", []string{"2.1.319", "file changed"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newReproveWorld(t, "")
			harnessScript(t, w.first, "in-place", c.version, "")
			if !HarnessOwed(w.p) {
				t.Fatal("the file's contents changed")
			}
			if err := CheckWallSettings(w.p, w.c.approver); err == nil {
				t.Fatal("no stage starts before the changed file has passed")
			}
			said, err := ReproveHarness(context.Background(), w.p, w.c.approver)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range c.want {
				if !strings.Contains(said, want) {
					t.Errorf("%q lacks %q", said, want)
				}
			}
			if err := CheckWallSettings(w.p, w.c.approver); err != nil {
				t.Fatalf("the check holds on what passed: %v", err)
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
}

// TestNoHarnessLeftIsARefusal: the file gone and nothing on PATH to take its place.
func TestNoHarnessLeftIsARefusal(t *testing.T) {
	w := newReproveWorld(t, "")
	if err := os.Remove(w.first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.link); err != nil {
		t.Fatal(err)
	}
	_, err := ReproveHarness(context.Background(), w.p, w.c.approver)
	var we *WallError
	if !errors.As(err, &we) || we.Wall != WallLaunch {
		t.Fatalf("the launch's wall: %v", err)
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
