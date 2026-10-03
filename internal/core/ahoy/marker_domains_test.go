package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// defaultDomainsList is the paragraph of the managed block that lists the
// default domains: from its heading to the next.
func defaultDomainsList(t *testing.T, block string) string {
	t.Helper()
	const head = "### Default domains"
	start := strings.Index(block, head)
	if start < 0 {
		t.Fatalf("the managed block has no %q section", head)
	}
	rest := block[start+len(head):]
	if end := strings.Index(rest, "\n### "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

// TestManagedBlockNamesEveryBundledDomain holds the managed block's
// default-domain list to the domains the binary bundles, GRILL included, and
// to GRILL's dormant escape (spc-2610030944505997 step 4): a repository that
// receives a domain through the binary reads, in its own conventions file,
// that it has it and how to silence it.
func TestManagedBlockNamesEveryBundledDomain(t *testing.T) {
	list := defaultDomainsList(t, string(markerInner))
	opening := list
	if i := strings.Index(list, ". Each carries"); i >= 0 {
		opening = list[:i]
	}
	named := map[string]bool{}
	for _, m := range regexp.MustCompile("`([A-Z]+)`").FindAllStringSubmatch(opening, -1) {
		named[m[1]] = true
	}
	var bundled []string
	for name := range rules.Defaults().Domains {
		bundled = append(bundled, name)
	}
	sort.Strings(bundled)
	for _, name := range bundled {
		if !named[name] {
			t.Errorf("the managed block's default-domain list does not name the bundled domain %s", name)
		}
		delete(named, name)
	}
	for name := range named {
		t.Errorf("the managed block's default-domain list names %s, which the binary does not bundle", name)
	}
	flat := strings.Join(strings.Fields(list), " ")
	if !strings.Contains(flat, "`{\"GRILL\": {\"state\": \"dormant\"}}`") {
		t.Error("the managed block does not name GRILL's dormant escape, {\"GRILL\": {\"state\": \"dormant\"}}")
	}
}

// TestAbcdsOwnConventionsFileCarriesTheManagedBlock holds abcd's own AGENTS.md
// to the block ahoy plants, so the default-domain list abcd's own sessions
// read and the one every managed repository receives stay matched.
func TestAbcdsOwnConventionsFileCarriesTheManagedBlock(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := markerBlockRe.FindSubmatch(data)
	if m == nil {
		t.Fatal("abcd's AGENTS.md carries no managed block")
	}
	if got, want := string(m[0]), string(synthesizeMarker(markerInner, []byte("\n"))); got != want {
		t.Errorf("abcd's AGENTS.md managed block differs from the block ahoy plants (internal/core/ahoy/defaults/claude-md-marker-block.md); copy one onto the other.\n--- AGENTS.md\n%s\n--- planted\n%s", got, want)
	}
}
