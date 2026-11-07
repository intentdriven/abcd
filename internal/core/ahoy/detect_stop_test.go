package ahoy

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runstop"
)

// TestTheSharedLineNamesAStopAnotherAccountLeft: which is not honoured, and is said.
func TestTheSharedLineNamesAStopAnotherAccountLeft(t *testing.T) {
	home, folder := t.TempDir(), t.TempDir()
	dir := filepath.Join(folder, runstop.StopDirName+"-"+strconv.Itoa(os.Getuid()+1))
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runstop.StopFileName), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldHome, oldKnown := runstop.AccountHome, runstop.WellKnownFolder
	t.Cleanup(func() { runstop.AccountHome, runstop.WellKnownFolder = oldHome, oldKnown })
	runstop.AccountHome = func() (string, error) { return home, nil }
	runstop.WellKnownFolder = func() string { return folder }
	if got := detectSharedFolder(); !strings.Contains(got, "not honoured") {
		t.Fatalf("the shared line: %q", got)
	}
}
