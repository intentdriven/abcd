package lint_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// A release cut runs its own macOS suite before anything is tagged or published,
// whatever the nightly run said (dcn-2611042202509334, iss-2611042224527148). With
// macOS out of the merge queue, a macOS-only regression merged after a nightly run
// would otherwise be tagged on Linux checks alone.
//
// release.yml runs the suite as one job per shard, each calling the same reusable
// workflow ci.yml's matrix calls, so a shard's steps are one definition. Its
// macos-gate job reads every shard's result by name and fails unless all succeeded,
// and every job that holds a write scope needs that job. One job per shard rather
// than a matrix because needs.<job>.result is a matrix's result as a whole: only
// separate jobs let the gate say which shard stopped the release without asking the
// forge for the run's jobs, so it needs no scope at all.

const (
	macOSSuiteWorkflow  = ".github/workflows/macos-suite.yml"
	macOSShardJob       = "shard"
	releaseWorkflowFile = ".github/workflows/release.yml"
	releaseMacOSGateJob = "macos-gate"
)

// releaseMacOSShardJob is the id of release.yml's job for one shard.
func releaseMacOSShardJob(shard int) string { return "macos-suite-" + strconv.Itoa(shard) }

// nonSuccessResults is every result a shard or the gate can be read as that must
// stop a release, a result that reads as nothing and one nobody has met among them.
// abandoned is what a shard that never got a runner reads as (run 41901793293).
var nonSuccessResults = []string{"failure", "cancelled", "skipped", "abandoned", "", "neutral"}

// workflowJobIDs lists a workflow's job ids in the order written.
func workflowJobIDs(workflow string) []string {
	_, jobs, ok := strings.Cut(workflow, "\njobs:\n")
	if !ok {
		return nil
	}
	var out []string
	for _, m := range regexp.MustCompile(`(?m)^  ([A-Za-z][A-Za-z0-9_-]*):\s*$`).FindAllStringSubmatch(jobs, -1) {
		out = append(out, m[1])
	}
	return out
}

// jobNeeds reads a job's needs, in either the bare or the list form.
func jobNeeds(job string) []string {
	raw, ok := jobKey(job, "needs")
	if !ok {
		return nil
	}
	raw = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
	var out []string
	for _, f := range strings.Split(raw, ",") {
		out = append(out, strings.TrimSpace(f))
	}
	return out
}

// jobGrantsWrite reports whether a job's own permissions hold a write scope. Every
// tag, Release and asset in this chain is made with the built-in token, so a job
// with no write scope can make none of them.
func jobGrantsWrite(job string) bool {
	return regexp.MustCompile(`(?m)^      [a-z-]+: write\b`).MatchString(job)
}

// jobPublishes reports whether a job can tag or publish: it holds a write scope, or
// it hands a called workflow anything beyond the macOS suite.
func jobPublishes(job string) bool {
	if jobGrantsWrite(job) {
		return true
	}
	uses, ok := jobKey(job, "uses")
	return ok && uses != "./"+macOSSuiteWorkflow
}

// releaseRun is one way release.yml starts, stated as the contexts its jobs read.
type releaseRun struct {
	name      string
	event     string
	refType   string
	createTag bool
	ref       string
	// publishes reports whether a green run of this kind reaches the publish jobs.
	publishes []string
}

var releaseRuns = []releaseRun{
	{name: "auto-release, fresh tag", event: "push", refType: "branch", createTag: true, publishes: []string{"tag", "release", "site"}},
	{name: "auto-release, heal", event: "push", refType: "branch", ref: strings.Repeat("a", 40), publishes: []string{"release", "site"}},
	{name: "rehearsal", event: "workflow_dispatch", refType: "branch"},
	{name: "pushed tag", event: "push", refType: "tag"},
}

