package scaffold

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// writeWorkflows lays the named workflow files under repo/.github/workflows.
func writeWorkflows(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	dir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// TestRequiredChecksKeepsOnlyChecksEveryPullRequestReports pins
// itd-2610290746077716's "only the checks that run" criterion: a check is
// required only when every pull request into the default branch reports it, so
// a path-filtered workflow, one whose base-branch filter leaves the default
// branch out, and one that does not run on the merge queue while a merge queue
// is on are each left out and named, never required.
func TestRequiredChecksKeepsOnlyChecksEveryPullRequestReports(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"ci.yml": `on:
  pull_request:
    branches: [main]
  merge_group:
jobs:
  build:
    runs-on: ubuntu-latest
  lint:
    name: lint
    runs-on: ubuntu-latest
  matrix-job:
    strategy:
      matrix:
        os: [a, b]
        go: [c, d]
`,
		"docs.yml": `on:
  pull_request:
    paths:
      - docs/**
  merge_group:
jobs:
  docs:
    runs-on: ubuntu-latest
`,
		"other-base.yml": `on:
  pull_request:
    branches:
      - release/*
  merge_group:
jobs:
  release-only:
    runs-on: ubuntu-latest
`,
		"no-queue.yml": `on: [pull_request]
jobs:
  pr-only:
    runs-on: ubuntu-latest
`,
		"push.yml": `on: push
jobs:
  nightly:
    runs-on: ubuntu-latest
`,
	})

	checks, notes := requiredChecksOf(t, repo, true)
	if want := []string{"build", "lint"}; !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %v, want %v", checks, want)
	}
	joined := strings.Join(notes, "\n")
	for _, want := range []string{"docs.yml", "other-base.yml", "no-queue.yml", "matrix-job"} {
		if !strings.Contains(joined, want) {
			t.Errorf("notes do not name %s:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "push.yml") {
		t.Errorf("a workflow that never runs on a pull request is noise in the notes:\n%s", joined)
	}

	// Without a merge queue, the pull-request-only workflow gates every merge.
	checks, _ = requiredChecksOf(t, repo, false)
	if want := []string{"build", "lint", "pr-only"}; !reflect.DeepEqual(checks, want) {
		t.Errorf("without a merge queue, checks = %v, want %v", checks, want)
	}
}

// TestRequiredChecksWithNoWorkflowsIsEmpty: a repository with no CI is given
// no required check, so nothing can block its merges.
func TestRequiredChecksWithNoWorkflowsIsEmpty(t *testing.T) {
	checks, notes := requiredChecksOf(t, t.TempDir(), false)
	if len(checks) != 0 || len(notes) != 0 {
		t.Errorf("checks %v, notes %v; want none", checks, notes)
	}
}

// TestRequiredChecksAcceptsAGlobBaseFilterThatMatchesTheDefaultBranch: a
// `branches: ['**']` or `['ma*']` filter still covers the default branch.
func TestRequiredChecksAcceptsAGlobBaseFilterThatMatchesTheDefaultBranch(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"a.yml": "on:\n  pull_request:\n    branches: ['**']\njobs:\n  a:\n    runs-on: x\n",
		"b.yml": "on:\n  pull_request:\n    branches:\n      - \"ma*\"\njobs:\n  b:\n    runs-on: x\n",
		"c.yml": "on:\n  pull_request:\n    branches-ignore: [main]\njobs:\n  c:\n    runs-on: x\n",
		"d.yml": "on:\n  pull_request:\n    branches: [release/**]\njobs:\n  d:\n    runs-on: x\n",
	})
	checks, _ := requiredChecksOf(t, repo, false)
	if want := []string{"a", "b"}; !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %v, want %v", checks, want)
	}
}

// TestRequiredChecksExpandsASingleKeyMatrix: a job with one matrix key listed
// inline reports one check per value, `<name> (<value>)`, as GitHub names
// them; any other matrix shape is still left out and named.
func TestRequiredChecksExpandsASingleKeyMatrix(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"ci.yml": `on: pull_request
jobs:
  check:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
  named:
    name: unit
    strategy:
      matrix:
        go: ["1.25", '1.26']
  two-keys:
    strategy:
      matrix:
        os: [a, b]
        go: [c]
  with-include:
    strategy:
      matrix:
        os: [a]
        include:
          - os: b
`,
	})
	checks, notes := requiredChecksOf(t, repo, false)
	want := []string{"check (macos-latest)", "check (ubuntu-latest)", "unit (1.25)", "unit (1.26)"}
	if !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %v, want %v", checks, want)
	}
	joined := strings.Join(notes, "\n")
	for _, want := range []string{"two-keys", "with-include"} {
		if !strings.Contains(joined, want) {
			t.Errorf("notes do not name %s:\n%s", want, joined)
		}
	}
}

