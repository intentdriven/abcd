package scaffold

import (
	"strings"
	"testing"
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
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main")} {
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
		if !strings.Contains(publish, "needs: [verify, tag]\n") {
			t.Errorf("the publish job must need both verify and tag (Abcd=%v)", subs.Abcd)
		}
		for _, cond := range []string{
			"github.event_name != 'workflow_dispatch'",
			"needs.verify.result == 'success'",
			"(needs.tag.result == 'success' || needs.tag.result == 'skipped')",
		} {
			if !strings.Contains(publish, cond) {
				t.Errorf("the publish job's if must carry %q (Abcd=%v)", cond, subs.Abcd)
			}
		}

		// File order is job order read by a person: verify, tag, release.
		v := indexOf(t, rel, "\n  verify:\n", "release.yml")
		tg := indexOf(t, rel, "\n  tag:\n", "release.yml")
		p := indexOf(t, rel, "\n  release:\n", "release.yml")
		if !(v < tg && tg < p) {
			t.Errorf("release.yml job order: verify %d < tag %d < release %d must hold (Abcd=%v)", v, tg, p, subs.Abcd)
		}
	}
}
