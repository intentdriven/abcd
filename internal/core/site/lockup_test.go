package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lockupIdentity is an Identity block that carries the whole lockup: a name
// apart from its title, a descriptor and a purpose.
const lockupIdentity = `# Product

## Identity (canonical)

- **Name:** Probe
- **Title:** Probe Expanded Fixture Title
- **Descriptor:** the fixture engine for fixture people
- **Tagline:** A configuration layer for the fixture.
- **Purpose:** Express fixture ideas through working fixtures.
- **Pitch:** One binary that carries the why from idea to shipped reality.
`

// TestTheHeroRendersTheLockup holds the landing page to the lockup: the eyebrow
// is the headline (name and descriptor), the purpose sits under the tagline,
// the page title is the headline, and the brand is the declared name rather
// than the title's first word. The render passes every gate, and a drifted
// purpose is refused like any other hero span.
func TestTheHeroRendersTheLockup(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/development/brief/01-product/README.md", lockupIdentity)
	r.build()
	data, err := os.ReadFile(filepath.Join(r.out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, want := range []string{
		`<title>Probe — the fixture engine for fixture people</title>`,
		`>Probe — the fixture engine for fixture people</p>`,
		`<p class="purpose"`,
		`>Express fixture ideas through working fixtures.</p>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the landing page lacks %q", want)
		}
	}
	if strings.Contains(html, "Probe Expanded Fixture Title</p>") {
		t.Error("the eyebrow still renders the title; with a descriptor it renders the headline")
	}
	if res := r.check(); !res.OK() {
		t.Fatalf("a lockup render failed the gates: %+v", res.Findings)
	}
	r.breakOutput(">Express fixture ideas through working fixtures.</p>", ">Express nothing at all.</p>")
	wantFinding(t, r.check(), CheckHero, "purpose")
}