// TestRequiredChecksLeavesOutAWorkflowWhoseTypesSkipAPush pins the activity
// filter: a pull-request trigger with `types:` runs only on the activities it
// names, so a workflow that leaves out opened, synchronize or reopened does not
// report on every pull request and requiring its checks would hold every merge
// on a check that never arrives. A `types:` list that keeps all three, as the
// default does, still counts.
func TestRequiredChecksLeavesOutAWorkflowWhoseTypesSkipAPush(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"labelled.yml": `on:
  pull_request:
    types: [labeled]
jobs:
  label-check:
    runs-on: ubuntu-latest
`,
		"default-types.yml": `on:
  pull_request:
    types:
      - opened
      - synchronize
      - reopened
      - labeled
jobs:
  build:
    runs-on: ubuntu-latest
`,
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]", checks)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "labelled.yml") {
		t.Errorf("the labelled-only workflow is not named in the notes: %v", notes)
	}
}

// TestRequiredChecksLeavesOutAFlowStyleTrigger pins that a pull-request
// trigger written as a flow mapping has its filters seen, so it is left out and
// named rather than read as unfiltered: requiring a check a path or branch filter
// skips would hold merges on it.
func TestRequiredChecksLeavesOutAFlowStyleTrigger(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"flow-on.yml": `on: {pull_request: {paths: [docs/**]}}
jobs:
  docs:
    runs-on: ubuntu-latest
`,
		"flow-event.yml": `on:
  pull_request: {branches: [release]}
jobs:
  release-only:
    runs-on: ubuntu-latest
`,
		"list.yml": `on: [pull_request, push]
jobs:
  build:
    runs-on: ubuntu-latest
`,
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]", checks)
	}
	joined := strings.Join(notes, "\n")
	for _, f := range []string{"flow-on.yml", "flow-event.yml"} {
		if !strings.Contains(joined, f) {
			t.Errorf("%s is not named in the notes: %v", f, notes)
		}
	}
}

// TestRequiredChecksNeverReadsAStrategyAsTheBareJob pins that a job with a matrix
// is never read as its bare id, however its strategy is written: GitHub reports
// one check per matrix entry, so requiring the bare id would wait for ever. Written
// inline, the values are as readable as on their own lines; computed, only the run
// knows them, and the job is named.
func TestRequiredChecksNeverReadsAStrategyAsTheBareJob(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"ci.yml": `on:
  pull_request:
jobs:
  build:
    strategy: {matrix: {os: [a, b]}}
    runs-on: ubuntu-latest
  computed:
    strategy: ${{ fromJSON(x) }}
    runs-on: ubuntu-latest
  lint:
    runs-on: ubuntu-latest
`,
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if want := []string{"build (a)", "build (b)", "lint"}; !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %v, want %v", checks, want)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "job computed") {
		t.Errorf("the computed-strategy job is not named: %v", notes)
	}
}

// TestRequiredChecksReadsOnlyTheEventsOnNamesAtItsTopLevel pins that a key or
// list item named pull_request deeper inside `on:` (a dispatch input, a path
// filter) is not the trigger: a workflow that never runs on a pull request is
// never required.
func TestRequiredChecksReadsOnlyTheEventsOnNamesAtItsTopLevel(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"dispatch.yml": `on:
  workflow_dispatch:
    inputs:
      pull_request:
        description: PR number
jobs:
  manual:
    runs-on: ubuntu-latest
`,
		"push.yml": `on:
  push:
    paths:
      - '.github/workflows/pull_request.yml'
jobs:
  on-push:
    runs-on: ubuntu-latest
`,
		"ci.yml": `on:
  pull_request:
jobs:
  build:
    runs-on: ubuntu-latest
`,
		"with-input.yml": `on:
  pull_request:
  workflow_dispatch:
    inputs:
      pull_request: {description: PR, required: false}
jobs:
  test:
    runs-on: ubuntu-latest
`,
	})
	checks, _ := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build", "test"}) {
		t.Errorf("checks = %v, want [build test]", checks)
	}
}

