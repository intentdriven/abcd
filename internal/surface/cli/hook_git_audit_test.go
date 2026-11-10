package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The hooks' git calls and their deadlines (iss-2611100548341325,
// dcn-2611100854404911). A hook runs on the host's clock: a git that hangs in
// one holds the hook until the host's own timeout, so every git call a hook
// reaches is to run under that hook's context. Hundreds of call sites in the
// module still run with no deadline, so this audit is a ratchet, not a strict
// rule: it walks the calls from the binary's entry and each hook verb's
// handlers, counts every deadline-free git call it reaches, and compares the
// counts with testdata/hook-git-baseline.txt. A reached call missing from the
// baseline, or reached more times than it lists, fails; so does an entry
// reached fewer times or not at all, until the baseline is lowered to match.
// A line added to the baseline by hand is not caught here, only by review of
// the baseline's diff. The docs may say every hook git call is bound only
// once the baseline is empty.
//
// THE WALK. The module's packages are type-checked from source with the
// standard library alone (go list names the files, for darwin and for linux;
// go/parser and go/types do the rest; golang.org/x/tools is not a
// dependency), and each system is counted on its own. The graph's nodes are
// the module's declared functions and methods, its function literals and its
// package-level variables. Any use of a function, called or taken as a value,
// is an edge to it; a function literal is an edge from the function that
// holds it; a call of an interface method is an edge to that method on every
// module type that implements the interface; a use of a package variable is
// an edge to what its initialiser refers to, which is how a seam (var x =
// pkg.F) is followed. The roots are cmd/abcd's main, which builds the whole
// command tree, every package's init, and each hook verb's Run and RunE
// handlers. A command literal's Run and RunE are its only fields not followed
// from the function that builds it: a hook's are rooted, and every other
// verb's are left out, a call that builds one included. A hook verb whose
// literal sets no Run or RunE (its handler set by assignment) is refused at
// its entry, and so is a handler that is not a literal, a call or a module
// function.
//
// A FLAGGED CALL is a use of exec.Command, os.StartProcess, syscall.Exec,
// ForkExec or StartProcess; an exec.Cmd composite literal, declaration or
// new(exec.Cmd); a use of a gitutil or localtier function or method that has a
// sibling named <Name>Context; a call from outside those two packages of one
// of theirs that takes no context but reaches a flagged call; and any
// deadline-free context passed to a context parameter of a module function or
// method (an interface method included, a variadic context parameter, and a
// method expression's, its arguments read after the receiver) or of
// exec.CommandContext, whether or not that callee reaches git. That last over-counts on purpose: a wrapper or
// a constructor that carries the context on is caught where it is passed, and
// an entry is lowered only by passing a real deadline. A context is
// deadline-free when it is context.Background() or TODO(); WithCancel,
// WithCancelCause, WithValue or WithoutCancel of one; a cobra command's
// Context(), which is Background as run executes it; a call of a module
// function that returns one; or a variable or struct field assigned or
// keyed-initialised one. These are marked to one fixpoint over the whole
// module before anything is walked, so the order of declarations and of
// packages does not matter. Each flagged call is keyed on the system, the
// caller (module-relative package and function, a literal as its enclosing
// function with $lit, a handler built by a call with $run) and the callee,
// with the number of times reached code makes it, never a line.
//
// NOT MATCHED, so not counted (a static walk cannot follow every route a
// value takes): a deadline-free context passed through a call of a
// function-typed variable or field (the Run seam of tools.Installer in
// internal/core/tools/install.go is one), held in a positional struct
// literal, a map, slice or channel, returned as one of several results
// (ctx, err := f()), returned by an interface method or a function literal,
// or made in a function literal called in place (the over-count catches most
// of these where the context is passed on); a method expression on an
// instantiated generic type (T[int].M); exec.Cmd or a context under an alias
// type; an exec.Cmd held as a struct field or a slice or map element (a
// variable declared as one is matched); a Run or RunE set by assignment on a verb that is not a hook,
// which is walked with the function that sets it; a handler that calls on
// through a struct field holding a function; a call the standard library or
// a dependency makes back into the module through an interface (String,
// MarshalJSON, the fmt and errors callbacks Error, Is, As and Unwrap, a
// sort.Interface, and a module type implementing a dependency's interface
// whose method takes a context); and reflection. The hook scripts are read for the binary
// called as "$g", "$x" or "$r/abcd"; a call by another spelling is not seen.

// hookGitBaseline is the ratchet's file, beside this test.
const hookGitBaseline = "testdata/hook-git-baseline.txt"

// hookGitBaselineHeader heads the baseline when the update rewrites it.
const hookGitBaselineHeader = `# Deadline-free git calls the hook verbs reach (iss-2611100548341325,
# dcn-2611100854404911). One entry per line, sorted, each system on its own:
#   <goos> <caller> -> <callee> x<times reached code makes the call>
# The forms counted, and those the walk does not match, are named in the
# test's comment.
# TestHookGitCallsMatchBaseline (hook_git_audit_test.go) refuses a reached
# call not listed here or made more times than listed, and fails on an entry
# made fewer times or no longer reached until it is lowered or removed.
# A line added here by hand is not caught by the test, only by review of this
# file's diff: add none, bind the call to the hook's context instead. After
# binding calls, lower the counts with
#   ABCD_UPDATE_BASELINE=1 go test -run TestHookGitCallsMatchBaseline ./internal/surface/cli
# which only lowers and removes, and refuses to write while any call is new
# or rising, or while the hook verbs found differ from those hooks/ runs.
`

// hookEntry names one hook verb (as hooks/*.sh run it) and its entry: the
// command literal whose Use begins with use, inside the top-level function fn
// of package cli. Its Run and RunE handlers are roots.
type hookEntry struct{ verb, fn, use string }

// hookEntries are the verbs hooks/*.sh run; TestHookEntriesAreTheScriptsVerbs
// holds the list to the scripts.
var hookEntries = []hookEntry{
	{"guard hook", "newGuardHookCommand", "hook"},
	{"hook dogfood-binary", "newDogfoodBinaryCommand", "dogfood-binary"},
	{"hook permission-request", "newPermissionRequestCommand", "permission-request"},
	{"hook prompt-router", "newHookCommand", "prompt-router"},
	{"hook prompt-router-reset", "newHookCommand", "prompt-router-reset"},
	{"hook question-answered", "newHookCommand", "question-answered"},
	{"hook session-end", "newHookCommand", "session-end"},
	{"hook session-start", "newHookCommand", "session-start"},
	{"hook stop", "newHookStopCommand", "stop"},
	{"hook stop-failure", "newHookStopFailureCommand", "stop-failure"},
	{"hook subagent-stop", "newSubagentStopCommand", "subagent-stop"},
}

// auditSpec says what one walk starts from.
type auditSpec struct {
	entryPkg string      // module-relative package holding the entries
	entries  []hookEntry // command literals whose handlers are roots
	roots    []string    // module-relative functions, pkg.Func, also rooted
	cmdPkg   string      // the package whose struct's Run and RunE are handlers
}

var hookAuditSpec = auditSpec{
	entryPkg: "internal/surface/cli",
	entries:  hookEntries,
	roots:    []string{"cmd/abcd.main"},
	cmdPkg:   "github.com/spf13/cobra",
}

// hookCounts maps "<caller> -> <callee>" to the times reached code makes it.
type hookCounts map[string]int

