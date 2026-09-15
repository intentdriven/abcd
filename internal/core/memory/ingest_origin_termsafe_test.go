package memory

import (
	"os"
	"strings"
	"testing"
)

// ingest_origin_termsafe_test.go — iss-357. materialFromFetched sets the stored
// material.origin to the fetched FinalURL, which is redirect-controlled: net/url
// preserves raw non-ASCII in the query, so C1/bidi/zero-width runes in a hostile
// Location reach the durable sources registry (JSON, not the frontmatter dumper)
// and the returned citation verbatim — the exact class iss-359 fixed one door
// over in the cite fetcher. The origin must get iss-359's treatment
// (percent-encode the hidden runes losslessly) before it is stored. Attack runes
// are numeric so this file carries none of them.
func TestIngestEncodesHiddenRunesInFetchedOrigin(t *testing.T) {
	repo := t.TempDir()

	attacks := map[string]rune{
		"ESC":          0x1b,
		"C1 CSI":       0x9b,
		"RLO override": 0x202e,
		"zero-width":   0x200b,
	}
	poison := "https://example.com/a?q=v"
	for _, r := range attacks {
		poison += string(r)
	}

	fetcher := func(string) (FetchedSource, error) {
		return FetchedSource{
			FinalURL: poison,
			Headers:  map[string]string{"Content-Type": "text/plain"},
			Body:     []byte("Rotate tokens every 24 hours.\n"),
		}, nil
	}
	echoDistiller := func(normalised string, _ map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": "topic", "domain": "auth", "slug": "tokens", "body": normalised,
		}}, nil
	}

	res, err := Ingest(IngestRequest{
		RepoRoot:  repo,
		Source:    "https://example.com/a",
		Distiller: echoDistiller,
		Fetcher:   fetcher,
		Now:       fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}

	// The durable registry is JSON and does NOT pass through the frontmatter
	// dumper, so a raw hidden rune there is the iss-357 leak proper.
	raw, err := os.ReadFile(SourcesIndexPath(repo))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	for name, r := range attacks {
		if strings.ContainsRune(string(raw), r) {
			t.Errorf("iss-357: sources registry stores a raw %s (U+%04X) from the redirect-controlled origin; it must be percent-encoded like iss-359\nregistry:\n%s", name, r, raw)
		}
		if o, _ := res.Citation["origin"].(string); strings.ContainsRune(o, r) {
			t.Errorf("iss-357: returned citation origin carries a raw %s (U+%04X): %q", name, r, o)
		}
	}
	// The legitimate address survives (lossless): scheme/host/query still present.
	if o, _ := res.Citation["origin"].(string); !strings.Contains(o, "example.com") || !strings.Contains(o, "q=v") {
		t.Fatalf("iss-357: encoding must not drop the legitimate address: %q", o)
	}
}
