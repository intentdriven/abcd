package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// matrixOf is a job's strategy with one expandable matrix key, to end a fixture
// job with.
const matrixOf = `
    strategy:
      matrix:
        os: [a, b]
`

// both names the two contexts a fixture job with matrixOf reports.
func both(job string) []string { return []string{job + " (a)", job + " (b)"} }

// wantChecks fails unless the required checks are exactly want, and every job in
// noted is named in the notes.
func wantChecks(t *testing.T, repo string, want []string, noted ...string) {
	t.Helper()
	checks, notes, err := readRequired(repo, false)
	if err != nil {
		t.Fatalf("the read was refused: %v", err)
	}
	slices.Sort(want)
	if !reflect.DeepEqual(checks, want) {
		t.Errorf("checks = %q\nwant %q\nnotes %q", checks, want, notes)
	}
	joined := strings.Join(notes, "\n")
	for _, job := range noted {
		if !strings.Contains(joined, "job "+job+" ") {
			t.Errorf("the notes do not name %s:\n%s", job, joined)
		}
	}
}

// TestRequiredChecksFollowsImplicitSuccessThroughEveryAncestor: with no status
// function of its own, a job carries an implicit success(), which reads every
// ancestor, not only the jobs it names. A job with `if: always()` runs and succeeds
// behind a skipped job, and a job needing it is skipped all the same
// (actions/runner#491). The always() job itself runs, as does a job that says
// always() for itself.
func TestRequiredChecksFollowsImplicitSuccessThroughEveryAncestor(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  plain:
    runs-on: x
  gated:
    if: github.event_name != 'pull_request'
    runs-on: x
  stand-in:
    needs: gated
    if: always()
    runs-on: x
  clean-always:
    needs: plain
    if: always()
    runs-on: x
  after:
    needs: stand-in` + matrixOf + `
  after-two-hops:
    needs: [plain, after-plain]` + matrixOf + `
  after-plain:
    needs: stand-in
    runs-on: x
  says-success:
    needs: stand-in
    if: success()` + matrixOf + `
  says-always:
    needs: stand-in
    if: always()` + matrixOf + `
  says-not-cancelled:
    needs: gated
    if: ${{ !cancelled() }}` + matrixOf + `
  after-clean-always:
    needs: clean-always` + matrixOf + `
`})
	want := []string{"plain", "gated", "stand-in", "clean-always", "after-plain"}
	for _, job := range []string{"says-always", "says-not-cancelled", "after-clean-always"} {
		want = append(want, both(job)...)
	}
	wantChecks(t, repo, want, "after", "after-two-hops", "says-success")
}

// TestRequiredChecksReadsAConditionWhole: an `if:` is its whole YAML value, however
// it is written across lines, so text after its first line is never lost, and a
// block scalar that holds only always() is always().
func TestRequiredChecksReadsAConditionWhole(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  plain-lines:
    if: always()
      && github.event_name != 'pull_request'` + matrixOf + `
  folded:
    if: >-
      always()
      && github.event_name != 'pull_request'` + matrixOf + `
  literal-always:
    if: |
      always()` + matrixOf + `
  "quoted-key":
    "if": github.event_name != 'pull_request'
    "strategy":
      "matrix":
        os: [a, b]
`})
	wantChecks(t, repo, both("literal-always"), "plain-lines", "folded", "quoted-key")
}

