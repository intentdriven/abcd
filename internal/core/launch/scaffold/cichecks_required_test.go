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
