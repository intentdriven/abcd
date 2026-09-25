package scaffold

// itd-93 AC2: a repository with the scaffolded gate and a green rehearsal on
// record cuts its first release through the merge path, and the release
// PUBLISHES — the gate armed against the reviewed content commit, never the
// tagged merge commit, so the receipt-vs-tag self-reference cannot fail it.
//
// The test runs the rendered workflows themselves, not a description of them:
// a small runner reads each job and step out of the rendered YAML, evaluates
// every `if:`, `env:`, `with:` and output expression with the strict
// actionsexpr evaluator, and runs every `run:` script with bash in a fresh
// clone, the way a runner does. What it cannot run it fakes at the edge, and
// only there: a local bare repository is the forge's git remote, a fake `gh`
// keeps the forge's releases and attestations in a directory, `actions/checkout`
// is a clone, `actions/setup-go` is the Go toolchain already on PATH, and
// `actions/attest` records the subject digests the fake `gh attestation verify`
// later checks. `go run ./cmd/record-lint` — abcd's own program, which the
// semantic profile's receipt gate invokes — runs as a record-lint built from
// this tree. No real forge is ever contacted.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestScaffoldedGateCutsAFirstReleaseThatPublishes drives two profiles through
// the whole merge path: the bare profile `launch scaffold` writes, and the same
// profile with a semantic gate configured, whose receipt gate is the one the
// self-reference once failed.
func TestScaffoldedGateCutsAFirstReleaseThatPublishes(t *testing.T) {
	if testing.Short() {
		t.Skip("drives the release workflows end to end")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to run the workflow scripts")
	}
	for _, semantic := range []bool{false, true} {
		name := "bare"
		if semantic {
			name = "semantic-gate"
		}
		t.Run(name, func(t *testing.T) { cutFirstRelease(t, semantic) })
	}
}