// TestRequiredChecksReadsEveryFormOfNeeds: an empty list, a trailing comma, a key
// with only a comment after it and a null all name exactly the jobs they list.
func TestRequiredChecksReadsEveryFormOfNeeds(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  plain:
    runs-on: x
  gated:
    if: github.event_name != 'pull_request'
    runs-on: x
  empty-list:
    needs: []` + matrixOf + `
  trailing-comma:
    needs: [plain, ]` + matrixOf + `
  comment-then-list:
    needs: # the gates
      - plain` + matrixOf + `
  null-needs:
    needs:` + matrixOf + `
  quoted-need:
    needs: ["plain"]` + matrixOf + `
  gated-in-a-list:
    needs:
    - plain
    - "gated"` + matrixOf + `
`})
	want := []string{"plain", "gated"}
	for _, job := range []string{"empty-list", "trailing-comma", "comment-then-list", "null-needs", "quoted-need"} {
		want = append(want, both(job)...)
	}
	wantChecks(t, repo, want, "gated-in-a-list")
}

// TestRequiredChecksNamesWhatItCannotTell: a condition that adds nothing to the
// implicit one does not count against a matrix job. One this reader cannot judge,
// such as a guard on the repository's name, is left out, and the note says abcd
// could not tell and what the person can do, since every run may well report it.
func TestRequiredChecksNamesWhatItCannotTell(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  gated:
    if: github.event_name != 'pull_request'
    runs-on: x
  says-success:
    if: success()` + matrixOf + `
  says-success-expr:
    if: ${{ success() }}` + matrixOf + `
  says-true:
    if: true` + matrixOf + `
  says-true-expr:
    if: ${{ true }}` + matrixOf + `
  says-success-behind-gated:
    needs: gated
    if: success()` + matrixOf + `
  fork-guard:
    if: github.repository == 'owner/repo'` + matrixOf + `
`})
	want := []string{"gated"}
	for _, job := range []string{"says-success", "says-success-expr", "says-true", "says-true-expr"} {
		want = append(want, both(job)...)
	}
	wantChecks(t, repo, want, "says-success-behind-gated", "fork-guard")
	_, notes, _ := readRequired(repo, false)
	joined := strings.Join(notes, "\n")
	for _, say := range []string{"cannot tell", "github.required_checks"} {
		if !strings.Contains(joined, say) {
			t.Errorf("the notes do not say %q:\n%s", say, joined)
		}
	}
}

// TestRequiredChecksReadsEveryFormOfAJob: a job whose id is quoted, whose value
// carries an anchor or is an alias, or is written inline is read as a job, and so
// is every job after it.
func TestRequiredChecksReadsEveryFormOfAJob(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  first:
    runs-on: x
  "quoted":
    runs-on: x
  anchored: &shared
    runs-on: x
  aliased: *shared
  inline: {runs-on: x}
  spaced :
    runs-on: x
  last:
    runs-on: x
`})
	wantChecks(t, repo, []string{"first", "quoted", "anchored", "aliased", "inline", "spaced", "last"})
}

// TestRequiredChecksReadsAMatrixOnlyUnderItsStrategy: the word matrix anywhere else
// in a job, in a script, under a step's `with:`, in the job's `env:`, makes no
// matrix, and a strategy with no matrix makes none either.
func TestRequiredChecksReadsAMatrixOnlyUnderItsStrategy(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  in-a-script:
    runs-on: x
    steps:
      - run: |
          matrix:
            os: [fake]
  under-with:
    runs-on: x
    steps:
      - uses: a/b@v1
        with:
          matrix:
            os: [fake]
  in-env:
    runs-on: x
    env:
      matrix: fake
  gated-with-the-word:
    if: github.event_name != 'pull_request'
    runs-on: x
    steps:
      - run: |
          matrix:
            os: [fake]
  no-matrix:
    strategy:
      fail-fast: false
    runs-on: x
`})
	wantChecks(t, repo, []string{"in-a-script", "under-with", "in-env", "gated-with-the-word", "no-matrix"})
}

// TestRequiredChecksExpandsAMatrixHoweverItIsWritten: flow and block lists are the
// same value. A value GitHub would print differently from its text (an unquoted
// 1.10 reports as 1.1) and a computed matrix report names only the run knows.
func TestRequiredChecksExpandsAMatrixHoweverItIsWritten(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{"ci.yml": `on: pull_request
jobs:
  block:
    strategy:
      matrix:
        os:
          - a
          - b
  inline:
    strategy: {matrix: {os: [a, b]}}
  numbers:
    strategy:
      matrix:
        shard: [1, 2]
  float:
    strategy:
      matrix:
        go: [1.10]
  computed:
    strategy:
      matrix: ${{ fromJSON(needs.x.outputs.y) }}
  computed-strategy:
    strategy: ${{ fromJSON(needs.x.outputs.y) }}
`})
	want := append(append(both("block"), both("inline")...), "numbers (1)", "numbers (2)")
	wantChecks(t, repo, want, "float", "computed", "computed-strategy")
}

