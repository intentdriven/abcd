package positioning

import (
	"strings"
	"testing"
)

// lockupBrief is a block that carries the whole lockup: a name apart from its
// title, a descriptor and a purpose, beside the title, tagline, pitch and
// listing line.
const lockupBrief = `# Product

## Identity (canonical)

- **Name:** abcd
- **Title:** Agent-Based Configuration for Development
- **Descriptor:** the prototyping engine for product thinkers
- **Tagline:** Intend it. See it work. Keep it on record.
- **Purpose:** Express your product ideas through working prototypes and
  demonstrators.
- **Pitch:** A single Go binary, usable as a plugin in compatible agent harnesses.
- **Listing:** The open way to build your idea with the agent you already use.
  abcd keeps your idea as plain files.
`

func TestParseBlockReadsTheLockupFields(t *testing.T) {
	root := writeRepo(t, map[string]string{"brief.md": lockupBrief})
	got, err := ParseBlock(root, BlockLocation{File: "brief.md", Heading: "Identity (canonical)"})
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	for _, c := range []struct{ field, want string }{
		{"name", "abcd"},
		{"title", "Agent-Based Configuration for Development"},
		{"descriptor", "the prototyping engine for product thinkers"},
		{"tagline", "Intend it. See it work. Keep it on record."},
		{"purpose", "Express your product ideas through working prototypes and demonstrators."},
		{"pitch", "A single Go binary, usable as a plugin in compatible agent harnesses."},
	} {
		v, ok := got.Field(c.field)
		if !ok {
			t.Errorf("Field(%q) is not a block field", c.field)
			continue
		}
		if v != c.want {
			t.Errorf("Field(%q) = %q, want %q", c.field, v, c.want)
		}
	}
	if got.Brand() != "abcd" {
		t.Errorf("Brand() = %q, want the declared name", got.Brand())
	}
	if want := "abcd — the prototyping engine for product thinkers"; got.Headline() != want {
		t.Errorf("Headline() = %q, want %q", got.Headline(), want)
	}
}

// A block written before the lockup fields existed reads exactly as it did: the
// brand is the title's first word and the headline is the title, so no
// repository that adopted a block earlier changes what its surfaces say.
func TestBlockWithoutLockupFieldsKeepsItsTitle(t *testing.T) {
	b := Block{Title: "acme — a widget shop", Tagline: "Widgets."}
	if b.Brand() != "acme" {
		t.Errorf("Brand() = %q, want the title's first word", b.Brand())
	}
	if b.Headline() != b.Title {
		t.Errorf("Headline() = %q, want the title %q", b.Headline(), b.Title)
	}
	// A name with no descriptor heads the block on its own title.
	named := Block{Name: "acme", Title: "Acme Widget Shop", Tagline: "Widgets."}
	if want := "acme — Acme Widget Shop"; named.Headline() != want {
		t.Errorf("Headline() with a name and no descriptor = %q, want %q", named.Headline(), want)
	}
}

// A surface may require and template the lockup fields, and the config refuses
// a field outside the vocabulary by naming the whole vocabulary.
func TestSurfaceTemplatesTheLockupFields(t *testing.T) {
	root := writeRepo(t, map[string]string{"brief.md": lockupBrief})
	b, err := ParseBlock(root, BlockLocation{File: "brief.md", Heading: "Identity (canonical)"})
	if err != nil {
		t.Fatal(err)
	}
	s := Surface{ID: "x", Kind: KindRegexp, Files: []string{"README.md"}, Patterns: []string{`(?s)<p>(.*?)</p>`},
		Requires: []string{"name", "descriptor", "tagline", "purpose"}, Template: "{name} — {descriptor}. {tagline} {purpose}"}
	if err := s.validate(0); err != nil {
		t.Fatalf("a surface over the lockup fields is refused: %v", err)
	}
	want := "abcd — the prototyping engine for product thinkers. Intend it. See it work. Keep it on record. Express your product ideas through working prototypes and demonstrators."
	if got := s.render(b); got != want {
		t.Errorf("render = %q, want %q", got, want)
	}
	bad := s
	bad.Requires = []string{"slogan"}
	err = bad.validate(0)
	if err == nil || !strings.Contains(err.Error(), "name, title, descriptor, tagline, purpose, pitch or listing") {
		t.Errorf("unknown field error = %v, want it to name the whole vocabulary", err)
	}
}

// The listing line is a field of its own (product thinker, 2026-11-01): the
// sentence a tool listing shows beside other tools, written whole, so a surface
// can carry it as written instead of assembling it from the descriptor and the
// pitch, and any later surface can use it too.
func TestTheListingLineIsAFieldOfItsOwn(t *testing.T) {
	root := writeRepo(t, map[string]string{"brief.md": lockupBrief})
	b, err := ParseBlock(root, BlockLocation{File: "brief.md", Heading: "Identity (canonical)"})
	if err != nil {
		t.Fatal(err)
	}
	want := "The open way to build your idea with the agent you already use. abcd keeps your idea as plain files."
	if got, ok := b.Field("listing"); !ok || got != want {
		t.Fatalf("Field(listing) = %q, %v; want %q", got, ok, want)
	}
	s := Surface{ID: "manifest", Kind: KindJSONField, Files: []string{"plugin.json"}, Field: "description",
		Requires: []string{"listing"}, Template: "{listing}"}
	if err := s.validate(0); err != nil {
		t.Fatalf("a surface over the listing field is refused: %v", err)
	}
	if got := s.render(b); got != want {
		t.Errorf("render = %q, want the listing line as written: %q", got, want)
	}
	// A block written before the field existed reads exactly as it did.
	if got, _ := (Block{Title: "acme", Tagline: "Widgets."}).Field("listing"); got != "" {
		t.Errorf("a block with no listing line returns %q", got)
	}
}
