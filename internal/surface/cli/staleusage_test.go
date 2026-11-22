package cli

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdrepo"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// staleusage_test.go — the unknown-command / unknown-flag path on a binary that
// can prove it is stale (iss-2608230943088357).
//
// A plugin update lands a newer command surface (commands/*.md) beside a binary
// the bootstrap has not re-provisioned, so the page tells the reader to run a
// verb the binary predates. Cobra answers `unknown command "update"` (or
// `unknown flag: --yes`), which reads as a malformed invocation rather than a
// stale install. These tests pin the named refusal: the cobra line stays
// byte-for-byte, one sentence follows naming what the disk proves and the way
// out, and nothing is added when the disk proves nothing.

const pluginUpdateRemedy = "take a plugin update in the host (e.g. /plugin update abcd)"

// stalePluginRoot builds a plugin root whose command surface can be made newer
// than this binary: the layout resolvePluginRoot validates (hooks/), a commands/
// surface, and a binary file for the executable seam to point at. HOME is
// isolated so the machine's own path-entry record cannot supply a real root,
// the data dir is cleared so no cache meta answers, and cwd is a bare temp dir
// so the vintage comparison has no checkout to read.
func stalePluginRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"hooks", "commands"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Chdir(t.TempDir())
	return root
}

func writeCommandPage(t *testing.T, root, verb, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "commands", verb+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setExecutable points the running-binary seam at path, so the tests can place
// the binary inside or outside the plugin root without re-execing anything.
func setExecutable(t *testing.T, path string) {
	t.Helper()
	prev := osExecutable
	osExecutable = func() (string, error) { return path, nil }
	t.Cleanup(func() { osExecutable = prev })
}

// runMain drives the same entry point cmd/abcd does, so the rendered stderr —
// not just the error value — is what is asserted.
// unknownCommandLine is the line Run prints for an unknown sub-command of the
// command at path: the command named, the refused argument named by its
// position, and the sub-commands listed from the tree Run executes (cobra adds
// its help and completion verbs inside Execute; help is never listed).
func unknownCommandLine(t *testing.T, path ...string) string {
	t.Helper()
	root := NewRootCommand()
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	cmd := findByPath(root, path)
	if cmd == nil {
		t.Fatalf("no command at %v", path)
	}
	var names []string
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() {
			names = append(names, sub.Name())
		}
	}
	return "unknown command for \"" + strings.Join(append([]string{"abcd"}, path...), " ") +
		"\" (the first argument, not repeated here; its sub-commands are " + strings.Join(names, ", ") + ")"
}

// The refusal-code line Run adds last (spc-2610301431108175) is cut from the
// stderr returned, after checking it is the last line, so these tests assert
// the refusal's own lines; refusal_envelope_test.go asserts that line.
func runMain(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb)
	return code, out.String(), withoutRefusalCodeLine(errb.String())
}

// withoutRefusalCodeLine cuts the refusal-code line from the end of a
// refusal's stderr, leaving everything before it as it was.
func withoutRefusalCodeLine(stderr string) string {
	body := strings.TrimSuffix(stderr, "\n")
	i := strings.LastIndex(body, "\n")
	last := body[i+1:]
	if strings.HasPrefix(last, "abcd: ") && refusalCodeLineRe.MatchString(last) {
		return body[:i+1]
	}
	return stderr
}

// refusalCodeLineRe is the refusal-code line's ending: the call that explains
// the code.
var refusalCodeLineRe = regexp.MustCompile(`\(abcd explain [a-z0-9-]+/[a-z0-9-]+\)$`)