// simulateRelease evaluates every job's if: in the order written, the way GitHub
// does: a job that names no status function runs only when every job it depends on,
// directly or not, succeeded. verify and the gate conclude as stated, and any other
// job that runs concludes success. Only a job's direct needs are in its context, so
// a condition reading any other job's result fails closed. It reports each job's
// result, or an error when a condition cannot be evaluated.
func simulateRelease(workflow string, run releaseRun, gateResult string) (map[string]string, error) {
	results := map[string]string{}
	ancestors := map[string][]string{}
	ids := workflowJobIDs(workflow)
	// Written order is not dependency order, so go round until every job is placed.
	for pass := 0; pass <= len(ids) && len(results) < len(ids); pass++ {
	nextJob:
		for _, id := range ids {
			if _, done := results[id]; done {
				continue
			}
			block, _ := workflowJobBlock(workflow, id)
			needs := jobNeeds(block)
			var all []string
			for _, need := range needs {
				if !slices.Contains(ids, need) {
					return nil, fmt.Errorf("job %s needs %s, which the workflow does not define", id, need)
				}
				if _, placed := results[need]; !placed {
					continue nextJob
				}
				all = append(append(all, need), ancestors[need]...)
			}
			ancestors[id] = all
			failed, skippedOrWorse := false, false
			for _, a := range all {
				if results[a] != "success" {
					skippedOrWorse = true
				}
				if results[a] == "failure" {
					failed = true
				}
			}
			ctx := map[string]any{
				"github.event_name": run.event,
				"github.ref_type":   run.refType,
				"github.ref_name":   "v1.2.3",
				"inputs.tag":        "v1.2.3",
				"inputs.create_tag": run.createTag,
				"inputs.ref":        run.ref,
				"success()":         !skippedOrWorse,
				"failure()":         failed,
				"cancelled()":       false,
			}
			for _, need := range needs {
				ctx["needs."+need+".result"] = results[need]
			}
			cond, ok := jobKey(block, "if")
			if !ok {
				cond = "success()"
			}
			will, err := actionsexpr.EvalIf(cond, ctx)
			if err != nil {
				return nil, fmt.Errorf("job %s: if %q: %w", id, cond, err)
			}
			switch {
			case !will:
				results[id] = "skipped"
			case id == releaseMacOSGateJob:
				results[id] = gateResult
			default:
				results[id] = "success"
			}
		}
	}
	if len(results) < len(ids) {
		return nil, fmt.Errorf("the jobs' needs form a cycle: placed %d of %d", len(results), len(ids))
	}
	return results, nil
}

// publishBypasses returns every way a tag, a Release or an asset could be made in
// release.yml without the macOS suite having passed on the released commit.
func publishBypasses(workflow string) []string {
	var found []string
	bad := func(format string, args ...any) { found = append(found, fmt.Sprintf(format, args...)) }

	head, _, ok := strings.Cut(workflow, "\njobs:\n")
	if !ok {
		return []string{"no jobs: key; the parser or the workflow changed shape"}
	}
	if regexp.MustCompile(`(?m)^  [a-z-]+: write\b`).MatchString(head) {
		bad("the workflow's own permissions hold a write scope, so a job declaring none could publish")
	}
	gate, ok := workflowJobBlock(workflow, releaseMacOSGateJob)
	if !ok {
		return []string{"release.yml has no " + releaseMacOSGateJob + " job"}
	}
	if _, ok := jobKey(gate, "continue-on-error"); ok {
		bad("the gate carries continue-on-error, so a failure reads as success to the jobs that need it")
	}
	if needs := jobNeeds(gate); len(needs) != macOSShardCount {
		bad("the gate needs %q; want all %d shards", needs, macOSShardCount)
	} else {
		for n := 1; n <= macOSShardCount; n++ {
			if !slices.Contains(needs, releaseMacOSShardJob(n)) {
				bad("the gate does not need shard job %s", releaseMacOSShardJob(n))
			}
		}
	}
	// The gate must run when a shard did not succeed, or it says nothing.
	if cond, ok := jobKey(gate, "if"); !ok {
		bad("the gate carries no if; with the implicit success() it is skipped when a shard fails, and says nothing")
	} else if run, err := actionsexpr.EvalIf(cond, map[string]any{
		"github.event_name": "push", "github.ref_type": "branch",
		"success()": false, "failure()": true, "cancelled()": false,
	}); err != nil {
		bad("the gate's if %q cannot be evaluated: %v", cond, err)
	} else if !run {
		bad("the gate's if %q skips it when a shard failed, so no message names the shard", cond)
	}

	var publishers []string
	for _, id := range workflowJobIDs(workflow) {
		block, _ := workflowJobBlock(workflow, id)
		if !jobPublishes(block) {
			continue
		}
		publishers = append(publishers, id)
		if !slices.Contains(jobNeeds(block), releaseMacOSGateJob) {
			bad("job %s can publish and does not need %s", id, releaseMacOSGateJob)
		}
		if _, ok := jobKey(block, "continue-on-error"); ok {
			bad("job %s can publish and carries continue-on-error", id)
		}
		// Its own condition stops it, with every other need green: a job must not
		// lean on another publishing job to stop it, since that job can move.
		cond, ok := jobKey(block, "if")
		if !ok {
			cond = "success()"
		}
		for _, result := range nonSuccessResults {
			ctx := map[string]any{
				"github.event_name": "push",
				"github.ref_type":   "branch",
				"inputs.create_tag": true,
				"success()":         false,
				"failure()":         result == "failure",
				"cancelled()":       false,
			}
			for _, need := range jobNeeds(block) {
				ctx["needs."+need+".result"] = "success"
			}
			ctx["needs."+releaseMacOSGateJob+".result"] = result
			run, err := actionsexpr.EvalIf(cond, ctx)
			if err != nil {
				bad("job %s: if %q: %v", id, cond, err)
			} else if run {
				bad("job %s's if %q runs it with the gate reading %q and every other need green", id, cond, result)
			}
		}
	}
	for _, want := range []string{"tag", "release", "site"} {
		if !slices.Contains(publishers, want) {
			bad("job %s is no longer read as publishing; the parser or the workflow changed shape", want)
		}
	}

	for _, run := range releaseRuns {
		for _, result := range append([]string{"success"}, nonSuccessResults...) {
			got, err := simulateRelease(workflow, run, result)
			if err != nil {
				bad("%s, gate %q: %v", run.name, result, err)
				continue
			}
			for _, id := range publishers {
				ran := got[id] == "success"
				want := result == "success" && slices.Contains(run.publishes, id)
				if ran != want {
					bad("%s: job %s runs=%v with the gate reading %q; want %v", run.name, id, ran, result, want)
				}
			}
		}
	}
	return found
}