func cutFirstRelease(t *testing.T, semantic bool) {
	goEnv := hostGoEnv(t) // before gittest pins HOME, so the build cache stays warm
	f := newFakeForge(t, goEnv)
	r := f.work

	// A managed repository with no release workflow: a Go module, a changelog,
	// and its own pull-request CI.
	r.Write("go.mod", "module example.com/fixture\n\ngo "+strings.TrimPrefix(runtime.Version(), "go")+"\n")
	r.Write("main.go", "package main\n\nfunc main() {}\n")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n")
	r.Write(".github/workflows/ci.yml", "name: ci\non: [pull_request]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go build ./...\n")

	if semantic {
		// The semantic profile, rendered through the same templates the
		// scaffold uses, with one required detector.
		subs := BareSubstitutions("main")
		subs.CIChecks = DeriveCIChecks(r.Root())
		subs.SemanticGates = []string{"docs-currency-reviewer"}
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		r.Write(ReleaseYMLPath, string(rendered.ReleaseYML))
		r.Write(AutoReleaseYMLPath, string(rendered.AutoReleaseYML))
		r.Write(RunbookPath, string(rendered.Runbook))
		r.Write(CheckReviewsPath, string(rendered.CheckReviews))
		r.Write(".abcd/record-lint.json", `{"rules":{"receipt_gate":{"enabled":false,"severity":"blocker","receipts_dir":".abcd/work/reviews","required_gates":[]}}}`+"\n")
		f.recordLint = buildRecordLint(t, goEnv)
	} else {
		rep, err := Scaffold(Request{RepoRoot: r.Root()})
		if err != nil {
			t.Fatalf("scaffold: %v", err)
		}
		if rep.Wrote != 4 || len(rep.CIChecks) != 1 || rep.CIChecks[0] != "build" {
			t.Fatalf("scaffold report %+v: want four files wired to the repo's own `build` check", rep)
		}
	}
	r.Commit("adopt the scaffolded release gate")
	r.Git("push", "-q", "origin", "main")
	scaffolded := r.Git("rev-parse", "HEAD")

	// 1. The rehearsal, on record and green: a workflow_dispatch of release.yml.
	rehearsal := f.run(ReleaseYMLPath, event{name: "workflow_dispatch", sha: scaffolded, refName: "main"})
	for job, want := range map[string]string{"verify": "success", "rehearsal": "success", "tag": "skipped", "release": "skipped"} {
		if got := rehearsal[job].result; got != want {
			t.Fatalf("rehearsal: job %s = %s, want %s\n%s", job, got, want, f.log.String())
		}
	}
	if tags := r.Git("ls-remote", "--tags", "origin"); tags != "" || len(f.releases()) != 0 {
		t.Fatalf("the rehearsal must publish nothing; tags %q, releases %v", tags, f.releases())
	}

	// 2. The release branch: the CHANGELOG roll (the reviewed content commit),
	// then — with a semantic gate — the receipts naming it.
	r.Git("switch", "-q", "-c", "release")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-09-25\n\n### Added\n\n- the first release.\n")
	r.Commit("release: roll the changelog to 0.1.0")
	content := r.Git("rev-parse", "HEAD")
	if semantic {
		r.Write(".abcd/work/reviews/"+content+"/docs-currency-reviewer.json",
			`{"subject":{"digest":{"gitCommit":"`+content+`"}},"verificationResult":"PROMOTE",`+
				`"policy":{"detector":"docs-currency-reviewer"},"judgeModel":"claude-opus-4-8"}`+"\n")
		r.Commit("release: receipts for the roll")
	}

	// 3. The merge: the release pull request lands on main.
	r.Git("switch", "-q", "main")
	r.Git("merge", "-q", "--no-ff", "-m", "Merge the 0.1.0 release", "release")
	r.Git("push", "-q", "origin", "main")
	merged := r.Git("rev-parse", "HEAD")

	// 4. The push to main runs auto-release, which calls release.yml.
	runs := f.run(AutoReleaseYMLPath, event{name: "push", sha: merged, refName: "main"})
	if runs["detect"].result != "success" || runs["release"].result != "success" {
		t.Fatalf("auto-release did not complete: detect=%s release=%s\n%s",
			runs["detect"].result, runs["release"].result, f.log.String())
	}

	// The release PUBLISHED, from the merged commit, under the tag the chain made.
	rel := f.releases()
	if got := rel["v0.1.0"]; got != merged {
		t.Fatalf("published releases %v: want v0.1.0 built from the merged commit %s\n%s", rel, merged, f.log.String())
	}
	if tagged := r.Git("ls-remote", "origin", "refs/tags/v0.1.0^{}"); !strings.HasPrefix(tagged, merged) {
		t.Errorf("tag v0.1.0 = %q, want it at the merged commit %s", tagged, merged)
	}

	if testing.Verbose() {
		t.Log("workflow run log:\n" + f.log.String())
	}
	if semantic {
		// The receipt gate armed against the reviewed CONTENT commit — the roll —
		// not the tagged merge commit, whose tree can never hold a receipt naming
		// itself. That is the self-reference, and it did not fire.
		armed := runs["release/verify"].outputs["content_sha"]
		if armed != content {
			t.Errorf("the receipt gate armed against %q, want the reviewed content commit %s", armed, content)
		}
		if armed == merged {
			t.Error("the gate armed against the tagged commit: the receipt-vs-tag self-reference")
		}
		if !f.attested(".abcd/work/reviews/" + content + "/docs-currency-reviewer.json") {
			t.Error("the release did not attest the receipts of the commit the gate admitted")
		}
	}
}

// ---------------------------------------------------------------------------
// The fake forge and the workflow runner.

type event struct {
	name    string
	sha     string
	refName string
	inputs  map[string]any
}

type jobRun struct {
	result  string
	outputs map[string]string
}

type fakeForge struct {
	t          *testing.T
	work       *gittest.Repo
	origin     string // the forge's git remote (a bare repository)
	state      string // releases/ and attestations/ live here
	bin        string // the fake gh
	env        []string
	recordLint string
	log        strings.Builder
}

