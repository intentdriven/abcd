package landing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/ghcontract"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// queueRuleset gates the default branch through a merge queue behind a
// pull-request rule requiring one approving review, as .abcd/work/rulesets/
// holds it.
const queueRuleset = `{"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
	`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"},` +
	`{"parameters":{"required_approving_review_count":1},"type":"pull_request"},` +
	`{"parameters":{"merge_method":"MERGE"},"type":"merge_queue"}],"target":"branch"}` + "\n"

// unreviewedQueueRuleset gates the default branch through a merge queue with
// no rule requiring a person's approval (abcd's own ruleset's shape).
const unreviewedQueueRuleset = `{"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
	`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"},` +
	`{"parameters":{"merge_method":"MERGE"},"type":"merge_queue"}],"target":"branch"}` + "\n"

// noQueueRuleset gates the default branch with no merge queue.
const noQueueRuleset = `{"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
	`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"}],"target":"branch"}` + "\n"

// fakeForge answers the forge client's calls from its fields and records
// every call. It never reaches a network.
type fakeForge struct {
	repoURL string
	// nwo is the forge's owner/name for the repository; "" is o/r.
	nwo string
	// sshURL is the forge's ssh URL for the repository; "" is
	// git@example.com:o/r.git.
	sshURL string
	// def is the forge's default branch; "" is main.
	def string
	// head replaces @HEAD@ in a view, so a view reports the pull request's
	// head on the forge.
	head string
	prs  string
	// merged answers the listing of the branch's merged pull requests; ""
	// is none. mergedErr fails that listing alone.
	merged    string
	mergedErr error
	views     []string
	// answer, when set, is the whole answer to the state query, its @HEAD@
	// replaced as a view's is.
	answer string
	// answers are whole answers to the state query, one a call, the last
	// repeated, each @HEAD@ replaced as a view's is; they come before views.
	answers []string
	calls   [][]string
	// onRepoView runs when the repository is viewed, before the answer.
	onRepoView func()
	// failArm fails the arm call with this ordinal (1 is the first), 0 none.
	failArm int
	arms    int
	// err fails every call.
	err error
}

func (f *fakeForge) run(_ string, args ...string) (string, error) {
	f.calls = append(f.calls, args)
	if f.err != nil {
		return "", f.err
	}
	// The real forge client refuses a --json field it does not have, and so
	// does this one, from gh's own lists.
	if err := ghcontract.Check(args); err != nil {
		return "", err
	}
	switch strings.Join(args[:min(2, len(args))], " ") {
	case "repo view":
		if f.onRepoView != nil {
			f.onRepoView()
		}
		ssh, def := f.sshURL, f.def
		if ssh == "" {
			ssh = "git@example.com:o/r.git"
		}
		if def == "" {
			def = "main"
		}
		return `{"nameWithOwner":"` + f.owner() + `","url":"` + f.repoURL + `","sshUrl":"` + ssh + `","defaultBranchRef":{"name":"` + def + `"}}`, nil
	case "pr list":
		if slices.Contains(args, "--state=merged") {
			if f.mergedErr != nil {
				return "", f.mergedErr
			}
			if f.merged == "" {
				return "[]", nil
			}
			return f.merged, nil
		}
		if f.prs == "" {
			return "[]", nil
		}
		return f.prs, nil
	case "api graphql":
		// The pull request's state is asked of the repository the forge
		// resolved (o/r), through the one pinned query.
		n, ok := strings.CutPrefix(args[len(args)-1], "n=")
		owner, name, _ := strings.Cut(f.owner(), "/")
		if !ok || !slices.Equal(args, prStateArgs(owner, name, n)) {
			return "", fmt.Errorf("fake forge: unexpected query %v", args)
		}
		// The real forge refuses a field its schema lacks, and so does this
		// one, from the schema as ghcontract pins it.
		if err := ghcontract.CheckGraphQL(PRStateQuery); err != nil {
			return "", err
		}
		if f.answer != "" {
			return strings.ReplaceAll(f.answer, "@HEAD@", f.head), nil
		}
		if len(f.answers) > 0 {
			a := f.answers[0]
			if len(f.answers) > 1 {
				f.answers = f.answers[1:]
			}
			return strings.ReplaceAll(a, "@HEAD@", f.head), nil
		}
		if len(f.views) == 0 {
			return "", errors.New("fake forge: no view left")
		}
		v := f.views[0]
		if len(f.views) > 1 {
			f.views = f.views[1:]
		}
		return `{"data":{"repository":{"pullRequest":` + strings.ReplaceAll(v, "@HEAD@", f.head) + `}}}`, nil
	case "pr merge":
		if slices.Contains(args, "--auto") {
			if f.arms++; f.arms == f.failArm {
				return "", errors.New("fake forge: the arm failed")
			}
		}
		return "", nil
	}
	return "", fmt.Errorf("fake forge: unexpected call %v", args)
}

