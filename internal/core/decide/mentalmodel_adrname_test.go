package decide

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestMentalModelNamesADRFilesAsTheStoreHoldsThem holds the mental model's
// fourth-capture-verdict paragraph to the filenames the ADR store actually
// carries (iss-2610262030578776). The decide chapter says an ADR's filename
// is its stamp followed by its slug, and the ordinals 0001-0058 keep theirs;
// no file in the store carries an `adr-` prefix. The paragraph once gave the
// shape `adr-N-<slug>.md`, so a reader looking an ADR up by it found nothing.
func TestMentalModelNamesADRFilesAsTheStoreHoldsThem(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	// The store itself: every record is an ordinal or a stamp, never adr-N.
	store := filepath.Join(root, ".abcd", "development", "decisions", "adrs")
	entries, err := os.ReadDir(store)
	if err != nil {
		t.Fatalf("read ADR store: %v", err)
	}
	ordinal := regexp.MustCompile(`^[0-9]{4}-[a-z0-9-]+\.md$`)
	stamp := regexp.MustCompile(`^[0-9]{16}-[a-z0-9-]+\.md$`)
	var sawOrdinal, sawStamp bool
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "README.md" || !strings.HasSuffix(name, ".md") {
			continue
		}
		switch {
		case ordinal.MatchString(name):
			sawOrdinal = true
		case stamp.MatchString(name):
			sawStamp = true
		case strings.HasPrefix(name, "adr-"):
			t.Errorf("ADR store holds %s, an adr- prefixed file: the shape this test rules out of the prose is real after all", name)
		}
	}
	if !sawOrdinal || !sawStamp {
		t.Fatalf("ADR store holds ordinals=%v stamps=%v: the two vintages this test names are not both present, so it proves nothing", sawOrdinal, sawStamp)
	}

	path := filepath.Join(root, ".abcd", "development", "brief", "01-product", "03-mental-model.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mental model: %v", err)
	}
	var para string
	for _, p := range strings.Split(string(raw), "\n\n") {
		if strings.Contains(p, "A fourth capture verdict") {
			para = p
			break
		}
	}
	if para == "" {
		t.Fatalf("%s has no fourth-capture-verdict paragraph", path)
	}
	if strings.Contains(para, "adr-N-") {
		t.Errorf("%s names the ADR store's files `adr-N-<slug>.md`; the store holds `<stamp>-<slug>.md` and ordinals `NNNN-<slug>.md`", path)
	}
	for _, want := range []string{"`<stamp>-<slug>.md`", "`NNNN-<slug>.md`"} {
		if !strings.Contains(para, want) {
			t.Errorf("%s's fourth-capture-verdict paragraph does not name the ADR filename shape %s", path, want)
		}
	}
}
