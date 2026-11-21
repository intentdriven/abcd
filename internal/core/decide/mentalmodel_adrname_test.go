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

// adrStoreNamingFaults reports how a fourth-capture-verdict paragraph that
// routes a decision to the ADR store misnames the store's files, or nothing
// when it names them by the id without its adr- prefix: the stamp form, as
// `<stamp>-<slug>.md` or spelled out as `<yymmddHHMMSS><rrrr>-<slug>.md`,
// and the ordinals 0001-0058 keeping `NNNN-<slug>.md`.
func adrStoreNamingFaults(para string) []string {
	var faults []string
	if strings.Contains(para, "adr-N-") {
		faults = append(faults, "names the files with an adr- prefix, which no file in the store carries")
	}
	if !strings.Contains(para, "`<stamp>-<slug>.md`") && !strings.Contains(para, "`<yymmddHHMMSS><rrrr>-<slug>.md`") {
		faults = append(faults, "does not name the stamp form `<stamp>-<slug>.md` every ADR minted since adr-45 takes")
	}
	if !strings.Contains(para, "`NNNN-<slug>.md`") || !strings.Contains(para, "`0001`–`0058`") {
		faults = append(faults, "does not confine `NNNN-<slug>.md` to the ordinals `0001`–`0058`")
	}
	return faults
}

// TestBriefNamesADRFilesTheSameWayAtBothEnds holds the pair
// iss-2610262030446141 found: the mental model and the intent chapter each
// route a confirmed decision verdict to the same ADR store, and named its
// files by shapes the other did not describe (`adr-N-<slug>.md` against
// `NNNN-<slug>.md`, zero-padded). Each passage must name the files by the id
// without its adr- prefix, the stamp form and the ordinals both.
func TestBriefNamesADRFilesTheSameWayAtBothEnds(t *testing.T) {
	// The two passages as they stood before their fixes (03-mental-model at
	// 68be032a1^, 05-intent at f8de4f61c^): the check must refuse each.
	for name, old := range map[string]string{
		"mental model before 68be032a1":   "A confirmed `decision` routes capture to the existing **ADR store** (`.abcd/development/decisions/adrs/`, `adr-N-<slug>.md`)",
		"intent chapter before f8de4f61c": "A confirmed `decision` DIVERTS capture to the existing ADR store (`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded)",
	} {
		if len(adrStoreNamingFaults(old)) == 0 {
			t.Errorf("the check accepts the %s, which names a shape the other end does not describe", name)
		}
	}

	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("01-product", "03-mental-model.md"),
		filepath.Join("04-surfaces", "05-intent.md"),
	} {
		path := filepath.Join(root, ".abcd", "development", "brief", rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		var para string
		for _, p := range strings.Split(string(raw), "\n\n") {
			if strings.Contains(p, "A fourth capture verdict") && strings.Contains(p, "decisions/adrs/") {
				para = p
				break
			}
		}
		if para == "" {
			t.Fatalf("%s has no fourth-capture-verdict paragraph naming the ADR store", path)
		}
		for _, f := range adrStoreNamingFaults(para) {
			t.Errorf("%s's fourth-capture-verdict paragraph %s", path, f)
		}
	}
}