func TestHookGitCallsMatchBaseline(t *testing.T) {
	if raceEnabled {
		t.Skip("a type-check of the module gains nothing under -race; the uninstrumented run asserts it")
	}
	if testing.Short() {
		t.Skip("type-checks the module from source")
	}
	root := moduleRoot(t)
	reached, err := hookGitAuditAll(root, "./cmd/abcd", hookAuditSpec)
	if err != nil {
		t.Fatal(err)
	}
	base, err := readHookGitBaseline(hookGitBaseline)
	if err != nil {
		t.Fatal(err)
	}
	added, stale := diffHookGitBaseline(reached, base)
	if len(stale) > 0 && os.Getenv("ABCD_UPDATE_BASELINE") == "1" {
		verbs, err := hookVerbsInScripts(root)
		if err == nil {
			err = hookEntriesMatch(verbs)
		}
		var next hookCounts
		if err == nil {
			next, err = updatedBaseline(base, reached)
		}
		if err != nil {
			t.Fatalf("the update wrote nothing: %v", err)
		}
		if err := writeHookGitBaseline(hookGitBaseline, next); err != nil {
			t.Fatal(err)
		}
		t.Logf("lowered or removed %d entries of %s", len(stale), hookGitBaseline)
		stale = nil
	}
	var msg strings.Builder
	if len(added) > 0 {
		fmt.Fprintf(&msg, "%d deadline-free git call(s) a hook reaches are new or made more often than %s lists; bind each to the hook's context (never raise the baseline):\n", len(added), hookGitBaseline)
		for _, k := range added {
			fmt.Fprintf(&msg, "  %s\n", k)
		}
	}
	if len(stale) > 0 {
		fmt.Fprintf(&msg, "%d entr(ies) of %s are made fewer times or no longer reached; lower them (ABCD_UPDATE_BASELINE=1 does):\n", len(stale), hookGitBaseline)
		for _, k := range stale {
			fmt.Fprintf(&msg, "  %s\n", k)
		}
	}
	if msg.Len() > 0 {
		t.Fatal(msg.String())
	}
}

// TestHookGitAuditFollowsAndFlags plants calls in a module of its own and
// proves the walk reaches them through a helper, an interface, a seam, a
// literal, the tree's constructor and its Args, and a handler built by a
// call; that it flags the context-free forms, a deadline-free context passed
// six ways (a method expression's and a variadic parameter's among them), an
// exec.Cmd literal and one a package variable holds, a method with a Context sibling and a call
// of a wrapper that reaches one; that it counts each call; and that it leaves
// out a verb's own handler, an unreached caller and a git-free helper.
func TestHookGitAuditFollowsAndFlags(t *testing.T) {
	dir := writeAuditFixture(t)
	spec := auditSpec{
		entryPkg: "cmd/fx",
		entries:  []hookEntry{{"hook hook", "newHook", "hook"}, {"hook wrapped", "newWrapped", "wrapped"}},
		roots:    []string{"cmd/fx.main"},
		cmdPkg:   "example.com/fx/cmd/fx",
	}
	got, err := hookGitAuditAll(dir, "./cmd/fx", spec)
	if err != nil {
		t.Fatal(err)
	}
	common := []string{
		"cmd/fx.(*disk).Read -> exec.Command x1",
		"cmd/fx.helper -> gitutil.Run x1",
		"cmd/fx.loadPlain -> localtier.Untracked x1",
		"cmd/fx.newHook$lit -> exec.Cmd x3",
		"cmd/fx.newHook$lit -> gitutil.(Repo).Status x1",
		"cmd/fx.newHook$lit -> gitutil.Run x2",
		"cmd/fx.newHook$lit -> gitutil.RunContext(Background) x3",
		"cmd/fx.newHook$lit -> gitutil.Wrapped x1",
		"cmd/fx.tree -> gitutil.Run x1",
		"cmd/fx.tree$lit -> exec.Command x1",
		"cmd/fx.viaLit$lit -> exec.Command x1",
		"cmd/fx.wrapped -> gitutil.Run x1",
		"internal/gitutil.(Repo).Status -> gitutil.(Repo).StatusContext(Background) x1",
		"internal/gitutil.Run -> gitutil.RunContext(Background) x1",
		"internal/gitutil.Wrapped -> gitutil.Run x1",
		"internal/gitutil.init -> exec.Command x1",
		"internal/localtier.Untracked -> localtier.UntrackedContext(Background) x1",
		// A context through a chain of functions declared out of order, an
		// interface method an implementation of which reaches git, and a
		// struct field set two ways; a process started without exec.
		"cmd/fx.viaChain -> gitutil.RunContext(Background) x1",
		"cmd/fx.viaField -> gitutil.RunContext(Background) x2",
		"cmd/fx.viaIface -> cmd/fx.(runner).Do(Background) x1",
		// A context passed to a method expression, after its receiver, and to
		// a variadic context parameter; an exec.Cmd a package variable holds.
		"cmd/fx.viaMethodExpr -> cmd/fx.(gitRunner).Do(Background) x1",
		"cmd/fx.viaVariadic -> cmd/fx.withAll(Background) x1",
		"cmd/fx.heldCmd -> exec.Cmd x1",
		"cmd/fx.viaProcess -> os.StartProcess x1",
		"cmd/fx.viaProcess -> syscall.Exec x1",
		"cmd/fx.viaProcess -> syscall.ForkExec x1",
		"cmd/fx.viaProcess -> syscall.StartProcess x1",
		// A field another package's code passes to git, set in this one; and a
		// deadline-free context passed to any context parameter, a wrapper's
		// and a constructor's, whether or not that callee reaches git.
		"cmd/fx.viaParam -> cmd/fx.newT(Background) x1",
		"cmd/fx.viaParam -> cmd/fx.withLog(Background) x1",
		"internal/loop.(X).Run -> gitutil.RunContext(Background) x1",
	}
	// Each system is counted on its own: osCall makes two calls on darwin and
	// one on linux, so a rise on linux is not hidden by darwin's count.
	perOS := map[string][]string{"darwin": {"cmd/fx.osCall -> gitutil.Run x2"}, "linux": {"cmd/fx.osCall -> gitutil.Run x1"}}
	var want []string
	for _, goos := range hookGitAuditGOOS {
		for _, l := range append(slices.Clone(common), perOS[goos]...) {
			want = append(want, goos+" "+l)
		}
	}
	sort.Strings(want)
	if g := got.lines(); !slices.Equal(g, want) {
		t.Fatalf("reached:\n  %s\nwant:\n  %s", strings.Join(g, "\n  "), strings.Join(want, "\n  "))
	}
	// A handler in a form the walk cannot root (a variable) is refused, so a
	// verb's path is never lost silently.
	spec.entries = []hookEntry{{"hook var", "newVar", "var"}}
	if _, err := hookGitAuditAll(dir, "./cmd/fx", spec); err == nil || !strings.Contains(err.Error(), "newVar") {
		t.Fatalf("a variable handler was rooted without a refusal (err = %v)", err)
	}
}