// TestRequiredChecksProposesNothingFromAWorkflowItCannotRead: a shorter list reads
// as an answer, so a workflow the read cannot take whole refuses the read: no check
// is proposed from any workflow, and the error names the file. A workflow that does
// not run on a pull request is judged too when it does not parse, since the trigger
// is part of what could not be read.
func TestRequiredChecksProposesNothingFromAWorkflowItCannotRead(t *testing.T) {
	const good = "on: pull_request\njobs:\n  build:\n    runs-on: x\n"
	for name, c := range map[string]struct {
		body string
		set  func(t *testing.T, dir string)
	}{
		"syntax":          {body: "on: pull_request\njobs:\n  a:\n\t\truns-on: x\n"},
		"unknown alias":   {body: "on: pull_request\njobs:\n  a: *nowhere\n"},
		"not on a pr":     {body: "on: push\njobs: [\n"},
		"duplicate job":   {body: "on: pull_request\njobs:\n  a:\n    runs-on: x\n  a:\n    runs-on: x\n"},
		"job not a map":   {body: "on: pull_request\njobs:\n  a: 3\n"},
		"jobs not a map":  {body: "on: pull_request\njobs: [a]\n"},
		"no jobs":         {body: "on: pull_request\n"},
		"need of nothing": {body: "on: pull_request\njobs:\n  a:\n    needs: nowhere\n    runs-on: x\n"},
		"need not a name": {body: "on: pull_request\njobs:\n  a:\n    needs: {b: c}\n    runs-on: x\n"},
		"cycle":           {body: "on: pull_request\njobs:\n  a:\n    needs: b\n    runs-on: x\n  b:\n    needs: a\n    runs-on: x\n"},
		"odd job id":      {body: "on: pull_request\njobs:\n  \"a: b\":\n    runs-on: x\n"},
		"too large":       {body: "on: pull_request\njobs:\n  a:\n    runs-on: x\n" + strings.Repeat("# pad\n", maxWorkflowBytes/6+1)},
		"refused file": {set: func(t *testing.T, dir string) {
			if err := os.Symlink(filepath.Join(dir, "ci.yml"), filepath.Join(dir, "broken.yml")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"ci.yml": good}
			if c.set == nil {
				files["broken.yml"] = c.body
			}
			repo := writeWorkflows(t, files)
			if c.set != nil {
				c.set(t, filepath.Join(repo, ".github", "workflows"))
			}
			checks, _, err := readRequired(repo, false)
			if err == nil {
				t.Fatalf("the read went ahead, with checks %q", checks)
			}
			if len(checks) != 0 {
				t.Errorf("checks = %q; a refused read proposes none", checks)
			}
			if !strings.Contains(err.Error(), "broken.yml") {
				t.Errorf("the error does not name the file: %v", err)
			}
		})
	}
	t.Run("more files than it reads", func(t *testing.T) {
		files := map[string]string{}
		for i := 0; i <= maxWorkflowFiles; i++ {
			files[fmt.Sprintf("w%03d.yml", i)] = good
		}
		if checks, _, err := readRequired(writeWorkflows(t, files), false); err == nil {
			t.Errorf("the read went ahead over %d files, with checks %q", len(files), checks)
		}
	})
}

// readRequired is RequiredChecks for the fixtures above.
func readRequired(repo string, queue bool) ([]string, []string, error) {
	return RequiredChecks(repo, "main", queue)
}

// requiredChecksOf is RequiredChecks for a fixture every file of which it reads.
func requiredChecksOf(t *testing.T, repo string, queue bool) (checks, notes []string) {
	t.Helper()
	checks, notes, err := readRequired(repo, queue)
	if err != nil {
		t.Fatalf("the read was refused: %v", err)
	}
	return checks, notes
}
