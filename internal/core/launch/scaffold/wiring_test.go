package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
	"github.com/intentdriven/abcd/internal/gittest"
)

// managedCI is a managed repository's own CI: a pull-request workflow with a
// plainly named job, a display-named job, a matrix job and an expression-named
// job (whose check names only the run knows), a reusable-workflow call, and a
// push-only workflow that gates no merge.
const managedCI = `name: ci
"on":
  pull_request:
  push:
    branches: [trunk]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - run: echo lint
  unit:
    name: Unit tests (linux)  # the context GitHub reports
    runs-on: ubuntu-latest
    steps:
      - run: echo test
  matrixed:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - run: echo
  hostile:
    name: "x ${{ github.event.pull_request.title }}"
    runs-on: ubuntu-latest
    steps:
      - run: echo
  injected:
    name: "ok: }}\n  evil: true"
    runs-on: ubuntu-latest
  reused:
    uses: ./.github/workflows/other.yml
`

const pushOnly = `name: nightly
on:
  schedule:
    - cron: '0 0 * * *'
jobs:
  nightly:
    runs-on: ubuntu-latest
    steps:
      - run: echo
`

const mergeQueue = `name: queue
on: [merge_group]
jobs:
  queue-check:
    runs-on: ubuntu-latest
    steps:
      - run: echo
`

// TestDeriveCIChecksReadsTheRepositorysOwnPullRequestChecks is itd-93 AC1's
// missing half: the check names are the managed repository's own, derived from
// its pull-request and merge-queue workflows, never abcd's; a name GitHub only
// knows at run time is omitted rather than guessed; and nothing outside the
// injection-safe allowlist is ever returned.
func TestDeriveCIChecksReadsTheRepositorysOwnPullRequestChecks(t *testing.T) {
	dir := t.TempDir()
	wf := filepath.Join(dir, ".github", "workflows")
	mustWrite(t, filepath.Join(wf, "ci.yml"), managedCI)
	mustWrite(t, filepath.Join(wf, "nightly.yaml"), pushOnly)
	mustWrite(t, filepath.Join(wf, "queue.yml"), mergeQueue)
	// The scaffold's own workflows are never counted as the merge gate.
	mustWrite(t, filepath.Join(wf, "release.yml"), "on: [pull_request]\njobs:\n  verify:\n    runs-on: x\n")

	got := DeriveCIChecks(dir)
	want := []string{"Unit tests (linux)", "lint", "queue-check"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DeriveCIChecks = %q, want %q", got, want)
	}
	for _, c := range got {
		if !checkNameRe.MatchString(c) {
			t.Errorf("derived check %q escapes the allowlist", c)
		}
	}
	if DeriveCIChecks(t.TempDir()) != nil {
		t.Error("a repository with no workflows derives no checks")
	}
}