// owner is the forge's owner/name for the repository.
func (f *fakeForge) owner() string {
	if f.nwo == "" {
		return "o/r"
	}
	return f.nwo
}

// called reports whether a call began with prefix.
func (f *fakeForge) called(prefix ...string) int {
	n := 0
	for _, c := range f.calls {
		if len(c) >= len(prefix) && slices.Equal(c[:len(prefix)], prefix) {
			n++
		}
	}
	return n
}

// fixture is a repository whose default branch is on a local bare origin,
// with the branch feat/x one commit ahead of it and checked out.
type fixture struct {
	t        *testing.T
	repo     *gittest.Repo
	bare     string
	forge    *fakeForge
	gitCalls [][]string
	gates    []string
	// gate is what the injected gate does; mint (the default) mints a
	// receipt for the checkout's head.
	gate func(dir string) error
	home string
	// beforeGit runs before each networked git call the landing makes.
	beforeGit func(args []string)
}

func newFixture(t *testing.T, ruleset string) *fixture {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write("Makefile", "preflight:\n\t@true\n")
	if ruleset != "" {
		repo.Write(".abcd/work/rulesets/main-protection.json", ruleset)
	}
	repo.Commit("the base")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	repo.Git("checkout", "-q", "-b", "feat/x")
	repo.Write("feature.txt", "the change\n")
	repo.Commit("the change")
	f := &fixture{t: t, repo: repo, bare: bare, forge: &fakeForge{repoURL: bare}, home: t.TempDir()}
	f.gate = f.mint
	return f
}

// head is the checkout's HEAD.
func (f *fixture) head() string { return f.repo.Git("rev-parse", "HEAD") }

// mint writes the preflight receipt for dir's HEAD, as `make preflight` does.
func (f *fixture) mint(dir string) error {
	head, err := gitutil.Run(dir, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	p := filepath.Join(dir, filepath.FromSlash(ReceiptsRelDir), head)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, nil, 0o644)
}

func (f *fixture) options() Options {
	return Options{
		Roots: layered.Roots{Repo: f.repo.Root(), Home: f.home},
		Forge: f.forge.run,
		Git: func(dir string, args ...string) (string, error) {
			f.gitCalls = append(f.gitCalls, args)
			if f.beforeGit != nil {
				f.beforeGit(args)
			}
			return gitutil.Net(dir, args...)
		},
		Gate: func(dir string) error {
			f.gates = append(f.gates, dir)
			return f.gate(dir)
		},
	}
}

func (f *fixture) land(o Options) (Result, error) {
	f.t.Helper()
	if f.forge.head == "" {
		f.forge.head = f.head()
	}
	return Land(f.repo.Root(), o)
}

// pushes are the push calls the landing made.
func (f *fixture) pushes() [][]string {
	var out [][]string
	for _, c := range f.gitCalls {
		if len(c) > 0 && c[0] == "push" {
			out = append(out, c)
		}
	}
	return out
}

// onOrigin is the branch's tip on the bare origin, "" when it has none.
func (f *fixture) onOrigin(branch string) string {
	out, err := gitutil.Run(f.bare, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		return ""
	}
	return out
}