func newFakeForge(t *testing.T, goEnv []string) *fakeForge {
	t.Helper()
	r := gittest.NewRepo(t)
	f := &fakeForge{t: t, work: r, origin: t.TempDir(), state: t.TempDir(), bin: t.TempDir()}
	bare := exec.Command("git", "init", "-q", "--bare", "--initial-branch=main", f.origin)
	bare.Env = r.Env()
	if out, err := bare.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	r.Git("remote", "add", "origin", f.origin)
	mustWrite(t, filepath.Join(f.bin, "gh"), fakeGH)
	if err := os.Chmod(filepath.Join(f.bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append([]string{}, r.Env()...)
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = "PATH=" + f.bin + string(os.PathListSeparator) + strings.TrimPrefix(kv, "PATH=")
		}
	}
	f.env = append(append(env, goEnv...),
		"FORGE_STATE="+f.state, "FORGE_ORIGIN="+f.origin,
		"GITHUB_REPOSITORY=example/fixture", "GITHUB_ACTIONS=true", "GOTOOLCHAIN=local",
		"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
	return f
}

// fakeGH is the forge's command line, answering exactly the calls the
// scaffolded workflows make and refusing anything else loudly.
const fakeGH = `#!/usr/bin/env bash
set -euo pipefail
case "${1:-} ${2:-}" in
"release view")
  [ -f "$FORGE_STATE/releases/$3" ] ;;
"release create")
  tag="$3"; shift 3
  verify=0
  while [ $# -gt 0 ]; do
    case "$1" in --verify-tag) verify=1 ;; --title) shift ;; *) ;; esac
    shift
  done
  [ "$verify" = 1 ] || { echo "fake gh: release create without --verify-tag" >&2; exit 98; }
  [ ! -f "$FORGE_STATE/releases/$tag" ] || { echo "release $tag already exists" >&2; exit 1; }
  commit="$(git --git-dir="$FORGE_ORIGIN" rev-parse -q --verify "refs/tags/$tag^{commit}")" || {
    echo "tag $tag does not exist on the forge" >&2; exit 1; }
  mkdir -p "$FORGE_STATE/releases"
  printf '%s\n' "$commit" > "$FORGE_STATE/releases/$tag" ;;
"run list")
  printf '\n' ;;
"attestation verify")
  digest="$(git hash-object "$3")"
  [ -f "$FORGE_STATE/attestations/$digest" ] || { echo "no attestation for $3" >&2; exit 1; } ;;
"api "*)
  case "$2" in
  repos/*/git/ref/heads/*)
    git --git-dir="$FORGE_ORIGIN" rev-parse "refs/heads/${2##*/heads/}" ;;
  *) echo "fake gh: unsupported api $2" >&2; exit 97 ;;
  esac ;;
*)
  echo "fake gh: unsupported: $*" >&2; exit 97 ;;
esac
`

// releases is the forge's published releases: tag -> the commit it was built from.
func (f *fakeForge) releases() map[string]string {
	out := map[string]string{}
	entries, _ := os.ReadDir(filepath.Join(f.state, "releases"))
	for _, e := range entries {
		data, _ := os.ReadFile(filepath.Join(f.state, "releases", e.Name()))
		out[e.Name()] = strings.TrimSpace(string(data))
	}
	return out
}

// attested reports whether the forge holds an attestation for the file at rel
// in the released tree.
func (f *fakeForge) attested(rel string) bool {
	cmd := exec.Command("git", "-C", f.work.Root(), "rev-parse", "HEAD:"+rel)
	cmd.Env = f.work.Env()
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(f.state, "attestations", strings.TrimSpace(string(out))))
	return err == nil
}

// run executes one workflow file, as committed at ev.sha, for one event, and
// returns every job's result keyed by job id ("<caller job>/<job>" for the jobs
// of a called workflow).
func (f *fakeForge) run(path string, ev event) map[string]jobRun {
	f.t.Helper()
	src := f.work.Git("show", ev.sha+":"+path)
	wf := parseYAML(src)
	results := map[string]jobRun{}
	f.runWorkflow(wf, ev, "", results)
	return results
}

