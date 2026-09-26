package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// curlyQuoteGlyphs are the four typographic quotes. gofmt's doc-comment
// reformatter (go/doc/comment) rewrites a doubled backtick in doc-comment prose to
// U+201C and a doubled apostrophe to U+201D, the TeX convention, so a comment
// spelling a shell escape or a markdown code span is rewritten by the formatter
// every editor and hook runs. The result compiles, `make fmt-check` then DEMANDS
// the rewritten form, and a read-back looks right, so only a diff shows that a
// comment about quoting no longer spells what it documents
// (iss-2608301844363341). The remedy is an indented code block in the comment,
// which gofmt keeps verbatim, or a rewording.
const curlyQuoteGlyphs = "\u2018\u2019\u201c\u201d"

// curlyQuoteWriters is every Go file allowed to carry one of the glyphs, with the
// number it carries and why. The count is exact, so a substitution landing in an
// allowlisted file is caught too. Anything else holding one is a corruption.
var curlyQuoteWriters = map[string]struct {
	count  int
	reason string
}{
	"internal/core/lint/persona.go":          {1, "the attribution regexp admits a curly apostrophe in a persona name"},
	"internal/core/memory/coverage.go":       {2, "the quote-span scanner indexes the curly double-quote pair"},
	"internal/core/positioning/check.go":     {4, "the positioning normaliser maps each curly quote to its straight twin"},
	"internal/core/lint/curlyquotes_test.go": {0, "this guard spells the glyphs as escapes only"},
	"internal/core/launch/gates.go":          {9, "the narration gate tokenises and breaks clauses on the curly double quotes and apostrophe"},
}

// scanCurlyQuotes counts the glyphs in every .go file (tests included — a test
// comment is as falsifiable as any other) under root's dirs, keyed by the
// slash-separated path relative to root.
func scanCurlyQuotes(root string, dirs []string) (map[string]int, error) {
	hits := map[string]int{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			n := 0
			for _, r := range string(data) {
				if strings.ContainsRune(curlyQuoteGlyphs, r) {
					n++
				}
			}
			if n > 0 {
				rel, _ := filepath.Rel(root, path)
				hits[filepath.ToSlash(rel)] = n
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return hits, nil
}

// curlyQuoteOffenders renders every hit the allowlist does not account for.
func curlyQuoteOffenders(hits map[string]int) []string {
	var out []string
	for rel, n := range hits {
		w, ok := curlyQuoteWriters[rel]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s (carries %d curly quote(s) and is not allowlisted)", rel, n))
		case w.count != n:
			out = append(out, fmt.Sprintf("%s (carries %d curly quote(s); the allowlist names %d)", rel, n, w.count))
		}
	}
	sort.Strings(out)
	return out
}

// A planted glyph in an unlisted file is refused, and a planted extra one in a
// listed file is refused on its count: the guard is watched failing on a tree it
// did not write.
func TestCurlyQuoteGuardRefusesAPlantedGlyph(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/x/a.go", "package x\n\n// spelled '\\\u201d as the escape\n")
	writeFile(t, root, "internal/core/lint/persona.go", "package lint\n// \u2019 \u2019\n")
	writeFile(t, root, "cmd/y/clean.go", "package main\n// '\\'' stays straight\n")
	hits, err := scanCurlyQuotes(root, []string{"internal", "cmd"})
	if err != nil {
		t.Fatal(err)
	}
	got := curlyQuoteOffenders(hits)
	want := []string{
		"internal/core/lint/persona.go (carries 2 curly quote(s); the allowlist names 1)",
		"internal/x/a.go (carries 1 curly quote(s) and is not allowlisted)",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("offenders:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// The source tree carries a curly quote only where the allowlist says, and the
// allowlist names no file that no longer carries its count.
func TestNoCurlyQuotesInGoSource(t *testing.T) {
	root := filepath.Join("..", "..", "..") // internal/core/lint -> repository root
	hits, err := scanCurlyQuotes(root, []string{"internal", "cmd", "evals"})
	if err != nil {
		t.Fatal(err)
	}
	if off := curlyQuoteOffenders(hits); len(off) > 0 {
		t.Errorf("curly quotes in Go source (gofmt rewrites a doubled backtick or apostrophe in doc-comment prose to one; move the spelling into an indented code block or reword it, or allowlist a deliberate use with its count and reason):\n  %s",
			strings.Join(off, "\n  "))
	}
	for rel, w := range curlyQuoteWriters {
		if w.count > 0 && hits[rel] == 0 {
			t.Errorf("allowlist entry %s names %d curly quote(s) and the file carries none; remove the entry", rel, w.count)
		}
	}
}