func (f *fixture) prFor(sha string) string {
	return `[{"number":7,"url":"https://example.com/o/r/pull/7","isCrossRepository":false,"headRefOid":"` + sha + `","baseRefName":"main"}]`
}

// step is the result's step named name.
func step(t *testing.T, res Result, name string) Step {
	t.Helper()
	for _, s := range res.Steps {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("no %s step in %+v", name, res.Steps)
	return Step{}
}

func wantRefusal(t *testing.T, err error, says string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if !strings.Contains(r.Error(), says) {
		t.Fatalf("refusal %q does not say %q", r.Error(), says)
	}
}

func TestLandRefusesTheDefaultBranch(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("checkout", "-q", "main")
	_, err := f.land(f.options())
	wantRefusal(t, err, "default branch")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

func TestLandRefusesADetachedHead(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("checkout", "-q", "--detach")
	_, err := f.land(f.options())
	wantRefusal(t, err, "detached")
	if len(f.pushes()) != 0 {
		t.Fatalf("pushed %v", f.pushes())
	}
}

func TestLandRefusesADirtyTree(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Write("stray.txt", "uncommitted\n")
	_, err := f.land(f.options())
	wantRefusal(t, err, "uncommitted")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

func TestABranchCheckedOutNowhereIsRefused(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("branch", "feat/y")
	o := f.options()
	o.Branch = "feat/y"
	_, err := f.land(o)
	wantRefusal(t, err, "not checked out")
}

// The gate runs in the named branch's own checkout, not where land is run.
func TestTheGateRunsInTheNamedBranchsCheckout(t *testing.T) {
	f := newFixture(t, "")
	wt := filepath.Join(t.TempDir(), "wt")
	f.repo.Git("worktree", "add", "-q", "-b", "feat/y", wt, "feat/x")
	f.forge.prs = "[]"
	o := f.options()
	o.Branch = "feat/y"
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.gates) != 1 || fsRealPath(f.gates[0]) != fsRealPath(wt) {
		t.Fatalf("gated in %v, want %s", f.gates, wt)
	}
	if res.Branch != "feat/y" || f.onOrigin("feat/y") == "" {
		t.Fatalf("result %+v; origin holds %q", res, f.onOrigin("feat/y"))
	}
}

func fsRealPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func TestAReceiptForHeadSkipsTheGate(t *testing.T) {
	f := newFixture(t, "")
	if err := f.mint(f.repo.Root()); err != nil {
		t.Fatal(err)
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.gates) != 0 {
		t.Fatalf("the gate ran in %v, want it skipped", f.gates)
	}
	if s := step(t, res, "gate"); s.Status != StatusSkipped {
		t.Fatalf("gate step %+v, want skipped", s)
	}
}

func TestNoPushWithoutAReceiptAfterTheGate(t *testing.T) {
	f := newFixture(t, "")
	f.gate = func(string) error { return nil } // passes, mints nothing
	_, err := f.land(f.options())
	wantRefusal(t, err, "receipt")
	if len(f.gates) != 1 || len(f.pushes()) != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("gated %v, pushed %v: want one gate and no push", f.gates, f.pushes())
	}
}

func TestAFailingGateRefusesAndPushesNothing(t *testing.T) {
	f := newFixture(t, "")
	f.gate = func(string) error { return errors.New("make: *** [preflight] Error 1") }
	_, err := f.land(f.options())
	wantRefusal(t, err, "preflight")
	if len(f.pushes()) != 0 {
		t.Fatalf("pushed %v", f.pushes())
	}
}

func TestAHeadThatMovesDuringTheGateIsRefused(t *testing.T) {
	f := newFixture(t, "")
	f.gate = func(dir string) error {
		f.repo.Write("late.txt", "a late commit\n")
		f.repo.Commit("a late commit")
		return f.mint(dir)
	}
	_, err := f.land(f.options())
	wantRefusal(t, err, "moved")
	if len(f.pushes()) != 0 {
		t.Fatalf("pushed %v", f.pushes())
	}
}

func TestThePushIsPlainAndReachesOrigin(t *testing.T) {
	f := newFixture(t, "")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	pushes := f.pushes()
	want := []string{"push", "--porcelain", "origin", f.head() + ":refs/heads/feat/x"}
	if len(pushes) != 1 || !slices.Equal(pushes[0], want) {
		t.Fatalf("pushes %v, want exactly %v", pushes, want)
	}
	for _, a := range pushes[0] {
		if a == "--force" || a == "-f" || a == "--no-verify" || strings.HasPrefix(a, "--force") || strings.HasPrefix(a, "+") {
			t.Fatalf("push argument %q forces or skips a hook", a)
		}
	}
	if got := f.onOrigin("feat/x"); got != f.head() {
		t.Fatalf("origin holds feat/x at %q, want %s", got, f.head())
	}
	if s := step(t, res, "push"); s.Status != StatusDone {
		t.Fatalf("push step %+v", s)
	}
}

func TestPushOffPrintsTheCommandAndPushesNothing(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Write(".abcd/config.json", `{"land":{"push":false}}`)
	f.repo.Commit("push off")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.pushes()) != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("pushed %v", f.pushes())
	}
	s := step(t, res, "push")
	if s.Status != StatusOff || s.Command != "git push origin refs/heads/feat/x:refs/heads/feat/x" || !strings.Contains(s.Note, "repo layer") {
		t.Fatalf("push step %+v, want off naming the repo layer with the command", s)
	}
}