func (f *fakeForge) runWorkflow(wf *ynode, ev event, prefix string, results map[string]jobRun) {
	jobs := wf.get("jobs")
	if jobs == nil {
		f.t.Fatal("workflow has no jobs")
	}
	local := map[string]jobRun{}
	for _, id := range jobs.keys {
		job := jobs.m[id]
		needs := listOf(job.get("needs"))
		ctx := f.baseContext(ev)
		allGreen := true
		for _, n := range needs {
			nr := local[n]
			ctx["needs."+n+".result"] = nr.result
			for k, v := range nr.outputs {
				ctx["needs."+n+".outputs."+k] = v
			}
			if nr.result != "success" {
				allGreen = false
			}
		}
		ctx["success()"], ctx["failure()"], ctx["cancelled()"] = allGreen, false, false
		run := true
		if cond := job.str("if"); cond != "" {
			run = f.evalIf(cond, ctx)
		} else {
			run = allGreen
		}
		if !run {
			local[id] = jobRun{result: "skipped"}
			results[prefix+id] = local[id]
			fmt.Fprintf(&f.log, "job %s%s: skipped\n", prefix, id)
			continue
		}
		var jr jobRun
		if uses := job.str("uses"); uses != "" {
			jr = f.callWorkflow(job, uses, ev, ctx, prefix+id+"/", results)
		} else {
			jr = f.runJob(prefix+id, job, wf, ev, ctx)
		}
		local[id] = jr
		results[prefix+id] = jr
	}
}

// callWorkflow runs a reusable workflow (`uses: ./.github/workflows/x.yml`) as
// GitHub does: same event and commit, inputs from the caller's `with:`.
func (f *fakeForge) callWorkflow(job *ynode, uses string, ev event, ctx map[string]any, prefix string, results map[string]jobRun) jobRun {
	if !strings.HasPrefix(uses, "./") {
		f.t.Fatalf("unsupported reusable workflow %q", uses)
	}
	inputs := map[string]any{}
	if with := job.get("with"); with != nil {
		for _, k := range with.keys {
			// Typed, as GitHub hands a called workflow its inputs: a boolean
			// input stays a boolean, so `if: inputs.create_tag` reads false as false.
			inputs[k] = f.evalAny(with.m[k].scalar, ctx)
		}
	}
	called := parseYAML(f.work.Git("show", ev.sha+":"+strings.TrimPrefix(uses, "./")))
	// Declared inputs the caller did not pass take their defaults.
	if on := called.get("on"); on != nil {
		if wc := on.get("workflow_call"); wc != nil {
			if decl := wc.get("inputs"); decl != nil {
				for _, k := range decl.keys {
					if _, ok := inputs[k]; !ok {
						d := decl.m[k].str("default")
						if d == "false" || d == "true" {
							inputs[k] = d == "true"
						} else {
							inputs[k] = strings.Trim(d, `'"`)
						}
					}
				}
			}
		}
	}
	sub := map[string]jobRun{}
	f.runWorkflow(called, event{name: ev.name, sha: ev.sha, refName: ev.refName, inputs: inputs}, prefix, sub)
	res := jobRun{result: "success"}
	for k, v := range sub {
		results[k] = v
		if v.result == "failure" {
			res.result = "failure"
		}
	}
	return res
}

func (f *fakeForge) baseContext(ev event) map[string]any {
	ctx := map[string]any{
		"github.sha": ev.sha, "github.ref_name": ev.refName, "github.event_name": ev.name,
		"github.repository": "example/fixture", "github.token": "fake-token", "secrets.GITHUB_TOKEN": "fake-token",
		"github.event.repository.default_branch": "main", "github.event.repository.private": false,
	}
	for k, v := range ev.inputs {
		ctx["inputs."+k] = v
	}
	return ctx
}