// TestRequiredChecksSeesTheFiltersInEveryForm pins the filter spellings the
// reader must not miss: a quoted filter key, a flow mapping on the line after
// the event, and an alias, each of which would otherwise read as unfiltered.
func TestRequiredChecksSeesTheFiltersInEveryForm(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"quoted.yml":    "on:\n  pull_request:\n    \"paths\": [\"docs/**\"]\njobs:\n  docs:\n    runs-on: x\n",
		"next-line.yml": "on:\n  pull_request:\n    {paths: [docs/**]}\njobs:\n  docs2:\n    runs-on: x\n",
		"alias.yml":     "x-pr: &pr\n  paths: [x]\non:\n  pull_request: *pr\njobs:\n  aliased:\n    runs-on: x\n",
		"merge-key.yml": "x-filters: &filters\n  paths: [x]\non:\n  pull_request:\n    <<: *filters\njobs:\n  merged:\n    runs-on: x\n",
		"ci.yml":        "on:\n  pull_request:\njobs:\n  build:\n    runs-on: x\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]; notes %v", checks, notes)
	}
}

// TestRequiredChecksSeesThroughAnAnchor pins that an anchor (`&name`) in
// front of a trigger's value hides nothing from the reader: a flow mapping
// behind one, on the event, on `on:` itself or on the line after the key,
// is still left out, while an anchor over block filters still has them read.
// It also pins the merge-key and alias forms the reader must see past the first
// line and on the line after the key.
func TestRequiredChecksSeesThroughAnAnchor(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"anchored-event.yml": "on:\n  pull_request: &f {paths: [docs/**]}\n  push: *f\njobs:\n  docs:\n    runs-on: x\n",
		"anchored-on.yml":    "on: &t {pull_request: {paths: [x]}}\njobs:\n  docs2:\n    runs-on: x\n",
		"anchored-next.yml":  "on:\n  pull_request:\n    &f {paths: [x]}\njobs:\n  docs3:\n    runs-on: x\n",
		"anchored-block.yml": "on:\n  pull_request: &f\n    paths: [x]\njobs:\n  docs4:\n    runs-on: x\n",
		"merge-after.yml":    "x-filters: &filters\n  paths: [x]\non:\n  pull_request:\n    types: [opened, synchronize, reopened]\n    <<: *filters\njobs:\n  merged:\n    runs-on: x\n",
		"alias-next.yml":     "x-pr: &pr\n  paths: [x]\non:\n  pull_request:\n    *pr\njobs:\n  aliased:\n    runs-on: x\n",
		"anchored-bare.yml":  "on:\n  pull_request: &f\njobs:\n  plain:\n    runs-on: x\n",
		"ci.yml":             "on:\n  pull_request:\njobs:\n  build:\n    runs-on: x\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build", "plain"}) {
		t.Errorf("checks = %v, want [build plain]; notes %v", checks, notes)
	}
}

// TestRequiredChecksLeavesOutAnyTriggerShapeItDoesNotRead pins the fallback the
// read rests on: under a pull-request trigger it accepts only GitHub's five filter
// keys, and at the events' level only plain keys. A key behind an anchor or a tag or
// written with `?` is still that key, so its filter is seen, and any other shape (a
// merge key, an unknown key) leaves the workflow out rather than read it as unfiltered.
func TestRequiredChecksLeavesOutAnyTriggerShapeItDoesNotRead(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"anchor-key.yml":   "on:\n  pull_request:\n    &p paths: [x]\njobs:\n  a:\n    runs-on: x\n",
		"tag-key.yml":      "on:\n  pull_request:\n    !!str paths: [x]\njobs:\n  b:\n    runs-on: x\n",
		"complex-key.yml":  "on:\n  pull_request:\n    ? paths\n    : [x]\njobs:\n  c:\n    runs-on: x\n",
		"tagged-event.yml": "on:\n  !!str pull_request:\n    paths: [x]\njobs:\n  d:\n    runs-on: x\n",
		"merge-on.yml":     "x-triggers: &triggers\n  push:\non:\n  <<: *triggers\n  pull_request:\njobs:\n  e:\n    runs-on: x\n",
		"unknown-key.yml":  "on:\n  pull_request:\n    pathz: [x]\njobs:\n  f:\n    runs-on: x\n",
		"filters.yml":      "on:\n  pull_request:\n    # all five, plainly\n    'branches': [main]\n    types: [opened, synchronize, reopened]\n  push:\n    branches: [main]\njobs:\n  build:\n    runs-on: x\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]; notes %v", checks, notes)
	}
}