// TestScaffoldWiresTheRepositorysOwnCIChecks carries the derivation through the
// write path: the report names the checks, the runbook lists them as the
// contexts to require on the repository's own default branch, and release.yml's
// verify header names them — and no hostile name reaches either file.
func TestScaffoldWiresTheRepositorysOwnCIChecks(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, ".github", "workflows", "ci.yml"), managedCI)
	gitInit(t, dir, "trunk")

	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Unit tests (linux)", "lint"}; !reflect.DeepEqual(rep.CIChecks, want) {
		t.Errorf("report ci_checks = %q, want %q", rep.CIChecks, want)
	}
	book := readFile(t, filepath.Join(dir, filepath.FromSlash(RunbookPath)))
	rel := readFile(t, filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath)))
	for _, want := range []string{"- `Unit tests (linux)`", "- `lint`", "Require each of them on\n`trunk`"} {
		if !strings.Contains(book, want) {
			t.Errorf("runbook lacks %q", want)
		}
	}
	for _, want := range []string{"#   - Unit tests (linux)\n", "#   - lint\n", "Require them on trunk"} {
		if !strings.Contains(rel, want) {
			t.Errorf("release.yml verify header lacks %q", want)
		}
	}
	for _, doc := range []string{book, rel} {
		for _, bad := range []string{"pull_request.title", "evil", "matrixed", "reused"} {
			if strings.Contains(doc, bad) {
				t.Errorf("a name GitHub computes at run time, or a hostile one, reached a scaffolded file: %q", bad)
			}
		}
	}

	// No pull-request CI at all: the files say so, never a borrowed list.
	bare := t.TempDir()
	mustWrite(t, filepath.Join(bare, "go.mod"), "module example.com/y\n\ngo 1.22\n")
	gitInit(t, bare, "main")
	if rep, err = Scaffold(Request{RepoRoot: bare}); err != nil {
		t.Fatal(err)
	}
	if len(rep.CIChecks) != 0 {
		t.Errorf("no CI derives no checks, got %q", rep.CIChecks)
	}
	if !contains(t, filepath.Join(bare, filepath.FromSlash(RunbookPath)), "found no workflow in this repository") ||
		!contains(t, filepath.Join(bare, filepath.FromSlash(ReleaseYMLPath)), "(none found") {
		t.Error("with no CI the runbook and release.yml must say none was found")
	}
}

