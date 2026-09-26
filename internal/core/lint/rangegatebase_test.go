package lint_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// The shas each event carries in the simulated contexts below. They are
// distinct values rather than one placeholder so that "which rung of the
// fallback chain answered" is a fact the assertions can read, not a guess:
// a chain that resolved the wrong field on the right event still fails.
//
// mergeGroupBaseSHA lives in mergeGroupContext (mergequeueconcurrency_test.go),
// which is the canonical merge_group payload for both files.
const (
	mergeGroupBaseSHA  = "9f1b7c4e2a6d8035b1e4c7a92f6d0b3e8c5a1d7f"
	pullRequestBaseSHA = "3e5d9a1c7f2b4086d3a5c9e1b7f4d2a6c8e0b5d9"
	pushBeforeSHA      = "7c2a4e6b8d0f1937a5c3e9b1d7f5a2c4e6b8d0f2"
	// What GitHub puts in `before` when a push has no predecessor to name — a
	// branch created by the push, or a force-push whose old tip it declines to
	// cite. It is a well-formed sha that resolves to nothing, which is why the
	// steps must refuse it rather than hand it to git.
	absentSHA = "0000000000000000000000000000000000000000"
)

// TestEveryRangeScopedGateResolvesABaseOnEveryEvent holds ci.yml's range-scoped
// gates to one property: on every event the workflow triggers on, the step's
// BASE_SHA expression resolves to the base that event actually carries
// (iss-2609091258115663).
//
// The defect it closes was silent and it disarmed the run that matters most.
// Three steps derived their base as `github.event.pull_request.base.sha ||
// github.event.before`. A merge_group payload carries NEITHER — it names its
// base as `merge_group.base_sha` — so on a merge-queue entry the expression
// resolved to the empty string and each step took its own no-base branch: the
// issue-resolution range check and the decisions-append check announced "no
// usable base ref for this event" and exited 0, and record-lint downgraded to
// its unarmed run. That inverts the protection. The queue entry is the run that
// actually gates the merge, and the ONLY run that sees the final base after a
// competing pull request merged ahead of it — which is exactly the race a range
// check exists to catch. Every earlier run reported green against a base that
// no longer existed by the time the merge happened.
//
// The check is SEMANTIC, like its sibling in mergequeueconcurrency_test.go: it
// evaluates each step's own expression in a simulated event context and asks
// what value GitHub would hand the shell, so a chain written in a spelling
// nobody anticipated passes and a chain that merely looks right fails.
//
// The SET of steps is DERIVED from the file, so a fourth range gate added later
// is covered by existing. A step is in scope when it declares a `BASE_SHA:`
// env; it is out of scope only when it EARNS the exemption by judging the event
// itself — reading `github.event_name` into its environment and branching on
// it in its script, which is what the changed-paths classifier does (it stands
// down on any non-pull_request event before it ever looks at BASE_SHA, and its
// stand-down runs the full matrix rather than skipping a check). Delegating the
// event question entirely to the expression and then not covering an event is
// the defect; a step that answers the question itself has not delegated it.
func TestEveryRangeScopedGateResolvesABaseOnEveryEvent(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)

	steps, err := workflowSteps(workflow)
	if err != nil {
		t.Fatalf("%s: %v\n\nA step this test cannot read is a step whose base resolution "+
			"nobody is checking.", rel, err)
	}
	if len(steps) == 0 {
		t.Fatalf("%s parsed into no steps; the step parser has lost its shape and this gate "+
			"is asserting nothing", rel)
	}

	// One context per event ci.yml triggers on. Each names only what that event
	// carries; every other base field is null, the way GitHub resolves a
	// property its payload does not hold. The whole `github.event.merge_group.*`
	// subtree is cleared for the two non-queue events rather than only the field
	// in today's chain: a context that leaves a sibling field populated would
	// let a chain reaching for the WRONG merge-group field pass on a push, which
	// is the shape of the defect this test exists for.
	pullRequest := withoutSubtree(contextFor(map[string]any{
		"github.event_name":                  "pull_request",
		"github.ref":                         "refs/pull/620/merge",
		"github.ref_name":                    "620/merge",
		"github.head_ref":                    "fix/merge-group-base",
		"github.base_ref":                    "main",
		"github.event.pull_request.number":   float64(620),
		"github.event.pull_request.base.sha": pullRequestBaseSHA,
		"github.event.before":                nil,
	}), "github.event.merge_group.")
	defaultBranchPush := withoutSubtree(contextFor(map[string]any{
		"github.event_name":                "push",
		"github.ref":                       "refs/heads/main",
		"github.ref_name":                  "main",
		"github.event.pull_request.number": nil,
		"github.event.before":              pushBeforeSHA,
	}), "github.event.merge_group.")
	// A force-push, where `before` is the one rung that can be PRESENT and
	// unusable. GitHub either cites the rewritten tip — a sha no longer
	// reachable from anything — or the all-zeroes placeholder. The zeroes case
	// is the one an expression can see, and the assertion is that the chain
	// still yields it rather than papering over it with another event's field:
	// refusing a bad base is the step's job, and it can only do it if the bad
	// base reaches it.
	forcePush := withoutSubtree(contextFor(map[string]any{
		"github.event_name":                "push",
		"github.ref":                       "refs/heads/main",
		"github.ref_name":                  "main",
		"github.event.pull_request.number": nil,
		"github.event.before":              absentSHA,
	}), "github.event.merge_group.")

	events := []struct {
		name string
		ctx  map[string]any
		want string
		why  string
	}{
		{"merge_group", mergeGroupContext, mergeGroupBaseSHA,
			"the queue entry's own base — the run that actually gates the merge, and the only " +
				"one that sees the base a competing pull request left behind"},
		{"pull_request", pullRequest, pullRequestBaseSHA,
			"the pull request's base, which these gates have always resolved and must keep resolving"},
		{"push", defaultBranchPush, pushBeforeSHA,
			"the previous tip of the default branch, so the range is the merge that just landed"},
		{"push (force-push, no predecessor)", forcePush, absentSHA,
			"the all-zeroes placeholder, handed through UNCHANGED so the step's own guard is what " +
				"refuses it; a chain that swallowed it would run the gate against a base that does not exist"},
	}

	declaring, inScope, exempt := 0, 0, 0
	for _, step := range steps {
		raw, ok := step.env["BASE_SHA"]
		if !ok {
			continue
		}
		declaring++
		if judgesTheEventItself(step) {
			exempt++
			continue
		}
		inScope++

		t.Run(fmt.Sprintf("%s:%d", strings.NewReplacer("/", "_", " ", "_").Replace(step.name), step.line), func(t *testing.T) {
			for _, ev := range events {
				got, err := actionsexpr.EvalValue(raw, ev.ctx)
				if err != nil {
					t.Errorf("%s line %d, step %q: cannot evaluate BASE_SHA %s for a %s event: %v\n\n"+
						"This gate fails closed — an expression it cannot resolve is reported rather "+
						"than assumed correct. Simplify the expression, or teach internal/actionsexpr "+
						"the shape.", rel, step.envLine["BASE_SHA"], step.name, raw, ev.name, err)
					continue
				}
				if actionsexpr.Stringify(got) == ev.want {
					continue
				}
				t.Errorf("%s line %d, step %q resolves the wrong base on a %s event.\n\n"+
					"  BASE_SHA: %s\n"+
					"    -> %q\n"+
					"  want %q — %s\n\n"+
					"A step that cannot name a base takes its no-base branch: the issue-resolution "+
					"and decisions-append checks announce that they were skipped and exit 0, and "+
					"record-lint drops its unbumped-edit arm. Extend the fallback chain so this "+
					"event resolves a base, or make the step judge the event itself.",
					rel, step.envLine["BASE_SHA"], step.name, ev.name, raw, actionsexpr.Stringify(got), ev.want, ev.why)
			}

			// The chain hands the all-zeroes placeholder straight through (the
			// force-push case above), so every step must refuse a base rather
			// than assume the expression only ever yields a good one. Either
			// shape counts: comparing against the placeholder, or asking git
			// whether the commit exists.
			if !strings.Contains(step.body, absentSHA) && !strings.Contains(step.body, "git cat-file -e") &&
				!delegatesBaseGuard(step.body) {
				t.Errorf("%s line %d, step %q accepts whatever BASE_SHA holds without refusing an "+
					"unusable base.\n\nA force-push leaves `github.event.before` as the all-zeroes "+
					"placeholder or a rewritten sha that resolves to nothing, and the expression "+
					"passes it through by design. Guard on %q, or probe with `git cat-file -e`, "+
					"before handing it to a range.", rel, step.line, step.name, absentSHA)
			}
		})
	}

	// Fail-closed floors. The parsers are hand-written against a YAML shape, and
	// a shape change that made either return nothing would turn this whole sweep
	// into a pass over an empty set.
	if declaring == 0 {
		t.Errorf("no step in %s declares a `BASE_SHA:` env; the env parser has lost its shape, "+
			"and this gate is asserting nothing", rel)
	}
	if inScope < 3 {
		t.Errorf("%s holds %d range-scoped gate(s) declaring BASE_SHA without judging the event "+
			"themselves (%d exempt); three existed when this test was written — record-lint's "+
			"`-agent-diff` arm, the issue-resolution range check, and the decisions-append check.\n\n"+
			"This is a FLOOR, not the set: the set is derived, so a fourth gate is covered by "+
			"existing. A count below the floor means either the step parser lost its shape or a "+
			"gate stopped being range-scoped, and both deserve a look.", rel, inScope, exempt)
	}
}

