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
// shapes that could tell them apart, and each must give the same identity or the
// same error, for an author and a committer alike.
func TestTheListingAnswersAsAReadOfEachKeyDoes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string // appended to the repository's .git/config
		env    map[string]string
		cwd    func(t *testing.T, repo string) string
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
		{name: "no repository", cwd: func(t *testing.T, _ string) string { return t.TempDir() }},
		{name: "no directory", cwd: func(t *testing.T, _ string) string { return filepath.Join(t.TempDir(), "gone") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			if tc.config != "" {
				f, err := os.OpenFile(filepath.Join(r.Root(), ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.WriteString(tc.config); err != nil {
					t.Fatal(err)
				}
				f.Close()
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
