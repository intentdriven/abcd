package mdrecord

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// fenceDelimiterRe matches a fence delimiter written into Go source, in either
// form a hand-rolled tracker spells it: the literal three-character run a
// HasPrefix compares against, and the `{3,} / ~{3,} quantifier a regexp uses.
var fenceDelimiterRe = regexp.MustCompile("(?:```|~~~|`\\{3|~\\{3)")

// fenceWriter is one allowlisted file: how many delimiter occurrences it holds
// and why none of them is a second fence rule.
type fenceWriter struct {
	count  int
	reason string
}

// fenceWriters names every non-test Go file outside this package that writes a
// fence delimiter, with the number of delimiter occurrences it holds and the
// reason it is not a second fence rule. The count is pinned so a delimiter
// added to a file already on the list is a new claim too: a tracker written
// into an allowlisted file changes its count and fails until a reviewer reads
// the new reason. The default for a file this test names is to route it
// through Read.
var fenceWriters = map[string]fenceWriter{
	"internal/adapter/openaiapi/client.go":          {3, "judges one model answer whole: unfence strips a single fence wrapping the entire answer, and refuses to when another delimiter sits inside; it reads no document and tracks no lines"},
	"internal/adapter/scanner/scanner.go":           {1, "a comment quoting a regexp quantifier (`{36,}`); no delimiter is written or read"},
	"internal/core/glossary/index.go":               {2, "a WRITER: RenderLayout wraps the generated layout tree in one fence; it reads no fences"},
	"internal/core/history/reconstruct_render.go":   {1, "a WRITER: writeFenced opens a fence longer than any backtick run in the body, the floor of three; it reads no fences"},
	"internal/core/lifeboat/sources_conventions.go": {3, "judges one line or the whole text: a README prose measure skips a delimiter line, and a presence test asks whether any fence exists; neither tracks which lines a fence covers"},
	"internal/core/reading/project.go":              {2, "fenceDelimiterRe judges one frontmatter line and refuses it; which lines a fence covers is floorFences, which reads mdrecord"},
	"internal/core/release/page.go":                 {2, "a presence test: a headline carrying any delimiter is refused; it tracks nothing"},
	"internal/core/site/compose.go":                 {1, "judges one block the site's Blocks already cut by mdrecord's reading: does it open with a fence"},
	"internal/core/site/markdown.go":                {10, "the site renderer: it renders a block Blocks already cut by mdrecord's reading, refuses a fence form it does not render (tilde, four or more backticks), a list line that opens a fence and an indented code block; it keeps no fence state of its own"},
	"internal/surface/cli/history_reconstruct.go":   {2, "a WRITER: the telemetry block is emitted inside one json fence; it reads no fences"},
	"internal/surface/cli/reference.go":             {4, "a WRITER: the reference page emits its flag and example blocks as fences; it reads no fences"},
}

// TestNoSecondFenceRule is the one-canonical-primitive detector for the fence
// rule, the counterpart of fsutil's atomic-write and guarded-read detectors.
//
// mdrecord is the tree's one notion of a fence and an HTML comment. Private
// toggles disagreed with it — on tildes, on closing-run length, on an info
// string after a closer, on indentation — and a reader and a gate that disagree
// about where a code block ends disagree about which lines are content, which is
// the question each of them is asking. Three captures in one sweep were that
// disagreement (iss-2609250955207041, iss-2609250955051598,
// iss-2609250955209513), one of them the reading exclusion floor.
//
// The check is delimiter-only. It once also required a tracker spelling (a
// named flag, a boolean flipped in place), and two common styles of a toggle —
// run tracking kept in a variable called `open`, and a flip written as an
// if/else — carried none of them and escaped (iss-2609251510124162). Every
// writer of a delimiter is now named with a reason, which is a short list: most
// of the tree never spells one. A delimiter assembled at run time rather than
// written as a literal is outside its reach, and is left to review.
func TestNoSecondFenceRule(t *testing.T) {
	root := filepath.Join("..", "..", "..") // internal/core/mdrecord -> repository root
	var offenders []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == filepath.Join(root, "internal", "core", "mdrecord") {
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
			n := len(fenceDelimiterRe.FindAllIndex(data, -1))
			if n == 0 {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			seen[rel] = true
			w, ok := fenceWriters[rel]
			switch {
			case !ok:
				offenders = append(offenders, fmt.Sprintf("%s (writes %d fence delimiter(s) and is not allowlisted)", rel, n))
			case w.count != n:
				offenders = append(offenders, fmt.Sprintf("%s (writes %d fence delimiter(s); the allowlist names %d)", rel, n, w.count))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("fence delimiters outside mdrecord (route through mdrecord.Read, or allowlist with a reason and the count):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	// An allowlist entry for a file that no longer writes a delimiter is a
	// stale claim.
	for rel := range fenceWriters {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer writes a fence delimiter; remove the entry", rel)
		}
	}
}
