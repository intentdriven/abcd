package lifeboat

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// The embark chapter once said embark carries the pass-B exemption through "as
// the first line of both reports" (iss-2611060809529676). The `embark probe`
// report opens on its heading line and the exemption follows it; only `embark
// from` opens with the exemption. The renders and the chapter are both held to
// that here.
func TestBriefEmbarkChapterPlacesTheExemptionAfterTheProbeHeading(t *testing.T) {
	cov := &CoverageHandoff{PassBExemption: &PassBExemption{Reason: "no transcript tier"}}
	const exemption = "pass B (transcripts): declared exempt — no transcript tier"

	probe := strings.Split(EmbarkPlan{SourceName: "src", ManifestVerified: true, Coverage: cov}.Render(), "\n")
	if !strings.HasPrefix(probe[0], "embark probe: src") {
		t.Fatalf("the probe report does not open on its heading: %q", probe[0])
	}
	var first string
	for _, l := range probe[1:] {
		if l != "" {
			first = l
			break
		}
	}
	if first != exemption {
		t.Errorf("the probe report's first line after its heading is %q, want the exemption", first)
	}
	from := strings.Split(EmbarkResult{SourceName: "src", Coverage: cov}.Render(), "\n")
	if from[0] != exemption {
		t.Errorf("the from report does not open with the exemption: %q", from[0])
	}

	const rel = ".abcd/development/brief/04-surfaces/03-embark.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	body := strings.Join(strings.Fields(string(raw)), " ")
	if strings.Contains(body, "first line of both reports") {
		t.Errorf("%s still says the exemption is the first line of both reports; the probe report opens on its heading", rel)
	}
	for _, want := range []string{
		"before the blanks in both reports",
		"after the probe's heading line",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s does not say %q about where the exemption prints", rel, want)
		}
	}
}

// The embark chapter's packing-history example once showed `"dest":
// "../abcd-lifeboat"` (iss-2611060809546274), but Pack resolves the destination
// before it logs it, so the field never holds a relative path
// (TestPackVoyageRecordsTheResolvedDest holds the code to that). The example
// must show an absolute path, and the chapter must say the field holds the
// resolved destination.
func TestBriefEmbarkHistoryExampleShowsTheResolvedDest(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/03-embark.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	text := string(raw)
	const lead = "`disembark/history.jsonl` is append-only"
	i := strings.Index(text, lead)
	if i < 0 {
		t.Fatalf("%s no longer introduces the history example with %q", rel, lead)
	}
	rest := text[i:]
	open := strings.Index(rest, "```json\n")
	if open < 0 {
		t.Fatalf("%s has no json example after %q", rel, lead)
	}
	rest = rest[open+len("```json\n"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatalf("%s: the history example's fence never closes", rel)
	}
	dec := json.NewDecoder(strings.NewReader(rest[:end]))
	dec.DisallowUnknownFields()
	var e voyageEntry
	if err := dec.Decode(&e); err != nil {
		t.Fatalf("%s: the history example is not a voyage entry: %v", rel, err)
	}
	if !path.IsAbs(e.Dest) {
		t.Errorf("%s: the history example's dest is %q; the ledger records the resolved, absolute destination", rel, e.Dest)
	}
	body := strings.Join(strings.Fields(rest[end:]), " ")
	if !strings.Contains(body, "the destination resolved to an absolute path") {
		t.Errorf("%s does not say the history entry's dest holds the destination resolved to an absolute path", rel)
	}
}
