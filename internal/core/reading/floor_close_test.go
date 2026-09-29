package reading

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// TestTheFloorsBlockEndsNoEarlierThanTheCanonicalClose: the exclusion floor
// opens a block more broadly than the canonical reader, and that is safe only
// while its scans also END no earlier than the canonical reader's close
// (iss-2609281627055603).
//
// The floor's closer fires on any column-0 line opening with three dashes, so a
// `----` or a `--- x` ended its block while frontmatter.Fields, which closes on
// IsDelimiter alone, read on to the real `---`. The keys between the two closes
// are frontmatter to the canonical reader and were invisible to the floor, so an
// excluded key there travelled under a manifest asserting its refusal. Each case
// first proves the premise — Fields reads the key — and then that the floor
// refuses it.
func TestTheFloorsBlockEndsNoEarlierThanTheCanonicalClose(t *testing.T) {
	const warm = "ABCD-WARM-ORIGIN"
	for name, doc := range map[string]string{
		"a four-dash line":          "---\nid: spc-1\n----\norigin: " + warm + "\n---\n\n# A record\n",
		"a delimiter carrying text": "---\nid: spc-1\n--- x\norigin: " + warm + "\n---\n\n# A record\n",
		"a BOM-led opener":          "\ufeff---\nid: spc-1\n----\norigin: " + warm + "\n---\n\n# A record\n",
		"CRLF line ends":            "---\r\nid: spc-1\r\n----\r\norigin: " + warm + "\r\n---\r\n\r\n# A record\r\n",
	} {
		if _, ok := frontmatter.Fields(strings.Split(doc, "\n"))["origin"]; !ok {
			t.Fatalf("%s: the premise does not hold — the canonical reader does not read the key", name)
		}
		err := refuses(t, "spc-1-a-record.md", doc, refusalKeys, refusalHeadings)
		if err == nil {
			t.Errorf("%s: an excluded key the canonical reader reads as frontmatter was admitted; "+
				"the floor's block ended before the canonical close", name)
			continue
		}
		for _, want := range []string{"spc-1-a-record.md", `"origin"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: the refusal does not name %s: %v", name, want, err)
			}
		}
	}

	// The shape scan reads the same extent. A fence delimiter between the two
	// closes sits inside the block to the canonical reader; the fence mask starts
	// after the floor's own close, so honouring the mask there would hide the
	// key the fence wraps.
	fenced := "---\nid: spc-1\n----\n```\norigin: " + warm + "\n```\n---\n\n# A record\n"
	if err := refuses(t, "spc-1-a-record.md", fenced, refusalKeys, refusalHeadings); err == nil {
		t.Error("a fenced key between the floor's close and the canonical close was admitted")
	}

	// What the canonical reader reads as body stays body: a rule under a closed
	// block, and a four-dash line in the prose, refuse nothing.
	for name, doc := range map[string]string{
		"a rule under the block":   "---\nid: spc-1\n---\n\n# A record\n\n----\n\norigin: prose, not a key\n",
		"a four-dash line in body": "---\nid: spc-1\n---\n\nProse.\n\n--- x\n\nMore prose.\n",
	} {
		if err := refuses(t, "spc-1-a-record.md", doc, refusalKeys, refusalHeadings); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}
