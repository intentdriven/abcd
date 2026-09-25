package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// TestNoWorkflowMakesAMergeQueueRunCancellable holds every workflow that
// triggers on `merge_group` to one property: a merge-queue run of it can never
// be cancelled by concurrency (iss-2609020716560392).
//
// The failure this closes is silent and it removed one pull request from the
// queue four times in a night. A merge-queue entry runs on a temporary ref of
// the shape `refs/heads/gh-readonly-queue/<base>/pr-<n>-<base sha>`, which is
// never the default branch — so ci.yml's `cancel-in-progress: ${{ github.ref !=
// format('refs/heads/{0}', github.event.repository.default_branch) }}`
// evaluated TRUE there, and a merge-group run was cancellable. GitHub's queue
// reads a cancelled required check as a failure: it drops the entry and disarms
// auto-merge, with no comment on the pull request. The cancel expression was
// written to make a new push to a PR supersede the run in flight for it, and it
// still does; the queue refs were simply not in view when it was written.
//
// The check is SEMANTIC, not textual. It evaluates each concurrency block's own
// expressions in a simulated merge_group context and asks what GitHub would do,
// so a future workflow that grows a `concurrency:` block fails this test by
// being cancellable rather than by failing to match a pattern, and a correct
// block written in a spelling nobody anticipated passes. Two shapes are safe
// and both are in the tree today:
//
//   - The cancel expression is false for a merge_group event — ci.yml, which
//     guards on `github.event_name`.
//   - The group key is unique per run, so no in-flight run can share it and
//     there is nothing to cancel — attribution.yml and external-review.yml,
//     whose `${{ github.event.pull_request.number || github.run_id }}` falls
//     through to the run id on an event that carries no pull request.
//
// It fails CLOSED. An expression the evaluator cannot resolve — an unknown
// context path, an unsupported function, a block scalar it cannot fold — is
// reported as a violation, not skipped: the whole point is that nobody notices
// this defect in production, so an unprovable block must stop the push rather
// than pass quietly. Extending the evaluator is the way past, and the failure
// names what it could not read.
//
// The roster is DERIVED — every file under .github/workflows — so a new
// workflow is covered by existing, and the two floors below are the backstop
// that makes a broken parser fail rather than pass an empty sweep.
func TestNoWorkflowMakesAMergeQueueRunCancellable(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const dir = ".github/workflows"

	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext == ".yml" || ext == ".yaml" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s holds no workflow files; the sweep would pass by finding nothing", dir)
	}

	queueWorkflows, parsedBlocks := 0, 0
	for _, name := range names {
		rel := dir + "/" + name
		workflow := readRepoFile(t, root, rel)

		blocks, err := concurrencyBlocks(workflow)
		if err != nil {
			t.Errorf("%s: %v\n\nA concurrency block this test cannot read is a block whose "+
				"merge-queue behaviour nobody is checking.", rel, err)
			continue
		}
		parsedBlocks += len(blocks)

		if !triggersOnMergeGroup(workflow) {
			// No merge_group trigger, so no merge-queue run of it exists to
			// cancel. Its blocks still counted toward the parser floor above.
			continue
		}
		queueWorkflows++

		for _, b := range blocks {
			t.Run(fmt.Sprintf("%s:%d", name, b.line), func(t *testing.T) {
				if b.cancel == "" {
					// Absent is `false`: GitHub never cancels an in-progress run
					// for a group whose block does not ask it to.
					return
				}
				cancel, err := actionsexpr.EvalValue(b.cancel, mergeGroupContext)
				if err != nil {
					t.Errorf("%s line %d: cannot evaluate cancel-in-progress %q for a merge_group "+
						"event: %v\n\nThis gate fails closed — an expression it cannot resolve is "+
						"reported rather than assumed safe. Simplify the expression, or teach "+
						"internal/actionsexpr the shape.", rel, b.line, b.cancel, err)
					return
				}
				if !actionsexpr.Truthy(cancel) {
					return
				}

				// Cancellation is armed. It is still harmless if the group key
				// is unique per run, because then no other run is ever in the
				// group to be cancelled.
				group, err := actionsexpr.EvalValue(b.group, mergeGroupContext)
				if err != nil {
					t.Errorf("%s line %d: cancel-in-progress evaluates true for a merge_group event, "+
						"and the group key %q cannot be evaluated to prove it unique per run: %v",
						rel, b.line, b.group, err)
					return
				}
				if strings.Contains(actionsexpr.Stringify(group), runIDSentinel) {
					return
				}
				t.Errorf("%s line %d makes a merge-queue run cancellable.\n\n"+
					"  group:               %s\n"+
					"    -> %q\n"+
					"  cancel-in-progress:  %s\n"+
					"    -> true, on a merge_group event with github.ref = %q\n\n"+
					"GitHub reads a cancelled required check as a FAILED one: it removes the entry "+
					"from the merge queue and disarms auto-merge, silently. Either guard the cancel "+
					"expression on `github.event_name != 'merge_group'`, or key the group on "+
					"github.run_id so a queue run shares its group with nothing.",
					rel, b.line, b.group, actionsexpr.Stringify(group), b.cancel, mergeQueueRef)
			})
		}
	}

	// Two fail-closed floors. Both parsers are hand-written against a YAML
	// shape, and a shape change that made either return nothing would turn this
	// whole sweep into a pass over an empty set.
	if queueWorkflows == 0 {
		t.Errorf("no workflow under %s parsed as triggering on `merge_group`, and the merge queue "+
			"requires checks from several; the trigger parser has lost its shape", dir)
	}
	if parsedBlocks == 0 {
		t.Errorf("no `concurrency:` block parsed from any workflow under %s; the block parser has "+
			"lost its shape, and this gate is asserting nothing", dir)
	}
}

