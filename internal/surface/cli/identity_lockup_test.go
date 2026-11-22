package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/positioning"
)

// TestIdentityPrintsTheLockupFields holds the identity report and onboarding's
// summary to every field the block carries, in the block's own order: a block
// with the lockup fields prints its name, descriptor, purpose and listing line
// beside the title, tagline and pitch, and a block written before those fields
// existed prints only what it holds (iss-2610311024153333).
func TestIdentityPrintsTheLockupFields(t *testing.T) {
	full := positioning.Block{
		Name:       "acme",
		Title:      "Automated Catalogue Maintenance Engine",
		Descriptor: "the catalogue tool for small shops",
		Tagline:    "List it once. Sell it everywhere.",
		Purpose:    "Keep every storefront's catalogue true to the one you keep.",
		Pitch:      "A command-line tool that syncs one product list.",
		Listing:    "List your products once and sell them on every marketplace.",
		File:       "brief.md",
		Line:       3,
	}
	order := []string{
		"name:", "acme",
		"title:", "Automated Catalogue Maintenance Engine",
		"descriptor:", "the catalogue tool for small shops",
		"tagline:", "List it once. Sell it everywhere.",
		"purpose:", "Keep every storefront's catalogue true to the one you keep.",
		"pitch:", "A command-line tool that syncs one product list.",
		"listing:", "List your products once and sell them on every marketplace.",
	}
	var rep, ini bytes.Buffer
	renderIdentity(&rep, positioning.Report{Block: full, Severity: "warn"})
	renderIdentityInit(&ini, positioning.InitResult{Block: full, Location: positioning.BlockLocation{File: "brief.md", Heading: "Identity (canonical)"}})
	for name, out := range map[string]string{"identity": rep.String(), "identity init": ini.String()} {
		at := 0
		for _, want := range order {
			i := strings.Index(out[at:], want)
			if i < 0 {
				t.Errorf("%s output lacks %q after byte %d:\n%s", name, want, at, out)
				break
			}
			at += i + len(want)
		}
	}

	var old bytes.Buffer
	renderIdentity(&old, positioning.Report{Block: positioning.Block{Title: "acme", Tagline: "Widgets.", File: "brief.md"}, Severity: "warn"})
	for _, absent := range []string{"name:", "descriptor:", "purpose:", "pitch:", "listing:"} {
		if strings.Contains(old.String(), absent) {
			t.Errorf("a block without the field prints %q:\n%s", absent, old.String())
		}
	}
	for _, want := range []string{"title:", "acme", "tagline:", "Widgets."} {
		if !strings.Contains(old.String(), want) {
			t.Errorf("a block without the lockup fields lacks %q:\n%s", want, old.String())
		}
	}
}

// TestIdentityFieldColumnFitsTheBlock holds the value column to the labels the
// block actually prints: a block without the lockup fields prints byte for byte
// as it did before they existed, and onboarding's wrote: lines sit in the same
// column as the fields above them, whichever fields the block carries.
func TestIdentityFieldColumnFitsTheBlock(t *testing.T) {
	var old bytes.Buffer
	renderBlockFields(&old, positioning.Block{Title: "acme", Tagline: "Widgets.", Pitch: "A shop."})
	if want := "  title:   acme\n  tagline: Widgets.\n  pitch:   A shop.\n"; old.String() != want {
		t.Errorf("an old block prints\n%q\nwant\n%q", old.String(), want)
	}

	for _, blk := range []positioning.Block{
		{Title: "acme", Tagline: "Widgets."},
		{Name: "acme", Title: "Acme", Descriptor: "a shop", Tagline: "Widgets.", Listing: "Buy widgets."},
	} {
		var out bytes.Buffer
		renderIdentityInit(&out, positioning.InitResult{Block: blk, Wrote: []string{".abcd/positioning.json"},
			Location: positioning.BlockLocation{File: "brief.md", Heading: "Identity (canonical)"}})
		col := -1
		for _, line := range strings.Split(out.String(), "\n") {
			label, rest, ok := strings.Cut(strings.TrimPrefix(line, "  "), ":")
			if !ok || !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") || strings.Contains(label, " ") {
				continue
			}
			c := len(line) - len(strings.TrimLeft(rest, " "))
			if col == -1 {
				col = c
			} else if c != col {
				t.Errorf("value column %d on %q, want %d like the line above:\n%s", c, line, col, out.String())
			}
		}
		if col == -1 {
			t.Fatalf("no field lines in:\n%s", out.String())
		}
	}
}

// TestIdentityPrintsWhyAQuotedSurfaceDrifted holds the report to the reason a
// surface carries when it drifted for something other than a missing field: a
// quoted string that does not decode.
func TestIdentityPrintsWhyAQuotedSurfaceDrifted(t *testing.T) {
	var out bytes.Buffer
	renderIdentity(&out, positioning.Report{
		Block:    positioning.Block{Title: "acme", Tagline: "Widgets.", File: "brief.md"},
		Severity: "warn",
		Surfaces: []positioning.SurfaceResult{{ID: "citation-abstract", Status: positioning.StatusDrifted, File: "CITATION.cff", Line: 3,
			Found: `A "b" c`, Canonical: "acme", Reason: "the quoted YAML string does not decode: yaml: line 1: did not find expected key"}},
	})
	if !strings.Contains(out.String(), "reason:    the quoted YAML string does not decode") {
		t.Errorf("the report does not give the reason:\n%s", out.String())
	}
}
