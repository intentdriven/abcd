package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestGrillCostInTheChapterMatchesTheBlock holds the configuration chapter's
// figure for what GRILL costs to what *GRILL actually injects: the chapter
// once said about 1,100 tokens while the block came to about 1,600 at four
// bytes a token (iss-2611060809322689), and the block has grown since. A
// figure stated in the GRILL paragraph must sit within a quarter of the full
// block's size at four bytes a token; the chapter now states none.
func TestGrillCostInTheChapterMatchesTheBlock(t *testing.T) {
	d, ok := Defaults().Domains[GrillDomain]
	if !ok {
		t.Fatalf("the %s domain is not bundled", GrillDomain)
	}
	block := Render([]ResolvedDomain{{Name: GrillDomain, Source: SourceBundled, Domain: d}})
	tokens := len(block) / 4

	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".abcd", "development", "brief", "05-internals", "03-configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	start := strings.Index(doc, "`GRILL` is the second")
	if start < 0 {
		t.Fatal("the configuration chapter has no GRILL paragraph")
	}
	para := doc[start:]
	if end := strings.Index(para, "\n\n"); end >= 0 {
		para = para[:end]
	}
	for _, m := range regexp.MustCompile(`about\s+([\d,]+)\s+tokens`).FindAllStringSubmatch(para, -1) {
		n, err := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
		if err != nil {
			t.Fatal(err)
		}
		if n*4 < tokens*3 || n*4 > tokens*5 {
			t.Errorf("the chapter says GRILL lands at about %d tokens; *%s injects %d bytes, about %d tokens", n, GrillDomain, len(block), tokens)
		}
	}
}
