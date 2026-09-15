package memory

import (
	"strings"
	"testing"
)

// TestDerivedClassesUnionsBothShapes pins the lint path's class derivation to
// the UNION of the two shapes. derivedClasses used to read the plural
// `source.classes` list in an else-if that shadowed the scalar `source.class`
// entirely, so a page declaring both — `class: external_pdf` beside
// `classes: [session_memory]`, with no licence — presented only the harmless
// internal class to every check that shares this derivation, and the ML001
// blocker passed it at exit 0. The write-path exclusivity check
// (validateSourceBlock, which refuses a block mixing the shapes) never runs on
// the lint path, so nothing else catches the combination: lint reads a page's
// raw on-disk bytes by design.
//
// The union is the fail-closed and honest reading — "the classes this page
// declares" — and it must not depend on which key the author wrote first.
func TestDerivedClassesUnionsBothShapes(t *testing.T) {
	hash := strings.Repeat("a", 64)

	// The regression: both shapes present, the scalar one external, no licence.
	unlicensed := "---\nsource:\n  class: external_pdf\n  classes: [session_memory]\n  source_hash: " + hash + "\n---\n# Body\n"
	lr := lintPage(t, "note_privacy_gdpr.md", unlicensed)
	if !hasBlocker(lr, "ML001") || lr.ExitCode != 1 {
		t.Fatalf("a page whose external scalar class was shadowed by a plural list passed the licence blocker: exit=%d findings=%+v", lr.ExitCode, lr.Findings)
	}

	// Key order must not decide the outcome.
	reordered := "---\nsource:\n  classes: [session_memory]\n  class: external_pdf\n  source_hash: " + hash + "\n---\n# Body\n"
	if lr := lintPage(t, "note_privacy_gdpr.md", reordered); !hasBlocker(lr, "ML001") {
		t.Fatalf("declaration order changed the verdict: %+v", lr.Findings)
	}

	// Control: the same both-shapes page WITH a licence is clean. It mixes two
	// classes, so it also carries the weighting_note MS002 requires — which is
	// the union's visible effect on the checks that share this derivation, and
	// the honest one: the page really does declare two classes.
	licensed := "---\nsource:\n  class: external_pdf\n  classes: [session_memory]\n  licence: MIT\n" +
		"  weighting_note: pdf outweighs the session note\n  source_hash: " + hash + "\n---\n# Body\n"
	lrOK := lintPage(t, "note_privacy_gdpr.md", licensed)
	if hasBlocker(lrOK, "ML001") || lrOK.ExitCode != 0 {
		t.Fatalf("a licensed both-shapes page was blocked: exit=%d findings=%+v", lrOK.ExitCode, lrOK.Findings)
	}

	// Unit-level: the union is deduplicated, and a class named by both shapes
	// appears once.
	l := newMemoryLinter("p.md", t.TempDir(),
		"---\nsource:\n  class: external_pdf\n  classes: [external_pdf, session_memory]\n---\n# Body\n", nil)
	got := l.derivedClasses()
	want := []string{"external_pdf", "session_memory"}
	if len(got) != len(want) {
		t.Fatalf("derivedClasses = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("derivedClasses = %v, want %v", got, want)
		}
	}
}
