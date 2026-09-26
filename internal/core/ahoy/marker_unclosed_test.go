package ahoy

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// A file with no frontmatter and no live H1 takes the block at end of file,
// and a fence (or comment) nothing closes runs to end of file: the block was
// appended INSIDE the open span, where no reader sees it as markdown and the
// byte regex still classifies it current. The install now refuses the file,
// leaves it untouched, and detection reports it as a gap to resolve by hand.
func TestMarkerRefusesToAppendInsideAnUnclosedSpan(t *testing.T) {
	for name, original := range map[string]string{
		"an unclosed fence":   "intro\n\n```\nunclosed code\n",
		"an unclosed tilde":   "~~~\nunclosed code\n",
		"an unclosed comment": "intro\n\n<!--\nparked text\n",
	} {
		path := filepath.Join(t.TempDir(), "CLAUDE.md")
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, ok := installMarkerFile(path); ok {
			t.Errorf("%s: install reported ok", name)
		}
		if got, _ := os.ReadFile(path); !bytes.Equal(got, []byte(original)) {
			t.Errorf("%s: the file was changed:\n%s", name, got)
		}
		if st := classifyMarker(path); st != markerUnplaceable {
			t.Errorf("%s: classifyMarker = %q, want %q", name, st, markerUnplaceable)
		}
		if _, err := EnsureMarker(path, true); err == nil {
			t.Errorf("%s: the embark probe predicted a write it cannot make", name)
		}
	}
	// An H1 above the unclosed span still takes the block after it: the span
	// never reaches the insertion point.
	path := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# Title\n\n```\nunclosed code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if wrote, ok := installMarkerFile(path); !ok || !wrote {
		t.Errorf("an H1 above an unclosed fence: wrote=%v ok=%v", wrote, ok)
	}
}
