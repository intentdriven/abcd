//go:build unix

package statusline

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestASettingBehindASymlinkedAbcdHomeIsIgnored: the setting's
// previous_command is a shell command the harness runs, and it was read
// through a symlinked ~/.abcd the rules loader refuses (iss-2609281017573862).
// Behind the link the setting is not the caller's word: the defaults render
// and a note says why. The same file in a real ~/.abcd is taken.
func TestASettingBehindASymlinkedAbcdHomeIsIgnored(t *testing.T) {
	dotfiles := t.TempDir()
	writeSettings(t, dotfiles, `{"schema_version":1,"previous_command":"/bin/echo hi"}`)
	home := t.TempDir()
	if err := os.Symlink(abcdhome.Path(dotfiles), abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.PreviousCommand != "" || got.Installed {
		t.Fatalf("a setting behind a symlinked ~/.abcd was taken: %+v", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "~/.abcd is a symlink") {
		t.Fatalf("notes = %q, want one naming the symlinked ~/.abcd", notes)
	}

	if got, _, err := LoadFrom(dotfiles); err != nil || got.PreviousCommand != "/bin/echo hi" {
		t.Fatalf("the same setting in a real ~/.abcd must be taken: %+v, %v", got, err)
	}
}