func TestAnOriginThatIsNotTheForgesRepositoryIsRefused(t *testing.T) {
	f := newFixture(t, "")
	f.forge.repoURL = "https://example.com/someone-else/r"
	_, err := f.land(f.options())
	wantRefusal(t, err, "origin")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: the origin is checked before the gate", f.pushes(), f.gates)
	}
}

func TestNoPullRequestPrintsTheCreateCommand(t *testing.T) {
	f := newFixture(t, queueRuleset)
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || s.Command != "gh pr create --base main --head feat/x" {
		t.Fatalf("enqueue step %+v, want stopped with the create command", s)
	}
	if f.forge.called("pr", "create") != 0 || f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v: land opens and arms nothing", f.forge.calls)
	}
}

func TestACrossRepositoryPullRequestIsRefused(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = strings.Replace(f.prFor(f.head()), `"isCrossRepository":false`, `"isCrossRepository":true`, 1)
	_, err := f.land(f.options())
	wantRefusal(t, err, "another repository")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
}

func TestAPullRequestAtAnotherHeadIsNotArmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(strings.Repeat("a", 40))
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusStopped || f.forge.called("pr", "merge") != 0 {
		t.Fatalf("enqueue step %+v, calls %v: want stopped and unarmed", s, f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "pull it") || !strings.Contains(s, "wait for the forge") {
		t.Fatalf("enqueue note %q: want both remedies, pulling a newer forge head and waiting for an older one", s)
	}
}

const (
	viewBlocked = `{"mergeable":"MERGEABLE","mergeStateStatus":"BLOCKED","isInMergeQueue":false,"autoMergeRequest":null,"headRefOid":"@HEAD@"}`
	viewQueued  = `{"mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","isInMergeQueue":true,"autoMergeRequest":{"mergeMethod":"MERGE"},"headRefOid":"@HEAD@"}`
	viewCleanUn = `{"mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","isInMergeQueue":false,"autoMergeRequest":{"mergeMethod":"MERGE"},"headRefOid":"@HEAD@"}`
	viewDirty   = `{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY","isInMergeQueue":false,"autoMergeRequest":null,"headRefOid":"@HEAD@"}`
	// viewArmedByOther is a pull request someone else armed (squash), CLEAN
	// and outside the queue.
	viewArmedByOther = `{"mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","isInMergeQueue":false,"autoMergeRequest":{"mergeMethod":"SQUASH"},"headRefOid":"@HEAD@"}`
)

func TestAQueueBehindApprovalIsArmedAndConfirmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 || f.forge.called("pr", "merge", "7", "--disable-auto") != 0 {
		t.Fatalf("forge calls %v, want one arm pinned to the head and no toggle", f.forge.calls)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusDone || !strings.Contains(s.Note, "in the merge queue") {
		t.Fatalf("enqueue step %+v, want armed and confirmed queued", s)
	}
}

