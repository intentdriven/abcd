package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
func runMain(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

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
		want := "abcd: unknown command \"frobnicate\" for \"abcd\"\n" +
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
		code, _, stderr := runMain(t, "frobnicate", "--yes", "--json")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
		want := "abcd: unknown flag: --yes\n" +
			"abcd: this binary predates the `frobnicate` command its plugin surface documents — " + pluginUpdateRemedy + "\n"
		if stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
	})

	t.Run("json envelope carries the note", func(t *testing.T) {
		code, _, stderr := runMain(t, "--json", "frobnicate")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
		var env struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(stderr), &env); err != nil {
			t.Fatalf("stderr is not the JSON error envelope: %v\n%s", err, stderr)
		}
		if !strings.HasPrefix(env.Error, "unknown command \"frobnicate\" for \"abcd\"") ||
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
	want := "abcd: unknown command \"frobnicate\" for \"abcd docs\"\n" +
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

		_, _, stderr := runMain(t, "frobnicate")
		want := "abcd: this binary predates the `frobnicate` command the plugin surface it was provisioned from documents — this PATH copy is stale; run `abcd update`\n"
		if !strings.HasSuffix(stderr, want) {
			t.Fatalf("stderr =\n%q\nwant suffix\n%q", stderr, want)
		}
	})
}

// Without disk evidence the cobra line is all there is: a typo, a verb no page
// documents, and a flag no page documents all stay byte-for-byte.
func TestUnknownVerbWithoutSurfaceEvidenceStaysVerbatim(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "lint", "# `/abcd:lint`\n\nRun `\"${CLAUDE_PLUGIN_ROOT}/abcd\" lint`.\n")
	setExecutable(t, filepath.Join(root, "abcd"))

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"frobnicate"}, "abcd: unknown command \"frobnicate\" for \"abcd\"\n"},
		{[]string{"lnit"}, "abcd: unknown command \"lnit\" for \"abcd\"\n"},
		{[]string{"lint", "--frob"}, "abcd: unknown flag: --frob\n"},
		{[]string{"docs", "frobnicate"}, "abcd: unknown command \"frobnicate\" for \"abcd docs\"\n"},
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
		want := "abcd: unknown command \"frobnicate\" for \"abcd\"\n" +
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
		if want := "abcd: unknown command \"frobnicate\" for \"abcd\"\n"; stderr != want {
			t.Fatalf("stderr =\n%q\nwant\n%q", stderr, want)
		}
	})
}