func TestUnknownVerbDocumentedByPluginSurfaceNamesStaleBinary(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "# `/abcd:frobnicate`\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" frobnicate --yes --json\n```\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	t.Run("bare verb", func(t *testing.T) {
		code, stdout, stderr := runMain(t, "frobnicate")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2 (cobra's usage-error code is unchanged)", code)
		}
		if stdout != "" {
			t.Fatalf("stdout must stay empty, got %q", stdout)
		}
		want := "abcd: " + unknownCommandLine(t) + "\n" +
			"abcd: this binary predates the `frobnicate` command its plugin surface documents — " + pluginUpdateRemedy + "\n"
		if stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
	})

	t.Run("verb with a flag the binary cannot parse either", func(t *testing.T) {
		// The documented invocation carries flags; cobra parses them BEFORE it
		// validates the positional, so the failure is `unknown flag: --yes` and
		// the verb never appears in cobra's own line. The note must still name
		// the verb, which is what the surface documents and the binary lacks.
		code, _, stderr := runMain(t, "frobnicate", "--yes")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
		want := "abcd: unknown flag: --yes\n" +
			"abcd: this binary predates the `frobnicate` command its plugin surface documents — " + pluginUpdateRemedy + "\n"
		if stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
		// The documented invocation also carries --json, which the parse never
		// reached; the caller asked for the envelope all the same, and it
		// carries the note (iss-2609292352131344).
		code, stdout, stderr := runMain(t, "frobnicate", "--yes", "--json")
		if code != 2 || stderr != "" {
			t.Fatalf("under --json: exit %d, stderr %q; want 2 and nothing on stderr", code, stderr)
		}
		var env struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout is not the JSON error envelope: %v\n%s", err, stdout)
		}
		if !strings.HasPrefix(env.Error, "unknown flag: --yes") || !strings.Contains(env.Error, "predates the `frobnicate` command") {
			t.Fatalf("envelope error = %q", env.Error)
		}
	})

	t.Run("json envelope carries the note", func(t *testing.T) {
		code, stdout, _ := runMain(t, "--json", "frobnicate")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
		var env struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout is not the JSON error envelope: %v\n%s", err, stdout)
		}
		if !strings.HasPrefix(env.Error, unknownCommandLine(t)) ||
			!strings.Contains(env.Error, "predates the `frobnicate` command") ||
			!strings.Contains(env.Error, pluginUpdateRemedy) {
			t.Fatalf("envelope error = %q", env.Error)
		}
	})
}

func TestUnknownSubverbDocumentedByParentPageNamesStaleBinary(t *testing.T) {
	root := stalePluginRoot(t)
	// The parent page documents the sub-verb as an invocation, the way every
	// command page hands one over.
	writeCommandPage(t, root, "docs", "# `/abcd:docs`\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" docs frobnicate --json\n```\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	code, _, stderr := runMain(t, "docs", "frobnicate")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	want := "abcd: " + unknownCommandLine(t, "docs") + "\n" +
		"abcd: this binary predates the `docs frobnicate` command its plugin surface documents — " + pluginUpdateRemedy + "\n"
	if stderr != want {
		t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
	}
}

func TestUnknownFlagDocumentedByPageNamesStaleBinary(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "lint", "# `/abcd:lint`\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" lint --frob\n```\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	code, _, stderr := runMain(t, "lint", "--frob")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	want := "abcd: unknown flag: --frob\n" +
		"abcd: this binary predates the `--frob` flag of `abcd lint` its plugin surface documents — " + pluginUpdateRemedy + "\n"
	if stderr != want {
		t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
	}
}

// The remedy follows where the binary sits: a source checkout is rebuilt, a
// plugin-root binary is replaced by a plugin update, and a PATH copy that was
// provisioned from a now-newer surface takes `abcd update`.
func TestStaleUsageRemedyFollowsWhereTheBinarySits(t *testing.T) {
	t.Run("source checkout rebuilds", func(t *testing.T) {
		root := stalePluginRoot(t)
		writeCommandPage(t, root, "frobnicate", "# frobnicate\n")
		if err := os.MkdirAll(filepath.Join(root, "cmd", "abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "cmd", "abcd", "main.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// The dogfood layout: repo-root `abcd` is a link into bin/, so the binary
		// sits BELOW the root, not beside its hooks/.
		if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		setExecutable(t, filepath.Join(root, "bin", "abcd-test"))

		_, _, stderr := runMain(t, "frobnicate")
		want := "abcd: this binary predates the `frobnicate` command its command surface documents — the plugin root is a source checkout, so rebuild it with `make build`\n"
		if !strings.HasSuffix(stderr, want) {
			t.Fatalf("stderr =\n%q\nwant suffix\n%q", stderr, want)
		}
		if strings.Contains(stderr, pluginUpdateRemedy) {
			t.Fatalf("a source checkout must not be told to take a plugin update:\n%s", stderr)
		}
	})

	t.Run("PATH copy updates", func(t *testing.T) {
		root := stalePluginRoot(t)
		writeCommandPage(t, root, "frobnicate", "# frobnicate\n")
		setExecutable(t, filepath.Join(t.TempDir(), "abcd"))
		// A copy `abcd update` provisioned is a release binary, stamped with its
		// tag; an unstamped build is not told to update (iss-2610120921425348).
		prev := core.Version
		core.Version = "v1.2.3"
		t.Cleanup(func() { core.Version = prev })

		_, _, stderr := runMain(t, "frobnicate")
		want := "abcd: this binary predates the `frobnicate` command the plugin surface it was provisioned from documents — this PATH copy is stale; run `abcd update`\n"
		if !strings.HasSuffix(stderr, want) {
			t.Fatalf("stderr =\n%q\nwant suffix\n%q", stderr, want)
		}
	})
}

// Without disk evidence the refusal line is all there is: a typo, a verb no
// page documents, and a flag no page documents each print it alone.
func TestUnknownVerbWithoutSurfaceEvidenceStaysVerbatim(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "lint", "# `/abcd:lint`\n\nRun `\"${CLAUDE_PLUGIN_ROOT}/abcd\" lint`.\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"frobnicate"}, "abcd: " + unknownCommandLine(t) + "\n"},
		{[]string{"lnit"}, "abcd: " + unknownCommandLine(t) + "\n"},
		{[]string{"lint", "--frob"}, "abcd: unknown flag: --frob\n"},
		{[]string{"docs", "frobnicate"}, "abcd: " + unknownCommandLine(t, "docs") + "\n"},
	} {
		code, _, stderr := runMain(t, tc.args...)
		if code != 2 {
			t.Fatalf("%v: exit code = %d, want 2", tc.args, code)
		}
		if stderr != tc.want {
			t.Fatalf("%v: stderr =\n%q\nwant\n%q", tc.args, stderr, tc.want)
		}
	}
}