func TestAnArmedPullRequestStillOutOfTheQueueSaysWhatHoldsIt(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewBlocked}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 0 {
		t.Fatalf("toggled a pull request that is not CLEAN: %v", f.forge.calls)
	}
	if s := step(t, res, "enqueue"); !strings.Contains(s.Note, "BLOCKED") {
		t.Fatalf("enqueue step %+v, want what holds it named", s)
	}
}

func TestAConflictingPullRequestIsRefusedUnarmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewDirty}
	_, err := f.land(f.options())
	wantRefusal(t, err, "conflict")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
}

func TestNoQueueLeavesTheMergeToAPerson(t *testing.T) {
	f := newFixture(t, noQueueRuleset)
	f.forge.prs = f.prFor(f.head())
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if f.forge.called("pr", "merge") != 0 || s.Status != StatusLeft || !strings.Contains(s.Note, "no merge queue") {
		t.Fatalf("enqueue step %+v, calls %v: want nothing armed and the person told", s, f.forge.calls)
	}
}

// Ruling AM1, which human mode is today: a queue no approval guards is left
// for a person to merge.
func TestAQueueWithoutApprovalIsLeftForAPerson(t *testing.T) {
	f := newFixture(t, unreviewedQueueRuleset)
	f.forge.prs = f.prFor(f.head())
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "enqueue"); f.forge.called("pr", "merge") != 0 || s.Status != StatusLeft || !strings.Contains(s.Note, "approval") {
		t.Fatalf("enqueue step %+v, calls %v", s, f.forge.calls)
	}
}

// The rule is read at the merge base with the default branch: a ruleset the
// branch itself adds cannot arm its own merge.
func TestARulesetOnTheBranchTipDoesNotDecideTheMerge(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Write(".abcd/work/rulesets/main-protection.json", queueRuleset)
	f.repo.Commit("a ruleset the branch adds")
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "enqueue"); f.forge.called("pr", "merge") != 0 || s.Status != StatusLeft {
		t.Fatalf("enqueue step %+v, calls %v: want the base's rule (none), nothing armed", s, f.forge.calls)
	}
}

func TestEnqueueOffArmsNothing(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	o := f.options()
	o.NoEnqueue = true
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if f.forge.called("pr", "merge") != 0 || s.Status != StatusOff || !strings.Contains(s.Note, "--no-enqueue") {
		t.Fatalf("enqueue step %+v, calls %v", s, f.forge.calls)
	}
	if f.onOrigin("feat/x") != f.head() {
		t.Fatal("the push should still run with only enqueue off")
	}
}

func TestARepositoryWithNoPreflightTargetGetsNoPush(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Remove("Makefile")
	f.repo.Commit("no preflight")
	o := f.options()
	o.Gate = nil // the default gate, which looks for the target before running anything
	_, err := f.land(o)
	wantRefusal(t, err, "no preflight")
	if len(f.pushes()) != 0 {
		t.Fatalf("pushed %v", f.pushes())
	}
}

func TestMergeGateAppliesAM1(t *testing.T) {
	for _, c := range []struct {
		rule MergeRule
		arm  bool
	}{
		{MergeRule{}, false},
		{MergeRule{Method: "MERGE"}, false},
		{MergeRule{Approval: true}, false},
		{MergeRule{Method: "SQUASH", Approval: true}, true},
	} {
		if d := mergeGate(c.rule, "main"); d.arm != c.arm || d.why == "" {
			t.Errorf("mergeGate(%+v) = %+v, want arm %v with a reason", c.rule, d, c.arm)
		}
	}
}