// TestHookGitBaselineRefusesNewRisingAndStale: a call the baseline lacks or
// counts lower is new; one it counts higher or nothing reaches is stale.
func TestHookGitBaselineRefusesNewRisingAndStale(t *testing.T) {
	reached := hookCounts{"a.f -> gitutil.Run": 2, "b.g -> exec.Command": 1, "d.k -> gitutil.Toplevel": 1}
	base := hookCounts{"a.f -> gitutil.Run": 1, "c.h -> localtier.Untracked": 1, "d.k -> gitutil.Toplevel": 3}
	added, stale := diffHookGitBaseline(reached, base)
	if want := []string{"a.f -> gitutil.Run x2 (baseline x1)", "b.g -> exec.Command x1 (baseline x0)"}; !slices.Equal(added, want) {
		t.Errorf("added = %q, want %q", added, want)
	}
	if want := []string{"c.h -> localtier.Untracked x0 (baseline x1)", "d.k -> gitutil.Toplevel x1 (baseline x3)"}; !slices.Equal(stale, want) {
		t.Errorf("stale = %q, want %q", stale, want)
	}
	if a, s := diffHookGitBaseline(reached, reached); len(a)+len(s) != 0 {
		t.Errorf("an exact baseline reads as %q new and %q stale", a, s)
	}
	if _, err := updatedBaseline(base, reached); err == nil {
		t.Error("the update lowered a baseline while a call was new or rising")
	}
	next, err := updatedBaseline(hookCounts{"a.f -> gitutil.Run": 3, "c.h -> localtier.Untracked": 1}, hookCounts{"a.f -> gitutil.Run": 1})
	if err != nil || !slices.Equal(next.lines(), []string{"a.f -> gitutil.Run x1"}) {
		t.Errorf("update = %q, %v; want the one entry lowered to x1 and the other removed", next.lines(), err)
	}
}

// TestHookGitBaselineFileIsSortedAndUnique: the reader refuses a baseline out
// of order, with a repeat, or with an entry not shaped with a count.
func TestHookGitBaselineFileIsSortedAndUnique(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"unsorted":  "# h\nb.g -> exec.Command x1\na.f -> gitutil.Run x1\n",
		"repeated":  "a.f -> gitutil.Run x1\na.f -> gitutil.Run x2\n",
		"shapeless": "a.f gitutil.Run x1\n",
		"uncounted": "a.f -> gitutil.Run\n",
		"zero":      "a.f -> gitutil.Run x0\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readHookGitBaseline(p); err == nil {
			t.Errorf("%s baseline read without error", name)
		}
	}
	p := filepath.Join(dir, "good")
	if err := os.WriteFile(p, []byte("# h\ndarwin a.f -> gitutil.Run x3\nlinux a.f -> gitutil.Run x1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c, err := readHookGitBaseline(p); err != nil || c["linux a.f -> gitutil.Run"] != 1 || c["darwin a.f -> gitutil.Run"] != 3 {
		t.Errorf("a per-system baseline read as %v, %v", c, err)
	}
	p = filepath.Join(dir, "nosystem")
	if err := os.WriteFile(p, []byte("plan9 a.f -> gitutil.Run x1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readHookGitBaseline(p); err == nil {
		t.Error("an entry for a system the audit does not walk read without error")
	}
}

// TestHookEntriesAreTheScriptsVerbs: the audit's roots are exactly the verbs
// the hook scripts hooks.json names run, so a new hook verb cannot go
// unwalked and a dropped one cannot linger.
func TestHookEntriesAreTheScriptsVerbs(t *testing.T) {
	verbs, err := hookVerbsInScripts(moduleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := hookEntriesMatch(verbs); err != nil {
		t.Fatal(err)
	}
	if err := hookEntriesMatch(append(slices.Clone(verbs), "hook planted")); err == nil {
		t.Error("a verb the scripts run that hookEntries lacks was not refused")
	}
	// A call of the binary the verb pattern does not classify is refused,
	// not passed over.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"hooks/hooks.json": `{"hooks":{"Stop":[{"hooks":[{"command":"sh \"${CLAUDE_PLUGIN_ROOT}/hooks/stop.sh\""}]}]}}`,
		"hooks/stop.sh":    "case \"$x\" in /*) ;; esac\n[ -x \"$r/abcd\" ] && \"$g\" hook stop; \"$g\" ahoy install\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if v, err := hookVerbsInScripts(dir); err == nil || !strings.Contains(err.Error(), "ahoy install") {
		t.Errorf("an unclassified call of the binary was passed over (verbs %q, err %v)", v, err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			t.Fatalf("no go.mod above %s", dir)
		}
	}
}

// auditFixture is a module of its own: a fake gitutil and localtier, and a
// command package whose hook verbs reach deadline-free calls many ways.
var auditFixture = map[string]string{
	"go.mod": "module example.com/fx\n\ngo 1.22\n",
	"internal/gitutil/gitutil.go": `package gitutil

import (
	"context"
	"os/exec"
)

func init() { _ = exec.Command("true") }

func Run(args ...string) string { return RunContext(context.Background(), args...) }

func RunContext(ctx context.Context, args ...string) string {
	_ = exec.CommandContext(ctx, "git")
	return ""
}

func Plain() string { return "" }

func Wrapped() string { return Run("x") }

type Repo struct{}

func (Repo) Status() string { return Repo{}.StatusContext(context.Background()) }

func (Repo) StatusContext(ctx context.Context) string { return RunContext(ctx) }
`,
	"internal/localtier/localtier.go": `package localtier

import "context"

func Untracked(p string) bool { return UntrackedContext(context.Background(), p) }

func UntrackedContext(ctx context.Context, p string) bool { return ctx.Err() == nil && p != "" }
`,
	"internal/loop/loop.go": `package loop

import (
	"context"

	"example.com/fx/internal/gitutil"
)

type X struct{ Context context.Context }

func (x X) Run() { _ = gitutil.RunContext(x.Context) }
`,
	"cmd/fx/os_darwin.go": "package main\n\nimport \"example.com/fx/internal/gitutil\"\n\nfunc osCall() { _ = gitutil.Run(\"a\"); _ = gitutil.Run(\"b\") }\n",
	"cmd/fx/os_linux.go":  "package main\n\nimport \"example.com/fx/internal/gitutil\"\n\nfunc osCall() { _ = gitutil.Run(\"a\") }\n",
	"cmd/fx/main.go": `package main

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"example.com/fx/internal/gitutil"
	"example.com/fx/internal/localtier"
	"example.com/fx/internal/loop"
)

type Command struct {
	Use  string
	Args func() error
	RunE func() error
}

type reader interface{ Read() string }

type disk struct{}

func (*disk) Read() string { _ = exec.Command("git"); return "" }

type quiet struct{}

func (quiet) Read() string { return gitutil.Plain() }

var load = loadPlain

func loadPlain() bool { return localtier.Untracked("x") }

func helper() string { return gitutil.Run("status") }

func viaLit() func() { return func() { _ = exec.Command("git") } }

func unreached() string { return gitutil.Run("log") }

func bg() context.Context { return context.TODO() }

func wrap(h func() error) func() error { return func() error { return h() } }

func wrapped() error { _ = gitutil.Run("w"); return nil }

var handlerVar = func() error { return nil }

func a() context.Context { return b() }

func b() context.Context { return context.Background() }

func viaChain() { _ = gitutil.RunContext(a()) }

type opts struct{ ctx context.Context }

func viaField() {
	o := opts{ctx: context.Background()}
	_ = gitutil.RunContext(o.ctx)
	var p opts
	p.ctx = context.TODO()
	_ = gitutil.RunContext(p.ctx)
}

type runner interface{ Do(ctx context.Context) }

type gitRunner struct{}

func (gitRunner) Do(ctx context.Context) { _ = gitutil.RunContext(ctx) }

func viaIface(r runner) { r.Do(context.Background()) }

func viaMethodExpr() { gitRunner.Do(gitRunner{}, context.Background()) }

func withAll(ctxs ...context.Context) {}

func viaVariadic() { withAll(context.TODO(), context.Background()) }

var heldCmd exec.Cmd

func viaHeld() { _ = heldCmd.Path }

func viaProcess() {
	_, _ = os.StartProcess("git", nil, nil)
	_ = syscall.Exec("git", nil, nil)
	_, _ = syscall.ForkExec("git", nil, nil)
	_, _, _ = syscall.StartProcess("git", nil, nil)
}

func viaPkg() { loop.X{Context: context.Background()}.Run() }

func withLog(ctx context.Context) context.Context { return ctx }

type T struct{ ctx context.Context }

func newT(ctx context.Context) *T { return &T{ctx: ctx} }

func (t *T) Run() { _ = gitutil.RunContext(t.ctx) }

func viaParam() {
	_ = gitutil.RunContext(withLog(context.Background()))
	newT(context.Background()).Run()
}

func tree() *Command {
	_ = gitutil.Run("tree")
	return &Command{
		Use:  "other",
		Args: func() error { _ = exec.Command("args"); return nil },
		RunE: func() error { unreached(); return nil },
	}
}

func newHook(r reader) *Command {
	return &Command{Use: "hook [flags]", RunE: func() error {
		_ = gitutil.Run("rev-parse")
		_ = gitutil.Run("status")
		_ = gitutil.RunContext(context.Background())
		_ = gitutil.RunContext(bg())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = gitutil.RunContext(ctx)
		_ = gitutil.Plain()
		_ = gitutil.Wrapped()
		_ = gitutil.Repo{}.Status()
		_ = &exec.Cmd{Path: "git"}
		var c exec.Cmd
		_ = c
		_ = new(exec.Cmd)
		helper()
		r.Read()
		load()
		viaLit()()
		viaChain()
		viaField()
		viaIface(gitRunner{})
		viaMethodExpr()
		viaVariadic()
		viaHeld()
		viaProcess()
		osCall()
		viaPkg()
		viaParam()
		return nil
	}}
}

func newWrapped() *Command { return &Command{Use: "wrapped", RunE: wrap(wrapped)} }

func newVar() *Command { return &Command{Use: "var", RunE: handlerVar} }

func main() { _ = newHook(&disk{}); _ = tree(); _ = newWrapped(); _ = newVar() }
`,
}

func writeAuditFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range auditFixture {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// lines renders the counts as sorted baseline lines.
func (c hookCounts) lines() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = fmt.Sprintf("%s x%d", k, c[k])
	}
	return out
}

