package cli

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The cloud-session scripts (spc-2610120941441282), run the way
// githook_prepush_test.go runs the pre-push hook: in a throwaway clone carrying
// copies of the committed scripts, with a bare repository as its remote, and
// with fakes first on PATH for every tool whose answer a test decides. Nothing
// here needs the network, a hosted session, root or gitleaks.

type cloudCase struct {
	t      *testing.T
	dir    string // the working clone
	origin string // its bare remote
	bin    string // the fakes, first on PATH
	calls  string // the fakes' call log
	env    []string
}

// cloudFixtureFiles are the committed files the cloud scripts read, besides
// everything under scripts/cloud/: the attribution gate and its identity list
// (the identity check), the gate-tool list and go.mod (the environment check).
var cloudFixtureFiles = []string{
	"scripts/check-attribution.sh",
	"internal/core/identity/tool-identities.txt",
	"scripts/gate-tools.tsv",
	"go.mod",
}

// cloudScrubbed are the variables a test decides for itself: the identity git
// would otherwise take from the environment, and the host's remote marker.
var cloudScrubbed = []string{
	"PATH=", "GIT_AUTHOR_NAME=", "GIT_AUTHOR_EMAIL=", "GIT_COMMITTER_NAME=", "GIT_COMMITTER_EMAIL=",
	"EMAIL=", "CLAUDE_CODE_REMOTE=", "FAKE_UID=", "FAKE_GOVERSION=", "FAKE_GOMODCACHE=",
}

func newCloudCase(t *testing.T) *cloudCase {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	realGit, _ := exec.LookPath("git")
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))

	base := t.TempDir()
	c := &cloudCase{
		t: t, dir: filepath.Join(base, "work"), origin: filepath.Join(base, "origin.git"),
		bin: filepath.Join(base, "bin"), calls: filepath.Join(base, "calls.log"),
	}
	for _, kv := range gittest.Env(t) {
		keep := true
		for _, p := range cloudScrubbed {
			if strings.HasPrefix(kv, p) {
				keep = false
			}
		}
		if keep {
			c.env = append(c.env, kv)
		}
	}
	c.env = append(c.env, "PATH="+c.bin+string(os.PathListSeparator)+os.Getenv("PATH"), "CALLS="+c.calls)

	gittest.InitBare(t, c.origin)
	if out, err := c.run(base, "git", "init", "-q", "-b", "main", c.dir); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	cloudDir := filepath.Join(root, "scripts", "cloud")
	if err := filepath.WalkDir(cloudDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		c.copyFrom(root, filepath.ToSlash(rel))
		return nil
	}); err != nil && !os.IsNotExist(err) {
		t.Fatalf("copy scripts/cloud: %v", err)
	}
	for _, rel := range cloudFixtureFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			c.copyFrom(root, rel)
		}
	}
	c.write("seed.md", "seed\n", 0o644)
	c.git("add", "-A")
	c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
		"commit", "-q", "-m", "seed")
	c.git("remote", "add", "origin", c.origin)
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")

	// The fakes. Each logs its argv, so a test can prove a step never ran.
	c.fake("id", `echo "id $*" >>"$CALLS"
if [ "${1:-}" = -u ]; then echo "${FAKE_UID:-1000}"; exit 0; fi
exec /usr/bin/id "$@"`)
	c.fake("go", `echo "go $*" >>"$CALLS"
case "$*" in
"env GOVERSION") echo "${FAKE_GOVERSION:-}" ;;
"env GOMODCACHE") echo "${FAKE_GOMODCACHE:-/nonexistent}" ;;
"env GOOS") echo testos ;;
"env GOARCH") echo testarch ;;
*) echo "fake go: unexpected $*" >&2; exit 3 ;;
esac`)
	c.fake("git", `echo "git $*" >>"$CALLS"
exec '`+realGit+`' "$@"`)
	c.resetCalls()
	return c
}

func (c *cloudCase) run(dir string, name string, args ...string) (string, error) {
	c.t.Helper()
	return c.runEnv(dir, nil, name, args...)
}

func (c *cloudCase) runEnv(dir string, extra []string, name string, args ...string) (string, error) {
	c.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(append([]string{}, c.env...), extra...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// script runs one cloud script from the clone's root with extra environment.
func (c *cloudCase) script(extra []string, rel string, args ...string) (string, int) {
	c.t.Helper()
	out, err := c.runEnv(c.dir, extra, "bash", append([]string{filepath.Join(c.dir, filepath.FromSlash(rel))}, args...)...)
	return out, cloudExitCode(c.t, err)
}

func cloudExitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("running a cloud script: %v", err)
	}
	return ee.ExitCode()
}

