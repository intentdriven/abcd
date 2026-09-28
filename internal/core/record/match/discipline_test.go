package match

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The itd-84 discipline names a capture-time validator as its next rung; the
// record this package delivers marks that rung delivered (criterion 5 of
// itd-2609212137116617). The test reads the committed discipline record, so a
// later edit that drops the mark fails here rather than leaving the rung
// reading as unbuilt.
func TestTheDisciplineRecordsTheDeliveredRung(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package directory")
		}
		dir = parent
	}
	b, err := os.ReadFile(filepath.Join(dir, ".abcd/development/intents/disciplines/itd-84-intent-decomposition.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"**Delivered rung: the capture-time candidate pass.**", "itd-2609212137116617", "duplicates", "refines"} {
		if !strings.Contains(s, want) {
			t.Fatalf("itd-84 does not carry %q: the capture-time candidate pass does not read as delivered", want)
		}
	}
}