// TestEveryPublishingJobNeedsTheMacOSGate holds release.yml's publishing jobs to
// the gate twice over: each names it in its needs, and each condition is evaluated
// with the gate reading every result, on every way the workflow starts. Evaluated
// rather than matched, since a clause can be present and mean something else, and a
// job whose if: names a status function runs past a need that failed.
//
// A job counts as publishing by what it can do, not by name, so a job added later
// that holds a write scope is held to the same rule.
func TestEveryPublishingJobNeedsTheMacOSGate(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	workflow := readRepoFile(t, root, releaseWorkflowFile)
	for _, finding := range publishBypasses(workflow) {
		t.Error(finding)
	}

	// Each way of bypassing it, applied to the real file, must be caught: a check
	// that passes on the tree says nothing until it has been seen to refuse.
	replaceInJob := func(job, from, to string) func(string) string {
		return func(wf string) string {
			block, ok := workflowJobBlock(wf, job)
			if !ok {
				t.Fatalf("no %s job", job)
			}
			if !strings.Contains(block, from) {
				t.Fatalf("job %s does not read %q", job, from)
			}
			return strings.Replace(wf, block, strings.Replace(block, from, to, 1), 1)
		}
	}
	const extraJob = "  extra-publish:\n    needs: verify\n    runs-on: ubuntu-latest\n" +
		"    permissions:\n      contents: write\n    steps:\n      - run: gh release upload\n\n"
	for name, mutate := range map[string]func(string) string{
		"the tag job does not need the gate":     replaceInJob("tag", "needs: [verify, "+releaseMacOSGateJob+"]", "needs: verify"),
		"the release job does not need the gate": replaceInJob("release", "needs: [verify, "+releaseMacOSGateJob+", tag]", "needs: [verify, tag]"),
		"the release job's if ignores the gate":  replaceInJob("release", " && needs."+releaseMacOSGateJob+".result == 'success'", ""),
		"the site job does not need the gate":    replaceInJob("site", "needs: [release, "+releaseMacOSGateJob+"]", "needs: release"),
		"the site job's if ignores the gate":     replaceInJob("site", " && needs."+releaseMacOSGateJob+".result == 'success'", ""),
		"the gate leaves a shard out":            replaceInJob(releaseMacOSGateJob, ", "+releaseMacOSShardJob(macOSShardCount)+"]", "]"),
		"the gate is skipped when a shard fails": replaceInJob(releaseMacOSGateJob, "always() && ", ""),
		"the gate's failure reads as success":    replaceInJob(releaseMacOSGateJob, "    timeout-minutes:", "    continue-on-error: true\n    timeout-minutes:"),
		"a new job publishes round the gate": func(wf string) string {
			return strings.Replace(wf, "  # The manual rehearsal (workflow_dispatch).", extraJob+"  # The manual rehearsal (workflow_dispatch).", 1)
		},
		"the workflow's floor can write": func(wf string) string {
			return strings.Replace(wf, "permissions:\n  contents: read\n\njobs:", "permissions:\n  contents: write\n\njobs:", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if len(publishBypasses(mutate(workflow))) == 0 {
				t.Errorf("publishBypasses found nothing wrong once %s", name)
			}
		})
	}
}

