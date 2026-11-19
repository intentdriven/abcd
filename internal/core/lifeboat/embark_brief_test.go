package lifeboat

import (
	"os"
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
