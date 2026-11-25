//go:build live

package runner_test

// The live proof of a session's short launch (itd-2611251725203320,
// spc-2611251757236507): the launch `abcd start` makes before it opens a session, on the
// account's own profile, with the arguments the verb would build for this checkout and no
// mode and no prompt. It spends a little of the person's subscription and writes one
// ledger entry, so it sits behind the `live` build tag and nothing runs it by default:
//
//	GOFLAGS=-trimpath go test -tags live -count=1 -v -run TestTheShortLaunchHoldsOnALoggedInProfile ./internal/core/runner
//
// The login is the person's: where no host is on PATH, the account holds no profile, or
// the profile is not logged in, the check is skipped and says so on stderr as well as in
// the test's own output. ABCD_LIVE_PROFILE names the profile where there are several.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// skipSessionLoudly skips the live proof, naming why on stderr too, so a run that meant to
// prove and did not cannot read as a pass.
func skipSessionLoudly(t *testing.T, why string) {
	t.Helper()
	msg := "SKIPPED the live proof of a session's short launch: " + why
	fmt.Fprintln(os.Stderr, msg)
	t.Skip(msg)
}

func TestTheShortLaunchHoldsOnALoggedInProfile(t *testing.T) {
	if _, err := exec.LookPath(runner.Claude); err != nil {
		skipSessionLoudly(t, "no claude host is on PATH")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	p, err := runner.DriveProfile(home, os.Getenv("ABCD_LIVE_PROFILE"))
	switch {
	case errors.Is(err, runner.ErrNoProfile):
		skipSessionLoudly(t, "the account holds no such profile")
	case errors.Is(err, runner.ErrSeveralProfiles):
		skipSessionLoudly(t, "the account holds several profiles; name one in ABCD_LIVE_PROFILE")
	case err != nil:
		t.Fatal(err)
	}
	if entry := runner.ProfileLoads(p); entry != "" {
		t.Fatalf("the profile holds %s, which a launch would load", entry)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := gitutil.Toplevel(cwd)
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = root
	// The person's terminal holds no such marker, and a host may read it as its own.
	t.Setenv("CLAUDECODE", "")
	if err := os.Unsetenv("CLAUDECODE"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := runner.CheckLogin(ctx, p); err != nil {
		skipSessionLoudly(t, "the profile "+p.Name()+" is not logged in, and the login is the person's: "+runner.LoginCommand(p)+" ("+err.Error()+")")
	}
	kept, err := runner.KeepSessionBinary(home, filepath.Join(root, ".abcd", ".work.local", "bin", "abcd"))
	if err != nil {
		t.Fatalf("the checkout's own build is kept (any scripts/abcd.sh call makes it): %v", err)
	}
	harness, err := runner.SessionHarness(p, root)
	if err != nil {
		t.Fatal(err)
	}
	roots, _ := layered.RootsFor(root)
	cfg, err := runner.Load(roots)
	if err != nil {
		t.Fatal(err)
	}
	conventions, err := os.ReadFile(filepath.Join(root, loop.ConventionsFile))
	if err != nil {
		t.Fatal(err)
	}
	// The event at a sub-agent's start is left out where the kept build lacks its verb, as
	// the verb leaves it out.
	var hooks []runner.SessionHook
	for _, h := range runner.SessionHooks() {
		if h.Event == "SubagentStart" && exec.Command(kept, "explain", "hook", "subagent-start").Run() != nil {
			t.Log("this build has no hook for a sub-agent's start, so the settings leave that event out")
			continue
		}
		hooks = append(hooks, h)
	}
	s := runner.Session{Profile: p, Bin: kept, Hooks: hooks, ConventionsName: loop.ConventionsFile, Conventions: conventions}
	proof, err := runner.ProveSession(ctx, s, harness, root, cfg.Ledger())
	if err != nil {
		t.Fatalf("the short launch does not hold: %v", err)
	}
	t.Logf("tools reported: %q", proof.Reported)
	t.Logf("named and not reported: %q", proof.Unreported)
	t.Logf("plugins: %q", proof.Plugins)
	t.Logf("skills: %q", proof.Skills)
	t.Logf("agents: %q", proof.Agents)
	t.Logf("host %s", proof.HarnessVersion)
	if _, cleared := s.Env(os.Environ()); len(cleared) > 0 {
		t.Logf("left out of the environment: %q", cleared)
	}
}
