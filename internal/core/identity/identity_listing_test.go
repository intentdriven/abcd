package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// readByKey is the identity resolved with one git read per key, the reading the
// listing replaced, kept as what the listing is held to.
func readByKey(root string, role Role) (Effective, error) {
	field := func(envKey string, keys ...string) (string, error) {
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			return v, nil
		}
		for _, key := range keys {
			v, err := gitConfig(root, key)
			if err != nil || v != "" {
				return v, err
			}
		}
		return "", nil
	}
	env := "GIT_" + strings.ToUpper(string(role)) + "_"
	name, err := field(env+"NAME", string(role)+".name", "user.name")
	if err != nil {
		return Effective{}, err
	}
	email, err := field(env+"EMAIL", string(role)+".email", "user.email")
	if err != nil {
		return Effective{}, err
	}
	return Effective{Name: name, Email: email}, nil
}

// TestTheListingAnswersAsAReadOfEachKeyDoes runs both readings over the config
// shapes that could tell them apart, among them every scope, includes and
// conditional includes, and each must give the same identity or the same error,
// for an author and a committer alike. A shape that names an author holds both to
// it, so it cannot pass by both reading nothing.
func TestTheListingAnswersAsAReadOfEachKeyDoes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string // appended to the repository's .git/config
		global string // the person's ~/.gitconfig
		xdg    string // the person's $XDG_CONFIG_HOME/git/config
		system string // the system config, through GIT_CONFIG_SYSTEM
		// files are written beside the configs, in a folder {INC} names; a name
		// starting .git/ goes in the repository instead. {ROOT} is the repository.
		files  map[string]string
		env    map[string]string
		cwd    func(t *testing.T, repo string) string
		author *Effective // what the author must read, so a shape cannot pass by reading nothing
	}{
		{name: "nothing set"},
		{name: "user only", config: "[user]\n\tname = U Ser\n\temail = u@example.com\n"},
		{name: "author and committer outrank user", config: "[user]\n\tname = U\n\temail = u@example.com\n[author]\n\tname = A\n\temail = a@example.com\n[committer]\n\tname = C\n\temail = c@example.com\n"},
		{name: "one field from each", config: "[user]\n\temail = u@example.com\n[author]\n\tname = A\n"},
		{name: "several values take the last", config: "[user]\n\tname = First\n\tname = Last\n\temail = a@example.com\n\temail = b@example.com\n"},
		{name: "a blank value falls through", config: "[user]\n\tname = U\n[author]\n\tname = \"\"\n"},
		{name: "a value of spaces falls through", config: "[user]\n\tname = U\n[author]\n\tname = \"   \"\n"},
		{name: "a key with no value falls through", config: "[user]\n\tname = U\n[committer]\n\tname\n"},
		{name: "a key with no value and nothing behind it", config: "[user]\n\tname\n"},
		{name: "a value holding a newline", config: "[user]\n\tname = \"Two\\nLines\"\n"},
		{name: "a value padded with spaces", config: "[user]\n\tname = \"  Padded  \"\n"},
		{name: "section case is canonical", config: "[User]\n\tName = U\n\tEMAIL = u@example.com\n"},
		{name: "a subsection is another key", config: "[user \"x\"]\n\tname = Not Me\n"},
		{name: "an unrelated key that starts the same", config: "[user]\n\tnames = Not Me\n[authorx]\n\tname = Not Me\n"},
		{name: "a config git cannot parse", config: "[user\n\tname = broken\n"},
		{name: "the environment outranks the config", config: "[user]\n\tname = U\n\temail = u@example.com\n",
			env: map[string]string{"GIT_AUTHOR_NAME": "From Env", "GIT_COMMITTER_EMAIL": "env@example.com"}},
		{name: "a blank variable is unset", config: "[user]\n\tname = U\n", env: map[string]string{"GIT_AUTHOR_NAME": "  "}},
		{name: "a broken config with the environment answering all of it", config: "[user\n",
			env: map[string]string{"GIT_AUTHOR_NAME": "A", "GIT_AUTHOR_EMAIL": "a@example.com", "GIT_COMMITTER_NAME": "C", "GIT_COMMITTER_EMAIL": "c@example.com"}},
		{name: "a command-line entry", config: "[user]\n\tname = U\n",
			env: map[string]string{"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "committer.name", "GIT_CONFIG_VALUE_0": "From Dash C"}},
		{name: "a malformed command-line entry", env: map[string]string{"GIT_CONFIG_COUNT": "x"}},
		{name: "the global config", global: "[user]\n\tname = G\n\temail = g@example.com\n", author: &Effective{"G", "g@example.com"}},
		{name: "the global config under XDG", xdg: "[user]\n\tname = X\n\temail = x@example.com\n", author: &Effective{"X", "x@example.com"}},
		{name: "the system config", system: "[user]\n\tname = S\n\temail = s@example.com\n", author: &Effective{"S", "s@example.com"}},
		{name: "local over global over system",
			system: "[user]\n\tname = S\n\temail = s@example.com\n", global: "[user]\n\tname = G\n", config: "[user]\n\tname = L\n",
			author: &Effective{"L", "s@example.com"}},
		{name: "a role key in a wider scope outranks user in a narrower",
			global: "[author]\n\tname = Wide\n", config: "[user]\n\tname = Narrow\n\temail = n@example.com\n",
			author: &Effective{"Wide", "n@example.com"}},
		{name: "a blank local value over a global one", global: "[user]\n\tname = G\n", config: "[user]\n\tname = \"\"\n"},
		{name: "an include", config: "[include]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\tname = I\n\temail = i@example.com\n"},
			author: &Effective{"I", "i@example.com"}},
		{name: "an include beside the config", config: "[include]\n\tpath = extra\n", files: map[string]string{".git/extra": "[user]\n\tname = Near\n"},
			author: &Effective{"Near", ""}},
		{name: "an include in the global config", global: "[include]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[author]\n\tname = I\n"},
			author: &Effective{"I", ""}},
		{name: "a value after an include wins", config: "[include]\n\tpath = {INC}/a\n[user]\n\tname = After\n", files: map[string]string{"a": "[user]\n\tname = I\n"},
			author: &Effective{"After", ""}},
		{name: "an include after a value wins", config: "[user]\n\tname = Before\n[include]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\tname = I\n"},
			author: &Effective{"I", ""}},
		{name: "an include of an include", config: "[include]\n\tpath = {INC}/a\n",
			files:  map[string]string{"a": "[include]\n\tpath = {INC}/b\n", "b": "[user]\n\tname = Deep\n"},
			author: &Effective{"Deep", ""}},
		{name: "a valueless key from an include", config: "[include]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\tname\n"}},
		{name: "a missing include", config: "[user]\n\tname = U\n[include]\n\tpath = {INC}/gone\n", author: &Effective{"U", ""}},
		{name: "an include git cannot parse", config: "[include]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user\n"}},
		{name: "includeIf gitdir", config: "[includeIf \"gitdir:{ROOT}/\"]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\temail = if@example.com\n"},
			author: &Effective{"", "if@example.com"}},
		{name: "includeIf gitdir elsewhere", config: "[includeIf \"gitdir:/nowhere/\"]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\temail = if@example.com\n"},
			author: &Effective{"", ""}},
		{name: "includeIf onbranch", config: "[includeIf \"onbranch:main\"]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[user]\n\tname = OnMain\n"},
			author: &Effective{"OnMain", ""}},
		{name: "includeIf hasconfig", config: "[remote \"origin\"]\n\turl = https://example.com/r.git\n[includeIf \"hasconfig:remote.*.url:https://example.com/**\"]\n\tpath = {INC}/a\n",
			files: map[string]string{"a": "[user]\n\tname = Has\n"}, author: &Effective{"Has", ""}},
		{name: "includeIf in the global config", global: "[includeIf \"gitdir:{ROOT}/\"]\n\tpath = {INC}/a\n", files: map[string]string{"a": "[committer]\n\tname = C\n[user]\n\tname = U\n"},
			author: &Effective{"U", ""}},
		{name: "a worktree config", config: "[core]\n\trepositoryformatversion = 1\n[extensions]\n\tworktreeConfig = true\n",
			files: map[string]string{".git/config.worktree": "[user]\n\tname = W\n"}, author: &Effective{"W", ""}},
		{name: "pager.config names a program", config: "[user]\n\tname = U\n[pager]\n\tconfig = touch {INC}/pager-ran\n", author: &Effective{"U", ""}},
		{name: "core.pager names a program", global: "[core]\n\tpager = touch {INC}/pager-ran\n[pager]\n\tconfig = true\n", author: &Effective{"", ""}},
		{name: "no repository", cwd: func(t *testing.T, _ string) string { return t.TempDir() }},
		{name: "no directory", cwd: func(t *testing.T, _ string) string { return filepath.Join(t.TempDir(), "gone") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			home, xdg, inc := t.TempDir(), filepath.Join(t.TempDir(), "xdg"), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			unsetEnv(t, "GIT_CONFIG_GLOBAL")
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			unsetEnv(t, "GIT_CONFIG_SYSTEM")
			// git matches gitdir: against the physical path; a temporary directory
			// can sit behind a link.
			physical, err := filepath.EvalSymlinks(r.Root())
			if err != nil {
				t.Fatal(err)
			}
			text := strings.NewReplacer("{INC}", inc, "{ROOT}", physical)
			put := func(path, body string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(text.Replace(body)), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.config != "" {
				f, err := os.OpenFile(filepath.Join(r.Root(), ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.WriteString(text.Replace(tc.config)); err != nil {
					t.Fatal(err)
				}
				f.Close()
			}
			if tc.global != "" {
				put(filepath.Join(home, ".gitconfig"), tc.global)
			}
			if tc.xdg != "" {
				put(filepath.Join(xdg, "git", "config"), tc.xdg)
			}
			if tc.system != "" {
				path := filepath.Join(inc, "system-config")
				put(path, tc.system)
				t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
				t.Setenv("GIT_CONFIG_SYSTEM", path)
			}
			for name, body := range tc.files {
				if rel, ok := strings.CutPrefix(name, ".git/"); ok {
					put(filepath.Join(r.Root(), ".git", rel), body)
				} else {
					put(filepath.Join(inc, name), body)
				}
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			root := r.Root()
			if tc.cwd != nil {
				root = tc.cwd(t, root)
			}
			for _, role := range []Role{RoleAuthor, RoleCommitter} {
				want, wantErr := readByKey(root, role)
				got, gotErr := effective(root, role)
				if got != want {
					t.Errorf("%s: the listing read %+v, a read of each key %+v", role, got, want)
				}
				switch {
				case (gotErr == nil) != (wantErr == nil):
					t.Errorf("%s: the listing's error is %v, a read of each key's %v", role, gotErr, wantErr)
				case gotErr != nil && gotErr.Error() != wantErr.Error():
					t.Errorf("%s: the listing says %q, a read of each key %q", role, gotErr, wantErr)
				}
				if role == RoleAuthor && tc.author != nil && (got != *tc.author || gotErr != nil) {
					t.Errorf("the author read %+v (%v), want %+v", got, gotErr, *tc.author)
				}
			}
			// Through a pipe, neither reading runs what config names.
			if _, err := os.Stat(filepath.Join(inc, "pager-ran")); err == nil {
				t.Error("a read ran the pager the config names")
			}
		})
	}
}

// TestACheckRunsOneIdentityListing: an author and a committer take one git
// process between them, and an identity wholly from the environment takes none.
func TestACheckRunsOneIdentityListing(t *testing.T) {
	r := gittest.NewRepo(t)
	calls := gittest.SlowPipeGit(t, "--get-regexp", 0)
	c := &identityConfig{root: r.Root()}
	for _, role := range []Role{RoleAuthor, RoleCommitter, RoleAuthor} {
		if _, err := c.effective(role); err != nil {
			t.Fatal(err)
		}
	}
	if n := calls(); n != 1 {
		t.Errorf("%d listings for two roles, want 1", n)
	}
	t.Setenv("GIT_AUTHOR_NAME", "A")
	t.Setenv("GIT_AUTHOR_EMAIL", "a@example.com")
	if _, err := (&identityConfig{root: r.Root()}).effective(RoleAuthor); err != nil {
		t.Fatal(err)
	}
	if n := calls(); n != 1 {
		t.Errorf("an identity from the environment ran git: %d listings, want still 1", n)
	}
}
