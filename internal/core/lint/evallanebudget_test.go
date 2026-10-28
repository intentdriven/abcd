package lint_test

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The eval lanes' time budget is declared, not inherited (iss-2610241228178135).
//
// `make smoke` and `make evals-cold-reading` ran `go test` with no -timeout, so
// the evals package lived under go test's ten-minute default. On a loaded
// development machine the package ran past it with no test hung and failed a
// preflight whose change touched no eval. The local gate therefore takes its
// budget from EVAL_TIMEOUT, defaulting to minLocalEvalBudget.
//
// The CI half is the race lane's argument (racelanebudget_test.go): a package
// timeout that reaches past its job's timeout-minutes never fires, because the
// runner cancels the job first and a cancellation carries no goroutine dump. So
// every workflow step that runs an eval lane sets EVAL_TIMEOUT itself (in its env,
// or as a make argument where the step is rendered from the release scaffold,
// whose gates carry a run line only), and the
// job's timeout-minutes must hold that budget after everything before it: the
// -timeout of every earlier step in the job (release.yml's verify job runs smoke
// after its 40m race step) plus evalJobHeadroom for the untimed steps.

const minLocalEvalBudget = 20 * time.Minute

// evalJobHeadroom is what a job spends in steps that declare no -timeout before
// its eval step: checkout, setup-go and the load check took about half a minute
// in CI's smoke and cold-reading-evals jobs (merge-group runs of 2026-10-24), and
// release.yml's verify job 2.8 minutes before its race step (racelanebudget_test.go).
// Rounded up to three.
const evalJobHeadroom = 3 * time.Minute

var (
	evalTargets     = []string{"smoke", "evals-cold-reading"}
	evalTimeoutVar  = regexp.MustCompile(`(?m)^EVAL_TIMEOUT \?= (\S+)\s*$`)
	evalStepTimeout = regexp.MustCompile(`(?m)^\s+EVAL_TIMEOUT: "?([^"\s]+)"?\s*$`)
	evalRun         = regexp.MustCompile(`^\s+run: make (smoke|evals-cold-reading)(?: EVAL_TIMEOUT=(\S+))?\s*$`)
	jobHeader       = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	stepHeader      = regexp.MustCompile(`^\s+- `)
	jobTimeout      = regexp.MustCompile(`(?m)^    timeout-minutes: (\d+)\s*$`)
	stepTestTimeout = regexp.MustCompile(`^\s+run: .*\bgo test\b.*\s-timeout[ =](\S+)`)
)

func TestEvalLaneBudgetIsDeclaredAndFitsItsJob(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	makefile := readRepoFile(t, root, "Makefile")

	m := evalTimeoutVar.FindStringSubmatch(makefile)
	if m == nil {
		t.Fatal("Makefile declares no `EVAL_TIMEOUT ?= <duration>`; the eval lanes inherit go test's ten-minute default")
	}
	local, err := time.ParseDuration(m[1])
	if err != nil {
		t.Fatalf("Makefile EVAL_TIMEOUT %q is not a duration: %v", m[1], err)
	}
	if local < minLocalEvalBudget {
		t.Errorf("Makefile EVAL_TIMEOUT is %s, below the %s a loaded development machine needs", local, minLocalEvalBudget)
	}
	for _, target := range evalTargets {
		recipe, ok := makeRecipe(makefile, target)
		if !ok {
			t.Errorf("Makefile declares no `%s:` recipe", target)
			continue
		}
		n := 0
		for _, line := range strings.Split(recipe, "\n") {
			if !strings.Contains(line, "go test") {
				continue
			}
			n++
			if !strings.Contains(line, "-timeout $(EVAL_TIMEOUT)") {
				t.Errorf("Makefile %s runs %q without -timeout $(EVAL_TIMEOUT)", target, strings.TrimSpace(line))
			}
		}
		if n == 0 {
			t.Errorf("Makefile %s runs no go test; the parser or the recipe changed shape", target)
		}
	}

	steps := 0
	for _, file := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		for _, s := range evalSteps(readRepoFile(t, root, file)) {
			steps++
			where := file + " job " + s.job + " (make " + s.target + ")"
			sm := evalStepTimeout.FindStringSubmatch(s.step)
			if s.arg != "" {
				sm = []string{"", s.arg}
			}
			if sm == nil {
				t.Errorf("%s sets no EVAL_TIMEOUT, so the package timeout is the Makefile's %s and the job is cancelled before it fires", where, local)
				continue
			}
			pkg, err := time.ParseDuration(sm[1])
			if err != nil {
				t.Errorf("%s: EVAL_TIMEOUT %q is not a duration: %v", where, sm[1], err)
				continue
			}
			jm := jobTimeout.FindStringSubmatch(s.jobBlock)
			if jm == nil {
				t.Errorf("%s: the job declares no timeout-minutes", where)
				continue
			}
			if s.before < 0 {
				t.Errorf("%s: an earlier step's go test -timeout is not a duration", where)
				continue
			}
			mins, _ := strconv.Atoi(jm[1])
			if need := evalJobHeadroom + s.before + pkg; need >= time.Duration(mins)*time.Minute {
				t.Errorf("%s: EVAL_TIMEOUT %s after %s of earlier step timeouts and %s of headroom (%s) "+
					"does not fit under the job's timeout-minutes %d; a hang would be cancelled with no goroutine dump",
					where, pkg, s.before, evalJobHeadroom, need, mins)
			}
		}
	}
	if steps == 0 {
		t.Error("no workflow step runs make smoke or make evals-cold-reading; the parser or the workflows changed shape")
	}
}

// evalStep is one workflow step that runs an eval lane, with the job it runs in
// and the sum of the go test -timeout values of the job's earlier steps (-1 when
// one of them is not a duration).
type evalStep struct {
	job, target, step, jobBlock string
	arg                         string // EVAL_TIMEOUT given as a make argument
	before                      time.Duration
}

// evalSteps finds every step in a workflow whose run line is `make smoke` or
// `make evals-cold-reading`. Hand-parsed for the reason workflowJobBlock is.
func evalSteps(workflow string) []evalStep {
	lines := strings.Split(workflow, "\n")
	var out []evalStep
	job, jobAt, stepAt := "", -1, -1
	var before time.Duration
	for i, l := range lines {
		if m := jobHeader.FindStringSubmatch(l); m != nil {
			job, jobAt, stepAt, before = m[1], i, -1, 0
			continue
		}
		if m := stepTestTimeout.FindStringSubmatch(l); m != nil && before >= 0 {
			if d, err := time.ParseDuration(m[1]); err == nil {
				before += d
			} else {
				before = -1
			}
		}
		if stepHeader.MatchString(l) {
			stepAt = i
		}
		m := evalRun.FindStringSubmatch(l)
		if m == nil || jobAt < 0 || stepAt < 0 {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if stepHeader.MatchString(lines[j]) || jobHeader.MatchString(lines[j]) {
				end = j
				break
			}
		}
		block, _ := workflowJobBlock(workflow, job)
		out = append(out, evalStep{job: job, target: m[1], step: strings.Join(lines[stepAt:end], "\n"), jobBlock: block, arg: m[2], before: before})
	}
	return out
}