// diffHookGitBaseline returns the calls reached more often than the baseline
// lists (a missing entry lists none) and the entries reached less often (a
// call no longer reached is reached none), each sorted, with both counts.
func diffHookGitBaseline(reached, base hookCounts) (added, stale []string) {
	for k, n := range reached {
		if n > base[k] {
			added = append(added, fmt.Sprintf("%s x%d (baseline x%d)", k, n, base[k]))
		}
	}
	for k, n := range base {
		if reached[k] < n {
			stale = append(stale, fmt.Sprintf("%s x%d (baseline x%d)", k, reached[k], n))
		}
	}
	sort.Strings(added)
	sort.Strings(stale)
	return added, stale
}

// updatedBaseline lowers each entry to the times it is reached and removes
// the ones no longer reached. It never adds or raises: while any call is new
// or rising it refuses and writes nothing.
func updatedBaseline(base, reached hookCounts) (hookCounts, error) {
	if added, _ := diffHookGitBaseline(reached, base); len(added) > 0 {
		return nil, fmt.Errorf("%d call(s) are new or rising (first: %s); the update only lowers", len(added), added[0])
	}
	next := hookCounts{}
	for k := range base {
		if n := reached[k]; n > 0 {
			next[k] = n
		}
	}
	return next, nil
}

// readHookGitBaseline reads the entries, skipping blank and # lines, and
// refuses one not shaped <caller> -> <callee> x<N> with N at least one, or
// out of order, or repeated.
func readHookGitBaseline(path string) (hookCounts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := hookCounts{}
	prev := ""
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		goos, tail, _ := strings.Cut(line, " ")
		caller, rest, ok := strings.Cut(tail, " -> ")
		ok = ok && slices.Contains(hookGitAuditGOOS, goos)
		f := strings.Fields(rest)
		var times int
		if ok && len(f) == 2 && strings.HasPrefix(f[1], "x") {
			times, err = strconv.Atoi(f[1][1:])
		}
		if !ok || caller == "" || strings.ContainsAny(caller, " \t") || len(f) != 2 || err != nil || times < 1 {
			return nil, fmt.Errorf("%s:%d: %q is not <goos> <caller> -> <callee> x<N>, goos one of %q, N at least 1", path, n, line, hookGitAuditGOOS)
		}
		key := goos + " " + caller + " -> " + f[0]
		if key <= prev {
			return nil, fmt.Errorf("%s:%d: %q is out of order or repeated; keep the file sorted, one entry per call", path, n, line)
		}
		prev = key
		out[key] = times
	}
	return out, sc.Err()
}