// unknownPathRe names the context path the strict evaluator could not find.
var unknownPathRe = regexp.MustCompile(`unknown context path "([^"]+)"`)

// withMissing retries an evaluation, supplying GitHub's null for an absent
// input, output or env value — and only those: any other unknown context is a
// failure, so an expression reading something the runner does not model can
// never pass by accident.
func (f *fakeForge) withMissing(ctx map[string]any, eval func() error) {
	f.t.Helper()
	for i := 0; i < 16; i++ {
		err := eval()
		if err == nil {
			return
		}
		m := unknownPathRe.FindStringSubmatch(err.Error())
		if m == nil || !(strings.HasPrefix(m[1], "inputs.") || strings.HasPrefix(m[1], "env.") ||
			(strings.HasPrefix(m[1], "steps.") || strings.HasPrefix(m[1], "needs.")) && strings.Contains(m[1], ".outputs.")) {
			f.t.Fatalf("expression evaluation: %v", err)
		}
		ctx[m[1]] = nil
	}
	f.t.Fatal("expression evaluation did not converge")
}

func (f *fakeForge) evalIf(cond string, ctx map[string]any) bool {
	var got bool
	f.withMissing(ctx, func() (err error) { got, err = actionsexpr.EvalIf(cond, ctx); return err })
	return got
}

func (f *fakeForge) eval(raw string, ctx map[string]any) string {
	return actionsexpr.Stringify(f.evalAny(raw, ctx))
}

func (f *fakeForge) evalAny(raw string, ctx map[string]any) any {
	var got any
	f.withMissing(ctx, func() (err error) { got, err = actionsexpr.EvalValue(raw, ctx); return err })
	return got
}

// runJob runs one job's steps in a fresh workspace.
func (f *fakeForge) runJob(id string, job, wf *ynode, ev event, ctx map[string]any) jobRun {
	f.t.Helper()
	ws := f.t.TempDir()
	jobEnv := map[string]string{}
	if we := wf.get("env"); we != nil {
		for _, k := range we.keys {
			jobEnv[k] = f.eval(we.m[k].scalar, ctx)
		}
	}
	failed := false
	steps := job.get("steps")
	for i, step := range steps.seq {
		sid := step.str("id")
		ctx["success()"], ctx["failure()"] = !failed, failed
		for k, v := range jobEnv {
			ctx["env."+k] = v
		}
		cond := step.str("if")
		if cond == "" {
			cond = "success()"
		}
		if !f.evalIf(cond, ctx) {
			fmt.Fprintf(&f.log, "  %s step %d %q: skipped\n", id, i, step.str("name"))
			continue
		}
		outFile := filepath.Join(f.t.TempDir(), "output")
		envFile := filepath.Join(f.t.TempDir(), "env")
		mustWrite(f.t, outFile, "")
		mustWrite(f.t, envFile, "")
		var ok bool
		var out string
		switch uses := step.str("uses"); {
		case strings.HasPrefix(uses, "actions/checkout@"):
			ref := ev.sha
			if with := step.get("with"); with != nil && with.get("ref") != nil {
				if v := f.eval(with.get("ref").scalar, ctx); v != "" {
					ref = v
				}
			}
			ok, out = f.shell(ws, jobEnv, nil, "git clone -q \"$FORGE_ORIGIN\" . && git checkout -q --detach "+ref, "", "")
		case strings.HasPrefix(uses, "actions/setup-go@"):
			ok = true // the toolchain on PATH is the one go.mod names (GOTOOLCHAIN=local)
		case strings.HasPrefix(uses, "actions/attest@"):
			subject := f.eval(step.get("with").get("subject-path").scalar, ctx)
			ok, out = f.shell(ws, jobEnv, nil, `mkdir -p "$FORGE_STATE/attestations" && for s in `+subject+
				`; do [ -f "$s" ] || exit 1; : > "$FORGE_STATE/attestations/$(git hash-object "$s")"; done`, "", "")
		case uses != "":
			f.t.Fatalf("job %s step %q uses %s, which this runner does not model", id, step.str("name"), uses)
		default:
			stepEnv := map[string]string{}
			if se := step.get("env"); se != nil {
				for _, k := range se.keys {
					stepEnv[k] = f.eval(se.m[k].scalar, ctx)
				}
			}
			script := step.str("run")
			if f.recordLint != "" {
				script = strings.ReplaceAll(script, "go run ./cmd/record-lint", f.recordLint)
			}
			ok, out = f.shell(ws, jobEnv, stepEnv, script, outFile, envFile)
		}
		fmt.Fprintf(&f.log, "  %s step %d %q: ok=%v\n%s", id, i, step.str("name"), ok, indent(out))
		for k, v := range readKV(f.t, outFile) {
			if sid != "" {
				ctx["steps."+sid+".outputs."+k] = v
			}
		}
		for k, v := range readKV(f.t, envFile) {
			jobEnv[k] = v
		}
		if !ok {
			failed = true
		}
	}
	jr := jobRun{result: "success", outputs: map[string]string{}}
	if failed {
		jr.result = "failure"
	}
	if outs := job.get("outputs"); outs != nil {
		for _, k := range outs.keys {
			jr.outputs[k] = f.eval(outs.m[k].scalar, ctx)
		}
	}
	fmt.Fprintf(&f.log, "job %s: %s %v\n", id, jr.result, jr.outputs)
	return jr
}