// baseGuardingGates are the range gates that derive their own base through
// gitutil.ResolveRangeBase — the one Go derivation of "is this base usable",
// which announces a skip for an empty value or the all-zeroes placeholder and
// refuses a sha that resolves to nothing. A step that hands BASE_SHA straight to
// one of them has not skipped the guard; it has moved it where it is written
// once and tested (TestResolveRangeBase, TestDecisionsAppendSkipsWithoutAUsableBase
// and TestDecisionsAppendFaults). A gate joins this list only with such a test.
var baseGuardingGates = []string{
	"go run ./cmd/record-lint decisions-append \"$BASE_SHA\"",
}

func delegatesBaseGuard(body string) bool {
	for _, g := range baseGuardingGates {
		if strings.Contains(body, g) {
			return true
		}
	}
	return false
}

// withoutSubtree nulls every context path under a prefix — the payload subtree
// an event does not carry. The keys stay present and resolve to null rather than
// being deleted, because a deleted path is an evaluator ERROR ("unknown context
// path") and would report a chain reaching for it as unreadable instead of as
// resolving to nothing, which is what GitHub actually does.
func withoutSubtree(ctx map[string]any, prefix string) map[string]any {
	for k := range ctx {
		if strings.HasPrefix(k, prefix) {
			ctx[k] = nil
		}
	}
	return ctx
}

