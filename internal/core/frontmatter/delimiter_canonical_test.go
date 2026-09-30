package frontmatter

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// delimiterSite is one allowlisted file: how many string literals spelling a
// `---` delimiter it holds, and why none of them is a second delimiter rule.
type delimiterSite struct {
	count  int
	reason string
}

// delimiterSites names every non-test Go file outside this package whose string
// literals spell a `---` delimiter at a line start, with the number of such
// literals and the reason each is not a private compare. The count is pinned so
// a compare added to a file already on the list is a new claim too. The default
// for a file this test names is to route it through IsDelimiter, Close or
// CloseAfter.
var delimiterSites = map[string]delimiterSite{
	// Writers: they emit a delimiter and judge none.
	"internal/core/capture/serialize.go":         {2, "a WRITER: buildIssueText emits the block's two delimiters; the reader side is frontmatterBounds, which asks IsDelimiter and CloseAfter"},
	"internal/core/decide/decide.go":             {2, "a WRITER: the ADR skeleton's two delimiters"},
	"internal/core/intent/create.go":             {2, "a WRITER: the minted intent's two delimiters"},
	"internal/core/intent/consistency.go":        {2, "a WRITER: the review record's two delimiters"},
	"internal/core/spec/spec.go":                 {2, "a WRITER: the minted spec's two delimiters"},
	"internal/core/report/report.go":             {4, "a WRITER: two report templates' delimiters; the report reader judges by IsDelimiter"},
	"internal/core/source/add.go":                {2, "a WRITER: a source entry's two delimiters"},
	"internal/core/lab/record.go":                {2, "a WRITER: a probe record's two delimiters"},
	"internal/core/lab/mint.go":                  {1, "a WRITER: the lab entry's block, both delimiters in one format string"},
	"internal/core/memory/schema.go":             {2, "a WRITER: rebuilds a region as a block to hand parseFrontmatter; it judges no delimiter"},
	"internal/gittest/repo.go":                   {2, "a WRITER: a test-fixture record's block's two delimiters"},
	"internal/surface/cli/history.go":            {1, "a WRITER: a separator line between rendered transcripts; not frontmatter"},
	"internal/core/positioning/render.go":        {1, "a WRITER: a unified diff's `--- a/` header; not frontmatter"},
	"internal/core/lint/subverbs.go":             {1, "a markdown TABLE's separator row (`|---|`), not a frontmatter delimiter"},
	"internal/core/implement/loop/brief.go":      {1, "a WRITER: the lane brief's thematic break above the intent section, in the body; not frontmatter"},
	"internal/core/implement/loop/issuebrief.go": {1, "a WRITER: the issue lane brief's thematic break above the issue section, in the body; not frontmatter"},
	// Deliberate, documented differences.
	"internal/core/memory/yaml.go":     {3, "the memory store's opener tolerates an indented delimiter (documented at frontmatterOpenIndex and textOpensFrontmatter); joinFileFrontmatter WRITES the block's two delimiters; every close is IsDelimiter"},
	"internal/core/memory/writer.go":   {3, "a WRITER rebuilding a region for parseFrontmatter, and a byte-0 test that leaves a page with a tolerated preamble alone because rebuilding it would drop the preamble"},
	"internal/core/history/store.go":   {6, "the transcript store's own record format, written by marshalRecord and read back byte-exact: a record this store did not write is refused, which is the point"},
	"internal/core/reading/project.go": {3, "the reading exclusion floor: it OPENS a block on any line beginning with three dashes, more broadly than IsDelimiter, and scans its keys and shapes to the LATER of its own prefix close (or `...`) and frontmatter.Close (blockScanEnd), so every line the canonical reader reads as frontmatter is scanned; its body scans start at the earlier close, so every line a renderer shows is scanned too"},
}

// TestNoPrivateDelimiterCompare is the one-canonical-primitive detector for the
// frontmatter delimiter rule, the counterpart of mdrecord's fence detector.
//
// IsDelimiter is the one delimiter rule and Close/CloseAfter the one closing
// walk. Private compares disagreed with them — a TrimSpace compare in the gates
// closed a block on an indented rule the reader reads as body, a bare prefix test
// in the site opened one on `----` — and a gate that disagrees with its reader
// about where the block ends passes the record the reader refuses
// (iss-2608270908348042).
//
// The check reads string LITERALS through go/scanner, so a comment quoting a
// delimiter is not a claim; a literal counts when its value opens with `---` or
// carries one at the start of a later line, a byte-order mark ahead of it or
// not. A delimiter assembled at run time is outside its reach, and is left to
// review.
func TestNoPrivateDelimiterCompare(t *testing.T) {
	root := filepath.Join("..", "..", "..") // internal/core/frontmatter -> repository root
	var offenders []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == filepath.Join(root, "internal", "core", "frontmatter") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			n := delimiterLiterals(data)
			if n == 0 {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			seen[rel] = true
			s, ok := delimiterSites[rel]
			switch {
			case !ok:
				offenders = append(offenders, fmt.Sprintf("%s (spells %d `---` delimiter literal(s) and is not allowlisted)", rel, n))
			case s.count != n:
				offenders = append(offenders, fmt.Sprintf("%s (spells %d `---` delimiter literal(s); the allowlist names %d)", rel, n, s.count))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("`---` delimiter literals outside frontmatter (route through IsDelimiter, Close or CloseAfter, or allowlist with a reason and the count):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	for rel := range delimiterSites {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer spells a `---` delimiter literal; remove the entry", rel)
		}
	}
}

// delimiterLiterals counts the Go string literals in src whose value opens with
// `---` or carries `---` at the start of a later line, either one optionally led
// by a byte-order mark.
func delimiterLiterals(src []byte) int {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	n := 0
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return n
		}
		if tok != token.STRING {
			continue
		}
		v, err := strconv.Unquote(lit)
		if err != nil {
			continue
		}
		if strings.HasPrefix(v, "---") || strings.Contains(v, "\n---") ||
			strings.HasPrefix(v, "\ufeff---") || strings.Contains(v, "\n\ufeff---") {
			n++
		}
	}
}

// TestDelimiterLiteralsReadsLiteralsNotComments pins the counter the detector
// stands on: a compare in code counts, a delimiter quoted in a comment does not,
// a raw string spelling a block counts once, and a delimiter led by a byte-order
// mark counts — a private compare against "\ufeff---" is a second opener rule
// that TrimBOM exists to make unnecessary, and a counter blind to it let one be
// substituted for an allowlisted literal with the pinned count unchanged.
func TestDelimiterLiteralsReadsLiteralsNotComments(t *testing.T) {
	t.Parallel()
	src := "package p\n// a comment quoting \"---\" is not a claim\nvar a = x == \"---\"\nvar b = `id: a\n---\n`\nvar c = \"-- \"\n" +
		"var d = x == \"\\ufeff---\"\nvar e = \"id: a\\n\\ufeff---\"\n"
	if got := delimiterLiterals([]byte(src)); got != 4 {
		t.Fatalf("delimiterLiterals = %d, want 4", got)
	}
}