// shell runs a script the way a runner's default bash shell does.
func (f *fakeForge) shell(dir string, jobEnv, stepEnv map[string]string, script, outFile, envFile string) (bool, string) {
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
	cmd.Dir = dir
	env := append([]string{}, f.env...)
	for _, m := range []map[string]string{jobEnv, stepEnv} {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			env = append(env, k+"="+m[k])
		}
	}
	env = append(env, "GITHUB_OUTPUT="+outFile, "GITHUB_ENV="+envFile, "RUNNER_TEMP="+f.t.TempDir())
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return err == nil, string(out)
}

func readKV(t *testing.T, path string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if path == "" {
		return out
	}
	fh, err := os.Open(path)
	if err != nil {
		return out
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok {
			out[k] = v
		}
	}
	return out
}

func indent(s string) string {
	if s == "" {
		return ""
	}
	return "    | " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    | ") + "\n"
}

// hostGoEnv is the test process's Go caches, so the workflow's go commands
// build against a warm cache under the fixture's hermetic HOME.
func hostGoEnv(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH").Output()
	if err != nil {
		t.Skipf("go env: %v", err)
	}
	vals := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(vals) != 3 {
		t.Fatalf("go env printed %q", out)
	}
	return []string{"GOCACHE=" + vals[0], "GOMODCACHE=" + vals[1], "GOPATH=" + vals[2], "GOFLAGS=", "CGO_ENABLED=" + cgo()}
}

func cgo() string {
	out, err := exec.Command("go", "env", "CGO_ENABLED").Output()
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(string(out))
}

// buildRecordLint builds abcd's record-lint from this tree — the program the
// semantic profile's receipt gate runs as `go run ./cmd/record-lint`.
func buildRecordLint(t *testing.T, goEnv []string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "record-lint")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/record-lint")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), goEnv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build record-lint: %v\n%s", err, out)
	}
	return bin
}

