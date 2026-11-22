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