func TestRepoKeysMatchAcrossURLForms(t *testing.T) {
	same := []string{"https://github.com/O/R", "https://github.com/o/r.git", "git@github.com:o/r.git", "ssh://git@github.com:22/o/r", "https://user@GitHub.com/o/r/"}
	want := repoKey(same[0])
	for _, u := range same[1:] {
		if got := repoKey(u); got != want || got == "" {
			t.Errorf("repoKey(%q) = %q, want %q", u, got, want)
		}
	}
	if repoKey("https://github.com/o/other") == want || repoKey("https://gitlab.com/o/r") == want {
		t.Error("a different repository matched")
	}
	if repoKey("/tmp/x/origin.git") != repoKey("file:///tmp/x/origin") {
		t.Error("a local path and its file URL differ")
	}
	// m9: a port other than the scheme's own is another server.
	for _, u := range []string{"https://github.com:8443/o/r", "ssh://git@github.com:2222/o/r"} {
		if repoKey(u) == want {
			t.Errorf("repoKey(%q) = %q matched the default port's repository", u, repoKey(u))
		}
	}
	if repoKey("https://github.com:443/o/r") != want {
		t.Error("https's own port named another server")
	}
}

// M1: the push sends the commit the gate checked, not whatever the branch
// names when the push runs, even when the branch moves after a skipped gate.
func TestThePushSendsTheGatedCommitWhenTheBranchMovesAfterTheGate(t *testing.T) {
	f := newFixture(t, "")
	gated := f.head()
	if err := f.mint(f.repo.Root()); err != nil {
		t.Fatal(err)
	}
	f.beforeGit = func(args []string) {
		if args[0] == "push" {
			f.repo.Write("late.txt", "a late commit\n")
			f.repo.Commit("a late commit")
			f.beforeGit = nil
		}
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if got := f.onOrigin("feat/x"); got != gated {
		t.Fatalf("origin holds feat/x at %q, want the gated %s", got, gated)
	}
	if s := step(t, res, "push"); res.Head != gated || res.Pushed != gated || !strings.Contains(s.Note, short(gated)) {
		t.Fatalf("result head %s pushed %s, push step %+v: want the gated %s reported", res.Head, res.Pushed, s, gated)
	}
}

// M1: a branch that moves while the origin is checked still pushes only the
// commit the receipt names.
func TestThePushSendsTheGatedCommitWhenTheBranchMovesDuringTheOriginCheck(t *testing.T) {
	f := newFixture(t, "")
	gated := f.head()
	if err := f.mint(f.repo.Root()); err != nil {
		t.Fatal(err)
	}
	f.forge.onRepoView = func() {
		f.repo.Write("late.txt", "a late commit\n")
		f.repo.Commit("a late commit")
		f.forge.onRepoView = nil
	}
	res, err := f.land(f.options())
	if err != nil {
		var r *Refusal
		if !errors.As(err, &r) {
			t.Fatal(err)
		}
	}
	if got := f.onOrigin("feat/x"); got != "" && got != gated {
		t.Fatalf("origin holds feat/x at %q, which no gate checked (gated %s)", got, gated)
	}
	if res.Pushed != "" && res.Pushed != f.onOrigin("feat/x") {
		t.Fatalf("reported pushed %s, origin holds %q", res.Pushed, f.onOrigin("feat/x"))
	}
}

// M2: a re-run once the branch has merged reports it landed, and pushes,
// gates and queues nothing, so the deleted branch is not recreated.
func TestARerunAfterTheMergeReportsLanded(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.repo.Git("push", "-q", "origin", "feat/x:main")
	f.forge.prs = f.prFor(f.head())
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.pushes()) != 0 || len(f.gates) != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("pushed %v, gated %v, origin feat/x %q: want nothing after the merge", f.pushes(), f.gates, f.onOrigin("feat/x"))
	}
	if f.forge.called("pr") != 0 {
		t.Fatalf("forge calls %v: want no pull request asked for or armed", f.forge.calls)
	}
	if !res.Landed || len(res.Steps) != 1 || res.Steps[0].Status != StatusLanded {
		t.Fatalf("result %+v, want one landed step", res)
	}
}