// The context a merge-queue run evaluates its expressions in. The ref shape is
// GitHub's own — `refs/heads/gh-readonly-queue/<base branch>/pr-<number>-<base
// sha>` — and it is the whole defect: it is never the default branch, so any
// expression that only excuses the default branch arms cancellation here.
//
// runIDSentinel is deliberately a value nothing else in the context contains,
// so "the evaluated group key contains the run id" is a sound test for "this
// group is unique per run".
const (
	mergeQueueRef = "refs/heads/gh-readonly-queue/main/pr-620-1c9a4f2d6b8e0a7c5d3f1b9e2a8c6d4f0e7b5a3c"
	runIDSentinel = "17253907741"
)

var mergeGroupContext = map[string]any{
	"github.event_name":                      "merge_group",
	"github.workflow":                        "ci",
	"github.ref":                             mergeQueueRef,
	"github.ref_name":                        strings.TrimPrefix(mergeQueueRef, "refs/heads/"),
	"github.ref_type":                        "branch",
	"github.run_id":                          runIDSentinel,
	"github.run_attempt":                     "1",
	"github.sha":                             "1c9a4f2d6b8e0a7c5d3f1b9e2a8c6d4f0e7b5a3c",
	"github.repository":                      "intentdriven/abcd",
	"github.actor":                           "github-merge-queue[bot]",
	"github.base_ref":                        "",
	"github.head_ref":                        "",
	"github.event.repository.default_branch": "main",
	"github.event.merge_group.head_ref":      mergeQueueRef,
	"github.event.merge_group.base_ref":      "refs/heads/main",
	"github.event.merge_group.head_sha":      "1c9a4f2d6b8e0a7c5d3f1b9e2a8c6d4f0e7b5a3c",
	// The base the queue entry was formed against — the commit `head_sha` sits
	// on top of. It is the one field on this event that names a base to diff
	// against, which is what the range-scoped gates need (rangegatebase_test.go).
	"github.event.merge_group.base_sha": mergeGroupBaseSHA,
	// A merge_group payload carries no pull request, no inputs and no pushed
	// branch. GitHub resolves a missing context property to null, and these are
	// spelled out rather than left to the unknown-path error so that reading a
	// PR field on this event is understood, not merely unresolvable.
	"github.event.pull_request.number":   nil,
	"github.event.pull_request.head.ref": nil,
	"github.event.pull_request.base.sha": nil,
	"github.event.before":                nil,
	"github.event.number":                nil,
	"inputs.tag":                         nil,
	"inputs.mode":                        nil,
}

// concurrency is one parsed `concurrency:` block — workflow-level or job-level.
// group and cancel hold the RAW value text, evaluated later; "" means the key
// is absent from the block.
type concurrency struct {
	line   int // 1-based, for the failure message
	group  string
	cancel string
}

var (
	concurrencyKeyRe = regexp.MustCompile(`^(\s*)concurrency:\s*(.*)$`)
	groupKeyRe       = regexp.MustCompile(`^\s*group:\s*(.*)$`)
	cancelKeyRe      = regexp.MustCompile(`^\s*cancel-in-progress:\s*(.*)$`)
)