func listOf(n *ynode) []string {
	if n == nil {
		return nil
	}
	s := strings.TrimSpace(n.scalar)
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// A reader for the block-style YAML subset the templates are written in:
// block mappings, block sequences, plain and quoted scalars, flow sequences
// kept as text, and literal block scalars. It is enough to run the workflows;
// it is not a YAML parser.

type ynode struct {
	scalar string
	keys   []string
	m      map[string]*ynode
	seq    []*ynode
}

func (n *ynode) get(k string) *ynode {
	if n == nil || n.m == nil {
		return nil
	}
	return n.m[k]
}

func (n *ynode) str(k string) string {
	if c := n.get(k); c != nil {
		return c.scalar
	}
	return ""
}

type yparser struct {
	lines []string
	i     int
}

func parseYAML(src string) *ynode {
	p := &yparser{lines: strings.Split(src, "\n")}
	p.skip()
	if p.i >= len(p.lines) {
		return &ynode{}
	}
	return p.block(p.indentOf(p.i))
}

func (p *yparser) indentOf(i int) int {
	return len(p.lines[i]) - len(strings.TrimLeft(p.lines[i], " "))
}

// skip moves past blank and comment-only lines.
func (p *yparser) skip() {
	for p.i < len(p.lines) {
		t := strings.TrimSpace(p.lines[p.i])
		if t != "" && !strings.HasPrefix(t, "#") {
			return
		}
		p.i++
	}
}

func (p *yparser) block(ind int) *ynode {
	p.skip()
	if p.i < len(p.lines) && strings.HasPrefix(strings.TrimSpace(p.lines[p.i]), "- ") {
		return p.sequence(ind)
	}
	return p.mapping(ind)
}

func (p *yparser) sequence(ind int) *ynode {
	n := &ynode{}
	for {
		p.skip()
		if p.i >= len(p.lines) || p.indentOf(p.i) != ind || !strings.HasPrefix(strings.TrimSpace(p.lines[p.i]), "- ") {
			return n
		}
		// Rewrite "- key: v" as a mapping line two columns in, and read the item.
		l := p.lines[p.i]
		p.lines[p.i] = l[:ind] + "  " + l[ind+2:]
		if !strings.Contains(strings.TrimSpace(p.lines[p.i]), ": ") && !strings.HasSuffix(strings.TrimSpace(p.lines[p.i]), ":") {
			n.seq = append(n.seq, &ynode{scalar: plain(strings.TrimSpace(p.lines[p.i]))})
			p.i++
			continue
		}
		n.seq = append(n.seq, p.mapping(ind+2))
	}
}

func (p *yparser) mapping(ind int) *ynode {
	n := &ynode{m: map[string]*ynode{}}
	for {
		p.skip()
		if p.i >= len(p.lines) || p.indentOf(p.i) != ind || strings.HasPrefix(strings.TrimSpace(p.lines[p.i]), "- ") {
			return n
		}
		t := strings.TrimSpace(p.lines[p.i])
		k, rest, _ := strings.Cut(t, ":")
		k = strings.Trim(k, `"'`)
		rest = strings.TrimSpace(rest)
		p.i++
		var child *ynode
		switch {
		case strings.HasPrefix(rest, "|") || strings.HasPrefix(rest, ">"):
			child = &ynode{scalar: p.blockScalar(ind)}
		case rest == "" || strings.HasPrefix(rest, "#"):
			p.skip()
			if p.i < len(p.lines) && p.indentOf(p.i) > ind {
				child = p.block(p.indentOf(p.i))
			} else {
				child = &ynode{}
			}
		default:
			child = &ynode{scalar: plain(rest)}
		}
		if _, seen := n.m[k]; !seen {
			n.keys = append(n.keys, k)
		}
		n.m[k] = child
	}
}

// blockScalar reads a literal block scalar's body, dedented.
func (p *yparser) blockScalar(ind int) string {
	var body []string
	bodyInd := -1
	for p.i < len(p.lines) {
		l := p.lines[p.i]
		if strings.TrimSpace(l) == "" {
			body = append(body, "")
			p.i++
			continue
		}
		li := len(l) - len(strings.TrimLeft(l, " "))
		if li <= ind {
			break
		}
		if bodyInd < 0 {
			bodyInd = li
		}
		body = append(body, l[bodyInd:])
		p.i++
	}
	return strings.TrimRight(strings.Join(body, "\n"), "\n") + "\n"
}

// plain reads a scalar: quotes stripped, a trailing comment dropped from an
// unquoted value.
func plain(v string) string {
	if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') {
		if end := strings.LastIndexByte(v, v[0]); end > 0 {
			return v[1:end]
		}
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
