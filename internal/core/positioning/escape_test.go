package positioning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// escapeBrief carries a quote and a backslash in every field the catalogue
// and citation surfaces render, the characters a JSON string or a YAML
// double-quoted scalar cannot hold raw.
const escapeBrief = `# Product

## Identity (canonical)

- **Name:** acme
- **Title:** Acme "Quoted" Engine
- **Descriptor:** the tool for C:\shops and "small" sellers
- **Tagline:** List it once.
- **Listing:** Say "hi" to every C:\store at once.
`

// TestEscapedSurfacesProposeValidFilesAndClear holds the two registered
// surfaces that sit inside a quoted string, the marketplace catalogue's JSON
// description and the CITATION.cff YAML abstract, to their file formats: a
// block value holding `"` or `\` proposes a file that still parses and says
// the value, and once that proposal is adopted the surface no longer drifts.
// The surfaces are this repository's own registrations.
func TestEscapedSurfacesProposeValidFilesAndClear(t *testing.T) {
	proposeAdoptAndClear(t, escapeBrief, staleCatalogue, staleCitation)
}

// TestYAMLRefusedCharactersAreEscaped carries the characters a YAML stream may
// not hold raw (DEL, the C1 controls, U+FFFE and U+FFFF) and NEL, which YAML
// reads as a line break and folds to a space inside a double-quoted scalar:
// the proposed CITATION.cff must still parse and say each of them exactly, so
// the surface clears once the proposal is adopted.
func TestYAMLRefusedCharactersAreEscaped(t *testing.T) {
	odd := "\u007f\u0080\u0085\u009f\ufffe\uffff"
	brief := "# Product\n\n## Identity (canonical)\n\n" +
		"- **Name:** acme\n" +
		"- **Title:** Acme " + odd + " Engine\n" +
		"- **Descriptor:** the tool for a\u0085b sellers " + odd + " too\n" +
		"- **Tagline:** List it once.\n" +
		"- **Listing:** Say " + odd + " to every store.\n"
	proposeAdoptAndClear(t, brief, staleCatalogue, staleCitation)
}

