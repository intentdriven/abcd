package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// identity_probe_pin_test.go — the semantics ProbeIdentity's git reads have,
// pinned shape by shape so a change to HOW the probe asks git (how many
// processes, which flags) is held to WHAT it answers (iss-2609281546130900).
// Every case builds real configuration files and runs the real git binary;
// the expectations are the answers the four-process probe gave, including the
// awkward ones (a value holding an escaped newline is split by the user.*
// read and kept whole by the persona read).

// probeAnswer is the git-derived part of an Identity. HomePath/HomeUser come
// from $HOME, not from git, and are pinned elsewhere.
type probeAnswer struct {
	Name, Email             string
	OtherNames, OtherEmails []string
	RemoteUser, RemoteRepo  string
}

func answerOf(id Identity) probeAnswer {
	return probeAnswer{
		Name: id.GitUserName, Email: id.GitUserEmail,
		OtherNames: id.OtherGitUserNames, OtherEmails: id.OtherGitUserEmails,
		RemoteUser: id.GitRemoteUsername, RemoteRepo: id.GitRemoteRepo,
	}
}

// pinCtx is one case's hermetic world: a HOME, a global and a system config
// file the probe's scrubbed environment honours, and a repository.
type pinCtx struct {
	home, global, system, work, repo string
}

// unsetenv removes k for the rest of the test and restores it afterwards.
// t.Setenv(k, "") would leave k present, and a present-but-empty
// GIT_CONFIG_PARAMETERS is still a command-line config entry.
func unsetenv(t *testing.T, k string) {
	t.Helper()
	t.Setenv(k, "")
	if err := os.Unsetenv(k); err != nil {
		t.Fatal(err)
	}
}

// clearIdentityEnv removes every variable the probe reads from the process
// environment, directly or through the scrubbed subprocess environment, so a
// case starts from nothing and adds only what it names.
func clearIdentityEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(k, "GIT_") {
			unsetenv(t, k)
		}
	}
}

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// newPinCtx builds the world. bare makes the repository a bare one, so the
// config it carries is read through discovery rules a `-c` entry can change.
func newPinCtx(t *testing.T, bare bool) *pinCtx {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	clearIdentityEnv(t)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	c := &pinCtx{
		home:   home,
		global: filepath.Join(home, "global.gitconfig"),
		system: filepath.Join(home, "system.gitconfig"),
		work:   filepath.Join(home, "work"),
	}
	c.repo = filepath.Join(c.work, "repo")
	for _, p := range []string{c.global, c.system} {
		appendFile(t, p, "")
	}
	if err := os.MkdirAll(c.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"init", "-q"}
	if bare {
		args = append(args, "--bare")
	}
	c.git(t, append(args, c.repo)...)
	t.Setenv("GIT_CONFIG_GLOBAL", c.global)
	t.Setenv("GIT_CONFIG_SYSTEM", c.system)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
	return c
}

// git runs a setup command under the hermetic test environment (global and
// system config neutralised), so setup never reads what the case configures.
func (c *pinCtx) git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (c *pinCtx) localConfig() string {
	if _, err := os.Stat(filepath.Join(c.repo, ".git")); err == nil {
		return filepath.Join(c.repo, ".git", "config")
	}
	return filepath.Join(c.repo, "config") // bare
}

type pinCase struct {
	name                   string
	bare                   bool
	system, global, local  string // raw config text appended to each file
	setup                  func(t *testing.T, c *pinCtx)
	env                    map[string]string
	root                   func(c *pinCtx) string // the repoRoot handed to the probe; default c.repo
	want                   probeAnswer
	skipAsRoot, skipAbsent bool
}