// A verb token is joined into a path under commands/, so only a verb-shaped
// token may reach the filesystem at all.
func TestStaleUsageNoteRefusesPathShapedTokens(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "# frobnicate\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	for _, arg := range []string{"../commands/frobnicate", "commands/frobnicate", "Frobnicate", ".md"} {
		_, _, stderr := runMain(t, arg)
		if strings.Contains(stderr, "predates") {
			t.Fatalf("%q must never be looked up as a command page:\n%s", arg, stderr)
		}
	}
}

// The (b) path: no page documents the verb, but the binary can still prove it
// is stale from disk alone — behind its own source checkout tip, or differing
// from the release the plugin cache pinned.
func TestUnknownVerbOnStaleBinaryWithoutPageNamesVintage(t *testing.T) {
	t.Run("dogfood binary behind its checkout tip", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("ABCD_PLUGIN_ROOT", "")
		t.Setenv("CLAUDE_PLUGIN_ROOT", "")
		t.Setenv("CLAUDE_PLUGIN_DATA", "")
		setExecutable(t, filepath.Join(t.TempDir(), "abcd"))

		repo := t.TempDir()
		gitCmd(t, repo, "init", "-q")
		if err := os.WriteFile(filepath.Join(repo, "a"), []byte("1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitCmd(t, repo, "add", "a")
		gitCommit(t, repo, "commit", "-q", "-m", "one")
		built := strings.TrimSpace(gitCmd(t, repo, "rev-parse", "HEAD"))
		if err := os.WriteFile(filepath.Join(repo, "a"), []byte("2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitCmd(t, repo, "add", "a")
		gitCommit(t, repo, "commit", "-q", "-m", "two")
		tip := strings.TrimSpace(gitCmd(t, repo, "rev-parse", "HEAD"))
		t.Chdir(repo)
		t.Cleanup(ahoy.SetCurrentVintageForTest(func() vintage.Current {
			return vintage.Current{Revision: built, Known: true}
		}))

		code, _, stderr := runMain(t, "frobnicate")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
		want := "abcd: " + unknownCommandLine(t) + "\n" +
			"abcd: this binary was built from commit " + built[:12] + " but this checkout is at " + tip[:12] +
			" — it is behind its own source, so rebuild it with `make build`\n"
		if stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
	})

	t.Run("PATH copy differing from the pinned release", func(t *testing.T) {
		stalePluginRoot(t)
		data := t.TempDir()
		if err := os.MkdirAll(filepath.Join(data, "cache"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(data, "cache", "binary-meta"), []byte("release_tag=v9.9.9\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CLAUDE_PLUGIN_DATA", data)
		setExecutable(t, filepath.Join(t.TempDir(), "abcd"))
		prev := core.Version
		core.Version = "v1.2.3"
		t.Cleanup(func() { core.Version = prev })
		t.Cleanup(ahoy.SetCurrentVintageForTest(func() vintage.Current {
			return vintage.Current{Revision: "0123456789abcdef0123456789abcdef01234567", Known: true}
		}))

		_, _, stderr := runMain(t, "frobnicate")
		want := "abcd: this binary is v1.2.3 and the plugin cache pinned release v9.9.9 — the two differ, so this PATH copy may be stale; run `abcd update`\n"
		if !strings.HasSuffix(stderr, want) {
			t.Fatalf("stderr =\n%q\nwant suffix\n%q", stderr, want)
		}
	})

	t.Run("fresh binary stays silent", func(t *testing.T) {
		stalePluginRoot(t)
		setExecutable(t, filepath.Join(t.TempDir(), "abcd"))
		_, _, stderr := runMain(t, "frobnicate")
		if want := "abcd: " + unknownCommandLine(t) + "\n"; stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
	})
}

// A binary the release did not stamp (version "dev") is a local build — `go
// run` from a source checkout, a hand-run `go build` — never a copy `abcd
// update` provisioned, so a page documenting what it lacks is no evidence it is
// stale and updating would change nothing. The note says the thing is not in
// this build and advises nothing (iss-2610120921425348).
func TestDevBuildOutsidePluginRootIsNotCalledAStalePATHCopy(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "# frobnicate\n")
	writeCommandPage(t, root, "lint", "```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" lint --frob\n```\n")
	// Where `go run` leaves its executable: the build cache, outside every root.
	setExecutable(t, filepath.Join(t.TempDir(), "go-build", "9a", "9a5c-d", "abcd"))
	prev := core.Version
	core.Version = "dev"
	t.Cleanup(func() { core.Version = prev })

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"frobnicate"}, "abcd: the plugin surface documents the `frobnicate` command, which is not in this build — an unstamped `dev` build, not a release binary, so the surface is no evidence that it is stale\n"},
		{[]string{"lint", "--frob"}, "abcd: the plugin surface documents the `--frob` flag of `abcd lint`, which is not in this build — an unstamped `dev` build, not a release binary, so the surface is no evidence that it is stale\n"},
	} {
		_, _, stderr := runMain(t, tc.args...)
		if !strings.HasSuffix(stderr, tc.want) {
			t.Errorf("%v: stderr =\n%q\nwant suffix\n%q", tc.args, stderr, tc.want)
		}
		for _, not := range []string{"abcd update", "PATH copy", "make build", pluginUpdateRemedy} {
			if strings.Contains(stderr, not) {
				t.Errorf("%v: a dev build must not be advised %q:\n%s", tc.args, not, stderr)
			}
		}
	}
}

// The binary scripts/abcd.sh builds sits inside the source checkout, under its
// local tier, but it was relinked from the tree as it stands when the script
// ran it: a page documenting what it lacks is ahead of the code, not proof the
// binary is old, so it takes the dev build's note, never `make build`, which
// does not touch it (iss-2610241240133009).
func TestCheckoutBuildInTheLocalTierIsNotToldToRebuild(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "# frobnicate\n")
	if err := os.MkdirAll(filepath.Join(root, "cmd", "abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "abcd", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, filepath.FromSlash(abcdrepo.CheckoutBuildRel))
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	setExecutable(t, exe)
	prev := core.Version
	core.Version = "dev"
	t.Cleanup(func() { core.Version = prev })

	t.Run("started by the script", func(t *testing.T) {
		setCheckoutMark(t, exe)
		_, _, stderr := runMain(t, "frobnicate")
		want := "abcd: the plugin surface documents the `frobnicate` command, which is not in this build — an unstamped `dev` build, not a release binary, so the surface is no evidence that it is stale\n"
		if !strings.HasSuffix(stderr, want) {
			t.Fatalf("stderr =\n%q\nwant suffix\n%q", stderr, want)
		}
		for _, not := range []string{"make build", "abcd update", pluginUpdateRemedy} {
			if strings.Contains(stderr, not) {
				t.Errorf("scripts/abcd.sh's build must not be advised %q:\n%s", not, stderr)
			}
		}
	})
	// A copy left in the tier and run directly carries no mark: it is the tree
	// as it stood when the script last ran, so a page naming what it lacks may
	// well be proof it is old, and the way to a current one is the script.
	// Judged by its own checkout's pages, so a plain terminal, where no plugin
	// root resolves, says the same.
	for _, env := range []string{"plugin root named", "plain terminal"} {
		t.Run("left there and run directly, "+env, func(t *testing.T) {
			setCheckoutMark(t, "")
			if env == "plain terminal" {
				t.Setenv("ABCD_PLUGIN_ROOT", "")
			}
			_, _, stderr := runMain(t, "frobnicate")
			for _, want := range []string{"`frobnicate` command", "run directly", "`scripts/abcd.sh`"} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr lacks %q:\n%s", want, stderr)
				}
			}
			for _, not := range []string{"no evidence that it is stale", "make build", "abcd update", pluginUpdateRemedy} {
				if strings.Contains(stderr, not) {
					t.Errorf("a left copy must not be told %q:\n%s", not, stderr)
				}
			}
		})
	}
}

// setCheckoutMark says which executable scripts/abcd.sh started, as the binary
// read it from its environment: "" for a run the script did not start.
func setCheckoutMark(t *testing.T, path string) {
	t.Helper()
	prev := checkoutMark
	checkoutMark = func() string { return path }
	t.Cleanup(func() { checkoutMark = prev })
}

// A flag is documented for the sub-verb typed only when that sub-verb's own
// synopsis line in a fenced block carries it. Prose elsewhere on the page, and
// another sub-verb's synopsis, are not evidence: `build next --dry-run` was
// called documented because build.md mentions `/abcd:drain --dry-run` in prose,
// and `implement log --run` because `implement step` takes --run
// (iss-2610120921497084).
func TestFlagNoteIsScopedToTheTypedSubverbsSynopsis(t *testing.T) {
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	// Lines as commands/build.md and commands/implement.md carry them, plus one
	// flag per synopsis that no binary has, for the positive cases.
	writeCommandPage(t, root, "build", "# `/abcd:build`\n\n"+
		"```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" build <itd-N> [--session <id>] [--frob-top] --json\n```\n\n"+
		"read as `/abcd:drain --dry-run` reads it, and that no peer holds it.\n\n"+
		"```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" build next [--session <id>] [--max <n> | --until-empty] [--frob-next] --json\n```\n")
	writeCommandPage(t, root, "implement", "# `/abcd:implement`\n\n"+
		"```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement log <event> --session <id> --field key=value … --json\n```\n\n"+
		"```bash\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement step [--run <run-id>] [--frob-step] --json\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement receipt <path> [--run <run-id>] --json\n"+
		"```\n\n"+
		"```bash\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement outage [--json]\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement outage record --session <id> --what <text> [--frob-record] [--json]\n"+
		"\"${CLAUDE_PLUGIN_ROOT}/abcd\" implement outage probe --session <id> \\\n"+
		"    [--model ok|fail] [--frob-wrapped] [--json]\n"+
		"```\n")
	writeCommandPage(t, root, "abcd", "```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" --frob-root --json\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" mode --frob-mode\n```\n")

	t.Run("prose or another sub-verb's synopsis is not evidence", func(t *testing.T) {
		for _, args := range [][]string{
			{"build", "next", "--dry-run"},
			{"implement", "log", "agent_start", "--run", "r", "--session", "s", "--field", "agent=x"},
			{"build", "next", "--frob-top"},          // build's own synopsis, not build next's
			{"build", "--frob-next"},                 // build next's synopsis, not build's
			{"implement", "outage", "--frob-record"}, // depth two: outage record's, not outage's
			{"--frob-mode"},                          // a verb's synopsis on the dispatcher page
		} {
			_, _, stderr := runMain(t, args...)
			want := "abcd: unknown flag: " + flagIn(args) + "\n"
			if stderr != want {
				t.Errorf("%v: stderr =\n%q\nwant\n%q", args, stderr, want)
			}
		}
	})

	t.Run("the typed sub-verb's own synopsis is", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			what string
		}{
			{[]string{"build", "next", "--frob-next"}, "the `--frob-next` flag of `abcd build next`"},
			{[]string{"build", "--frob-top"}, "the `--frob-top` flag of `abcd build`"},
			{[]string{"implement", "step", "--frob-step"}, "the `--frob-step` flag of `abcd implement step`"},
			{[]string{"implement", "outage", "record", "--frob-record"}, "the `--frob-record` flag of `abcd implement outage record`"},
			// A synopsis wrapped with a trailing backslash is one invocation.
			{[]string{"implement", "outage", "probe", "--frob-wrapped"}, "the `--frob-wrapped` flag of `abcd implement outage probe`"},
			{[]string{"--frob-root"}, "the `--frob-root` flag of `abcd`"},
		} {
			_, _, stderr := runMain(t, tc.args...)
			want := "abcd: this binary predates " + tc.what + " its plugin surface documents — " + pluginUpdateRemedy + "\n"
			if !strings.HasSuffix(stderr, want) {
				t.Errorf("%v: stderr =\n%q\nwant suffix\n%q", tc.args, stderr, want)
			}
		}
	})
}

// flagIn is the first `--` token of args, the one pflag refuses.
func flagIn(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			return a
		}
	}
	return ""
}

