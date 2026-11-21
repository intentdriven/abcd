package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestItd98NamesNoSupersededOwner holds itd-98 (solo vs. duo is measured) to
// owners that still exist: it keeps model-level benchmarking out of its scope
// by naming who owns it, and once named itd-17, which itd-2609221009495079
// superseded (iss-2610262043598494). Every line of itd-98 naming an intent
// that sits in superseded/ must say on that line that it is superseded, so
// no line hands scope or substrate to a record that no longer carries it.
func TestItd98NamesNoSupersededOwner(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	intents := filepath.Join(root, ".abcd", "development", "intents")
	superseded := map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(intents, "superseded"))
	if err != nil {
		t.Fatalf("read superseded intents: %v", err)
	}
	idOf := regexp.MustCompile(`^(itd-[0-9]+)-`)
	for _, e := range entries {
		if m := idOf.FindStringSubmatch(e.Name()); m != nil {
			superseded[m[1]] = true
		}
	}
	if !superseded["itd-17"] {
		t.Fatal("itd-17 is not in superseded/: the case this test guards no longer holds, so it proves nothing")
	}
	path := filepath.Join(intents, "drafts", "itd-98-solo-vs-duo-is-measured-not-debated.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read itd-98: %v", err)
	}
	ref := regexp.MustCompile(`\bitd-[0-9]+\b`)
	for i, line := range strings.Split(string(raw), "\n") {
		for _, id := range ref.FindAllString(line, -1) {
			if superseded[id] && !strings.Contains(line, "superseded") {
				t.Errorf("%s:%d names %s, which is superseded, as if it still owned its scope: %q", path, i+1, id, strings.TrimSpace(line))
			}
		}
	}
	if !strings.Contains(string(raw), "itd-64") {
		t.Errorf("%s no longer names itd-64, the owner of model-level benchmarking", path)
	}
}
