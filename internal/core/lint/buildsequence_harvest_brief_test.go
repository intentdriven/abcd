package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The delivery chapter's memory bullet once said a vendor memory harvest is an
// opt-in, read-only source over the memory substrate, in the present tense and
// unmarked, while the chapter marks its unbuilt items as design targets and no
// verb, flag or Go source offers a harvest (iss-2611040555055477). Until one
// ships, the clause that names the harvest carries the chapter's marker.
func TestBriefBuildSequenceMarksTheMemoryHarvestADesignTarget(t *testing.T) {
	const briefRel = ".abcd/development/brief/06-delivery/01-build-sequence.md"
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(briefRel)))
	if err != nil {
		t.Fatalf("read %s: %v", briefRel, err)
	}

	const opener = "- **memory**:"
	text := string(raw)
	start := strings.Index(text, opener)
	if start < 0 {
		t.Fatalf("%s has no bullet opening %q", briefRel, opener)
	}
	bullet := text[start:]
	if end := strings.Index(bullet[len(opener):], "\n- "); end >= 0 {
		bullet = bullet[:len(opener)+end]
	}
	if end := strings.Index(bullet, "\n\n"); end >= 0 {
		bullet = bullet[:end]
	}
	flat := strings.Join(strings.Fields(bullet), " ")

	i := strings.Index(flat, "harvest")
	if i < 0 {
		t.Skipf("the memory bullet of %s no longer names a harvest: %s", briefRel, flat)
	}
	// The clause runs from the semicolon before the harvest to the next
	// semicolon or full stop, so a marker on the shipping substrate does not
	// count for the harvest.
	from := strings.LastIndex(flat[:i], ";") + 1
	clause := flat[from:]
	if end := strings.IndexAny(flat[i:], ";."); end >= 0 {
		clause = flat[from : i+end]
	}
	if !strings.Contains(clause, "*(design target)*") {
		t.Errorf("the memory bullet of %s names a vendor memory harvest without marking it *(design target)*: %s", briefRel, flat)
	}
}
