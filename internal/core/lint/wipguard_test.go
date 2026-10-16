package lint_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The wip-guard required check (spc-2610120941441282 step 7,
// itd-2610120738075547 C7, adr-2610120937435856 decision 4, brief invariant
// 23). A cloud session's work-in-progress push needs no receipt, and the hook
// that admits it is skipped by --no-verify, so what keeps that work off the
// default branch is this job, required by the ruleset. Each test below holds
// one way that guarantee could be lost without the job ever going red.

const (
	wipGuardJob      = "wip-guard"
	ciWorkflow       = ".github/workflows/ci.yml"
	mainProtection   = ".abcd/work/rulesets/main-protection.json"
	wipGuardStepHint = "bash \"$check\" \"$source\" \"$BASE_SHA\" HEAD"
)

// wipGuardBlock is the job's block in ci.yml, failing when there is none.
func wipGuardBlock(t *testing.T) string {
	t.Helper()
	workflow := readRepoFile(t, filepath.Join("..", "..", ".."), ciWorkflow)
	block, ok := workflowJobBlock(workflow, wipGuardJob)
	if !ok {
		t.Fatalf("%s defines no %q job; the ruleset requires its context, and a required context "+
			"no job reports blocks every merge", ciWorkflow, wipGuardJob)
	}
	return block
}

// wipGuardStep is the job's step that runs the check, read from the job alone.
func wipGuardStep(t *testing.T) workflowStep {
	t.Helper()
	steps, err := workflowSteps(wipGuardBlock(t))
	if err != nil {
		t.Fatalf("%s's %s job: %v", ciWorkflow, wipGuardJob, err)
	}
	var found []workflowStep
	for _, s := range steps {
		if strings.Contains(s.body, wipGuardStepHint) {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s's %s job has %d steps running %s; want exactly one", ciWorkflow, wipGuardJob, len(found), wipGuardStepHint)
	}
	return found[0]
}

// TestWIPGuardJobRunsOnPullRequestAndMergeGroup: the workflow triggers on both
// events that can carry a change into the default branch, and the job's step
// judges each one, reading a pull request's source branch through an env var
// and a queue entry's through the REST API, with least-privilege permissions
// and every action pinned by its full commit.
func TestWIPGuardJobRunsOnPullRequestAndMergeGroup(t *testing.T) {
	workflow := readRepoFile(t, filepath.Join("..", "..", ".."), ciWorkflow)
	triggers := workflowTriggers(workflow)
	for _, want := range []string{"pull_request", "merge_group"} {
		if !slices.Contains(triggers, want) {
			t.Errorf("%s does not trigger on %s (it triggers on %q)", ciWorkflow, want, triggers)
		}
	}
	block := wipGuardBlock(t)
	step := wipGuardStep(t)

	if got := step.env["HEAD_REF"]; strings.Join(strings.Fields(got), " ") != "${{ github.head_ref }}" {
		t.Errorf("the step's HEAD_REF is %q; want the pull request's source branch, ${{ github.head_ref }}", got)
	}
	if got := step.env["QUEUE_REF"]; strings.Join(strings.Fields(got), " ") != "${{ github.event.merge_group.head_ref }}" {
		t.Errorf("the step's QUEUE_REF is %q; want the queue entry's ref, ${{ github.event.merge_group.head_ref }}", got)
	}
	// The source branch is attacker-chosen text: an expression inside the
	// script is pasted into it before bash parses it (script injection).
	script := stepScript(t, step)
	if strings.Contains(script, "${{") {
		t.Errorf("the step's script interpolates an expression; pass every value through env instead:\n%s", script)
	}
	for _, arm := range []string{"pull_request)", "merge_group)"} {
		if !strings.Contains(script, arm) {
			t.Errorf("the step's script has no %s arm, so it does not judge that event:\n%s", arm, script)
		}
	}
	if !strings.Contains(script, "gh api") || !strings.Contains(script, "/pulls/") {
		t.Errorf("the step does not read a queue entry's pull request through the REST API:\n%s", script)
	}

	perms := regexp.MustCompile(`(?m)^    permissions:\n((?:      .*\n)+)`).FindStringSubmatch(block + "\n")
	if perms == nil {
		t.Fatalf("the %s job declares no permissions of its own", wipGuardJob)
	}
	var granted []string
	for _, l := range strings.Split(strings.TrimSpace(perms[1]), "\n") {
		granted = append(granted, strings.Join(strings.Fields(l), " "))
	}
	slices.Sort(granted)
	if want := []string{"contents: read", "pull-requests: read"}; !slices.Equal(granted, want) {
		t.Errorf("the %s job grants %q; want exactly %q", wipGuardJob, granted, want)
	}
	if !strings.Contains(block, "persist-credentials: false") {
		t.Errorf("the %s job's checkout keeps the token in the clone's config", wipGuardJob)
	}

	uses := regexp.MustCompile(`(?m)uses:\s*(\S+)`).FindAllStringSubmatch(block, -1)
	if len(uses) == 0 {
		t.Fatalf("the %s job uses no action, so it has no checkout to judge", wipGuardJob)
	}
	for _, u := range uses {
		if !regexp.MustCompile(`@[0-9a-f]{40}$`).MatchString(u[1]) {
			t.Errorf("the %s job uses %s, not pinned by a full commit", wipGuardJob, u[1])
		}
	}
	if !strings.Contains(block, "scripts/check-wip-source-cases.sh") {
		t.Errorf("the %s job does not run the check's cases before the check", wipGuardJob)
	}
}

// TestWIPGuardNeverStoodDown: a required check must report on every event,
// and a job that stands down still reports its context green, a green for
// work that did not happen. So the job carries no condition and no
// dependency, the docs-only classifier included, and neither do its steps.
func TestWIPGuardNeverStoodDown(t *testing.T) {
	block := wipGuardBlock(t)
	for _, l := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, key := range []string{"if:", "needs:", "continue-on-error:"} {
			if strings.HasPrefix(trimmed, key) {
				t.Errorf("the %s job carries %q (%s), so it can stand down or pass on a failure",
					wipGuardJob, key, strings.TrimSpace(l))
			}
		}
		if strings.Contains(trimmed, "inert") {
			t.Errorf("the %s job reads the docs-only classifier (%s)", wipGuardJob, strings.TrimSpace(l))
		}
	}
}

// TestMainProtectionRequiresWIPGuard: the committed ruleset mirror names the
// job's context, in the mirror's sorted order. Applying it on the forge is the
// person's act; the mirror is the tree's record of what that act sets.
func TestMainProtectionRequiresWIPGuard(t *testing.T) {
	var ruleset struct {
		Rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				RequiredStatusChecks []struct {
					Context string `json:"context"`
				} `json:"required_status_checks"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, filepath.Join("..", "..", ".."), mainProtection)), &ruleset); err != nil {
		t.Fatalf("decoding %s: %v", mainProtection, err)
	}
	var required []string
	for _, rule := range ruleset.Rules {
		if rule.Type != "required_status_checks" {
			continue
		}
		for _, c := range rule.Parameters.RequiredStatusChecks {
			required = append(required, c.Context)
		}
	}
	if !slices.Contains(required, wipGuardJob) {
		t.Fatalf("%s does not require the %q context (it requires: %s); an unrequired check stops "+
			"no work-in-progress branch from merging", mainProtection, wipGuardJob, strings.Join(required, ", "))
	}
	if !slices.IsSorted(required) {
		t.Errorf("%s lists its required contexts out of order (%s)", mainProtection, strings.Join(required, ", "))
	}
}

// TestWIPGuardReadsEachEventsSourceBranch runs the step's own script in a
// scratch clone whose base commit holds a stand-in check that records what it
// was handed, and whose working tree (the change) holds a different one: a
// pull request's source branch arrives verbatim and is never run as shell; a
// queue entry's comes from its pull request through the API; and a queue ref
// with no pull-request number, an API that fails or answers nothing, or an
// event with no usable base refuses without judging anything. The check that
// runs is always the BASE commit's copy, so a change cannot edit the check it
// is judged by; only when the base holds none yet (the first landing) does the
// change's own copy run, and never for a source branch holding abcd-wip.
func TestWIPGuardReadsEachEventsSourceBranch(t *testing.T) {
	script := stepScript(t, wipGuardStep(t))
	const queue = "refs/heads/gh-readonly-queue/main/pr-42-3e5d9a1c7f2b4086d3a5c9e1b7f4d2a6c8e0b5d9"
	// recorder is a stand-in check that records who it is and what it was
	// handed; the change's copy also exits 0 whatever it is given, as a
	// pull request that edited the check to pass would.
	recorder := func(who string) string {
		return "printf '" + who + " ' >\"$PWD/handed\"; printf '%s ' \"$@\" >>\"$PWD/handed\"; exit 0\n"
	}
	type run struct {
		name       string
		env        map[string]string // "BASE" in a value is the base commit
		gh         string
		noBaseCopy bool // the base commit holds no check (the first landing)
		wantCode   int
		wantArgs   string // who ran and what it was handed; "" when nothing must run
	}
	runs := []run{
		{"a pull request", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "test/abcd-wip/iss-0000000000000000", "BASE_SHA": "BASE"},
			"exit 3", false, 0, "base test/abcd-wip/iss-0000000000000000 BASE HEAD"},
		{"a pull request whose branch is shell", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "x/$(touch pwned)`touch pwned`", "BASE_SHA": "BASE"},
			"exit 3", false, 0, "base x/$(touch pwned)`touch pwned` BASE HEAD"},
		{"a queue entry", map[string]string{"EVENT_NAME": "merge_group", "QUEUE_REF": queue, "BASE_SHA": "BASE", "REPO": "o/r"},
			`[ "$*" = "api repos/o/r/pulls/42 --jq .head.ref" ] || exit 4; echo x/abcd-wip/iss-0000000000000000`, false, 0,
			"base x/abcd-wip/iss-0000000000000000 BASE HEAD"},
		{"the first landing: the base holds no check yet", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "feat/x", "BASE_SHA": "BASE"},
			"exit 3", true, 0, "change feat/x BASE HEAD"},
		{"the first landing from a work-in-progress branch", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "x/abcd-wip/iss-0000000000000000", "BASE_SHA": "BASE"},
			"exit 3", true, 1, ""},
		{"the first landing from an upper-case work-in-progress branch", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "Testhost/ABCD-WIP/x", "BASE_SHA": "BASE"},
			"exit 3", true, 1, ""},
		{"a base this clone does not hold", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "feat/x", "BASE_SHA": "9f1b7c4e2a6d8035b1e4c7a92f6d0b3e8c5a1d7f"},
			"exit 3", false, 1, ""},
		{"a queue ref with no pull-request number", map[string]string{"EVENT_NAME": "merge_group", "QUEUE_REF": "refs/heads/main", "BASE_SHA": "BASE", "REPO": "o/r"},
			"echo feat/x", false, 1, ""},
		{"an API that fails", map[string]string{"EVENT_NAME": "merge_group", "QUEUE_REF": queue, "BASE_SHA": "BASE", "REPO": "o/r"},
			"exit 1", false, 1, ""},
		{"an API that answers nothing", map[string]string{"EVENT_NAME": "merge_group", "QUEUE_REF": queue, "BASE_SHA": "BASE", "REPO": "o/r"},
			"echo", false, 1, ""},
		{"a pull request with no base", map[string]string{"EVENT_NAME": "pull_request", "HEAD_REF": "feat/x", "BASE_SHA": ""},
			"exit 3", false, 1, ""},
		{"a queue entry with the placeholder base", map[string]string{"EVENT_NAME": "merge_group", "QUEUE_REF": queue, "BASE_SHA": "0000000000000000000000000000000000000000", "REPO": "o/r"},
			"echo feat/x", false, 1, ""},
		{"the nightly run", map[string]string{"EVENT_NAME": "schedule", "BASE_SHA": ""},
			"exit 3", false, 0, ""},
	}
	for _, r := range runs {
		t.Run(r.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := t.TempDir()
			fakeTool(t, bin, "gh", r.gh+"\n")
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				cmd.Env = append(gittest.Env(t),
					"GIT_AUTHOR_NAME=Test Person", "GIT_AUTHOR_EMAIL=person@example.com",
					"GIT_COMMITTER_NAME=Test Person", "GIT_COMMITTER_EMAIL=person@example.com")
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
				return strings.TrimSpace(string(out))
			}
			git("init", "-q")
			check := filepath.Join(dir, "scripts", "check-wip-source.sh")
			if err := os.MkdirAll(filepath.Dir(check), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "seed.md"), []byte("seed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if !r.noBaseCopy {
				if err := os.WriteFile(check, []byte(recorder("base")), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			git("add", "-A")
			git("commit", "-q", "-m", "base")
			base := git("rev-parse", "HEAD")
			// The change edits the check (or adds it) and leaves it in the tree.
			if err := os.WriteFile(check, []byte(recorder("change")), 0o755); err != nil {
				t.Fatal(err)
			}
			git("add", "-A")
			git("commit", "-q", "-m", "the change")
			env := map[string]string{}
			for k, v := range r.env {
				env[k] = strings.ReplaceAll(v, "BASE", base)
			}
			want := strings.ReplaceAll(r.wantArgs, "BASE", base)

			out, code := runScript(t, script, dir, bin, env)
			if code != r.wantCode {
				t.Fatalf("exit %d, want %d\n%s", code, r.wantCode, out)
			}
			handed, err := os.ReadFile(filepath.Join(dir, "handed"))
			switch {
			case want == "" && err == nil:
				t.Errorf("a check ran with %q, though the step should not have judged anything\n%s", handed, out)
			case want != "" && err != nil:
				t.Errorf("the check never ran\n%s", out)
			case want != "" && strings.TrimSpace(string(handed)) != want:
				t.Errorf("the check run was %q, want %q", strings.TrimSpace(string(handed)), want)
			}
			if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
				t.Errorf("the source branch's text ran as a command")
			}
		})
	}
}
