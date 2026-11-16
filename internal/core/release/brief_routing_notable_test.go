package release

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The launch chapter's model-tier routing paragraph ends on what a cut does
// with no routing table accepted and no override. It once said "nothing is
// printed", while the same paragraph says a ready emit prints a routing: line
// and an ingest a route: line, and both still print in that state: only the
// stderr fallback line is absent (iss-2611040554354084). TestLaunchShip-
// CarriesTheRequestAndTheReceipt in internal/surface/cli holds the behaviour;
// this holds the sentence to it.
func TestBriefLaunchNoTableSentenceKeepsTheRoutingLines(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/04-launch.md"
	body, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	var para string
	for _, p := range strings.Split(string(body), "\n\n") {
		if strings.HasPrefix(p, "**Model-tier routing.**") {
			para = strings.Join(strings.Fields(p), " ")
		}
	}
	if para == "" {
		t.Fatalf("%s has no **Model-tier routing.** paragraph", rel)
	}
	i := strings.Index(para, "With no table accepted and no override")
	if i < 0 {
		t.Fatalf("%s's routing paragraph says nothing of the no-table state:\n%s", rel, para)
	}
	sentence := para[i:]
	if regexp.MustCompile(`(?i)\bnothing is printed\b`).MatchString(sentence) {
		t.Errorf("%s says nothing is printed with no table accepted, but the routing: and route: lines still are:\n%s", rel, sentence)
	}
	for _, want := range []string{
		"no stderr fallback line is printed",
		"`routing:`",
		"`route:`",
		"`host-decides`",
		"`harness`",
	} {
		if !strings.Contains(sentence, want) {
			t.Errorf("%s's no-table sentence does not say %q:\n%s", rel, want, sentence)
		}
	}
}