// M3: the forge's own default branch decides, not the checkout's possibly
// stale origin/HEAD.
func TestAForgeDefaultThatDiffersFromTheLocalOneIsRefused(t *testing.T) {
	f := newFixture(t, "")
	f.forge.def = "develop"
	_, err := f.land(f.options())
	wantRefusal(t, err, "develop")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

// M3, the reproduction: the forge's default was renamed to develop after the
// clone, and land is run on develop.
func TestTheForgesDefaultBranchIsNeverPushed(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("checkout", "-q", "-b", "develop")
	f.forge.def = "develop"
	_, err := f.land(f.options())
	wantRefusal(t, err, "default branch")
	if len(f.pushes()) != 0 || len(f.gates) != 0 || f.onOrigin("develop") != "" {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

// M3: a forge that cannot be asked gets no push and no gate.
func TestAForgeThatCannotBeAskedGetsNoPush(t *testing.T) {
	f := newFixture(t, "")
	f.forge.err = errors.New("gh: could not resolve host")
	_, err := f.land(f.options())
	wantRefusal(t, err, "nothing was gated or pushed")
	if len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("pushed %v, gated %v: want neither", f.pushes(), f.gates)
	}
}

// m1: an arming pinned to the head, re-checked in the confirming view.
func TestAHeadThatMovesOnTheForgeAfterArmingIsRefused(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	moved := `{"mergeable":"MERGEABLE","mergeStateStatus":"BLOCKED","isInMergeQueue":false,"autoMergeRequest":{"mergeMethod":"MERGE"},"headRefOid":"` + strings.Repeat("b", 40) + `"}`
	f.forge.views = []string{viewBlocked, moved}
	_, err := f.land(f.options())
	wantRefusal(t, err, "--disable-auto")
}

// m2: a fork's pull request from a branch of the same name does not block
// the branch's own.
func TestAForksPullRequestDoesNotBlockTheBranchsOwn(t *testing.T) {
	f := newFixture(t, queueRuleset)
	fork := `{"number":8,"url":"https://example.com/o/r/pull/8","isCrossRepository":true,"headRefOid":"` + strings.Repeat("c", 40) + `","baseRefName":"main"}`
	f.forge.prs = "[" + fork + "," + strings.TrimPrefix(f.prFor(f.head()), "[")
	f.forge.views = []string{viewBlocked, viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if res.PR != 7 || f.forge.called("pr", "merge", "7", "--auto") != 1 || f.forge.called("pr", "merge", "8") != 0 {
		t.Fatalf("result %+v, calls %v: want #7 armed and the fork's left alone", res, f.forge.calls)
	}
	if f.forge.called("pr", "list", "--head=feat/x", "--state=open", "--json=number,url,isCrossRepository,headRefOid,baseRefName", "--limit=20") != 1 {
		t.Fatalf("calls %v: want the listing raised to 20", f.forge.calls)
	}
}

// m3: every fetch and push URL of origin must be the forge's repository.
func TestEveryOriginURLMustBeTheForgesRepository(t *testing.T) {
	for _, add := range [][]string{
		{"remote", "set-url", "--add", "origin", "https://example.com/someone-else/r"},
		{"remote", "set-url", "--add", "--push", "origin", "https://example.com/someone-else/r"},
	} {
		f := newFixture(t, "")
		if add[3] == "--push" {
			f.repo.Git("remote", "set-url", "--add", "--push", "origin", f.bare)
		}
		f.repo.Git(add...)
		_, err := f.land(f.options())
		wantRefusal(t, err, "someone-else")
		if len(f.pushes()) != 0 {
			t.Fatalf("%v: pushed %v", add, f.pushes())
		}
	}
}

// m5: an off in the landed branch's own checkout counts, even when land is
// run from another checkout.
func TestTheBranchsOwnCheckoutCanTurnAnActOff(t *testing.T) {
	f := newFixture(t, "")
	wt := filepath.Join(t.TempDir(), "wt")
	f.repo.Git("worktree", "add", "-q", "-b", "feat/y", wt, "feat/x")
	if err := os.MkdirAll(filepath.Join(wt, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".abcd", "config.json"), []byte(`{"land":{"push":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", wt, "add", "-A")
	f.repo.Git("-C", wt, "commit", "-q", "-m", "push off on feat/y")
	o := f.options()
	o.Branch = "feat/y"
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Push.On || f.onOrigin("feat/y") != "" {
		t.Fatalf("push %+v, origin feat/y %q: want the branch checkout's off honoured", res.Push, f.onOrigin("feat/y"))
	}
}

// m6: an arming someone else made is left as it is, reported, and its queue
// state confirmed; land neither re-arms nor toggles it.
func TestAnArmingSomeoneElseMadeIsLeftAlone(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewArmedByOther}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v: want the other arming untouched", f.forge.calls)
	}
	if s := step(t, res, "enqueue"); !strings.Contains(s.Note, "already armed") || !strings.Contains(s.Note, "CLEAN") {
		t.Fatalf("enqueue step %+v, want the arming reported and its queue state named", s)
	}
}

// m11: a push git refused names git's own per-ref rejection.
func TestARefusedPushNamesTheRejection(t *testing.T) {
	f := newFixture(t, "")
	diverged := f.repo.Git("commit-tree", "-p", "main", "-m", "diverged", "main^{tree}")
	f.repo.Git("push", "-q", "origin", diverged+":refs/heads/feat/x")
	_, err := f.land(f.options())
	wantRefusal(t, err, "[rejected]")
}

// m12: an ssh host alias is refused, saying so plainly.
func TestAnSSHHostAliasIsRefusedNamingIt(t *testing.T) {
	f := newFixture(t, "")
	f.forge.repoURL = "https://example.com/o/r"
	f.repo.Git("remote", "set-url", "origin", "git@example-work:o/r.git")
	_, err := f.land(f.options())
	wantRefusal(t, err, "example-work")
	wantRefusal(t, err, "alias")
	wantRefusal(t, err, "insteadOf")
}

// m15: the merge rule is read at origin's default branch as fetched, so a
// ruleset the default branch gained after the branch was cut counts.
func TestTheMergeRuleIsReadAtTheFetchedDefaultBranch(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/work/rulesets/main-protection.json", queueRuleset)
	f.repo.Commit("the default branch gains a ruleset")
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("checkout", "-q", "feat/x")
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "enqueue"); f.forge.called("pr", "merge", "7", "--auto", "--merge") != 1 || s.Status != StatusDone {
		t.Fatalf("enqueue step %+v, calls %v: want the fetched default branch's rule to arm it", s, f.forge.calls)
	}
}

// m10: with both acts off the preflight never runs.
func TestBothActsOffRunsNoGate(t *testing.T) {
	f := newFixture(t, "")
	o := f.options()
	o.NoPush, o.NoEnqueue = true, true
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.gates) != 0 || len(f.forge.calls) != 0 {
		t.Fatalf("gated %v, forge calls %v: want neither with both acts off", f.gates, f.forge.calls)
	}
	if s := step(t, res, "gate"); s.Status != StatusSkipped {
		t.Fatalf("gate step %+v, want skipped", s)
	}
}

// M1: a branch whose upstream is origin's default branch (as `git worktree
// add -b B <path> origin/main` sets it), under push.default=upstream, is
// pushed to its own name only: land names the destination ref itself and
// never reads @{u} or the configured upstream.
func TestABranchTrackingTheDefaultBranchPushesOnlyToItsOwnName(t *testing.T) {
	f := newFixture(t, "")
	f.repo.Git("config", "branch.feat/x.remote", "origin")
	f.repo.Git("config", "branch.feat/x.merge", "refs/heads/main")
	f.repo.Git("config", "push.default", "upstream")
	mainBefore := f.onOrigin("main")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if got := f.onOrigin("main"); got != mainBefore {
		t.Fatalf("origin's main moved from %s to %s", mainBefore, got)
	}
	if got := f.onOrigin("feat/x"); got != f.head() || res.Pushed != f.head() {
		t.Fatalf("origin feat/x %q, pushed %q: want the gated %s", got, res.Pushed, f.head())
	}
	for _, p := range f.pushes() {
		if p[len(p)-1] != f.head()+":refs/heads/feat/x" {
			t.Fatalf("push %v names a destination other than refs/heads/feat/x", p)
		}
	}
}