// concurrencyBlocks returns every concurrency block in a workflow, at any
// indent — the workflow-level one and any a job carries. Hand-parsed for the
// reason the workflow contracts beside it are: this repository carries no YAML
// parser for its own workflows and adds no dependency for one key.
func concurrencyBlocks(workflow string) ([]concurrency, error) {
	lines := strings.Split(workflow, "\n")
	var out []concurrency
	for i := 0; i < len(lines); i++ {
		m := concurrencyKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent := len(m[1])
		if inline := stripComment(m[2]); inline != "" {
			// `concurrency: <group>` shorthand. It names a group and cannot
			// arm cancellation, which defaults to false.
			out = append(out, concurrency{line: i + 1, group: inline})
			continue
		}
		block := concurrency{line: i + 1}
		for j := i + 1; j < len(lines); j++ {
			line := lines[j]
			if strings.TrimSpace(line) == "" {
				continue
			}
			if lineIndent(line) <= indent {
				break
			}
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			if g := groupKeyRe.FindStringSubmatch(line); g != nil {
				v, err := scalarValue(g[1], lines, j)
				if err != nil {
					return nil, fmt.Errorf("line %d: group: %w", j+1, err)
				}
				block.group = v
				continue
			}
			if c := cancelKeyRe.FindStringSubmatch(line); c != nil {
				v, err := scalarValue(c[1], lines, j)
				if err != nil {
					return nil, fmt.Errorf("line %d: cancel-in-progress: %w", j+1, err)
				}
				block.cancel = v
			}
		}
		out = append(out, block)
	}
	return out, nil
}

// scalarValue resolves one key's value: the rest of its own line, or — when
// that is a block-scalar marker — the folded body beneath it. A `>`-folded
// expression is a legal way to write a long cancel expression, so it must
// resolve rather than fail closed on shape alone.
func scalarValue(rest string, lines []string, at int) (string, error) {
	rest = strings.TrimSpace(rest)
	if rest != "" && !strings.HasPrefix(rest, ">") && !strings.HasPrefix(rest, "|") {
		return stripComment(rest), nil
	}
	if rest == "" {
		return "", fmt.Errorf("no value on the key line and no block scalar")
	}
	indent := lineIndent(lines[at])
	var body []string
	for j := at + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		if lineIndent(lines[j]) <= indent {
			break
		}
		body = append(body, strings.TrimSpace(lines[j]))
	}
	if len(body) == 0 {
		return "", fmt.Errorf("block scalar %q has no body", rest)
	}
	return strings.Join(body, " "), nil
}

func lineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

// stripComment removes a trailing YAML comment. A `#` inside an expression or a
// quoted string is not one, and the only `#` this repository's workflow values
// carry are comments, so the conservative rule is: a `#` outside `${{ }}` and
// outside quotes ends the value.
func stripComment(v string) string {
	inExpr, inQuote := false, byte(0)
	for i := 0; i < len(v); i++ {
		switch {
		case inQuote != 0:
			if v[i] == inQuote {
				inQuote = 0
			}
		case v[i] == '\'' || v[i] == '"':
			inQuote = v[i]
		case strings.HasPrefix(v[i:], "${{"):
			inExpr = true
		case strings.HasPrefix(v[i:], "}}"):
			inExpr = false
		case v[i] == '#' && !inExpr && (i == 0 || v[i-1] == ' '):
			return strings.TrimSpace(v[:i])
		}
	}
	return strings.TrimSpace(v)
}

var (
	onKeyRe         = regexp.MustCompile(`^on:\s*(.*)$`)
	topLevelKeyRe   = regexp.MustCompile(`^[A-Za-z"']`)
	mergeGroupKeyRe = regexp.MustCompile(`^\s*(-\s*)?merge_group\s*:?\s*$`)
)

// triggersOnMergeGroup reports whether a workflow runs on merge-queue entries.
// Only those can have a merge-queue run to cancel; the rest are out of scope,
// and a workflow that later gains the trigger is swept in by gaining it.
func triggersOnMergeGroup(workflow string) bool {
	lines := strings.Split(workflow, "\n")
	for i, line := range lines {
		m := onKeyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if inline := stripComment(m[1]); inline != "" {
			// `on: merge_group` or `on: [pull_request, merge_group]`.
			for _, f := range strings.FieldsFunc(strings.Trim(inline, "[]"), func(r rune) bool {
				return r == ',' || r == ' '
			}) {
				if f == "merge_group" {
					return true
				}
			}
			return false
		}
		for j := i + 1; j < len(lines); j++ {
			if topLevelKeyRe.MatchString(lines[j]) {
				break
			}
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "#") {
				continue
			}
			if mergeGroupKeyRe.MatchString(lines[j]) {
				return true
			}
		}
		return false
	}
	return false
}