func writeHookGitBaseline(path string, c hookCounts) error {
	var b strings.Builder
	b.WriteString(hookGitBaselineHeader)
	for _, l := range c.lines() {
		b.WriteString(l + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

var (
	auditScriptRef  = regexp.MustCompile(`hooks/([a-z][a-z-]*\.sh)`)
	auditScriptVerb = regexp.MustCompile(`"\$(?:g|x|r/abcd)" (guard hook|hook [a-z][a-z-]*)`)
	// auditScriptCall is any word after a quoted binary variable; "case $x
	// in" and a test's closing bracket are not calls.
	auditScriptCall = regexp.MustCompile(`(case )?"\$(?:g|x|r/abcd)" ([A-Za-z-][^\s;|&)]*(?: [a-z][a-z-]*)?)`)
)

// hookVerbsInScripts is every abcd verb the scripts hooks/hooks.json names
// run, sorted.
func hookVerbsInScripts(root string) ([]string, error) {
	cfg, err := os.ReadFile(filepath.Join(root, "hooks", "hooks.json"))
	if err != nil {
		return nil, err
	}
	scripts := map[string]bool{}
	for _, m := range auditScriptRef.FindAllStringSubmatch(string(cfg), -1) {
		scripts[m[1]] = true
	}
	verbs := map[string]bool{}
	for s := range scripts {
		body, err := os.ReadFile(filepath.Join(root, "hooks", s))
		if err != nil {
			return nil, err
		}
		for _, m := range auditScriptVerb.FindAllStringSubmatch(string(body), -1) {
			verbs[m[1]] = true
		}
		for _, m := range auditScriptCall.FindAllStringSubmatch(string(body), -1) {
			if m[1] != "" || auditScriptVerb.MatchString(m[0]) {
				continue
			}
			return nil, fmt.Errorf("hooks/%s calls the binary as %q, which the audit cannot classify as a hook verb", s, m[2])
		}
	}
	if len(verbs) == 0 {
		return nil, errors.New("hooks/hooks.json names no script that runs an abcd hook verb")
	}
	out := make([]string, 0, len(verbs))
	for v := range verbs {
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

// hookEntriesMatch refuses verbs that differ from hookEntries'.
func hookEntriesMatch(verbs []string) error {
	var have []string
	for _, e := range hookEntries {
		have = append(have, e.verb)
	}
	sort.Strings(have)
	want := slices.Clone(verbs)
	sort.Strings(want)
	if !slices.Equal(have, want) {
		return fmt.Errorf("hookEntries roots %q but the hook scripts run %q", have, want)
	}
	return nil
}

// hookGitAuditGOOS are the systems walked; each is counted on its own, its
// entries keyed with its name.
var hookGitAuditGOOS = []string{"darwin", "linux"}

func hookGitAuditAll(root, mainPkg string, spec auditSpec) (hookCounts, error) {
	out := hookCounts{}
	for _, goos := range hookGitAuditGOOS {
		c, err := hookGitAudit(root, mainPkg, spec, goos)
		if err != nil {
			return nil, fmt.Errorf("GOOS=%s: %w", goos, err)
		}
		for k, n := range c {
			out[goos+" "+k] = n
		}
	}
	return out, nil
}

// listedPkg is the part of go list's record the audit reads.
type listedPkg struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Module     *struct {
		Path string
		Main bool
	}
	ImportMap map[string]string
}

// goListDeps names mainPkg's packages for goos, each after its dependencies,
// through the test's own toolchain, without cgo (the walk needs types, not a
// build) and without the network.
func goListDeps(root, mainPkg, goos string) ([]listedPkg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	goBin, err := testToolchainGo(ctx)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, goBin, "list", "-deps", "-json=ImportPath,Dir,GoFiles,Module,ImportMap", mainPkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOPROXY=off", "GOTOOLCHAIN=local", "GOFLAGS=", "GOOS="+goos, "GOARCH="+runtime.GOARCH)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	var pkgs []listedPkg
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var p listedPkg
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			return pkgs, nil
		} else if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
}

// testToolchainGo is the go command of the toolchain that built this test.
// A -trimpath build records no GOROOT, so the go on PATH is asked for its
// root and version, under the inherited GOTOOLCHAIN so a pinned switch
// applies, and refused unless the version is this test's own. go list itself
// then runs that toolchain's go with GOTOOLCHAIN=local.
func testToolchainGo(ctx context.Context) (string, error) {
	goroot := runtime.GOROOT()
	if goroot == "" {
		path, err := exec.LookPath("go")
		if err != nil {
			return "", fmt.Errorf("the test's own toolchain is not named (runtime.GOROOT is empty) and no go is on PATH to ask: %v", err)
		}
		cmd := exec.CommandContext(ctx, path, "env", "GOROOT", "GOVERSION")
		cmd.Env = append(os.Environ(), "GOFLAGS=") // the inherited GOTOOLCHAIN picks the pinned one
		out, err := cmd.Output()
		f := strings.Fields(string(out))
		if err != nil || len(f) != 2 {
			return "", fmt.Errorf("go env GOROOT GOVERSION: %v %q", err, out)
		}
		if f[1] != runtime.Version() {
			return "", fmt.Errorf("the go on PATH is %s, not this test's %s; the audit uses no other toolchain", f[1], runtime.Version())
		}
		goroot = f[0]
	}
	goBin := filepath.Join(goroot, "bin", "go")
	if !fileIsThere(goBin) {
		return "", fmt.Errorf("the test's own toolchain has no go command at %s", goBin)
	}
	return goBin, nil
}

func fileIsThere(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// hookGitAudit type-checks mainPkg's packages for goos from source (bodies
// only for the main module's), builds the call graph and returns the counts
// of the flagged calls the spec's roots reach.
func hookGitAudit(root, mainPkg string, spec auditSpec, goos string) (hookCounts, error) {
	listed, err := goListDeps(root, mainPkg, goos)
	if err != nil {
		return nil, err
	}
	g := &callGraph{
		cmdPkg: spec.cmdPkg,
		edges:  map[any][]any{}, flags: map[any][]string{}, ifaceUses: map[any][]*types.Func{},
		bgCalls: map[any][]*types.Func{},
		names:   map[any]string{}, pkgs: map[string]*pkgDecls{}, funcs: map[string]*types.Func{},
		methodSets: map[types.Type]*types.MethodSet{}, resolved: map[*types.Func][]any{},
		bgFuncs: map[*types.Func]bool{}, bgVars: map[*types.Var]bool{},
	}
	for _, lp := range listed {
		if lp.Module != nil && lp.Module.Main {
			g.mod = lp.Module.Path
			break
		}
	}
	if g.mod == "" {
		return nil, errors.New("go list named no package of the main module")
	}
	type modPkg struct {
		pkg   *types.Package
		files []*ast.File
		info  *types.Info
	}
	var mods []modPkg
	fset := token.NewFileSet()
	checked := map[string]*types.Package{"unsafe": types.Unsafe}
	sizes := types.SizesFor("gc", runtime.GOARCH)
	for _, lp := range listed {
		if lp.ImportPath == "unsafe" {
			continue
		}
		mine := lp.Module != nil && lp.Module.Main
		files := make([]*ast.File, 0, len(lp.GoFiles))
		for _, name := range lp.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		var errs []error
		conf := types.Config{
			Importer: importerFunc(func(path string) (*types.Package, error) {
				if m, ok := lp.ImportMap[path]; ok {
					path = m
				}
				if p := checked[path]; p != nil {
					return p, nil
				}
				return nil, fmt.Errorf("%s imports %s, which go list did not name before it", lp.ImportPath, path)
			}),
			IgnoreFuncBodies: !mine,
			Sizes:            sizes,
			Error: func(err error) {
				if len(errs) < 5 {
					errs = append(errs, err)
				}
			},
		}
		var info *types.Info
		if mine {
			info = &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
		}
		pkg, _ := conf.Check(lp.ImportPath, fset, files, info)
		if len(errs) > 0 {
			return nil, fmt.Errorf("type-checking %s: %w", lp.ImportPath, errors.Join(errs...))
		}
		checked[lp.ImportPath] = pkg
		if mine {
			mods = append(mods, modPkg{pkg, files, info})
		}
	}
	// The deadline-free contexts are marked to one fixpoint over the whole
	// module before anything is walked: a field one package passes to git may
	// be set from Background in a package that imports it, and a function may
	// return one declared after it.
	for changed := true; changed; {
		changed = false
		for _, m := range mods {
			if g.markBgVars(m.files, m.info) {
				changed = true
			}
			if g.markBgFuncs(m.files, m.info) {
				changed = true
			}
		}
	}
	for _, m := range mods {
		g.addPackage(m.pkg, m.files, m.info)
	}
	return g.reach(spec)
}

// pkgDecls keeps what the entry finder needs of one module package.
type pkgDecls struct {
	funcs map[string]*ast.FuncDecl // top-level functions, not methods
	info  *types.Info
}

// callGraph's nodes are *types.Func (an origin), *ast.FuncLit, *ast.CallExpr
// (a handler built by a call) and *types.Var (a package-level variable).
type callGraph struct {
	mod, cmdPkg string
	edges       map[any][]any
	flags       map[any][]string      // flagged uses, one per use
	ifaceUses   map[any][]*types.Func // interface methods used
	bgCalls     map[any][]*types.Func // callees passed a deadline-free context
	names       map[any]string
	inits       []any
	named       []*types.Named
	pkgs        map[string]*pkgDecls
	funcs       map[string]*types.Func // declared functions by name, for roots
	methodSets  map[types.Type]*types.MethodSet
	resolved    map[*types.Func][]any
	bgFuncs     map[*types.Func]bool // module functions returning a deadline-free context
	bgVars      map[*types.Var]bool  // variables assigned one
}

func (g *callGraph) rel(path string) string {
	if path == g.mod {
		return "."
	}
	return strings.TrimPrefix(path, g.mod+"/")
}

func (g *callGraph) inModule(path string) bool {
	return path == g.mod || strings.HasPrefix(path, g.mod+"/")
}

// gitPkg says whether path is gitutil or localtier.
func (g *callGraph) gitPkg(path string) bool {
	return path == g.mod+"/internal/gitutil" || path == g.mod+"/internal/localtier"
}

// gitName says whether a node's name is in gitutil or localtier.
func (g *callGraph) gitName(name string) bool {
	return strings.HasPrefix(name, "internal/gitutil.") || strings.HasPrefix(name, "internal/localtier.")
}

func recvNamed(fn *types.Func) (*types.Named, string) {
	r := fn.Type().(*types.Signature).Recv()
	if r == nil {
		return nil, ""
	}
	t, star := r.Type(), ""
	if p, ok := t.(*types.Pointer); ok {
		t, star = p.Elem(), "*"
	}
	n, _ := t.(*types.Named)
	return n, star
}

// methodPart is "(T)." or "(*T)." for a method, "" for a function.
func methodPart(fn *types.Func) string {
	if fn.Type().(*types.Signature).Recv() == nil {
		return ""
	}
	n, star := recvNamed(fn)
	tn := fn.Type().(*types.Signature).Recv().Type().String()
	if n != nil {
		tn = n.Obj().Name()
	}
	return "(" + star + tn + ")."
}

func (g *callGraph) funcName(fn *types.Func) string {
	return strings.ReplaceAll(g.rel(fn.Pkg().Path())+"."+methodPart(fn)+fn.Name(), " ", "")
}

// calleeKey is gitutil.F, localtier.(T).M or exec.F for those packages, and
// the module-relative name for any other.
func (g *callGraph) calleeKey(fn *types.Func) string {
	if p := fn.Pkg().Path(); g.gitPkg(p) || p == "os/exec" {
		return fn.Pkg().Name() + "." + methodPart(fn) + fn.Name()
	}
	return g.funcName(fn)
}

func (g *callGraph) addPackage(pkg *types.Package, files []*ast.File, info *types.Info) {
	pd := &pkgDecls{funcs: map[string]*ast.FuncDecl{}, info: info}
	g.pkgs[g.rel(pkg.Path())] = pd
	for _, name := range pkg.Scope().Names() {
		if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && !tn.IsAlias() {
			if n, ok := tn.Type().(*types.Named); ok && !types.IsInterface(n) {
				g.named = append(g.named, n)
			}
		}
	}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				fn, ok := info.Defs[d.Name].(*types.Func)
				if !ok {
					continue
				}
				name := g.funcName(fn)
				g.names[fn] = name
				g.funcs[name] = fn
				if d.Recv == nil {
					pd.funcs[d.Name.Name] = d
					if d.Name.Name == "init" {
						g.inits = append(g.inits, fn)
					}
				}
				if d.Body != nil {
					g.walk(fn, d.Body, info, name)
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					vs, ok := s.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, id := range vs.Names {
						v, ok := info.Defs[id].(*types.Var)
						if !ok {
							continue
						}
						name := g.rel(pkg.Path()) + "." + id.Name
						g.names[v] = name
						if vs.Type != nil && g.isExecCmd(vs.Type, info) {
							g.flags[v] = append(g.flags[v], "exec.Cmd")
						}
						for _, e := range vs.Values {
							g.walk(v, e, info, name)
						}
					}
				}
			}
		}
	}
}

// baseName is a node name without its $lit or $run suffix.
func baseName(name string) string {
	if i := strings.Index(name, "$"); i >= 0 {
		return name[:i]
	}
	return name
}

// walk records n's uses under node. A literal is a node of its own, named
// for the declaration that holds it. A command literal's Run and RunE are
// walked as nodes of their own with no edge from node: a hook's are rooted,
// any other verb's are left unreached.
func (g *callGraph) walk(node any, n ast.Node, info *types.Info, name string) {
	ast.Inspect(n, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.FuncLit:
			lit := baseName(name) + "$lit"
			g.names[x] = lit
			g.edges[node] = append(g.edges[node], x)
			g.walk(x, x.Body, info, lit)
			return false
		case *ast.CompositeLit:
			if g.isExecCmd(x.Type, info) {
				g.flags[node] = append(g.flags[node], "exec.Cmd")
			}
			if !g.isCommandLit(x, info) {
				return true
			}
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok && g.isHandlerKey(kv, info) {
					g.walkHandler(kv.Value, info, name)
					continue
				}
				g.walk(node, el, info, name)
			}
			return false
		case *ast.ValueSpec:
			if x.Type != nil && g.isExecCmd(x.Type, info) {
				for range x.Names {
					g.flags[node] = append(g.flags[node], "exec.Cmd")
				}
			}
		case *ast.CallExpr:
			if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok && len(x.Args) == 1 {
				if b, ok := info.Uses[id].(*types.Builtin); ok && b.Name() == "new" && g.isExecCmd(x.Args[0], info) {
					g.flags[node] = append(g.flags[node], "exec.Cmd")
				}
			}
			g.checkContextArgs(node, x, info)
		case *ast.Ident:
			g.use(node, info.Uses[x])
		}
		return true
	})
}

