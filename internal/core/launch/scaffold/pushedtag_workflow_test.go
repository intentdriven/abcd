package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// TestPinnedProvenanceReleasesOnlyThroughAutoRelease holds a release
// workflow whose provenance is verified against release.yml on the default
// branch to one entry point that publishes: auto-release's call from that
// branch. A run started by a pushed version tag is refused before anything is
// built; releases are made by auto-release from the default branch. The
// workflow keeps its tag-push trigger, and such a run is refused by its first
// job, which checks out, builds and signs nothing, while verify — and so every
// job after it — is skipped. auto-release's call (its event is the push to the
// default branch) and the rehearsal run as before.
//
// That is abcd's own profile and a managed repository's with semantic gates,
// whose receipts verify step holds the same identity. A managed repository
// without semantic gates verifies no provenance, and its tag-push run still
// verifies and releases.
func TestPinnedProvenanceReleasesOnlyThroughAutoRelease(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(ReleaseYMLPath)))
	if err != nil {
		t.Fatalf("read committed %s: %v", ReleaseYMLPath, err)
	}
	abcd, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	semanticSubs := BareSubstitutions("main")
	semanticSubs.SemanticGates = []string{"docs-currency-reviewer"}
	semantic, err := Render(semanticSubs)
	if err != nil {
		t.Fatal(err)
	}
	type entry struct {
		name     string
		ctx      map[string]any
		tagPush  bool
		refusing bool
	}
	entries := []entry{
		{name: "a pushed tag", ctx: map[string]any{"github.event_name": "push", "github.ref_type": "tag"}, tagPush: true},
		{name: "auto-release's call from main", ctx: map[string]any{"github.event_name": "push", "github.ref_type": "branch"}},
		{name: "the rehearsal", ctx: map[string]any{"github.event_name": "workflow_dispatch", "github.ref_type": "branch"}},
	}
	for where, wf := range map[string]string{
		ReleaseYMLPath:                     string(committed),
		"release.yml.tmpl (abcd)":          string(abcd.ReleaseYML),
		"release.yml.tmpl (bare+semantic)": string(semantic.ReleaseYML),
	} {
		// The refusing job is the first job, so it reads first.
		jobs := wf[strings.Index(wf, "\njobs:\n"):]
		first := indexOf(t, jobs, "\n  pushed-tag:\n", where)
		if v := indexOf(t, jobs, "\n  verify:\n", where); v < first {
			t.Errorf("%s: the pushed-tag job must come before verify", where)
		}
		refuse := jobSection(t, wf, "pushed-tag")
		verify := jobSection(t, wf, "verify")
		for _, forbidden := range []string{"uses:", "contents: write", "id-token", "attestations", "secrets."} {
			if strings.Contains(refuse, forbidden) {
				t.Errorf("%s: the pushed-tag job must run nothing but its refusal; it carries %q", where, forbidden)
			}
		}
		for _, e := range entries {
			ctx := map[string]any{"success()": true, "failure()": false, "cancelled()": false}
			for k, v := range e.ctx {
				ctx[k] = v
			}
			runs, err := actionsexpr.EvalIf(jobIf(t, refuse, where+" pushed-tag"), ctx)
			if err != nil {
				t.Fatal(err)
			}
			if runs != e.tagPush {
				t.Errorf("%s: on %s the pushed-tag job runs=%v, want %v", where, e.name, runs, e.tagPush)
			}
			gated, err := actionsexpr.EvalIf(jobIf(t, verify, where+" verify"), ctx)
			if err != nil {
				t.Fatal(err)
			}
			if gated == e.tagPush {
				t.Errorf("%s: on %s verify runs=%v, want %v", where, e.name, gated, !e.tagPush)
			}
		}
		script := runBlock(t, refuse)
		cmd := exec.Command("bash", "-c", script)
		cmd.Env = []string{"TAG=v1.2.3", "PATH=/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("%s: the pushed-tag job's refusal exited 0:\n%s", where, out)
		}
		for _, want := range []string{"v1.2.3", "auto-release"} {
			if !strings.Contains(string(out), want) {
				t.Errorf("%s: the refusal must name %q:\n%s", where, want, out)
			}
		}
	}

	for _, subs := range []Substitutions{BareSubstitutions("main"), GateSubstitutions("main", "")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(rendered.ReleaseYML), "\n  pushed-tag:\n") {
			t.Errorf("a release workflow that verifies no provenance keeps its tag-push entry point (Gate=%v)", subs.Gate)
		}
	}
}
