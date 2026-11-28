package ideate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A verdict field naming the home in its cleaned spelling, under a HOME
// written with a doubled slash, has the home swept: the literal backstop reads
// the home cleaned as well as as written (iss-2609290055092630).
func TestRecordRedactorSweepsTheCleanedSpellingOfAnUncleanHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("HOME", base+"//maya")
	t.Setenv(abcdhome.StageHomeEnv, "")
	r, err := newRecordRedactor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clean := filepath.Join(base, "maya")
	got, _ := r.field("the prototype lives at " + clean + "/proto")
	if strings.Contains(got, clean) || !strings.Contains(got, "~/proto") {
		t.Errorf("the cleaned spelling of the home was not swept: %q", got)
	}
}
