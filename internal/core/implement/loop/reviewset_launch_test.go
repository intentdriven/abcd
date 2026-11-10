package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// launchRoots writes a machine configuration under a home of its own and
// returns the roots that read it, with a repository that sets nothing.
func launchRoots(t *testing.T, machine string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	path := abcdhome.Path(r.Home, "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(machine), 0o600); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestTheBundledReviewSetLaunchesOnASubscriptionOnlyMachine
// (iss-2611080057001367): a drive walls every role onto the claude harness, and
// on a machine that enables it and configures no provider, every reviewer of
// the bundled review set can be launched, so the drive's start lets it through.
func TestTheBundledReviewSetLaunchesOnASubscriptionOnlyMachine(t *testing.T) {
	r := launchRoots(t, `{"runner":{"claude":{}}}`)
	cfg, err := runner.Load(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckReviewersLaunch(Options{Roots: &r, Runners: cfg.Walled(runner.Profile{})}, r.Repo); err != nil {
		t.Fatalf("a subscription machine cannot launch the bundled review set: %v", err)
	}
}

// TestAReviewSetNoRouteCanLaunchIsRefusedAtTheStart: a reviewer model this
// machine's routes cannot launch is refused before anything is written,
// naming the role, the model and the configuration to set.
func TestAReviewSetNoRouteCanLaunchIsRefusedAtTheStart(t *testing.T) {
	for name, tc := range map[string]struct {
		machine       string
		want, without []string
	}{
		"denylisted": {`{"oracle":{"denylist":["anthropic/*"]},"runner":{"claude":{}}}`,
			[]string{"oracle.denylist", "models.classes"}, []string{"oracle.api.anthropic"}},
		"off its list": {`{"oracle":{"api":{"anthropic":{"base_url":"https://api.example.invalid/v1","models":["claude-haiku-4-5"]}}},"runner":{"claude":{}}}`,
			[]string{"oracle.api.anthropic"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			r := launchRoots(t, tc.machine)
			cfg, err := runner.Load(r)
			if err != nil {
				t.Fatal(err)
			}
			err = CheckReviewersLaunch(Options{Roots: &r, Runners: cfg.Walled(runner.Profile{})}, r.Repo)
			ref := mustRefusal(t, err)
			if ref.Stage != StageReview {
				t.Fatalf("want the review stage's refusal: %+v", ref)
			}
			for _, w := range append(tc.want, RoleRuthless, BundledClasses[ClassMiddle], "nothing was written") {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("the refusal %q does not name %q", err, w)
				}
			}
			// A walled drive runs every role on the harness, so a runner route
			// would change nothing there.
			for _, w := range append(tc.without, "roles."+RoleRuthless+".runner") {
				if strings.Contains(err.Error(), w) {
					t.Errorf("the refusal %q names %q, which does not apply", err, w)
				}
			}
		})
	}
}
