package ahoy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallModeReadsTheRecordWithoutAPluginRoot is iss-2610240916382802: a
// working copy at ~/.local/bin/abcd that ~/.abcd.noindex/path-entry records is
// an install whether or not a plugin root resolves. A plain-terminal run after
// the README's install line has no plugin root, and the detection pass must
// still say the copy is installed rather than leave install_mode empty.
func TestInstallModeReadsTheRecordWithoutAPluginRoot(t *testing.T) {
	home, _ := setupUserScope(t)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	target := filepath.Join(home, ".local", "bin", "abcd")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("#!/bin/sh\necho abcd release copy\n")
	if err := os.WriteFile(target, body, 0o755); err != nil {
		t.Fatal(err)
	}
	sha, ok := fileSHA256Hex(target)
	if !ok {
		t.Fatal("hashing the copy failed")
	}
	if err := writePathEntry(target, sha, ""); err != nil {
		t.Fatal(err)
	}

	if got := detectInstallMode("", false); got != "pinned" {
		t.Errorf("install mode with no plugin root = %q, want %q for the copy the record names", got, "pinned")
	}

	// A record whose copy no longer matches its hash vouches for nothing.
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho changed\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := detectInstallMode("", false); got != "" {
		t.Errorf("install mode for a copy whose bytes changed = %q, want empty", got)
	}
}
