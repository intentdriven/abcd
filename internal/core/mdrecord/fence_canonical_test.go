package mdrecord

import (
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

// fenceTrackerRe matches the shapes a line walk keeps fence state in: a named
// flag or run, a private mask, or a boolean flipped in place.
var fenceTrackerRe = regexp.MustCompile(`\b(?:inFence|inCode|openChar|openLen|fenceOpen|fenceMask\(|FenceMask\(|isFenceLine\()|\b(\w+) = !(\w+)\b`)

// secondFenceRuleAllowed names each file that writes a fence delimiter and keeps
// state across lines WITHOUT being a second fence rule, with the reason. A new
// entry is a claim a reviewer reads; the default for a file this test names is
// to route it through Read.
var secondFenceRuleAllowed = map[string]string{}

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
// The check is deliberately crude and errs toward flagging: a non-test Go file
// outside this package that writes a fence delimiter AND tracks state across
// lines is doing this package's job. A file that only mentions a delimiter in a
// comment or judges one line alone carries no tracking and is not matched.
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
			body := string(data)
			if !fenceDelimiterRe.MatchString(body) {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			for _, m := range fenceTrackerRe.FindAllStringSubmatch(body, -1) {
				// A flip is `x = !x`; `a = !b` is an assignment, not a toggle.
				if m[1] != "" && m[1] != m[2] {
					continue
				}
				seen[rel] = true
				if _, ok := secondFenceRuleAllowed[rel]; !ok {
					offenders = append(offenders, rel+" (writes a fence delimiter and tracks "+m[0]+")")
				}
				break
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("second fence rules (route through mdrecord.Read, or allowlist with a reason):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	// An allowlist entry for a file that no longer matches is a stale claim.
	for rel := range secondFenceRuleAllowed {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer writes a fence delimiter and tracks state; remove the entry", rel)
		}
	}
}