func identityPinCases() []pinCase {
	big := strings.Repeat("v", 70000) + " Name"
	return []pinCase{
		{name: "no configuration at all", want: probeAnswer{}},
		{name: "user.name only", global: "[user]\n\tname = Solo Name\n",
			want: probeAnswer{Name: "Solo Name"}},
		{name: "user.email only", global: "[user]\n\temail = solo@example.com\n",
			want: probeAnswer{Email: "solo@example.com"}},
		{name: "multi-valued keys in one file, the last effective",
			global: "[user]\n\tname = First Name\n\tname = Second Name\n\temail = one@example.com\n\temail = two@example.com\n",
			want: probeAnswer{Name: "Second Name", Email: "two@example.com",
				OtherNames: []string{"First Name"}, OtherEmails: []string{"one@example.com"}}},
		{name: "every file scope, the worktree effective",
			system: "[user]\n\tname = System Name\n\temail = system@example.com\n",
			global: "[user]\n\tname = Global Name\n\temail = global@example.com\n",
			local:  "[user]\n\tname = Local Name\n\temail = local@example.com\n",
			setup: func(t *testing.T, c *pinCtx) {
				c.git(t, "-C", c.repo, "config", "extensions.worktreeConfig", "true")
				c.git(t, "-C", c.repo, "config", "--worktree", "user.name", "Worktree Name")
				c.git(t, "-C", c.repo, "config", "--worktree", "user.email", "worktree@example.com")
			},
			want: probeAnswer{Name: "Worktree Name", Email: "worktree@example.com",
				OtherNames:  []string{"System Name", "Global Name", "Local Name"},
				OtherEmails: []string{"system@example.com", "global@example.com", "local@example.com"}}},
		{name: "an includeIf persona evaluated where it sits",
			global: "[user]\n\tname = Global Name\n\temail = global@example.com\n",
			setup: func(t *testing.T, c *pinCtx) {
				inc := filepath.Join(c.home, "persona.inc")
				appendFile(t, inc, "[user]\n\tname = Include Persona\n\temail = persona@example.com\n")
				appendFile(t, c.global, "[includeIf \"gitdir:"+c.work+"/\"]\n\tpath = "+inc+"\n[user]\n\tname = After Include\n")
			},
			want: probeAnswer{Name: "After Include", Email: "persona@example.com",
				OtherNames: []string{"Global Name", "Include Persona"}, OtherEmails: []string{"global@example.com"}}},
		{name: "an includeIf that does not match the repository",
			global: "[user]\n\tname = Global Name\n",
			setup: func(t *testing.T, c *pinCtx) {
				inc := filepath.Join(c.home, "elsewhere.inc")
				appendFile(t, inc, "[user]\n\tname = Elsewhere Persona\n")
				appendFile(t, c.global, "[includeIf \"gitdir:"+c.home+"/elsewhere/\"]\n\tpath = "+inc+"\n")
			},
			want: probeAnswer{Name: "Global Name"}},
		{name: "outside a repository only system and global are read",
			global: "[user]\n\tname = Global Name\n",
			root:   func(c *pinCtx) string { return c.home },
			want:   probeAnswer{Name: "Global Name"}},
		{name: "a repository root that does not exist",
			global: "[user]\n\tname = Global Name\n",
			root:   func(c *pinCtx) string { return filepath.Join(c.home, "absent") },
			want:   probeAnswer{}},
		{name: "case-variant duplicates collapse",
			global: "[user]\n\tname = Alice Example\n\temail = Alice@Example.com\n",
			local:  "[user]\n\tname = ALICE EXAMPLE\n\temail = alice@example.com\n",
			want:   probeAnswer{Name: "ALICE EXAMPLE", Email: "alice@example.com"}},
		{name: "a section spelled in another case is the same key",
			global: "[User]\n\tName = Upper Section\n",
			want:   probeAnswer{Name: "Upper Section"}},
		{name: "a valueless key and a blank value are dropped",
			global: "[user]\n\tname = Real Name\n\tname\n\temail = \"   \"\n",
			want:   probeAnswer{Name: "Real Name"}},
		{name: "quoted padding is trimmed",
			global: "[user]\n\tname = \"  Padded Name  \"\n",
			want:   probeAnswer{Name: "Padded Name"}},
		{name: "an escaped newline splits the user read and stays whole in the persona read",
			global: "[user]\n\tname = \"Line One\\nLine Two\"\n",
			want: probeAnswer{Name: "Line Two",
				OtherNames: []string{"Line One", "Line One\nLine Two"}}},
		{name: "control bytes and an equals sign survive verbatim",
			global: "[user]\n\tname = \"Tab\\tName\"\n\tname = Ctl\x01Name\n\tname = \"Back\\bspace\"\n\temail = \"key=value@example.com\"\n",
			want: probeAnswer{Name: "Back\bspace", Email: "key=value@example.com",
				OtherNames: []string{"Tab\tName", "Ctl\x01Name"}}},
		{name: "a very large value",
			global: "[user]\n\tname = " + big + "\n",
			want:   probeAnswer{Name: big}},
		{name: "author and committer keys join the others",
			global: "[user]\n\tname = User Name\n\temail = user@example.com\n[author]\n\tname = Author Key\n\temail = author@example.com\n[committer]\n\tname = User Name\n\temail = committer@example.com\n",
			want: probeAnswer{Name: "User Name", Email: "user@example.com",
				OtherNames: []string{"Author Key"}, OtherEmails: []string{"author@example.com", "committer@example.com"}}},
		{name: "author keys with no user identity",
			global: "[author]\n\tname = Author Only\n\temail = author@example.com\n",
			want:   probeAnswer{OtherNames: []string{"Author Only"}, OtherEmails: []string{"author@example.com"}}},
		{name: "keys that only resemble identity keys are ignored",
			global: "[user \"sub\"]\n\tname = Sub Name\n[user]\n\tnames = Plural\n\tusername = Handle\n[github]\n\tuser = gh\n",
			want:   probeAnswer{}},
		{name: "the environment persona joins the others",
			global: "[user]\n\tname = Config Name\n\temail = config@example.com\n",
			env: map[string]string{"GIT_AUTHOR_NAME": "Env Author", "GIT_AUTHOR_EMAIL": "env@example.com",
				"GIT_COMMITTER_NAME": "config name", "GIT_COMMITTER_EMAIL": "  "},
			want: probeAnswer{Name: "Config Name", Email: "config@example.com",
				OtherNames: []string{"Env Author"}, OtherEmails: []string{"env@example.com"}}},
		{name: "an injected counted -c identity is an other, never the effective",
			global: "[user]\n\tname = Real Name\n\temail = real@example.com\n",
			env: map[string]string{"GIT_CONFIG_COUNT": "2",
				"GIT_CONFIG_KEY_0": "user.email", "GIT_CONFIG_VALUE_0": "injected@example.com",
				"GIT_CONFIG_KEY_1": "remote.origin.url", "GIT_CONFIG_VALUE_1": "https://github.com/evil/repo"},
			want: probeAnswer{Name: "Real Name", Email: "real@example.com",
				OtherEmails: []string{"injected@example.com"}}},
		{name: "an injected GIT_CONFIG_PARAMETERS identity is an other, after the environment persona",
			global: "[user]\n\tname = Real Name\n",
			env: map[string]string{"GIT_CONFIG_PARAMETERS": `'user.name'='Param Name' 'committer.email'='param@example.com'`,
				"GIT_AUTHOR_NAME": "Env Author"},
			want: probeAnswer{Name: "Real Name",
				OtherNames: []string{"Env Author", "Param Name"}, OtherEmails: []string{"param@example.com"}}},
		{name: "a malformed -c entry blinds the persona read and nothing else",
			global: "[user]\n\tname = Real Name\n[author]\n\tname = Author Key\n[remote \"origin\"]\n\turl = git@github.com:octo/proj.git\n",
			env:    map[string]string{"GIT_CONFIG_PARAMETERS": `'bogus`, "GIT_AUTHOR_EMAIL": "env@example.com"},
			want: probeAnswer{Name: "Real Name", OtherEmails: []string{"env@example.com"},
				RemoteUser: "octo", RemoteRepo: "proj"}},
		{name: "a malformed counted -c entry blinds the persona read and nothing else",
			global: "[user]\n\temail = real@example.com\n[committer]\n\temail = committer@example.com\n",
			env:    map[string]string{"GIT_CONFIG_COUNT": "2", "GIT_CONFIG_KEY_0": "user.name", "GIT_CONFIG_VALUE_0": "Param Name"},
			want:   probeAnswer{Email: "real@example.com"}},
		{name: "a -c entry that changes discovery changes only the persona read",
			bare:   true,
			global: "[user]\n\tname = Global Name\n",
			local:  "[user]\n\tname = Bare Local\n[author]\n\tname = Bare Author\n[remote \"origin\"]\n\turl = https://github.com/octo/bare.git\n",
			env:    map[string]string{"GIT_CONFIG_PARAMETERS": `'safe.bareRepository'='explicit'`},
			want: probeAnswer{Name: "Bare Local", OtherNames: []string{"Global Name"},
				RemoteUser: "octo", RemoteRepo: "bare"}},
		{name: "the legacy GIT_CONFIG file is never read",
			global: "[user]\n\tname = Real Name\n",
			setup: func(t *testing.T, c *pinCtx) {
				alt := filepath.Join(c.home, "alt.gitconfig")
				appendFile(t, alt, "[user]\n\tname = Alt Name\n\temail = alt@example.com\n")
				t.Setenv("GIT_CONFIG", alt)
			},
			want: probeAnswer{Name: "Real Name"}},
		{name: "an inherited GIT_DIR cannot redirect the probe",
			local: "[user]\n\tname = This Repo\n",
			setup: func(t *testing.T, c *pinCtx) {
				other := filepath.Join(c.home, "other")
				c.git(t, "init", "-q", other)
				appendFile(t, filepath.Join(other, ".git", "config"), "[user]\n\tname = Other Repo\n")
				t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
				t.Setenv("GIT_WORK_TREE", other)
				t.Setenv("GIT_COMMON_DIR", filepath.Join(other, ".git"))
			},
			want: probeAnswer{Name: "This Repo"}},
		{name: "GIT_CONFIG_GLOBAL and the system file are honoured, NOSYSTEM too",
			system: "[user]\n\temail = system@example.com\n",
			global: "[user]\n\tname = Global Name\n",
			setup:  func(t *testing.T, c *pinCtx) { t.Setenv("GIT_CONFIG_NOSYSTEM", "1") },
			want:   probeAnswer{Name: "Global Name"}},
		{name: "remote: the last url wins, a pushurl and another remote are ignored",
			local: "[remote \"origin\"]\n\turl = https://github.com/first/one.git\n\turl = https://GitHub.com/Second/two.git\n\tpushurl = https://github.com/push/p\n[remote \"upstream\"]\n\turl = https://github.com/up/u\n",
			want:  probeAnswer{RemoteUser: "Second", RemoteRepo: "two"}},
		{name: "remote: a subsection spelled in another case is another remote",
			local: "[remote \"Origin\"]\n\turl = https://github.com/cased/c\n",
			want:  probeAnswer{}},
		{name: "remote: read from global config too",
			global: "[remote \"origin\"]\n\turl = git@github.com:globalowner/g.git\n",
			want:   probeAnswer{RemoteUser: "globalowner", RemoteRepo: "g"}},
		{name: "remote: a last url that is valueless leaves no remote",
			local: "[remote \"origin\"]\n\turl = https://github.com/first/one\n\turl\n",
			want:  probeAnswer{}},
		{name: "remote: a non-GitHub remote names no owner",
			local: "[remote \"origin\"]\n\turl = https://git.example.com/owner/repo\n",
			want:  probeAnswer{}},
		{name: "a malformed config file blinds every read",
			global: "[user]\n\tname = Real Name\n[broken\n",
			env:    map[string]string{"GIT_AUTHOR_NAME": "Env Author"},
			want:   probeAnswer{OtherNames: []string{"Env Author"}}},
		{name: "an unreadable global config is skipped, the rest still read",
			global:     "[user]\n\tname = Real Name\n",
			local:      "[user]\n\temail = local@example.com\n",
			setup:      func(t *testing.T, c *pinCtx) { chmod(t, c.global, 0) },
			skipAsRoot: true,
			want:       probeAnswer{Email: "local@example.com"}},
		{name: "an unreadable repository config blinds every read",
			global:     "[user]\n\tname = Real Name\n",
			local:      "[user]\n\temail = local@example.com\n",
			setup:      func(t *testing.T, c *pinCtx) { chmod(t, c.localConfig(), 0) },
			skipAsRoot: true,
			want:       probeAnswer{}},
		{name: "git absent",
			global: "[user]\n\tname = Real Name\n",
			env:    map[string]string{"GIT_COMMITTER_EMAIL": "env@example.com"},
			setup: func(t *testing.T, c *pinCtx) {
				t.Setenv("PATH", t.TempDir())
			},
			want: probeAnswer{OtherEmails: []string{"env@example.com"}}},
	}
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
}