// walkHandler walks a Run or RunE value as a node of its own: a literal as
// $lit, a call building one as $run. A named handler needs no walk.
func (g *callGraph) walkHandler(v ast.Expr, info *types.Info, name string) {
	switch v := ast.Unparen(v).(type) {
	case *ast.FuncLit:
		lit := baseName(name) + "$lit"
		g.names[v] = lit
		g.walk(v, v.Body, info, lit)
	case *ast.CallExpr:
		run := baseName(name) + "$run"
		g.names[v] = run
		g.walk(v, v, info, run)
	}
}

// isHandlerKey says whether kv sets the command struct's Run or RunE.
func (g *callGraph) isHandlerKey(kv *ast.KeyValueExpr, info *types.Info) bool {
	key, _ := kv.Key.(*ast.Ident)
	if key == nil || (key.Name != "Run" && key.Name != "RunE") {
		return false
	}
	f, ok := info.Uses[key].(*types.Var)
	return ok && f.IsField() && f.Pkg() != nil && f.Pkg().Path() == g.cmdPkg
}

func (g *callGraph) isCommandLit(cl *ast.CompositeLit, info *types.Info) bool {
	for _, el := range cl.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok && g.isHandlerKey(kv, info) {
			return true
		}
	}
	return false
}

func (g *callGraph) isExecCmd(t ast.Expr, info *types.Info) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	tn, ok := info.Uses[sel.Sel].(*types.TypeName)
	return ok && tn.Pkg() != nil && tn.Pkg().Path() == "os/exec" && tn.Name() == "Cmd"
}

