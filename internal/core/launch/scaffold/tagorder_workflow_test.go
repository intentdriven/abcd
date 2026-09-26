package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// TestTheTagWaitsOnTheVerifyGate holds the release chain to one order: detect,
// then verify, then tag, then build and publish (iss-2608231226347380, the
// residual of adr-52; iss-2609100513521322 is its duplicate).
//
// adr-52 moved the semantic receipt gate into release.yml's verify job, on the
// premise that verify runs before the tag. On the auto-release path it did not:
// auto-release's own tag job pushed the tag and only then called release.yml,
// so a red verify — deterministic or semantic — left an immutable tag naming a
// version with no Release, and the heal path rebuilt the same tagged commit on
// every later push. A gate placed after the act it guards can only report it.
//
// The tag is therefore made INSIDE release.yml, by a job that needs verify, and
// only when the caller asks for it (auto-release on the fresh-tag path). A
// refused gate then leaves no tag and the version stays free. Both profiles:
// a managed repository scaffolded from the bare template carries the same
// chain, and TestSelfScaffoldParity binds the Abcd rendering to the committed
// workflows.
func TestTheTagWaitsOnTheVerifyGate(t *testing.T) {
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main"), GateSubstitutions("main", "")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		auto, rel := string(rendered.AutoReleaseYML), string(rendered.ReleaseYML)

		// auto-release makes no tag of its own: no tag job, no push anywhere.
		if strings.Contains(auto, "\n  tag:\n") {
			t.Errorf("auto-release.yml must not carry a tag job; the tag is made after verify, inside release.yml (Abcd=%v)", subs.Abcd)
		}
		for _, verb := range []string{"git push", "git tag -a"} {
			if strings.Contains(auto, verb) {
				t.Errorf("auto-release.yml must not run %q: anything it tags precedes the gate (Abcd=%v)", verb, subs.Abcd)
			}
		}
		call := jobSection(t, auto, "release")
		if !strings.Contains(call, "needs: detect\n") {
			t.Errorf("auto-release's release job must need detect alone (Abcd=%v)", subs.Abcd)
		}
		if !strings.Contains(call, "create_tag: ${{ needs.detect.outputs.need_tag == 'true' }}") {
			t.Errorf("auto-release must ask release.yml to create the tag exactly when detect found none (Abcd=%v)", subs.Abcd)
		}

		// release.yml declares the input, off by default, so the tag-push and
		// rehearsal entry points never make a tag.
		if !strings.Contains(rel, "      create_tag:\n        required: false\n        type: boolean\n        default: false\n") {
			t.Errorf("release.yml must declare a boolean create_tag input defaulting to false (Abcd=%v)", subs.Abcd)
		}

		verify := jobSection(t, rel, "verify")
		tag := jobSection(t, rel, "tag")
		publish := jobSection(t, rel, "release")
		for _, verb := range []string{"git push", "git tag -a"} {
			if strings.Contains(verify, verb) {
				t.Errorf("verify must not run %q (Abcd=%v)", verb, subs.Abcd)
			}
		}
		if !strings.Contains(tag, "needs: verify\n") || !strings.Contains(tag, "if: inputs.create_tag\n") {
			t.Errorf("the tag job must need verify and run only when create_tag is set (Abcd=%v):\n%s", subs.Abcd, tag)
		}
		if !strings.Contains(tag, "COMMIT: ${{ inputs.ref || github.sha }}") ||
			!strings.Contains(tag, `git tag -a "$TAG"`) || !strings.Contains(tag, `git push origin "refs/tags/$TAG"`) {
			t.Errorf("the tag job must tag the exact commit verify checked out and push only the tag ref (Abcd=%v)", subs.Abcd)
		}
		// A managed profile publishes what its named build job built, so its
		// publish job needs that job too (iss-2608270559310755).
		needs := "needs: [verify, tag]\n"
		if !subs.Abcd {
			needs = "needs: [verify, tag, build]\n"
		}
		if !strings.Contains(publish, needs) {
			t.Errorf("the publish job must need both verify and tag (Abcd=%v, Gate=%v): want %q", subs.Abcd, subs.Gate, needs)
		}
		// The publish job's condition is evaluated, not pattern-matched, by
		// TestThePublishConditionNeedsAGreenVerify below.

		// File order is job order read by a person: verify, tag, release.
		v := indexOf(t, rel, "\n  verify:\n", "release.yml")
		tg := indexOf(t, rel, "\n  tag:\n", "release.yml")
		p := indexOf(t, rel, "\n  release:\n", "release.yml")
		if !(v < tg && tg < p) {
			t.Errorf("release.yml job order: verify %d < tag %d < release %d must hold (Abcd=%v)", v, tg, p, subs.Abcd)
		}
	}
}

