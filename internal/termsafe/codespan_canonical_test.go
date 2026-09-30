package termsafe

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// backtickScanRe matches a backtick sought in Go source, in the two forms a
// hand-rolled code-span pairer spells it: the rune literal a byte walk compares
// against, and a backtick string handed to a strings or bytes search.
var backtickScanRe = regexp.MustCompile("'`'|\\b(?:strings|bytes)\\.(?:Index|LastIndex|Count|Cut|Split)[A-Za-z]*\\([^)\\n]*\"`\"")

// backtickScanner is one allowlisted file: how many backtick scans it holds and
// why none of them is a second code-span pairer.
type backtickScanner struct {
	count  int
	reason string
}

// backtickScanners names every non-test Go file outside this package that
// seeks a backtick, with the number of scans it holds and the reason it is not
// a second pairer. The count is pinned, so a scan added to a file already on
// the list is a new claim too: a pairer written into an allowlisted file
// changes its count and fails until a reviewer reads the new reason. The
// default for a file this test names is to pair through PairCodeSpan.
var backtickScanners = map[string]backtickScanner{
	"internal/adapter/scanner/identity.go":        {1, "a delimiter set: a backtick is one of the characters that may end an identity token; nothing is paired"},
	"internal/core/capture/promote.go":            {1, "a WRITER: codeSpan measures the longest backtick run to choose a fence the value cannot close; nothing is paired"},
	"internal/core/guard/tokenize.go":             {23, "the shell tokenizer: a backtick there is command substitution, a shell grammar, not markdown"},
	"internal/core/guard/payload.go":              {4, "buildsName reads a shell line's raw text for a name an expansion builds: a backtick there opens or closes a command substitution, a shell grammar, not markdown; nothing is paired"},
	"internal/core/guard/unknown.go":              {2, "spellWord spells a default's or an alternative's shell word, and readPattern reads a trim's or a replacement's pattern: a backtick in either opens a command substitution, whose output the spelling drops or the pattern reads as unknown text; nothing is paired"},
	"internal/core/history/reconstruct_render.go": {1, "a WRITER: longestBacktickRun sizes a fence longer than any run in the body; nothing is paired"},
	"internal/core/ideate/render.go":              {1, "blockText asks whether a value opens with a backtick, then asks OpensBalancedCodeSpan, which pairs through PairCodeSpan"},
	"internal/core/lifeboat/mdrender.go":          {1, "escapeLeadingMarker asks whether a value opens with a backtick, then asks OpensBalancedCodeSpan, which pairs through PairCodeSpan"},
	"internal/core/lint/lint.go":                  {2, "stripInlineCode walks to each run and pairs it through PairCodeSpan, stepping over an unpaired run whole"},
	"internal/core/mdrecord/mdrecord.go":          {4, "OpensComment and CodeSpanRanges walk to each run and pair it through PairCodeSpan, runEnd steps over an unpaired run, and the fence opener refuses an info string holding a backtick"},
	"internal/core/mdrender/render.go":            {2, "the site renderer's inline walk hands each run to PairCodeSpan; the other is the escapable-punctuation range, which names a backtick among the ASCII punctuation"},
	"internal/core/surface/appendix.go":           {2, "codeRegions walks to each run and pairs it through PairCodeSpan, stepping over an unpaired run whole"},
}

// TestNoSecondCodeSpanPairer is the one-canonical-primitive detector for the
// code-span rule, the counterpart of mdrecord's TestNoSecondFenceRule.
//
// PairCodeSpan is the tree's one code-span pairer. The site renderer, the prose
// cleaner, the record's comment and span readers and the block escapers each
// paired runs by their own walk, and the renderer's closed a span where the
// opening run's text first recurred: it refused a span the escapers had judged
// balanced and left unescaped, failing the page (iss-2609262322244502). The
// guard's own first sweep found two more walks pairing single backticks inside
// longer runs (iss-2609262350446885).
//
// The check is spelling-only, as the fence detector's is: every file that seeks
// a backtick is named with a reason, which is a short list because most of the
// tree never seeks one. A backtick assembled at run time, or spelled as a
// number, is outside its reach and is left to review.
func TestNoSecondCodeSpanPairer(t *testing.T) {
	root := filepath.Join("..", "..") // internal/termsafe -> repository root
	self := filepath.Join(root, "internal", "termsafe")
	var offenders []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == self {
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
			n := len(backtickScanRe.FindAllIndex(data, -1))
			if n == 0 {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			seen[rel] = true
			w, ok := backtickScanners[rel]
			switch {
			case !ok:
				offenders = append(offenders, fmt.Sprintf("%s (seeks a backtick %d time(s) and is not allowlisted)", rel, n))
			case w.count != n:
				offenders = append(offenders, fmt.Sprintf("%s (seeks a backtick %d time(s); the allowlist names %d)", rel, n, w.count))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("backtick scans outside termsafe (pair through PairCodeSpan, or allowlist with a reason and the count):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	// An allowlist entry for a file that no longer seeks a backtick is a stale
	// claim.
	for rel := range backtickScanners {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer seeks a backtick; remove the entry", rel)
		}
	}
}
