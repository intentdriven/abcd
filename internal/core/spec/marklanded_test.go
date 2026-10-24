package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const markSpec = "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Steps\n\n" +
	"1. The parser\n   - packages: internal/core/spec\n   - tests: the parser\n\n" +
	"2. The loop\n   - packages: internal/core/implement\n   - landed: null\n" +
	"3. The page\n\n## Progress\n\nNothing yet.\n"

// TestMarkStepLandedWritesTheLineUnderTheStep (iss-2610110918418459): the
// landing marks the step it lands, so a later build or a --remainder close
// reads it as landed. The line goes under the step, after its own lines; a
// null landed line is replaced; every other line is kept as written.
func TestMarkStepLandedWritesTheLineUnderTheStep(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "   - tests: the parser\n   - landed: abc123\n\n2. The loop"},
		{2, "   - packages: internal/core/implement\n   - landed: abc123\n3. The page"},
		{3, "3. The page\n   - landed: abc123\n\n## Progress"},
	}
	for _, tc := range cases {
		got, changed, err := MarkStepLanded(markSpec, tc.n, "abc123")
		if err != nil || !changed {
			t.Fatalf("step %d: marking is a change: changed=%v err=%v", tc.n, changed, err)
		}
		if !strings.Contains(got, tc.want) {
			t.Fatalf("step %d: want %q in:\n%s", tc.n, tc.want, got)
		}
		steps, err := ParseSteps(got)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range steps {
			if wantLanded := s.Number == tc.n; (s.Landed == "abc123") != wantLanded {
				t.Fatalf("step %d: step %d reads landed %q", tc.n, s.Number, s.Landed)
			}
		}
		if n := strings.Count(got, "\n") - strings.Count(markSpec, "\n"); n != 1 && !(tc.n == 2 && n == 0) {
			t.Fatalf("step %d: the mark adds one line, or replaces the null one; the line count moved by %d", tc.n, n)
		}
	}
}

// TestMarkStepLandedKeepsAMarkAndRefusesAnother: the same value again changes
// nothing; another value over a landed step is refused, never overwritten; an
// unlisted step and an unstepped spec are refused; CRLF endings are kept.
func TestMarkStepLandedKeepsAMarkAndRefusesAnother(t *testing.T) {
	once, _, err := MarkStepLanded(markSpec, 1, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if again, changed, err := MarkStepLanded(once, 1, "abc123"); err != nil || changed || again != once {
		t.Fatalf("the same mark again changes nothing: changed=%v err=%v", changed, err)
	}
	if _, _, err := MarkStepLanded(once, 1, "def456"); err == nil || !strings.Contains(err.Error(), "abc123") {
		t.Fatalf("another mark over a landed step is refused naming the one it has: %v", err)
	}
	for _, n := range []int{0, 4} {
		if _, _, err := MarkStepLanded(markSpec, n, "abc123"); err == nil {
			t.Fatalf("step %d is not listed, and is refused", n)
		}
	}
	if _, _, err := MarkStepLanded("# alpha\n\n## Summary\n\nx\n", 1, "abc123"); err == nil {
		t.Fatal("a spec that lists no steps has none to mark")
	}
	for _, v := range []string{"", "a\nb", "null"} {
		if _, _, err := MarkStepLanded(markSpec, 1, v); err == nil {
			t.Fatalf("value %q is refused", v)
		}
	}
	crlf := strings.ReplaceAll(markSpec, "\n", "\r\n")
	got, _, err := MarkStepLanded(crlf, 1, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "\n") != strings.Count(got, "\r\n") || !strings.Contains(got, "   - landed: abc123\r\n") {
		t.Fatalf("CRLF endings are kept:\n%q", got)
	}
}

// TestMarkLandedRewritesTheOpenSpecInPlace: the store form reads the open
// spec, marks it and writes it back, keeping its mode; a closed spec is refused.
func TestMarkLandedRewritesTheOpenSpecInPlace(t *testing.T) {
	root := t.TempDir()
	open := filepath.Join(root, SpecsRelDir, StatusOpen)
	if err := os.MkdirAll(open, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(open, "spc-1-alpha.md")
	if err := os.WriteFile(p, []byte(markSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := MarkLanded(root, "spc-1", 1, "abc123")
	if err != nil || !changed {
		t.Fatalf("MarkLanded: changed=%v err=%v", changed, err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "   - landed: abc123\n") {
		t.Fatalf("the open spec carries the mark:\n%s", b)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("the spec keeps its mode: %v", fi.Mode().Perm())
	}
	if _, err := Close(root, "spc-1"); err != nil {
		t.Fatal(err)
	}
	if changed, err := MarkLanded(root, "spc-1", 1, "abc123"); err != nil || changed {
		t.Fatalf("a closed spec's mark already there is kept, so a resumed landing goes on: changed=%v err=%v", changed, err)
	}
	if _, err := MarkLanded(root, "spc-1", 2, "abc123"); err == nil {
		t.Fatal("a closed spec is not marked")
	}
}

// TestAMarkedStepLeavesTheRemainder: the remainder copy a --remainder close
// writes carries only the steps the marks leave unlanded.
func TestAMarkedStepLeavesTheRemainder(t *testing.T) {
	marked, _, err := MarkStepLanded(markSpec, 1, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	steps, err := ParseSteps(marked)
	if err != nil {
		t.Fatal(err)
	}
	carried, _ := CarryUnlanded(steps)
	if len(carried) != 2 || carried[0].Title != "The loop" || carried[1].Title != "The page" {
		t.Fatalf("the remainder carries steps 2 and 3 only: %+v", carried)
	}
}
