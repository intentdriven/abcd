package scaffold

import (
	"os"
	"path/filepath"
	"reflect"
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

	checks, notes := RequiredChecks(repo, "main", true)
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
	checks, _ = RequiredChecks(repo, "main", false)
	if want := []string{"build", "lint", "pr-only"}; !reflect.DeepEqual(checks, want) {
		t.Errorf("without a merge queue, checks = %v, want %v", checks, want)
	}
}

// TestRequiredChecksWithNoWorkflowsIsEmpty: a repository with no CI is given
// no required check, so nothing can block its merges.
func TestRequiredChecksWithNoWorkflowsIsEmpty(t *testing.T) {
	checks, notes := RequiredChecks(t.TempDir(), "main", false)
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
	checks, _ := RequiredChecks(repo, "main", false)
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
	checks, notes := RequiredChecks(repo, "main", false)
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
	checks, notes := RequiredChecks(repo, "main", false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]", checks)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "labelled.yml") {
		t.Errorf("the labelled-only workflow is not named in the notes: %v", notes)
	}
}

// TestRequiredChecksLeavesOutAFlowStyleTrigger pins that a pull-request
// trigger written as a flow mapping, whose filters this line reader cannot
// see, is left out and named rather than read as unfiltered: requiring a check
// a path or branch filter skips would hold merges on it.
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
	checks, notes := RequiredChecks(repo, "main", false)
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

// TestRequiredChecksLeavesOutAnInlineStrategy pins that a job whose strategy
// is written inline (`strategy: {matrix: …}`) is read as a matrix whose names
// only the run knows, never as the bare job id: GitHub reports one check per
// matrix entry, so requiring the bare id would wait for ever.
func TestRequiredChecksLeavesOutAnInlineStrategy(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"ci.yml": `on:
  pull_request:
jobs:
  build:
    strategy: {matrix: {os: [a, b]}}
    runs-on: ubuntu-latest
  lint:
    runs-on: ubuntu-latest
`,
	})
	checks, notes := RequiredChecks(repo, "main", false)
	if !reflect.DeepEqual(checks, []string{"lint"}) {
		t.Errorf("checks = %v, want [lint]", checks)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "job build") {
		t.Errorf("the inline-strategy job is not named: %v", notes)
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
	checks, _ := RequiredChecks(repo, "main", false)
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
		"alias.yml":     "on:\n  pull_request: *pr\njobs:\n  aliased:\n    runs-on: x\n",
		"merge-key.yml": "on:\n  pull_request:\n    <<: *filters\njobs:\n  merged:\n    runs-on: x\n",
		"ci.yml":        "on:\n  pull_request:\njobs:\n  build:\n    runs-on: x\n",
	})
	checks, notes := RequiredChecks(repo, "main", false)
	if !reflect.DeepEqual(checks, []string{"build"}) {
		t.Errorf("checks = %v, want [build]; notes %v", checks, notes)
	}
}