// A command page is read whole on an error path (up to maxCommandPageBytes),
// so the synopsis scan must stay linear in it. One fenced line of repeated
// invocations once rescanned the rest of the line per invocation (about two
// minutes at the cap for a leaf command), and a run of backslash-wrapped lines
// recopied the joined text per line (0.7s). A linear scan's pages here usually
// reach past the cap on a quick machine, about 1 MiB, and are smaller on a
// slow one; the verdict does not need them to reach it.
//
// The test judges growth, never a time: an absolute bound of 250ms failed on
// a slow shared runner (333ms) with the scan linear (iss-2611121820123919).
// scanGrowth times a page and one four times larger; a linear scan takes
// about four times as long, a quadratic one about sixteen, and the bound sits
// between them. A scan that stays linear but becomes a constant factor slower
// is not what this test catches. Under -race it skips: nothing here is
// concurrent for the detector to watch, and the CI step that runs the
// plain-only guards by name is derived from this skip (internal/core/lint's
// plain-guard check).
//
// A ratio over the bound is measured once more from the start, and the test
// fails only when the second is over it too: a false failure holds an
// unattended merge queue, and a pass costs nothing more.
func TestSynopsisScanIsLinearInThePage(t *testing.T) {
	if raceEnabled {
		t.Skip("asserts only the scan's growth, which the plain lane holds; nothing here is concurrent for -race to watch")
	}
	fence := func(line string) string { return "```bash\n" + line + "\n```\n" }
	for _, tc := range []struct {
		name string
		body func(size int) string
		skew usageSkew
	}{
		{"one line of leaf invocations", func(n int) string { return fence(strings.Repeat("abcd lint ", n/10)) }, usageSkew{known: []string{"lint"}, flag: "--frob"}},
		{"one line of invocations with sub-verbs", func(n int) string { return fence(strings.Repeat("abcd build ", n/11)) }, usageSkew{known: []string{"build"}, flag: "--frob", subVerbs: true}},
		{"a run of wrapped lines", func(n int) string { return fence(strings.Repeat("x \\\n", n/4) + "abcd lint") }, usageSkew{known: []string{"lint"}, flag: "--frob"}},
	} {
		// The verdict is checked on every page the judge reads rather than on
		// one more at the cap: a quadratic scan of the cap takes minutes, and
		// the judge's own pages fail it in seconds.
		found := false
		build := func(n int) func() {
			body := tc.body(n)
			return func() { found = synopsisCarriesFlag(body, tc.skew) || found }
		}
		var over []growth
		for attempt := 1; attempt <= 2; attempt++ {
			g := scanGrowth(t, build)
			if !g.judged {
				// No scan of these bodies is this quick, so a page that
				// stopped reaching the scan fails here rather than passing
				// with nothing measured.
				t.Errorf("%s: the least read of %d KiB took %v, under the %v floor at the largest size; the scan was not measured",
					tc.name, g.small>>10, g.smallTook, scanGrowthFloor)
				break
			}
			t.Logf("%s, measurement %d: %d KiB in %v, %d KiB in %v: growth %.2f", tc.name, attempt, g.small>>10, g.smallTook, 4*g.small>>10, g.largeTook, g.ratio)
			if g.ratio <= scanGrowthBound {
				break
			}
			over = append(over, g)
		}
		if found {
			t.Errorf("%s: no line names --frob, yet it was found", tc.name)
		}
		if len(over) == 2 {
			a, b := over[0], over[1]
			t.Errorf("%s: a page four times larger took more than %d times the CPU to scan in two measurements (%d KiB against %d KiB: %v against %v, %.1f; %d KiB against %d KiB: %v against %v, %.1f); the scan is not linear in the page",
				tc.name, scanGrowthBound,
				4*a.small>>10, a.small>>10, a.largeTook, a.smallTook, a.ratio,
				4*b.small>>10, b.small>>10, b.largeTook, b.smallTook, b.ratio)
		}
	}
}

