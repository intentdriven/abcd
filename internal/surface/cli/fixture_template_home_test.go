package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A template is kept only when seeding left nothing in HOME but what gittest.Env
// writes there, wherever os.UserConfigDir puts it on this machine: reading any
// platform's path as foreign means no template is ever kept, and the cut does
// nothing on that platform with every test still passing.
func TestOnlyGittestHomeAcceptsWhatGittestEnvWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gittest.Env(t)
	if !onlyGittestHome(t, home) {
		t.Error("what gittest.Env writes in HOME reads as foreign, so no fixture template is ever kept here")
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if onlyGittestHome(t, home) {
		t.Error("a file gittest.Env does not write reads as its own")
	}
}
