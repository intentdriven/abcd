package reading

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// setIdentityValue makes the identity probe read value for setting and
// nothing else: every git variable cleared, the global config a fresh file,
// HOME named after the account for the home folder's name.
func setIdentityValue(t *testing.T, setting, value string) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
	dir := t.TempDir()
	var text string
	home := filepath.Join(dir, "maya")
	switch setting {
	case "user.name":
		text = "[user]\n\tname = " + value + "\n"
	case "user.email":
		text = "[user]\n\temail = " + value + "\n"
	case "remote.origin.url":
		text = "[remote \"origin\"]\n\turl = git@github.com:" + value + "/notes.git\n"
	case "the home folder's name":
		home = filepath.Join(dir, value)
	}
	global := filepath.Join(dir, "global.gitconfig")
	if err := os.WriteFile(global, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv(abcdhome.StageHomeEnv, "")
	t.Setenv("HOME", home)
}

// treeFiles lists every file under root but the ingest's lock file, which
// every attempt takes, refused or not.
func treeFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() != ".lock" {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// An ingest refuses, writing nothing, while an identity value is not valid
// UTF-8: no pattern can match the value, so payload text holding it would be
// committed in clear. Every other degraded scanner still writes and notes.
func TestIngestRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, c := range []struct{ setting, value string }{
		{"user.name", "\xe9orvan Quill"}, {"user.name", "Zorv\xe9n Quill"}, {"user.name", "Zorvan Quill\xe9"},
		{"user.email", "\xe9orvan@example.com"}, {"user.email", "zorv\xe9n@example.com"}, {"user.email", "zorvan@example.co\xe9"},
		{"remote.origin.url", "\xe9orvan"}, {"remote.origin.url", "zorv\xe9n"}, {"remote.origin.url", "zorvan\xe9"},
		{"the home folder's name", "\xe9orvan"}, {"the home folder's name", "zorv\xe9n"}, {"the home folder's name", "zorvan\xe9"},
	} {
		f := newIngestFixture(t, "detection")
		out := f.writePayload(f.payload(1))
		setIdentityValue(t, c.setting, c.value)
		before := treeFiles(t, f.root)
		_, err := Ingest(IngestRequest{RepoRoot: f.root, OutputPath: out})
		if err == nil {
			t.Errorf("%s %+q: the ingest wrote, want a refusal", c.setting, c.value)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, c.setting) || !strings.Contains(msg, "UTF-8") || !strings.Contains(msg, "nothing written") {
			t.Errorf("%s %+q: the refusal does not name the setting to re-save: %q", c.setting, c.value, msg)
		}
		if low := strings.ToLower(msg); strings.Contains(low, "orv") || strings.Contains(low, "quill") ||
			!utf8.ValidString(msg) || strings.ContainsRune(msg, utf8.RuneError) {
			t.Errorf("%s %+q: the refusal prints the value: %+q", c.setting, c.value, msg)
		}
		if after := treeFiles(t, f.root); !slices.Equal(before, after) {
			t.Errorf("%s %+q: the refused ingest changed the tree: %v -> %v", c.setting, c.value, before, after)
		}
	}
}

// A UTF-8 identity leaves the ingest as it was.
func TestIngestWithAUTF8IdentityIsUnchanged(t *testing.T) {
	f := newIngestFixture(t, "detection")
	setIdentityValue(t, "user.name", "Zorvan Quillé")
	if res := f.mustIngest(f.payload(1)); res.Degraded != "" {
		t.Errorf("a UTF-8 identity noted the ingest degraded: %q", res.Degraded)
	}
}