// scanGrowthBound is the most times longer a scan of four times the input may
// take. A linear scan reads about 4 and a quadratic one about 16; 8 is their
// geometric mean, so either side has a factor of two of room for noise before
// the verdict turns. A quadratic scan reads below 16 where a fixed cost of a
// call (compiling the patterns) is a large share of the smaller read;
// scanGrowthFloor makes that share small.
const scanGrowthBound = 8

// scanGrowthFloor is the least CPU time the smaller read must take, judged on
// the least of its readings, the one compared. Below it the clock's
// resolution and the fixed costs of a call are a large share of the reading,
// and the ratio says little, so scanGrowth grows both inputs instead of
// judging.
const scanGrowthFloor = 10 * time.Millisecond

// scanGrowthLimit is the largest smaller input scanGrowth reads.
const scanGrowthLimit = 64 << 20

// scanGrowthRounds is how many times each input is read; the least reading of
// each size is the one the judge compares.
const scanGrowthRounds = 5

// growth is one scanGrowth reading: the smaller input's size and the least CPU
// time a read of it and of the input four times larger took. judged is false
// when the smaller read stayed under scanGrowthFloor at scanGrowthLimit, and
// the ratio is then not a verdict.
type growth struct {
	small                int
	smallTook, largeTook time.Duration
	ratio                float64
	judged               bool
}