// TestNothingOutsideReleaseYMLTagsOrPublishes closes the other way round the gate:
// another workflow that tags, creates a Release or starts the production site path
// would never meet release.yml's needs. Only auto-release calls release.yml, on its
// one job with a write scope, and only release.yml asks site.yml for production.
func TestNothingOutsideReleaseYMLTagsOrPublishes(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no workflows: %v", err)
	}
	for _, p := range paths {
		name := filepath.Base(p)
		wf := readRepoFile(t, root, ".github/workflows/"+name)
		steps, err := workflowSteps(wf)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, s := range steps {
			// A comment may name a verb: site.yml's say what made the Release it reads.
			for _, line := range strings.Split(s.body, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue
				}
				for _, verb := range []string{"git tag -a", `git push origin "refs/tags/`, "gh release create"} {
					if strings.Contains(line, verb) && name != "release.yml" {
						t.Errorf("%s step %q runs %q; a tag or a Release is made in release.yml alone, behind its macOS gate", name, s.name, verb)
					}
				}
			}
		}
		if strings.Contains(wf, "uses: ./.github/workflows/release.yml") && name != "auto-release.yml" {
			t.Errorf("%s calls release.yml; only auto-release does", name)
		}
		if regexp.MustCompile(`(?m)^\s+mode: production\s*$`).MatchString(wf) && name != "release.yml" {
			t.Errorf("%s asks site.yml for production; only release.yml does, behind its macOS gate", name)
		}
	}
}