// TestUndecodableQuotedStringIsNeverOK holds a quoted surface whose string
// does not decode in its own format to a finding, never to ok, even when the
// block's words sit in it as written: a CITATION.cff abstract with bare inner
// quotes is not YAML, and a catalogue description with an escape JSON rejects
// is not JSON. Each reports drifted with the reason, and the proposal that
// replaces it clears it.
func TestUndecodableQuotedStringIsNeverOK(t *testing.T) {
	catalogue := `{
  "name": "acme-marketplace",
  "plugins": [
    {
      "description": "\q Say \"hi\" to every C:\\store at once.",
      "name": "acme"
    }
  ]
}
`
	// The description holds `\q`, an escape JSON rejects, beside the listing
	// spelled as JSON would spell it.
	citation := "cff-version: 1.2.0\ntitle: \"acme\"\n" +
		`abstract: "Acme "Quoted" Engine: the tool for C:\shops and "small" sellers"` + "\nlicense: MIT\n"
	root, cfg := escapeRepo(t, escapeBrief, catalogue, citation)
	rep, err := Check(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range rep.Surfaces {
		if s.Status != StatusDrifted {
			t.Errorf("an undecodable %s reads %s, want drifted", s.ID, s.Status)
		}
		if !strings.Contains(s.Reason, "does not decode") {
			t.Errorf("an undecodable %s gives the reason %q, want it to say the string does not decode", s.ID, s.Reason)
		}
	}
	proposeAdoptAndClear(t, escapeBrief, catalogue, citation)
}

const staleCatalogue = `{
  "name": "acme-marketplace",
  "plugins": [
    {
      "description": "An old description.",
      "name": "acme"
    }
  ]
}
`

const staleCitation = `cff-version: 1.2.0
title: "acme"
abstract: "An old abstract."
license: MIT
`

// escapeRepo writes a fixture holding brief, the catalogue and the citation,
// and returns it with a config of this repository's own catalogue and
// citation surfaces.
func escapeRepo(t *testing.T, brief, catalogue, citation string) (string, Config) {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	own, ok, err := LoadConfig(repo)
	if err != nil || !ok {
		t.Fatalf("LoadConfig(this repository) = ok %v, err %v", ok, err)
	}
	var surfaces []Surface
	for _, s := range own.Surfaces {
		if s.ID == "catalogue-description" || s.ID == "citation-abstract" {
			surfaces = append(surfaces, s)
		}
	}
	if len(surfaces) != 2 {
		t.Fatalf("found %d of the catalogue and citation surfaces, want 2", len(surfaces))
	}
	root := writeRepo(t, map[string]string{
		"brief.md":                        brief,
		".claude-plugin/marketplace.json": catalogue,
		"CITATION.cff":                    citation,
	})
	return root, Config{SchemaVersion: 1, Block: BlockLocation{File: "brief.md", Heading: "Identity (canonical)"}, Surfaces: surfaces}
}

// proposeAdoptAndClear proposes both surfaces from brief, holds each proposed
// file to its format and to the block's value, adopts the proposals, and
// requires both surfaces to read ok afterwards.
func proposeAdoptAndClear(t *testing.T, brief, catalogue, citation string) {
	t.Helper()
	root, cfg := escapeRepo(t, brief, catalogue, citation)
	b, err := ParseBlock(root, cfg.Block)
	if err != nil {
		t.Fatal(err)
	}

	p, err := Propose(root, cfg)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(p.Diffs) != 2 {
		t.Fatalf("Propose gave %d diffs, want one per surface: %+v", len(p.Diffs), p.Report.Surfaces)
	}
	for _, d := range p.Diffs {
		switch d.File {
		case ".claude-plugin/marketplace.json":
			var cat struct {
				Plugins []struct {
					Description string `json:"description"`
				} `json:"plugins"`
			}
			if err := json.Unmarshal([]byte(d.Proposed), &cat); err != nil {
				t.Fatalf("the proposed catalogue is not JSON: %v\n%s", err, d.Proposed)
			}
			if len(cat.Plugins) != 1 || cat.Plugins[0].Description != b.Listing {
				t.Errorf("the proposed catalogue says %+v, want the listing %q", cat.Plugins, b.Listing)
			}
		case "CITATION.cff":
			var cff struct {
				Abstract string `yaml:"abstract"`
			}
			if err := yaml.Unmarshal([]byte(d.Proposed), &cff); err != nil {
				t.Fatalf("the proposed CITATION.cff is not YAML: %v\n%s", err, d.Proposed)
			}
			if want := b.Title + ": " + b.Descriptor; cff.Abstract != want {
				t.Errorf("the proposed abstract is %q, want %q", cff.Abstract, want)
			}
		default:
			t.Fatalf("unexpected diff for %s", d.File)
		}
		if err := os.WriteFile(filepath.Join(root, d.File), []byte(d.Proposed), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rep, err := Check(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range rep.Surfaces {
		if s.Status != StatusOK {
			t.Errorf("after the proposal is adopted, %s reads %s (found %q, missing %v)", s.ID, s.Status, s.Found, s.Missing)
		}
	}
}

// The escape setting names one of the two formats, and only on a regexp
// surface: a json_field surface is always decoded, and an unknown format is
// refused rather than read as plain text.
func TestSurfaceEscapeIsValidated(t *testing.T) {
	ok := Surface{ID: "x", Kind: KindRegexp, Files: []string{"a.cff"}, Patterns: []string{`(?m)^abstract: "(.*)"$`}, Requires: []string{"title"}}
	for _, e := range []string{"", EscapeJSON, EscapeYAML} {
		s := ok
		s.Escape = e
		if err := s.validate(0); err != nil {
			t.Errorf("escape %q is refused: %v", e, err)
		}
	}
	bad := ok
	bad.Escape = "toml"
	if err := bad.validate(0); err == nil {
		t.Error("an unknown escape format is accepted")
	}
	field := Surface{ID: "m", Kind: KindJSONField, Files: []string{"plugin.json"}, Field: "description", Requires: []string{"title"}, Escape: EscapeJSON}
	if err := field.validate(0); err == nil {
		t.Error("an escape on a json_field surface is accepted")
	}
}
