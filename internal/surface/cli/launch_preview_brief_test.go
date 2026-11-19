package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// The launch chapter's paragraph on the preview once opened by stating its
// shape: spelled `dry-run`, a flag and not a sub-verb, registered as no
// subcommand (iss-2610262033108186). itd-147 keeps every flag and sub-verb in
// the generated appendix, and TestSurfaceChapterProseStatesNoShape matches only
// the double-dash spelling, so the bare one passed it. The paragraph names no
// spelling and no shape and points at the appendix for them.
func TestLaunchPreviewParagraphStatesNoShape(t *testing.T) {
	rel := surface.BriefSurfacesDir + "/04-launch.md"
	raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	prose, err := surface.SplitChapter(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	var para string
	for _, p := range strings.Split(prose, "\n\n") {
		if p = strings.Join(strings.Fields(p), " "); strings.Contains(p, `It is **not** "ship minus publish"`) {
			para = p
			break
		}
	}
	if para == "" {
		t.Fatalf("%s: no paragraph says the preview is not ship minus publish", rel)
	}
	for _, banned := range []string{"`dry-run`", "spelled", "subcommand", "a flag, not a sub-verb", "names it as a flag"} {
		if strings.Contains(para, banned) {
			t.Errorf("%s: the preview paragraph states its shape (%q); the appendix alone states flags and sub-verbs", rel, banned)
		}
	}
	if !strings.Contains(para, "appendix") {
		t.Errorf("%s: the preview paragraph does not point at the generated appendix for the preview's spelling", rel)
	}
}