// TestReleaseMacOSShardsRunTheReleasedCommit holds the shard jobs to the commit the
// tag job tags, to the suite's one shard count, and to no more than they need: read
// access to the code, no secrets, no cache (this path publishes, and a cache a
// branch could have written has no business in it). Each is its own job with no
// fail-fast, so a shard that fails leaves the others running to a verdict and a
// re-run repeats only what failed.
func TestReleaseMacOSShardsRunTheReleasedCommit(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	workflow := readRepoFile(t, root, releaseWorkflowFile)

	tag, ok := workflowJobBlock(workflow, "tag")
	if !ok {
		t.Fatal("release.yml has no tag job")
	}
	commit, ok := regexpFirst(tag, `(?m)^\s+COMMIT: (.+?)\s*$`)
	if !ok {
		t.Fatal("the tag job's step no longer reads COMMIT; the test or the workflow changed shape")
	}
	verify, ok := workflowJobBlock(workflow, "verify")
	if !ok {
		t.Fatal("release.yml has no verify job")
	}
	verifyIf, _ := jobKey(verify, "if")

	for n := 1; n <= macOSShardCount; n++ {
		id := releaseMacOSShardJob(n)
		job, ok := workflowJobBlock(workflow, id)
		if !ok {
			t.Errorf("release.yml has no %s job", id)
			continue
		}
		if got, _ := jobKey(job, "uses"); got != "./"+macOSSuiteWorkflow {
			t.Errorf("%s uses %q; want ./%s, the steps ci.yml's suite runs", id, got, macOSSuiteWorkflow)
		}
		for key, want := range map[string]string{
			"shard":  strconv.Itoa(n),
			"shards": strconv.Itoa(macOSShardCount),
			"ref":    commit,
			"cache":  "false",
		} {
			if got, _ := regexpFirst(job, `(?m)^      `+key+`: (.+?)\s*$`); got != want {
				t.Errorf("%s passes %s: %q; want %q", id, key, got, want)
			}
		}
		if got, _ := jobKey(job, "if"); got != verifyIf {
			t.Errorf("%s runs on %q; want verify's %q, so it runs wherever verify does and on no pushed tag", id, got, verifyIf)
		}
		if jobGrantsWrite(job) {
			t.Errorf("%s holds a write scope; it reads the code and publishes nothing", id)
		}
		if !strings.Contains(job, "\n    permissions:\n      contents: read\n") {
			t.Errorf("%s does not state `contents: read`, the one scope it needs", id)
		}
		if _, ok := jobKey(job, "secrets"); ok {
			t.Errorf("%s passes secrets; the suite reads none", id)
		}
		if needs := jobNeeds(job); len(needs) != 0 {
			t.Errorf("%s needs %q; a need that failed would skip the suite and hide what it would have said", id, needs)
		}
		if strings.Contains(job, "strategy:") {
			t.Errorf("%s carries a strategy; each shard is one job, so the gate can read it by name", id)
		}
	}
	if extra := fmt.Sprintf("  %s:", releaseMacOSShardJob(macOSShardCount+1)); strings.Contains(workflow, "\n"+extra+"\n") {
		t.Errorf("release.yml defines %s, past the %d shards the suite runs", extra, macOSShardCount)
	}

	suite := readRepoFile(t, root, macOSSuiteWorkflow)
	head, _, _ := strings.Cut(suite, "\njobs:\n")
	if regexp.MustCompile(`(?m)^  [a-z-]+: write\b`).MatchString(head) {
		t.Errorf("%s's own permissions hold a write scope", macOSSuiteWorkflow)
	}
	shard, ok := workflowJobBlock(suite, macOSShardJob)
	if !ok {
		t.Fatalf("%s has no %s job", macOSSuiteWorkflow, macOSShardJob)
	}
	if jobGrantsWrite(shard) {
		t.Errorf("%s job %s holds a write scope", macOSSuiteWorkflow, macOSShardJob)
	}
	if !strings.Contains(shard, "\n    permissions:\n      contents: read\n") {
		t.Errorf("%s job %s does not state `contents: read`", macOSSuiteWorkflow, macOSShardJob)
	}
	if _, ok := jobKey(shard, "continue-on-error"); ok {
		t.Errorf("%s job %s carries continue-on-error, so a failed shard reads as success", macOSSuiteWorkflow, macOSShardJob)
	}
	if strings.Contains(shard, "secrets.") {
		t.Errorf("%s job %s reads a secret", macOSSuiteWorkflow, macOSShardJob)
	}
	checkout := ""
	for _, s := range jobSteps(shard) {
		if strings.HasPrefix(s.uses, "actions/checkout@") {
			checkout = s.body
		}
		if strings.HasPrefix(s.uses, "actions/setup-go@") && !strings.Contains(s.body, "cache: ${{ inputs.cache }}") {
			t.Errorf("%s: setup-go does not take its cache setting from inputs.cache, so the release path would "+
				"read a cache:\n%s", macOSSuiteWorkflow, s.body)
		}
	}
	if !strings.Contains(checkout, "ref: ${{ inputs.ref }}") {
		t.Errorf("%s: the checkout does not read inputs.ref, so a shard could test another commit than the one tagged:\n%s",
			macOSSuiteWorkflow, checkout)
	}
	if !strings.Contains(checkout, "persist-credentials: false") {
		t.Errorf("%s: the checkout persists its credentials:\n%s", macOSSuiteWorkflow, checkout)
	}
}

