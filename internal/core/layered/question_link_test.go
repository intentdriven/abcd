//go:build unix

package layered

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/question"
)

// TestQuestionLimitsAndTheSymlinkedHome lays out a dotfiles-symlinked
// ~/.abcd.noindex both ways (after home_link_test.go): holding no config.json
// it reads as absent and the limits are the defaults, with nothing said; holding
// one, even with no question key, it is an unread layer naming the file, so the
// gate fails open and loud on it (itd-2610301914350541 decision 8).
func TestQuestionLimitsAndTheSymlinkedHome(t *testing.T) {
	link := func(t *testing.T, f *fixture) string {
		dotfiles := t.TempDir()
		if err := os.Symlink(dotfiles, abcdhome.Path(f.roots.Home)); err != nil {
			t.Fatal(err)
		}
		return dotfiles
	}
	t.Run("no config.json behind the link", func(t *testing.T) {
		f := newFixture(t)
		link(t, f)
		v, err := QuestionLimits(f.roots)
		if err != nil {
			t.Fatalf("a symlinked ~/.abcd.noindex holding no config.json must read as absent: %v", err)
		}
		if !reflect.DeepEqual(v.V, question.Default) || v.Layer != Bundled {
			t.Errorf("got %+v from %s, want the defaults", v.V, v.Layer)
		}
	})
	assertUnread(t, map[string]func(t *testing.T, f *fixture) string{
		"a config.json with no question key behind the link": func(t *testing.T, f *fixture) string {
			f.write(filepath.Join(link(t, f), filepath.FromSlash(Config.MachineRel)), `{"pace": {"work_minutes": 5}}`)
			return machineOrigin
		},
		"a symlinked repository file": func(t *testing.T, f *fixture) string {
			target := filepath.Join(t.TempDir(), "elsewhere.json")
			f.write(target, `{"question": {"rows": 40}}`)
			p := filepath.Join(f.roots.Repo, filepath.FromSlash(Config.RepoRel))
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, p); err != nil {
				t.Fatal(err)
			}
			return Config.RepoOrigin()
		},
	})
}
