package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/homeignore"
)

// TestTheLaidIgnoreFileDoesNotAdmitACredential pins adr-2610140859394097's
// invariant: abcd's home ignoring itself (itd-2610140858048004) changes nothing
// for credentials. With the ignore file laid in a home that is a git working
// tree, the abcd home still refuses the value and writes nothing.
func TestTheLaidIgnoreFileDoesNotAdmitACredential(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := homeignore.Lay(home); err != nil || got != homeignore.Laid {
		t.Fatalf("Lay = %v, %v; want Laid", got, err)
	}
	_, err := Set(home, "svc", Choice{Home: HomeABCD, Value: secretValue})
	if err == nil || !strings.Contains(err.Error(), "working tree") {
		t.Fatalf("err = %v, want a working-tree refusal with the ignore file laid", err)
	}
	if _, statErr := os.Lstat(abcdhome.Path(home, StoreFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("credentials.json was written beside the laid ignore file")
	}
}