// TestMacOSSuiteCallersPassItsDeclaredInputs pins the reusable workflow's inputs to
// both callers' with: blocks. A mismatch is a parse-time failure that fails the
// whole calling run before any job starts, so it would stop ci.yml on every event,
// the merge queue's included, and no needs: can contain it. A job that calls a
// reusable workflow may not carry steps, runs-on or timeout-minutes either.
func TestMacOSSuiteCallersPassItsDeclaredInputs(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	suite := readRepoFile(t, root, macOSSuiteWorkflow)
	head, _, ok := strings.Cut(suite, "\njobs:\n")
	if !ok {
		t.Fatalf("%s: no jobs: key", macOSSuiteWorkflow)
	}
	if got := workflowTriggers(suite); !slices.Equal(got, []string{"workflow_call"}) {
		t.Errorf("%s triggers on %q; it runs only when called, on what the caller names", macOSSuiteWorkflow, got)
	}
	_, inputsPart, ok := strings.Cut(head, "\n    inputs:\n")
	if !ok {
		t.Fatalf("%s declares no workflow_call inputs", macOSSuiteWorkflow)
	}
	var declared []string
	for _, l := range strings.Split(inputsPart, "\n") {
		if m := regexp.MustCompile(`^      ([a-z_]+):\s*$`).FindStringSubmatch(l); m != nil {
			declared = append(declared, m[1])
		}
		if topLevelKeyRe.MatchString(l) {
			break
		}
	}
	slices.Sort(declared)
	if !slices.Equal(declared, []string{"cache", "ref", "shard", "shards"}) {
		t.Fatalf("%s declares inputs %q", macOSSuiteWorkflow, declared)
	}
	if n := strings.Count(inputsPart, "        required: true\n"); n != len(declared) {
		t.Errorf("%s: %d of its %d inputs are required; a caller states each, so none is had by omission",
			macOSSuiteWorkflow, n, len(declared))
	}

	withKeys := func(job string) []string {
		_, rest, ok := strings.Cut(job, "\n    with:\n")
		if !ok {
			return nil
		}
		var keys []string
		for _, l := range strings.Split(rest, "\n") {
			if m := regexp.MustCompile(`^      ([a-z_]+):`).FindStringSubmatch(l); m != nil {
				keys = append(keys, m[1])
			} else if l != "" && !strings.HasPrefix(l, "      ") && !strings.HasPrefix(strings.TrimSpace(l), "#") {
				break
			}
		}
		slices.Sort(keys)
		return keys
	}
	callers := map[string][]string{".github/workflows/ci.yml": {macOSSuiteJob}}
	for n := 1; n <= macOSShardCount; n++ {
		callers[releaseWorkflowFile] = append(callers[releaseWorkflowFile], releaseMacOSShardJob(n))
	}
	for file, jobs := range callers {
		wf := readRepoFile(t, root, file)
		for _, id := range jobs {
			job, ok := workflowJobBlock(wf, id)
			if !ok {
				t.Errorf("%s has no %s job", file, id)
				continue
			}
			if got, _ := jobKey(job, "uses"); got != "./"+macOSSuiteWorkflow {
				t.Errorf("%s job %s uses %q; want ./%s", file, id, got, macOSSuiteWorkflow)
			}
			if got := withKeys(job); !slices.Equal(got, declared) {
				t.Errorf("%s job %s passes %q; %s declares %q", file, id, got, macOSSuiteWorkflow, declared)
			}
			for _, key := range []string{"runs-on", "steps", "timeout-minutes", "env"} {
				if regexp.MustCompile(`(?m)^    ` + key + `:`).MatchString(job) {
					t.Errorf("%s job %s carries %s:, which GitHub refuses on a job that calls a workflow, "+
						"failing the whole run before any job starts", file, id, key)
				}
			}
		}
	}

	// Every other workflow leaves it alone: a caller this test does not read would
	// hold none of the above.
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		rel := ".github/workflows/" + filepath.Base(p)
		if _, known := callers[rel]; known {
			continue
		}
		if strings.Contains(readRepoFile(t, root, rel), "./"+macOSSuiteWorkflow) {
			t.Errorf("%s calls %s; add it to this test's callers", rel, macOSSuiteWorkflow)
		}
	}

	ci, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), macOSSuiteJob)
	if !ok {
		t.Fatal("ci.yml has no macOS suite job")
	}
	for key, want := range map[string]string{
		"shard":  "${{ matrix.shard }}",
		"shards": "${{ strategy.job-total }}",
		"ref":    "${{ github.sha }}",
		"cache":  "true",
	} {
		if got, _ := regexpFirst(ci, `(?m)^      `+key+`: (.+?)\s*$`); got != want {
			t.Errorf("ci.yml job %s passes %s: %q; want %q", macOSSuiteJob, key, got, want)
		}
	}
}