// TestCIStillSupersedesAnInFlightPullRequestRun holds the OTHER half of ci.yml's
// concurrency contract, so that exempting the merge queue cannot be "fixed"
// later by turning cancellation off altogether.
//
// The expression exists to make a new push to a pull request supersede the run
// still in flight for its previous revision — the matrix is expensive, and a run
// nobody will merge is pure waste. Two properties carry that, and the sibling
// test above cannot see either, because it only ever asks about a merge_group
// event:
//
//   - On a pull_request event, cancellation is armed and the group is keyed on
//     the PR's ref. Both are needed: a group keyed per RUN would arm a cancel
//     that can never find anything to cancel, which is the silent way to lose
//     this.
//   - On a push to the default branch, cancellation is off, so a merge is never
//     cancelled mid-run by the merge behind it.
func TestCIStillSupersedesAnInFlightPullRequestRun(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"

	blocks, err := concurrencyBlocks(readRepoFile(t, root, rel))
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	if len(blocks) != 1 {
		t.Fatalf("%s holds %d concurrency blocks, want the single workflow-level one; "+
			"this test asserts about that one and would otherwise be reading a job's", rel, len(blocks))
	}
	b := blocks[0]

	const prRef = "refs/pull/620/merge"
	pullRequest := contextFor(map[string]any{
		"github.event_name":                "pull_request",
		"github.ref":                       prRef,
		"github.ref_name":                  "620/merge",
		"github.head_ref":                  "fix/merge-queue-concurrency",
		"github.base_ref":                  "main",
		"github.event.pull_request.number": float64(620),
	})
	defaultBranchPush := contextFor(map[string]any{
		"github.event_name": "push",
		"github.ref":        "refs/heads/main",
		"github.ref_name":   "main",
	})

	cancel, err := actionsexpr.EvalValue(b.cancel, pullRequest)
	if err != nil {
		t.Fatalf("evaluating cancel-in-progress for a pull_request event: %v", err)
	}
	if !actionsexpr.Truthy(cancel) {
		t.Errorf("%s does not cancel an in-flight run on a pull_request event (cancel-in-progress "+
			"%s -> %v).\n\nSuperseding a PR's previous revision is what this expression is for; a "+
			"blanket false exempts the merge queue by giving up the property instead.",
			rel, b.cancel, cancel)
	}
	group, err := actionsexpr.EvalValue(b.group, pullRequest)
	if err != nil {
		t.Fatalf("evaluating group for a pull_request event: %v", err)
	}
	if !strings.Contains(actionsexpr.Stringify(group), prRef) {
		t.Errorf("%s keys its concurrency group on %q for a pull_request event, which does not "+
			"contain the PR ref %q; two runs of the same pull request must land in the same group "+
			"or neither can supersede the other", rel, actionsexpr.Stringify(group), prRef)
	}
	if strings.Contains(actionsexpr.Stringify(group), runIDSentinel) {
		t.Errorf("%s keys its concurrency group on the run id for a pull_request event (%q); every "+
			"run then has the group to itself, and the armed cancel supersedes nothing",
			rel, actionsexpr.Stringify(group))
	}

	cancel, err = actionsexpr.EvalValue(b.cancel, defaultBranchPush)
	if err != nil {
		t.Fatalf("evaluating cancel-in-progress for a default-branch push: %v", err)
	}
	if actionsexpr.Truthy(cancel) {
		t.Errorf("%s cancels an in-flight run on a push to the default branch (cancel-in-progress "+
			"%s -> %v); the merge behind a merge would cancel the checks on the merged tip",
			rel, b.cancel, cancel)
	}
}

// contextFor derives an event context from the merge_group one, which is the
// base because it is the event this file exists for. An override states what a
// different event carries differently — nothing else about the run changes.
func contextFor(overrides map[string]any) map[string]any {
	ctx := make(map[string]any, len(mergeGroupContext))
	for k, v := range mergeGroupContext {
		ctx[k] = v
	}
	for k, v := range overrides {
		ctx[k] = v
	}
	return ctx
}
