package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemoryIngestSanitizesLicence is the gh-262 detector: IngestResult.Licence
// is free text lifted verbatim from the ingested source's bytes (an
// SPDX-License-Identifier line here), not a validated SPDX token, so a licence
// carrying raw terminal control/escape/bidi/zero-width runes must reach the TTY
// defanged — matching the sibling memory render fields — not raw. Attack runes
// are built numerically so this source file carries none of the invisible
// characters it defends against.
func TestMemoryIngestSanitizesLicence(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)

	attacks := map[string]rune{
		"ESC":          0x1b,
		"BEL":          0x07,
		"C1 CSI":       0x9b,
		"RLO override": 0x202e,
	}
	// SPDX-License-Identifier: MIT<ESC>]0;pwned<BEL><CSI>...<RLO> — the licence
	// value carries every attack rune; the regex capture excludes only CR/LF.
	licenceTail := "]0;pwned"
	poisoned := "MIT"
	for _, r := range attacks {
		poisoned += string(r)
	}
	poisoned += licenceTail

	src := filepath.Join(repo, "article.txt")
	body := "SPDX-License-Identifier: " + poisoned + "\nRotate tokens every 24 hours.\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	pages := filepath.Join(repo, "pages.json")
	if err := os.WriteFile(pages, []byte(`[{"type":"topic","domain":"auth","slug":"tokens","body":"# Rotation\nRotate tokens every 24 hours."}]`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"memory", "ingest", src, "--pages-json", pages}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected a clean ingest\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "licence:") {
		t.Fatalf("gh-262: expected a licence line in the render:\n%s", out)
	}
	for name, r := range attacks {
		if strings.ContainsRune(out, r) {
			t.Errorf("gh-262: ingest render carries a raw %s (U+%04X) attack rune from the licence field; it must be defanged like the sibling memory render fields\noutput: %q", name, r, out)
		}
	}
	// The legitimate MIT token and the visible-but-inert tail must survive.
	if !strings.Contains(out, "MIT") || !strings.Contains(out, licenceTail) {
		t.Fatalf("gh-262: sanitising must not drop the legitimate licence content:\n%s", out)
	}
}