// TestMacOSGateStopsTheReleaseAndSaysWhich runs the gate's own script over every
// result each shard can read: it passes only when all of them succeeded, and otherwise
// names each shard that did not, what became of it, and the remedy, in the log and
// the step summary. A shard that never got a runner gets the words its job's
// annotation carries, since that is the line the person has just read on the run.
func TestMacOSGateStopsTheReleaseAndSaysWhich(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	workflow := readRepoFile(t, root, releaseWorkflowFile)
	job, ok := workflowJobBlock(workflow, releaseMacOSGateJob)
	if !ok {
		t.Fatalf("release.yml has no %s job", releaseMacOSGateJob)
	}
	for key, want := range map[string]string{"runs-on": "ubuntu-latest", "permissions": "{}"} {
		if got, _ := jobKey(job, key); got != want {
			t.Errorf("the gate's %s is %q; want %q", key, got, want)
		}
	}
	steps := jobSteps(job)
	if len(steps) != 1 {
		t.Fatalf("the gate has %d steps; want the one that reads the shards' results", len(steps))
	}
	step := steps[0]
	if step.cond != "always()" {
		t.Errorf("the gate's step reads `if: %s`; want always(): a step skipped on a cancelled run leaves a job "+
			"that concludes success", step.cond)
	}
	if step.uses != "" {
		t.Errorf("the gate's step uses %s; it reads results and takes no action", step.uses)
	}
	script := step.script()
	if m := toolchainCommandRe.FindString(script); m != "" {
		t.Errorf("the gate's step runs %q; it runs nothing but its verdict", strings.TrimSpace(m))
	}
	if strings.Contains(script, "${{") {
		t.Errorf("the gate's step interpolates an expression inside run:; pass values through env")
	}
	for n := 1; n <= macOSShardCount; n++ {
		key, want := fmt.Sprintf("SHARD_%d_RESULT", n), "${{ needs."+releaseMacOSShardJob(n)+".result }}"
		if got, _ := regexpFirst(step.body, `(?m)^\s+`+key+`: (.+?)\s*$`); got != want {
			t.Errorf("the gate's step: %s is %q; want %q", key, got, want)
		}
	}

	allGreen := func() [macOSShardCount]string {
		var results [macOSShardCount]string
		for i := range results {
			results[i] = "success"
		}
		return results
	}
	run := func(t *testing.T, results [macOSShardCount]string) (string, string, error) {
		t.Helper()
		summary := filepath.Join(t.TempDir(), "summary")
		// The flags the runner gives bash for a step that names no shell.
		cmd := exec.Command("bash", "-e", "-c", script)
		cmd.Env = append(os.Environ(), "GITHUB_RUN_ID=4242", "GITHUB_STEP_SUMMARY="+summary)
		for i, r := range results {
			cmd.Env = append(cmd.Env, fmt.Sprintf("SHARD_%d_RESULT=%s", i+1, r))
		}
		out, err := cmd.CombinedOutput()
		written, _ := os.ReadFile(summary)
		return string(out), string(written), err
	}

	if out, _, err := run(t, allGreen()); err != nil {
		t.Errorf("the gate failed with every shard green: %v\n%s", err, out)
	} else if strings.Contains(out, "::error") {
		t.Errorf("the gate raised an error with every shard green:\n%s", out)
	}

	for n := 1; n <= macOSShardCount; n++ {
		for _, result := range nonSuccessResults {
			t.Run(fmt.Sprintf("shard %d reads %q", n, result), func(t *testing.T) {
				results := allGreen()
				results[n-1] = result
				out, summary, err := run(t, results)
				if err == nil {
					t.Fatalf("the gate passed with shard %d reading %q:\n%s", n, result, out)
				}
				if !strings.Contains(out, "::error") {
					t.Errorf("the refusal raises no error annotation:\n%s", out)
				}
				for _, says := range []string{
					fmt.Sprintf("shard %d of %d", n, macOSShardCount),
					"re-run the failed jobs",
					"gh run rerun 4242 --failed",
					"made no tag",
				} {
					if !strings.Contains(out, says) || !strings.Contains(summary, says) {
						t.Errorf("the refusal does not say %q in both the log and the summary:\n%s\nsummary:\n%s", says, out, summary)
					}
				}
				for other := 1; other <= macOSShardCount; other++ {
					if other != n && strings.Contains(out, fmt.Sprintf("shard %d of %d", other, macOSShardCount)) {
						t.Errorf("the refusal names shard %d, which succeeded:\n%s", other, out)
					}
				}
				if result == "abandoned" && !strings.Contains(out, "not acquired by Runner") {
					t.Errorf("an abandoned shard's refusal does not carry the annotation's words:\n%s", out)
				}
			})
		}
	}

	t.Run("every shard that did not succeed is named", func(t *testing.T) {
		results := allGreen()
		results[0], results[macOSShardCount-1] = "failure", "abandoned"
		out, _, err := run(t, results)
		if err == nil {
			t.Fatalf("the gate passed:\n%s", out)
		}
		for _, says := range []string{
			fmt.Sprintf("shard 1 of %d", macOSShardCount), "'failure'",
			fmt.Sprintf("shard %d of %d", macOSShardCount, macOSShardCount), "'abandoned'",
		} {
			if !strings.Contains(out, says) {
				t.Errorf("the refusal does not say %q:\n%s", says, out)
			}
		}
	})
}