// TestProbeIdentityPinnedSemantics is the table: each shape's full answer.
func TestProbeIdentityPinnedSemantics(t *testing.T) {
	for _, tc := range identityPinCases() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipAsRoot && os.Geteuid() == 0 {
				t.Skip("root reads a mode-000 file")
			}
			c := newPinCtx(t, tc.bare)
			appendFile(t, c.system, tc.system)
			appendFile(t, c.global, tc.global)
			if tc.local != "" { // abcd-lint:allow
				appendFile(t, c.localConfig(), tc.local) // abcd-lint:allow
			}
			if tc.setup != nil {
				tc.setup(t, c)
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			root := c.repo
			if tc.root != nil {
				root = tc.root(c)
			}
			got := answerOf(ProbeIdentity(root))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ProbeIdentity answer drifted:\n got  %s\n want %s", fmtAnswer(got), fmtAnswer(tc.want))
			}
		})
	}
}

func fmtAnswer(a probeAnswer) string {
	short := func(s string) string {
		if len(s) > 40 {
			return s[:20] + "…" + s[len(s)-10:]
		}
		return s
	}
	shortAll := func(v []string) []string {
		if v == nil {
			return nil
		}
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = short(s)
		}
		return out
	}
	return strings.Join([]string{
		"name=" + quote(short(a.Name)), "email=" + quote(a.Email),
		"otherNames=" + quoteAll(shortAll(a.OtherNames)), "otherEmails=" + quoteAll(a.OtherEmails),
		"remote=" + quote(a.RemoteUser) + "/" + quote(a.RemoteRepo),
	}, " ")
}

func quote(s string) string { return "\"" + strings.ReplaceAll(s, "\n", `\n`) + "\"" }

func quoteAll(v []string) string {
	if v == nil {
		return "nil"
	}
	out := make([]string, len(v))
	for i, s := range v {
		out[i] = quote(s)
	}
	return "[" + strings.Join(out, ", ") + "]"
}