func calleeOf(call *ast.CallExpr, info *types.Info) *types.Func {
	fun := ast.Unparen(call.Fun)
	switch f := fun.(type) {
	case *ast.IndexExpr:
		fun = f.X
	case *ast.IndexListExpr:
		fun = f.X
	}
	var id *ast.Ident
	switch f := fun.(type) {
	case *ast.Ident:
		id = f
	case *ast.SelectorExpr:
		id = f.Sel
	}
	if id == nil {
		return nil
	}
	fn, _ := info.Uses[id].(*types.Func)
	if fn == nil {
		return nil
	}
	return fn.Origin()
}

func isContextType(t types.Type) bool {
	n, ok := t.(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "context" && n.Obj().Name() == "Context"
}

// isBg says whether e is a context with no deadline.
func (g *callGraph) isBg(e ast.Expr, info *types.Info) bool {
	switch e := ast.Unparen(e).(type) {
	case *ast.CallExpr:
		fn := calleeOf(e, info)
		if fn == nil || fn.Pkg() == nil {
			return false
		}
		if fn.Pkg().Path() == "context" {
			switch fn.Name() {
			case "Background", "TODO":
				return true
			case "WithCancel", "WithCancelCause", "WithValue", "WithoutCancel":
				return len(e.Args) > 0 && g.isBg(e.Args[0], info)
			}
			return false
		}
		if n, _ := recvNamed(fn); n != nil && fn.Name() == "Context" && fn.Pkg().Path() == g.cmdPkg {
			return true // a command's Context is Background as run executes it
		}
		return g.bgFuncs[fn]
	case *ast.Ident:
		v, ok := info.Uses[e].(*types.Var)
		return ok && g.bgVars[v.Origin()]
	case *ast.SelectorExpr:
		v, ok := info.Uses[e.Sel].(*types.Var)
		return ok && v.IsField() && g.bgVars[v.Origin()]
	}
	return false
}

// markBgVars records each variable and struct field assigned or
// initialised a deadline-free context, saying whether it marked any anew.
func (g *callGraph) markBgVars(files []*ast.File, info *types.Info) bool {
	changed := false
	mark := func(lhs ast.Expr, rhs ast.Expr) {
		var id *ast.Ident
		switch l := lhs.(type) {
		case *ast.Ident:
			id = l
		case *ast.SelectorExpr:
			id = l.Sel
		}
		if id == nil || !g.isBg(rhs, info) {
			return
		}
		obj := info.Defs[id]
		if obj == nil {
			obj = info.Uses[id]
		}
		if v, ok := obj.(*types.Var); ok && !g.bgVars[v.Origin()] {
			g.bgVars[v.Origin()] = true
			changed = true
		}
	}
	pair := func(lhs []ast.Expr, rhs []ast.Expr) {
		switch {
		case len(lhs) == len(rhs):
			for i := range lhs {
				mark(lhs[i], rhs[i])
			}
		case len(rhs) == 1 && len(lhs) > 0:
			mark(lhs[0], rhs[0])
		}
	}
	for _, f := range files {
		ast.Inspect(f, func(x ast.Node) bool {
			switch x := x.(type) {
			case *ast.AssignStmt:
				pair(x.Lhs, x.Rhs)
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(x.Names))
				for i, n := range x.Names {
					lhs[i] = n
				}
				pair(lhs, x.Values)
			case *ast.KeyValueExpr:
				if f, ok := x.Key.(*ast.Ident); ok {
					if v, ok := info.Uses[f].(*types.Var); ok && v.IsField() {
						mark(f, x.Value)
					}
				}
			}
			return true
		})
	}
	return changed
}

// markBgFuncs records each module function that returns a deadline-free
// context from its own body, saying whether it marked any anew.
func (g *callGraph) markBgFuncs(files []*ast.File, info *types.Info) bool {
	changed := false
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fn, ok := info.Defs[fd.Name].(*types.Func)
			if !ok {
				continue
			}
			ast.Inspect(fd.Body, func(x ast.Node) bool {
				switch x := x.(type) {
				case *ast.FuncLit:
					return false
				case *ast.ReturnStmt:
					for _, r := range x.Results {
						if !g.bgFuncs[fn] && g.isBg(r, info) {
							g.bgFuncs[fn] = true
							changed = true
						}
					}
				}
				return true
			})
		}
	}
	return changed
}

// checkContextArgs records a call passing a deadline-free context in a
// context parameter.
func (g *callGraph) checkContextArgs(node any, call *ast.CallExpr, info *types.Info) {
	fn := calleeOf(call, info)
	if fn == nil || fn.Pkg() == nil {
		return
	}
	if p := fn.Pkg().Path(); p != "os/exec" && !g.inModule(p) {
		return
	}
	sig := fn.Type().(*types.Signature)
	params := sig.Params()
	// A method expression (T.M(recv, ...)) takes its receiver as its first
	// argument, so its parameters start one argument on.
	args := call.Args
	if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok && len(args) > 0 && sig.Recv() != nil && namesType(sel.X, info) {
		args = args[1:]
	}
	for i, a := range args {
		t := types.Type(nil)
		switch {
		case i < params.Len()-1 || i < params.Len() && !sig.Variadic():
			t = params.At(i).Type()
		case sig.Variadic() && params.Len() > 0 && call.Ellipsis == token.NoPos:
			// Each argument a variadic parameter takes is of its element type.
			t = params.At(params.Len() - 1).Type().(*types.Slice).Elem()
		}
		if t != nil && isContextType(t) && g.isBg(a, info) {
			g.bgCalls[node] = append(g.bgCalls[node], fn)
			return
		}
	}
}

func (g *callGraph) use(node any, obj types.Object) {
	switch o := obj.(type) {
	case *types.Func:
		o = o.Origin()
		if k := g.flagKey(o); k != "" {
			g.flags[node] = append(g.flags[node], k)
		}
		if r := o.Type().(*types.Signature).Recv(); r != nil && types.IsInterface(r.Type()) {
			g.ifaceUses[node] = append(g.ifaceUses[node], o)
			return
		}
		if o.Pkg() != nil && g.inModule(o.Pkg().Path()) {
			g.edges[node] = append(g.edges[node], o)
		}
	case *types.Var:
		if o.Pkg() != nil && g.inModule(o.Pkg().Path()) && o.Parent() == o.Pkg().Scope() {
			g.edges[node] = append(g.edges[node], o)
		}
	}
}

// flagKey names a context-free form the audit lists, or "" for any other:
// exec.Command, and a gitutil or localtier function or method with a
// <Name>Context sibling.
func (g *callGraph) flagKey(fn *types.Func) string {
	p := fn.Pkg()
	if p == nil {
		return ""
	}
	if fn.Type().(*types.Signature).Recv() == nil {
		switch p.Path() + "." + fn.Name() {
		case "os/exec.Command":
			return "exec.Command"
		case "os.StartProcess":
			return "os.StartProcess"
		case "syscall.Exec", "syscall.ForkExec", "syscall.StartProcess":
			return "syscall." + fn.Name()
		}
	}
	if p.Path() == "os/exec" {
		return ""
	}
	if !g.gitPkg(p.Path()) {
		return ""
	}
	if fn.Type().(*types.Signature).Recv() != nil {
		n, _ := recvNamed(fn)
		if n == nil {
			return ""
		}
		if obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(n), true, p, fn.Name()+"Context"); obj != nil {
			if _, ok := obj.(*types.Func); ok {
				return g.calleeKey(fn)
			}
		}
		return ""
	}
	if _, ok := p.Scope().Lookup(fn.Name() + "Context").(*types.Func); ok {
		return g.calleeKey(fn)
	}
	return ""
}

