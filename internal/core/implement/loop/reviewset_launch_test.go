package loop

import (
	"os"
	"path/filepath"
	"slices"
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

// walledFor is r's runner configuration walled for a drive, holding the
// routes r's class map resolves, as the drive's start hands them over.
func walledFor(t *testing.T, r layered.Roots) *runner.Config {
	t.Helper()
	cfg, err := runner.Load(r)
	if err != nil {
		t.Fatal(err)
	}
	routes, err := ClassRoutes(r)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Walled(runner.Profile{}).WithClassRoutes(routes)
}

// TestTheBundledReviewSetLaunchesOnASubscriptionOnlyMachine
// (iss-2611080057001367): a drive walls every role onto the claude harness, and
// on a machine that enables it and configures no provider, every reviewer of
// the bundled review set can be launched, so the drive's start lets it through.
func TestTheBundledReviewSetLaunchesOnASubscriptionOnlyMachine(t *testing.T) {
	r := launchRoots(t, `{"runner":{"claude":{}}}`)
	if err := CheckReviewersLaunch(Options{Roots: &r, Runners: walledFor(t, r)}, r.Repo); err != nil {
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
			err := CheckReviewersLaunch(Options{Roots: &r, Runners: walledFor(t, r)}, r.Repo)
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

// TestTheMachinesClassMapBoundsWhatADriveTakesByName: ClassRoutes is the class
// map this machine resolves, its own classes over the bundled ones, and a
// drive's walled launch takes those routes by name and no others: the
// machine's route for a class runs with no provider block, and the bundled
// route it replaced is no longer one it takes.
func TestTheMachinesClassMapBoundsWhatADriveTakesByName(t *testing.T) {
	const override = "anthropic/claude-opus-6"
	r := launchRoots(t, `{"models":{"classes":{"middle":"`+override+`"}},"runner":{"claude":{}}}`)
	routes, err := ClassRoutes(r)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(routes, override) || slices.Contains(routes, BundledClasses[ClassMiddle]) ||
		!slices.Contains(routes, BundledClasses[ClassBelow]) {
		t.Fatalf("class routes = %q, want the machine's middle over the bundled one, and the bundled rest", routes)
	}
	cfg := walledFor(t, r)
	if err := CheckReviewersLaunch(Options{Roots: &r, Runners: cfg}, r.Repo); err != nil {
		t.Fatalf("the machine's class map cannot launch: %v", err)
	}
	if err := cfg.CanLaunch(RoleRuthless, BundledClasses[ClassMiddle], false); err == nil {
		t.Fatal("the bundled route the machine replaced is still taken by name")
	}
}

// TestARunStartedAfterTheClassMapChangedLaunchesItsReviewers: inside a drive,
// each run's start reads the class map again, and the routes it read join the
// walled configuration's, so a lane started after the machine's map changed
// launches its reviewer at the new route; a route no map read in the process
// names stays refused.
func TestARunStartedAfterTheClassMapChangedLaunchesItsReviewers(t *testing.T) {
	const changed = "anthropic/claude-opus-6"
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"runner":{"claude":{}}}`)
	roots, _ := layered.RootsFor(repo.Root())
	cfg := walledFor(t, roots)
	if err := cfg.CanLaunch(RoleRuthless, changed, false); err == nil {
		t.Fatal("a route no map has named yet is taken by name")
	}
	machineConfig(t, `{"models":{"classes":{"middle":"`+changed+`"}},"runner":{"claude":{}}}`)
	if _, err := Start(repo.Root(), "itd-10", Options{Runners: cfg, ClassMapRead: cfg.AddClassRoutes}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.CanLaunch(RoleRuthless, changed, false); err != nil {
		t.Fatalf("the run's reviewer at the map's new route cannot launch: %v", err)
	}
	if err := cfg.CanLaunch(RoleRuthless, "anthropic/claude-gone-1", false); err == nil {
		t.Fatal("a route no map read in the process names is taken by name")
	}
}