// TestThePublishConditionNeedsAGreenVerify evaluates the publish job's `if:`
// the way GitHub does, over every combination of the run's state, and holds it
// to one rule: publish exactly when the run is not a rehearsal, has not been
// cancelled, its verify gate is green, and its tag was either made or not asked
// for. The committed workflow and both template profiles are each evaluated.
//
// It evaluates rather than matches because the condition's clauses can all be
// present while it means something else: `(needs.verify.result == 'success' ||
// !cancelled())` carries the verify clause and publishes after a red verify on
// every uncancelled run, the tag-push and heal paths included, and a substring
// assertion passed it (review of the workflows lane, mutant M6).
func TestThePublishConditionNeedsAGreenVerify(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(ReleaseYMLPath)))
	if err != nil {
		t.Fatalf("read committed %s: %v", ReleaseYMLPath, err)
	}
	// Whether a source is the reusable gate comes from the profile that
	// rendered it, never from the condition under test: a condition that lost
	// its `inputs.publish` clause would otherwise be expected not to carry it.
	type source struct {
		wf   string
		gate bool
	}
	sources := map[string]source{ReleaseYMLPath: {wf: string(committed), gate: false}}
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main"), GateSubstitutions("main", "")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		sources[fmt.Sprintf("release.yml.tmpl (Abcd=%v, Gate=%v)", subs.Abcd, subs.Gate)] = source{wf: string(rendered.ReleaseYML), gate: subs.Gate}
	}

	results := []string{"success", "failure", "cancelled", "skipped"}
	for where, src := range sources {
		wf, gate := src.wf, src.gate
		cond := jobIf(t, jobSection(t, wf, "release"), where)
		// A managed profile publishes only what its named build job built
		// (iss-2608270559310755), and the gate publishes only when its caller
		// asks; abcd's own profile has neither, and its condition reads neither.
		hasBuild := strings.Contains(wf, "\n  build:\n")
		var buildCond string
		if hasBuild {
			buildCond = jobIf(t, jobSection(t, wf, "build"), where+" build job")
		}
		for _, event := range []string{"push", "workflow_dispatch"} {
			for _, verify := range results {
				for _, tag := range results {
					for _, build := range results {
						for _, publish := range []bool{true, false} {
							for _, cancelled := range []bool{false, true} {
								ctx := map[string]any{
									"github.event_name":   event,
									"needs.verify.result": verify,
									"needs.tag.result":    tag,
									"needs.build.result":  build,
									"inputs.publish":      publish,
									"cancelled()":         cancelled,
									"success()":           !cancelled && verify == "success" && tag == "success",
									"failure()":           verify == "failure" || tag == "failure",
								}
								got, err := actionsexpr.EvalIf(cond, ctx)
								if err != nil {
									t.Fatalf("%s: cannot evaluate the publish job's if %q: %v", where, cond, err)
								}
								want := event != "workflow_dispatch" && !cancelled && verify == "success" &&
									(tag == "success" || tag == "skipped") &&
									(!hasBuild || build == "success") && (!gate || publish)
								if got != want {
									t.Errorf("%s: the publish job's if %q is %v for event=%s verify=%s tag=%s build=%s "+
										"publish=%v cancelled=%v, want %v: a release publishes only after a green verify, "+
										"with its tag made or not asked for, on an uncancelled non-rehearsal run",
										where, cond, got, event, verify, tag, build, publish, cancelled, want)
								}
								if buildCond == "" {
									continue
								}
								// The build job obeys the same rule less its own
								// result: a caller that asked for no publish gets
								// no build either.
								gotBuild, err := actionsexpr.EvalIf(buildCond, ctx)
								if err != nil {
									t.Fatalf("%s: cannot evaluate the build job's if %q: %v", where, buildCond, err)
								}
								wantBuild := event != "workflow_dispatch" && !cancelled && verify == "success" &&
									(tag == "success" || tag == "skipped") && (!gate || publish)
								if gotBuild != wantBuild {
									t.Errorf("%s: the build job's if %q is %v for event=%s verify=%s tag=%s publish=%v "+
										"cancelled=%v, want %v", where, buildCond, gotBuild, event, verify, tag, publish, cancelled, wantBuild)
								}
							}
						}
					}
				}
			}
		}
	}
}

// jobIf returns the value of a job's own `if:` key, failing when the job has
// none, more than one, or a block scalar this reader does not fold.
func jobIf(t *testing.T, job, where string) string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(job, "\n") {
		if strings.HasPrefix(line, "    if:") {
			found = append(found, strings.TrimSpace(strings.TrimPrefix(line, "    if:")))
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: the job carries %d `if:` keys at job level; want exactly one", where, len(found))
	}
	if v := found[0]; v == "" || v[0] == '|' || v[0] == '>' {
		t.Fatalf("%s: the job's `if:` is a block scalar (%q); write it on one line so it can be evaluated", where, v)
	}
	return found[0]
}