// namesType says whether x names a type, T or *T, plain or package-qualified:
// the receiver part of a method expression. An instantiated generic type
// (T[int].M) is not recognised.
func namesType(x ast.Expr, info *types.Info) bool {
	x = ast.Unparen(x)
	if star, ok := x.(*ast.StarExpr); ok {
		x = ast.Unparen(star.X)
	}
	var id *ast.Ident
	switch e := x.(type) {
	case *ast.Ident:
		id = e
	case *ast.SelectorExpr:
		id = e.Sel
	}
	if id == nil {
		return false
	}
	_, ok := info.Uses[id].(*types.TypeName)
	return ok
}

// takesContext says whether fn has a context parameter.
func takesContext(fn *types.Func) bool {
	params := fn.Type().(*types.Signature).Params()
	for i := 0; i < params.Len(); i++ {
		if isContextType(params.At(i).Type()) {
			return true
		}
	}
	return false
}

// implementations is m on every module type, or its pointer, that implements
// m's interface; a generic type is matched on the method's name alone.
func (g *callGraph) implementations(m *types.Func) []any {
	if out, ok := g.resolved[m]; ok {
		return out
	}
	iface, _ := m.Type().(*types.Signature).Recv().Type().Underlying().(*types.Interface)
	var out []any
	for _, n := range g.named {
		for _, t := range []types.Type{n, types.NewPointer(n)} {
			ms := g.methodSets[t]
			if ms == nil {
				ms = types.NewMethodSet(t)
				g.methodSets[t] = ms
			}
			sel := ms.Lookup(m.Pkg(), m.Name())
			if sel == nil {
				continue
			}
			if n.TypeParams().Len() == 0 && iface != nil && !types.Implements(t, iface) {
				continue
			}
			out = append(out, sel.Obj().(*types.Func).Origin())
			break
		}
	}
	g.resolved[m] = out
	return out
}

// successors are a node's edges and its interface calls' implementations.
func (g *callGraph) successors(n any) []any {
	out := g.edges[n]
	for _, m := range g.ifaceUses[n] {
		out = append(out[:len(out):len(out)], g.implementations(m)...)
	}
	return out
}

// taintedGitNodes are the nodes of gitutil and localtier that make a flagged
// call or reach, within those packages, one that does.
func (g *callGraph) taintedGitNodes() map[any]bool {
	taint := map[any]bool{}
	for changed := true; changed; {
		changed = false
		for n, name := range g.names {
			if taint[n] || !g.gitName(name) {
				continue
			}
			hit := len(g.flags[n]) > 0 || len(g.bgCalls[n]) > 0
			for _, e := range g.edges[n] {
				if taint[e] {
					hit = true
					break
				}
			}
			if hit {
				taint[n] = true
				changed = true
			}
		}
	}
	return taint
}

// reach walks from the spec's roots and counts the flagged calls reached.
func (g *callGraph) reach(spec auditSpec) (hookCounts, error) {
	pd := g.pkgs[spec.entryPkg]
	if pd == nil {
		return nil, fmt.Errorf("the entry package %s was not loaded", spec.entryPkg)
	}
	roots := append([]any(nil), g.inits...)
	for _, name := range spec.roots {
		fn := g.funcs[name]
		if fn == nil {
			return nil, fmt.Errorf("the root %s is not declared", name)
		}
		roots = append(roots, fn)
	}
	for _, e := range spec.entries {
		got, err := g.entryRoots(pd, e)
		if err != nil {
			return nil, err
		}
		roots = append(roots, got...)
	}
	seen := map[any]bool{}
	for queue := roots; len(queue) > 0; {
		n := queue[0]
		queue = queue[1:]
		if seen[n] {
			continue
		}
		seen[n] = true
		queue = append(queue, g.successors(n)...)
	}
	tainted := g.taintedGitNodes()
	out := hookCounts{}
	for n := range seen {
		name := g.names[n]
		for _, k := range g.flags[n] {
			out[name+" -> "+k]++
		}
		// Any deadline-free context passed to a module function's or
		// method's context parameter, or to exec.CommandContext, is counted
		// whether or not that callee reaches git: a wrapper or a constructor
		// may carry it on, and the over-count is lowered only by passing a
		// real deadline.
		for _, fn := range g.bgCalls[n] {
			out[name+" -> "+g.calleeKey(fn)+"(Background)"]++
		}
		// A call from outside gitutil and localtier of one of theirs that
		// takes no context and has no Context sibling, but reaches a flagged
		// call, is counted at the caller, so a new caller of such a wrapper
		// is caught where it is made.
		if g.gitName(name) {
			continue
		}
		for _, e := range g.edges[n] {
			if fn, ok := e.(*types.Func); ok && tainted[fn] && g.flagKey(fn) == "" && !takesContext(fn) {
				out[name+" -> "+g.calleeKey(fn)]++
			}
		}
	}
	return out, nil
}

// entryRoots finds the command literal whose Use begins with e.use inside the
// function e.fn and returns its Run and RunE handlers. A handler must be a
// literal, a call (walked as a root of its own) or a module function; any
// other form, or no handler, is refused, so a verb's path is never dropped.
func (g *callGraph) entryRoots(pd *pkgDecls, e hookEntry) ([]any, error) {
	d := pd.funcs[e.fn]
	if d == nil {
		return nil, fmt.Errorf("hook entry %s: the function is not declared", e.fn)
	}
	var roots []any
	var bad error
	found := false
	ast.Inspect(d, func(x ast.Node) bool {
		cl, ok := x.(*ast.CompositeLit)
		if !ok || found || !useIs(cl, e.use) {
			return true
		}
		found = true
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok || !g.isHandlerKey(kv, pd.info) {
				continue
			}
			switch v := ast.Unparen(kv.Value).(type) {
			case *ast.FuncLit, *ast.CallExpr:
				roots = append(roots, v)
				continue
			case *ast.Ident:
				if fn, ok := pd.info.Uses[v].(*types.Func); ok && g.inModule(fn.Pkg().Path()) {
					roots = append(roots, fn.Origin())
					continue
				}
			case *ast.SelectorExpr:
				if fn, ok := pd.info.Uses[v.Sel].(*types.Func); ok && g.inModule(fn.Pkg().Path()) {
					roots = append(roots, fn.Origin())
					continue
				}
			}
			bad = fmt.Errorf("hook entry %s %q: its %s handler is not a function literal, a call or a module function, so the walk cannot root it", e.fn, e.use, kv.Key.(*ast.Ident).Name)
		}
		return false
	})
	if bad != nil {
		return nil, bad
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("hook entry %s %q: no command literal with a Run or RunE handler", e.fn, e.use)
	}
	return roots, nil
}

func useIs(cl *ast.CompositeLit, use string) bool {
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, _ := kv.Key.(*ast.Ident); key == nil || key.Name != "Use" {
			continue
		}
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return false
		}
		f := strings.Fields(s)
		return len(f) > 0 && f[0] == use
	}
	return false
}