// TestRequiredChecksLeavesOutAMatrixJobThatCanBeSkipped: a matrix job skipped on
// some run reports one check under its unexpanded name, so the `<name> (<value>)`
// contexts never arrive there, and requiring them holds that pull request for ever.
// A job can be skipped by its own `if:`, or, with no `if:` of its own, by the
// implicit success() on any of its ancestors. A job whose own condition reads only
// the run's state, always() or !cancelled(), runs on a run that is going ahead. A
// job with no matrix still counts: skipped, it reports its one name, which a ruleset
// reads as passed.
func TestRequiredChecksLeavesOutAMatrixJobThatCanBeSkipped(t *testing.T) {
	const matrix = `
    strategy:
      matrix:
        os: [a, b]`
	repo := writeWorkflows(t, map[string]string{
		"ci.yml": `on: pull_request
jobs:
  plain:
    runs-on: ubuntu-latest
  gated:
    if: github.event_name != 'pull_request'
    runs-on: ubuntu-latest
  through-gated:
    needs: gated
    runs-on: ubuntu-latest
  own-if:
    if: github.event_name != 'pull_request'` + matrix + `
  folded-if:
    if: >-
      github.event_name != 'pull_request'` + matrix + `
  needs-scalar:
    needs: gated` + matrix + `
  needs-flow:
    needs: [plain, gated] # a comment` + matrix + `
  needs-block:
    needs:
      - plain
      - gated` + matrix + `
  needs-block-level:
    needs:
    - plain
    - gated` + matrix + `
  needs-through:
    needs: through-gated` + matrix + `
  keeps-always:
    needs: gated
    if: always()` + matrix + `
  keeps-not-cancelled:
    needs: gated
    if: ${{ !cancelled() }}` + matrix + `
  keeps-quoted:
    if: "${{ always() }}"` + matrix + `
  keeps-need-on-plain:
    needs: [plain]` + matrix + `
  behind-always:
    needs: keeps-always` + matrix + `
`,
	})
	checks, notes := requiredChecksOf(t, repo, false)
	want := []string{"gated", "plain", "through-gated"}
	for _, job := range []string{"keeps-always", "keeps-need-on-plain", "keeps-not-cancelled", "keeps-quoted"} {
		want = append(want, job+" (a)", job+" (b)")
	}
	sort.Strings(want)
	if !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %q\nwant %q", checks, want)
	}
	joined := strings.Join(notes, "\n")
	for _, job := range []string{"own-if", "folded-if", "needs-scalar", "needs-flow", "needs-block", "needs-block-level", "needs-through", "behind-always"} {
		if !strings.Contains(joined, "job "+job+" ") {
			t.Errorf("notes do not name %s:\n%s", job, joined)
		}
	}
}

// TestRequiredChecksReadsThisRepositorysShardedSuite: this repository's macOS suite
// is a matrix a pull request skips, behind one job that reports the context the
// ruleset requires. Only that job's name is a required check, and every context the
// ruleset mirror requires is one the read proposes.
func TestRequiredChecksReadsThisRepositorysShardedSuite(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	checks, _ := requiredChecksOf(t, root, true)
	mirror, err := os.ReadFile(filepath.Join(root, ".abcd", "work", "rulesets", "main-protection.json"))
	if err != nil {
		t.Fatal(err)
	}
	contexts := regexp.MustCompile(`"context": "([^"]+)"`).FindAllStringSubmatch(string(mirror), -1)
	if len(contexts) < 2 {
		t.Fatalf("the ruleset mirror names %d contexts; the test or the mirror changed shape", len(contexts))
	}
	for _, m := range contexts {
		if !slices.Contains(checks, m[1]) {
			t.Errorf("RequiredChecks = %q, which leaves out %q", checks, m[1])
		}
	}
	for _, c := range checks {
		if strings.HasPrefix(c, "macos-suite") {
			t.Errorf("RequiredChecks names %q, which no pull request reports", c)
		}
	}
}