// TestScaffoldedCharterExemptsShaKeyedReceiptDirs is itd-93 AC5's missing
// half, run for real: the reviews-charter check the scaffold writes holds a
// dated review directory to its shape, and leaves a sha-keyed receipt
// directory alone — the collision abcd-cli once had cannot recur.
func TestScaffoldedCharterExemptsShaKeyedReceiptDirs(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to run the charter check")
	}
	rendered, err := Render(BareSubstitutions("main"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(r *gittest.Repo) (string, bool) {
		script := filepath.Join(t.TempDir(), "check-reviews.sh")
		mustWrite(t, script, string(rendered.CheckReviews))
		cmd := exec.Command("bash", script)
		cmd.Dir = r.Root()
		cmd.Env = r.Env()
		out, err := cmd.CombinedOutput()
		return string(out), err == nil
	}

	r := gittest.NewRepo(t)
	if out, ok := run(r); !ok {
		t.Fatalf("no reviews directory must pass:\n%s", out)
	}
	sha1 := strings.Repeat("a", 40)
	sha256 := strings.Repeat("b", 64)
	r.Write(".abcd/work/reviews/"+sha1+"/docs-currency-reviewer.json", "{}\n")
	r.Write(".abcd/work/reviews/"+sha256+"/other-gate.json", "{}\n")
	r.Write(".abcd/work/reviews/2026-09-01-plan-review/00-summary.md", "# summary\n")
	if out, ok := run(r); !ok {
		t.Fatalf("sha-keyed receipt directories (SHA-1 and SHA-256) must be exempt from RD001:\n%s", out)
	}

	r.Write(".abcd/work/reviews/"+sha1[:12]+"/docs-currency-reviewer.json", "{}\n")
	out, ok := run(r)
	if ok || !strings.Contains(out, "RD001") {
		t.Errorf("an abbreviated sha is not a receipt key and must fail RD001:\n%s", out)
	}
	if err := os.RemoveAll(filepath.Join(r.Root(), ".abcd", "work", "reviews", sha1[:12])); err != nil {
		t.Fatal(err)
	}

	r.Write(".abcd/work/reviews/2026-09-02-no-summary/01-notes.md", "# notes\n")
	if out, ok = run(r); ok || !strings.Contains(out, "missing required 00-summary.md") {
		t.Errorf("a dated review without its summary must fail RD001:\n%s", out)
	}

	// The verify job runs the charter as a numbered deterministic gate, and the
	// runbook lists it at the same number (gate_lockstep, by construction).
	if !strings.Contains(string(rendered.ReleaseYML), "run: bash "+CheckReviewsPath) {
		t.Error("the bare verify job must run the scaffolded charter check")
	}
}

// TestScaffoldedWorkflowsPassTheWorkflowAudit is itd-93 AC1's audit half. The
// intent names zizmor; it is not a dependency of this repository and is not
// installed where these tests run, so the two finding classes the criterion
// names are asserted here with the repository's own tools:
//
//   - duplicate keys: no mapping in either workflow repeats a key (a YAML
//     loader keeps the last one silently, which is how a gate goes missing);
//   - template injection: no `${{ }}` expression appears inside a run script,
//     and every expression anywhere in the file resolves, under the strict
//     actionsexpr evaluator, against contexts a pull request cannot write —
//     an attacker-controlled context such as a pull request's title fails the
//     evaluation instead of passing unnoticed.
//
// Every profile the templates render is audited, with the hostile managed CI
// above feeding the derived check names.
func TestScaffoldedWorkflowsPassTheWorkflowAudit(t *testing.T) {
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	withChecks := BareSubstitutions("trunk")
	withChecks.CIChecks = []string{"Unit tests (linux)", "lint"}
	profiles := map[string]Substitutions{
		"abcd":           AbcdSubstitutions(),
		"bare":           BareSubstitutions("main"),
		"bare+ci-checks": withChecks,
		"bare+semantic":  semantic,
	}
	for name, subs := range profiles {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		for file, doc := range map[string]string{
			"release.yml": string(rendered.ReleaseYML), "auto-release.yml": string(rendered.AutoReleaseYML),
		} {
			where := name + "/" + file
			for _, dup := range duplicateKeys(doc) {
				t.Errorf("%s: duplicate key %s", where, dup)
			}
			for _, inj := range expressionsInRunScripts(doc) {
				t.Errorf("%s: a ${{ }} expression inside a run script (template injection): %s", where, inj)
			}
			for _, expr := range workflowExpressions(doc) {
				if _, err := actionsexpr.EvalValue(expr, trustedContext); err != nil {
					t.Errorf("%s: expression %s reads a context outside the trusted set: %v", where, expr, err)
				}
			}
		}
	}
}

// trustedContext is every expression context the scaffolded workflows may
// read: the run's own identity, the caller's inputs, job and step outputs, and
// the built-in token. Nothing a pull request's author writes (a title, a body,
// a branch name, a commit message) is in it, so an expression reading one
// fails the strict evaluation.
var trustedContext = map[string]any{
	"inputs.tag": "v1.2.3", "inputs.ref": "", "inputs.create_tag": true,
	"github.sha": strings.Repeat("a", 40), "github.ref_name": "v1.2.3", "github.token": "t",
	"github.event_name": "push", "github.repository": "example/fixture",
	"github.event.repository.default_branch": "main", "github.event.repository.private": false,
	"secrets.GITHUB_TOKEN": "t",
	"needs.verify.result":  "success", "needs.tag.result": "success", "needs.release.result": "success",
	"needs.verify.outputs.content_sha":   strings.Repeat("b", 40),
	"needs.detect.outputs.version":       "1.2.3",
	"needs.detect.outputs.need_tag":      "true",
	"needs.detect.outputs.need_release":  "true",
	"needs.detect.outputs.release_ref":   "",
	"steps.detect.outputs.version":       "1.2.3",
	"steps.detect.outputs.need_tag":      "true",
	"steps.detect.outputs.need_release":  "true",
	"steps.detect.outputs.release_ref":   "",
	"steps.receipts.outputs.content_sha": strings.Repeat("b", 40),
	"env.CONTENT_SHA":                    strings.Repeat("b", 40),
	"cancelled()":                        false,
	"success()":                          true,
	"failure()":                          false,
	"always()":                           true,
}

var exprRe = regexp.MustCompile(`\$\{\{.*?\}\}`)

// workflowExpressions returns every `${{ }}` expression outside a comment.
func workflowExpressions(doc string) []string {
	var out []string
	for _, l := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		out = append(out, exprRe.FindAllString(l, -1)...)
	}
	return out
}