func (c *cloudCase) git(args ...string) string {
	c.t.Helper()
	out, err := c.run(c.dir, "git", args...)
	if err != nil {
		c.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

func (c *cloudCase) write(rel, content string, mode os.FileMode) {
	c.t.Helper()
	p := filepath.Join(c.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cloudCase) copyFrom(root, rel string) {
	c.t.Helper()
	src := filepath.Join(root, filepath.FromSlash(rel))
	b, err := os.ReadFile(src)
	if err != nil {
		c.t.Fatalf("read the committed %s: %v", rel, err)
	}
	st, err := os.Stat(src)
	if err != nil {
		c.t.Fatal(err)
	}
	c.write(rel, string(b), st.Mode().Perm())
}

func (c *cloudCase) fake(name, body string) {
	c.t.Helper()
	if err := os.MkdirAll(c.bin, 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cloudCase) resetCalls() {
	c.t.Helper()
	if err := os.WriteFile(c.calls, nil, 0o644); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cloudCase) callLog() string {
	c.t.Helper()
	b, err := os.ReadFile(c.calls)
	if err != nil {
		c.t.Fatal(err)
	}
	return string(b)
}

const connectorScript = "scripts/cloud/connector.sh"

// declare writes one connector's declaration, with an is-remote script beside
// it so only the field under test can be what is refused.
func (c *cloudCase) declare(name, conf string) {
	c.t.Helper()
	dir := "scripts/cloud/connectors/" + name
	c.write(dir+"/connector.conf", conf, 0o644)
	c.write(dir+"/is-remote.sh", "#!/usr/bin/env bash\nexit 1\n", 0o755)
}

func TestConnectorDeclarationLoads(t *testing.T) {
	c := newCloudCase(t)
	if out, code := c.script(nil, connectorScript, "validate"); code != 0 {
		t.Fatalf("the committed declaration does not validate (exit %d)\n%s", code, out)
	}
	if out, _ := c.script(nil, connectorScript, "names"); strings.TrimSpace(out) != "claude" {
		t.Errorf("names = %q, want the one committed connector", out)
	}
	if out, _ := c.script(nil, connectorScript, "default"); strings.TrimSpace(out) != "claude" {
		t.Errorf("default = %q, want the only connector declared", out)
	}
	if out, _ := c.script(nil, connectorScript, "prefixes"); strings.TrimSpace(out) != "claude/abcd-wip/" {
		t.Errorf("prefixes = %q, want claude/abcd-wip/", out)
	}
	out, code := c.script(nil, connectorScript, "get", "claude", "is_remote")
	want, err := filepath.EvalSymlinks(filepath.Join(c.dir, "scripts", "cloud", "connectors", "claude", "is-remote.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 || strings.TrimSpace(out) != want {
		t.Errorf("get claude is_remote = %q (exit %d), want the script's absolute path %s", out, code, want)
	}
	if out, code := c.script(nil, connectorScript, "get", "claude", "setup"); code == 0 {
		t.Errorf("get of a key the declaration does not carry succeeded: %q", out)
	}
	if out, code := c.script(nil, connectorScript, "get", "nosuch", "wip_prefix"); code == 0 {
		t.Errorf("get of an undeclared connector succeeded: %q", out)
	}

	// A second connector, with its own prefix, loads beside the first; the
	// default then refuses to guess between them.
	c.declare("other", "# another host\nname=other\nwip_prefix=abcd-wip/\nis_remote=is-remote.sh\n")
	if out, code := c.script(nil, connectorScript, "validate"); code != 0 {
		t.Fatalf("two connectors with distinct prefixes do not validate (exit %d)\n%s", code, out)
	}
	if out, _ := c.script(nil, connectorScript, "prefixes"); !strings.Contains(out, "claude/abcd-wip/") || !strings.Contains(out, "\nabcd-wip/") && !strings.HasPrefix(out, "abcd-wip/") {
		t.Errorf("prefixes = %q, want both connectors' prefixes", out)
	}
	out, code = c.script(nil, connectorScript, "default")
	if code == 0 || !strings.Contains(out, "refusing to guess") {
		t.Errorf("default with two connectors = %q (exit %d), want a refusal to guess", out, code)
	}
}

func TestConnectorRefusesBadPrefix(t *testing.T) {
	for _, tc := range []struct{ name, prefix string }{
		{"the default branch", "main"},
		{"the host namespace alone", "claude/"},
		{"no trailing slash", "abcd-wip"},
		{"two segments before abcd-wip", "a/b/abcd-wip/"},
		{"empty", ""},
		{"a segment after abcd-wip", "claude/abcd-wip/x/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCloudCase(t)
			c.declare("claude", "name=claude\nwip_prefix="+tc.prefix+"\nis_remote=is-remote.sh\n")
			for _, verb := range []string{"validate", "prefixes"} {
				out, code := c.script(nil, connectorScript, verb)
				if code == 0 {
					t.Fatalf("%s accepted wip_prefix=%q\n%s", verb, tc.prefix, out)
				}
				if !strings.Contains(out, "wip_prefix") || !strings.Contains(out, "connectors/claude/connector.conf") {
					t.Errorf("%s's refusal names neither the field nor the file\n%s", verb, out)
				}
			}
		})
	}
}

func TestConnectorRefusesUnknownKeyAndEscapingPath(t *testing.T) {
	outside := "#!/usr/bin/env bash\nexit 0\n"
	for _, tc := range []struct {
		name, conf, want string
		prep             func(c *cloudCase)
	}{
		{"an unknown key", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=is-remote.sh\ncolour=blue\n", "colour", nil},
		{"a line that is no key=value", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=is-remote.sh\njust words\n", "line 4", nil},
		{"a key given twice", "name=claude\nwip_prefix=claude/abcd-wip/\nwip_prefix=abcd-wip/\nis_remote=is-remote.sh\n", "twice", nil},
		{"a missing required key", "name=claude\nwip_prefix=claude/abcd-wip/\n", "is_remote", nil},
		{"a name that is not its directory", "name=other\nwip_prefix=claude/abcd-wip/\nis_remote=is-remote.sh\n", "name", nil},
		{"a parent-relative script", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=../../begin.sh\n", "is_remote", nil},
		{"an absolute script", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=/bin/sh\n", "is_remote", nil},
		{"a script that does not exist", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=gone.sh\n", "is_remote", nil},
		{"a symlink out of the directory", "name=claude\nwip_prefix=claude/abcd-wip/\nis_remote=link.sh\n", "is_remote", func(c *cloudCase) {
			c.write("outside.sh", outside, 0o755)
			if err := os.Symlink(filepath.Join(c.dir, "outside.sh"), filepath.Join(c.dir, "scripts/cloud/connectors/claude/link.sh")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCloudCase(t)
			c.declare("claude", tc.conf)
			if tc.prep != nil {
				tc.prep(c)
			}
			out, code := c.script(nil, connectorScript, "validate")
			if code == 0 {
				t.Fatalf("the declaration was accepted\n%s", out)
			}
			if !strings.Contains(out, tc.want) || !strings.Contains(out, "connectors/claude/connector.conf") {
				t.Errorf("the refusal does not name %q and the file\n%s", tc.want, out)
			}
			// Every verb reads through the same validation.
			if out, code := c.script(nil, connectorScript, "get", "claude", "is_remote"); code == 0 {
				t.Errorf("get answered from a refused declaration: %q", out)
			}
		})
	}
}

func TestConnectorRefusesDuplicatePrefix(t *testing.T) {
	c := newCloudCase(t)
	c.declare("other", "name=other\nwip_prefix=claude/abcd-wip/\nis_remote=is-remote.sh\n")
	out, code := c.script(nil, connectorScript, "validate")
	if code == 0 {
		t.Fatalf("two connectors declaring one prefix were accepted\n%s", out)
	}
	if !strings.Contains(out, "claude/abcd-wip/") || !strings.Contains(out, "two connectors") {
		t.Errorf("the refusal does not name the shared prefix\n%s", out)
	}
	if out, code := c.script(nil, connectorScript, "prefixes"); code == 0 {
		t.Errorf("prefixes answered with a prefix declared twice: %q", out)
	}
}

func TestIsRemoteOnlyWhenMarkerSet(t *testing.T) {
	c := newCloudCase(t)
	path, code := c.script(nil, connectorScript, "get", "claude", "is_remote")
	if code != 0 {
		t.Fatalf("the connector names no is_remote script\n%s", path)
	}
	path = strings.TrimSpace(path)
	for _, tc := range []struct {
		env  []string
		want int
	}{
		{nil, 1},
		{[]string{"CLAUDE_CODE_REMOTE="}, 1},
		{[]string{"CLAUDE_CODE_REMOTE=false"}, 1},
		{[]string{"CLAUDE_CODE_REMOTE=1"}, 1},
		{[]string{"CLAUDE_CODE_REMOTE=TRUE"}, 1},
		{[]string{"CLAUDE_CODE_REMOTE=true "}, 1},
		{[]string{"CLAUDE_CODE_REMOTE=true"}, 0},
	} {
		out, err := c.runEnv(c.dir, tc.env, path)
		if got := cloudExitCode(t, err); got != tc.want {
			t.Errorf("is-remote with %v exited %d, want %d\n%s", tc.env, got, tc.want, out)
		}
		if out != "" {
			t.Errorf("is-remote with %v printed %q; it answers by its exit code alone", tc.env, out)
		}
	}
}