// scanGrowth finds an input size whose read takes at least scanGrowthFloor of
// CPU, doubling from 4 KiB, then reads it and one four times larger
// scanGrowthRounds times each, alternating, and compares the least time of
// each size. When that least smaller reading is under the floor, it doubles
// both sizes and reads them again, so the floor holds for the reading that is
// compared. build makes the input outside the timing and returns the read.
//
// Process CPU is the clock, so a core taken by another process is not
// charged; another goroutine of this binary or the collector is, and the
// least of several readings drops a read one of them ran beside. Alternating
// the sizes keeps a burst of such work from covering every read of one size
// and none of the other. Starting small keeps a quadratic scan's larger read
// short: it reaches the floor at a small size, where sixteen times the floor
// is still well under a second.
//
// A reading left unjudged at scanGrowthLimit cannot come from a scan slower
// than linear: even a linear scan would have to get through over 6 GiB per
// second of CPU to keep 64 MiB under 10ms, and a scan whose cost grows faster
// than the page reaches the floor at a smaller size than a linear one does.
// The caller fails on it: it means the read did no scanning.
func scanGrowth(t *testing.T, build func(size int) func()) growth {
	t.Helper()
	timed := func(read func()) time.Duration {
		runtime.GC()
		start := processCPU()
		read()
		return processCPU() - start
	}
	n := 4 << 10
	for {
		read := build(n)
		read() // a warm-up: the first read pays one-time costs
		took := timed(read)
		if took >= scanGrowthFloor {
			break
		}
		if n >= scanGrowthLimit {
			return growth{small: n, smallTook: took}
		}
		n *= 2
	}
	for {
		small, large := build(n), build(4*n)
		g := growth{small: n, smallTook: time.Duration(math.MaxInt64), largeTook: time.Duration(math.MaxInt64)}
		for range scanGrowthRounds {
			g.smallTook = min(g.smallTook, timed(small))
			g.largeTook = min(g.largeTook, timed(large))
		}
		g.ratio = float64(g.largeTook) / float64(max(g.smallTook, time.Nanosecond))
		if g.smallTook >= scanGrowthFloor {
			g.judged = true
			return g
		}
		if n >= scanGrowthLimit {
			return g
		}
		n *= 2
	}
}
