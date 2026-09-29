package statusline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestReadSettingsFileJudgesTheFileItReads: the file whose bytes are read is
// the file whose mode and owner were judged. A settings file that passes every
// guard is renamed over by one anyone can write while the read is under way
// (inside the window fsutil.SwapHomeScopeVettedForTest opens, between a look at
// ~/.abcd and its open); the replacement's previous_command must never be
// handed back, because the harness runs it on every refresh
// (iss-2609290656491358). The read goes through fsutil.ReadHomeDeclaration,
// which judges the leaf on the descriptor of the ~/.abcd it opened, so the
// replacement is judged as itself and refused for its own mode.
func TestReadSettingsFileJudgesTheFileItReads(t *testing.T) {
	home := t.TempDir()
	path := writeSettings(t, home, `{"schema_version":1,"previous_command":"theirs-was-vetted"}`)
	swap := filepath.Join(home, ".abcd", "swap.json")
	if err := os.WriteFile(swap, []byte(`{"schema_version":1,"previous_command":"planted"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(swap, 0o666); err != nil {
		t.Fatal(err)
	}
	swapped := false
	t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(string) {
		if swapped {
			return
		}
		swapped = true
		if err := os.Rename(swap, path); err != nil {
			t.Errorf("rename: %v", err)
		}
	}))

	raw, why, err := ReadSettingsFile(home)
	if !swapped {
		t.Fatal("the read never opened ~/.abcd, so the window was not exercised")
	}
	if strings.Contains(string(raw), "planted") {
		t.Fatalf("ReadSettingsFile returned the swapped-in, world-writable file: %q", raw)
	}
	if err != nil || !strings.Contains(why, "writable by others") {
		t.Fatalf("why = %q, err = %v; want the replacement refused for its own mode", why, err)
	}
}