// workflowStep is one parsed `- name:` step: its declared env (raw value text,
// evaluated later) and the whole block, comments included, for the assertions
// that read the script.
type workflowStep struct {
	name    string
	line    int // 1-based, for the failure message
	body    string
	env     map[string]string
	envLine map[string]int
}

var (
	stepNameRe  = regexp.MustCompile(`^(\s*)-\s+name:\s*(.*)$`)
	envBlockRe  = regexp.MustCompile(`^(\s*)env:\s*$`)
	envEntryRe  = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$`)
	eventNameRe = regexp.MustCompile(`\$\{?EVENT_NAME\}?"?\s*(!=|==?)`)
)

// workflowSteps returns every named step in a workflow, at any indent. Hand-
// parsed for the reason its siblings are: this repository carries no YAML parser
// for its own workflows and adds no dependency for two keys.
func workflowSteps(workflow string) ([]workflowStep, error) {
	lines := strings.Split(workflow, "\n")
	var out []workflowStep
	for i := 0; i < len(lines); i++ {
		m := stepNameRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent := len(m[1])
		// The step ends at the next non-blank line indented no further than its
		// own `-` — the next step, a comment between steps, or the next key.
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if lineIndent(lines[j]) <= indent {
				end = j
				break
			}
		}
		step := workflowStep{
			name:    stripComment(m[2]),
			line:    i + 1,
			body:    strings.Join(lines[i:end], "\n"),
			env:     map[string]string{},
			envLine: map[string]int{},
		}
		for j := i + 1; j < end; j++ {
			e := envBlockRe.FindStringSubmatch(lines[j])
			if e == nil {
				continue
			}
			envIndent := len(e[1])
			for k := j + 1; k < end; k++ {
				if strings.TrimSpace(lines[k]) == "" {
					continue
				}
				if lineIndent(lines[k]) <= envIndent {
					break
				}
				if strings.HasPrefix(strings.TrimSpace(lines[k]), "#") {
					continue
				}
				kv := envEntryRe.FindStringSubmatch(lines[k])
				if kv == nil {
					return nil, fmt.Errorf("line %d: step %q has an env entry this parser cannot read: %q",
						k+1, step.name, lines[k])
				}
				v, err := scalarValue(kv[2], lines, k)
				if err != nil {
					return nil, fmt.Errorf("line %d: step %q env %s: %w", k+1, step.name, kv[1], err)
				}
				step.env[kv[1]] = v
				step.envLine[kv[1]] = k + 1
			}
		}
		out = append(out, step)
	}
	return out, nil
}

// judgesTheEventItself reports whether a step takes responsibility for the event
// dimension rather than delegating it to its BASE_SHA expression. Both halves
// are required: the event name must reach the script through the environment,
// AND the script must actually branch on it. A step that merely imports the
// value has not decided anything, and would otherwise buy an exemption with a
// declaration.
func judgesTheEventItself(step workflowStep) bool {
	if _, ok := step.env["EVENT_NAME"]; !ok {
		return false
	}
	return eventNameRe.MatchString(step.body)
}