// expressionsInRunScripts returns every `${{` that sits inside a run script,
// inline or block — the shape zizmor reports as template injection, because
// the runner splices the value into the script before the shell parses it.
func expressionsInRunScripts(doc string) []string {
	var out []string
	lines := strings.Split(doc, "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
		if !strings.HasPrefix(t, "run:") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(t, "run:"))
		if !strings.HasPrefix(val, "|") && !strings.HasPrefix(val, ">") {
			if strings.Contains(val, "${{") {
				out = append(out, val)
			}
			continue
		}
		indent := len(l) - len(strings.TrimLeft(l, " "))
		for i+1 < len(lines) {
			b := lines[i+1]
			if strings.TrimSpace(b) != "" && len(b)-len(strings.TrimLeft(b, " ")) <= indent {
				break
			}
			i++
			if strings.Contains(b, "${{") {
				out = append(out, strings.TrimSpace(b))
			}
		}
	}
	return out
}

// duplicateKeys walks a workflow's block mappings by indentation and reports
// every key a mapping repeats. Block scalars are skipped whole; a list item
// opens a fresh mapping.
func duplicateKeys(doc string) []string {
	type frame struct {
		indent int
		keys   map[string]bool
	}
	keyRe := regexp.MustCompile(`^("[^"]*"|'[^']*'|[A-Za-z0-9_.\-/]+):(\s|$)`)
	var stack []frame
	var dups []string
	lines := strings.Split(doc, "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		indent := len(l) - len(strings.TrimLeft(l, " "))
		item := false
		if strings.HasPrefix(t, "- ") {
			item = true
			t = strings.TrimSpace(t[2:])
			indent += 2
		}
		for len(stack) > 0 && (stack[len(stack)-1].indent > indent || (item && stack[len(stack)-1].indent == indent)) {
			stack = stack[:len(stack)-1]
		}
		m := keyRe.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		if len(stack) == 0 || stack[len(stack)-1].indent < indent {
			stack = append(stack, frame{indent: indent, keys: map[string]bool{}})
		}
		top := stack[len(stack)-1]
		key := strings.Trim(m[1], `"'`)
		if top.keys[key] {
			dups = append(dups, "line "+itoa(i+1)+": "+key)
		}
		top.keys[key] = true
		// A block scalar's body is text, not keys: skip it.
		rest := strings.TrimSpace(t[len(m[0]):])
		if strings.HasPrefix(rest, "|") || strings.HasPrefix(rest, ">") {
			for i+1 < len(lines) {
				b := lines[i+1]
				if strings.TrimSpace(b) != "" && len(b)-len(strings.TrimLeft(b, " ")) <= indent {
					break
				}
				i++
			}
		}
	}
	return dups
}

// TestDuplicateKeysDetectsARepeatedKey keeps the audit above honest: the
// checker must find a duplicate it is shown, at any depth, or its silence on
// the real workflows proves nothing.
func TestDuplicateKeysDetectsARepeatedKey(t *testing.T) {
	doc := "jobs:\n  verify:\n    steps:\n      - name: a\n        run: x\n        run: y\n  verify:\n    runs-on: z\n"
	got := duplicateKeys(doc)
	if len(got) != 2 {
		t.Errorf("want the repeated step key and the repeated job id, got %v", got)
	}
	if n := len(expressionsInRunScripts("      - run: |\n          echo ${{ github.event.issue.title }}\n")); n != 1 {
		t.Errorf("an expression in a block run script must be reported, got %d", n)
	}
	if _, err := actionsexpr.EvalValue("${{ github.event.pull_request.title }}", trustedContext); err == nil {
		t.Error("an attacker-controlled context must fail the strict evaluation")
	}

}
