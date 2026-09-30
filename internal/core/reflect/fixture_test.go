package reflect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	shippedDir = ".abcd/development/intents/shipped/"
	plannedDir = ".abcd/development/intents/planned/"
)

// auditedNotes is an Audit Notes section in the shape the intent auditor's
// ingest writes (internal/core/intent/audit.go ingestedBlock). The rationale
// text is distinctive so a test can prove the writer never copies it.
const auditedNotes = `## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0123456789ab -->
Fidelity review — receipt rcp-0123456789ab (verifier intent-fidelity-reviewer fixture).

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: RATIONALE-NEVER-COPIED the seed builder reads the tag
  evidence: internal/core/reflect/seed.go:1 — "package reflect"

Gap audit:
- honoured:
  - the seed opens from the release
    evidence: x.go:1 — "y"
  - the refusal names the tag
- diverged:
  - one link is relative
- missing: (none)
<!-- abcd-review: END receipt=rcp-0123456789ab -->
`

const placeholderNotes = "## Audit Notes\n\n_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._\n\n### Implementation notes\n\nA subsection is not an audit.\n"

const owedNotes = "## Audit Notes\n\n<!-- abcd-review: OWED receipt=rcp-aaaaaaaaaaaa -->\nFidelity review OWED (receipt rcp-aaaaaaaaaaaa).\n"

func intentDoc(id, impact, extraFM, title, notes string) string {
	return "---\nid: " + id + "\nimpact: " + impact + "\n" + extraFM + "---\n\n# " + title + "\n\n## Press Release\n\n> A line.\n\n" + notes
}

// releaseRepo builds a history with three tags:
//
//	v0.1.0 ships itd-1.
//	v0.2.0 ships itd-2 (audited), itd-3 (placeholder notes) and itd-8 (owed
//	       review); itd-4 also reaches shipped/ in that window but says
//	       `shipped_in: v0.1.0` (a hygiene sweep), so it is v0.1.0's. itd-5 is
//	       planned with `target_release: v0.2.0` and never shipped; itd-6 targets
//	       v0.3.0. The changelog carries a dated v0.2.0 section.
//	v0.3.0 ships nothing.
//
// After v0.3.0, itd-7 reaches shipped/ stamped `shipped_in: v0.2.0`, so it
// belongs to v0.2.0 although the tag's tree never held it.
func releaseRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(shippedDir+"itd-1-first.md", intentDoc("itd-1", "additive", "", "The first promise", auditedNotes))
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-09-01\n\n- The first promise (itd-1).\n")
	r.Commit("first release")
	r.Git("tag", "v0.1.0")

	r.Write(shippedDir+"itd-2-second.md", intentDoc("itd-2", "additive", "", "The second promise", auditedNotes))
	r.Write(shippedDir+"itd-3-third.md", intentDoc("itd-3", "fix", "", "The third promise", placeholderNotes))
	r.Write(shippedDir+"itd-8-eighth.md", intentDoc("itd-8", "fix", "", "The eighth promise", owedNotes))
	r.Write(shippedDir+"itd-4-swept.md", intentDoc("itd-4", "fix", "shipped_in: v0.1.0\n", "An old promise swept late", auditedNotes))
	r.Write(plannedDir+"itd-5-late.md", "---\nid: itd-5\ntarget_release: v0.2.0\n---\n\n# A promise that missed its release\n")
	r.Write(plannedDir+"itd-6-future.md", "---\nid: itd-6\ntarget_release: v0.3.0\n---\n\n# A promise for later\n")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-09-20\n\n### Added\n\n- The second promise (itd-2).\n\n## [0.1.0] - 2026-09-01\n\n- The first promise (itd-1).\n")
	r.Commit("second release")
	r.Git("tag", "v0.2.0")

	r.Commit("a release that shipped nothing")
	r.Git("tag", "v0.3.0")

	r.Write(shippedDir+"itd-7-stamped.md", intentDoc("itd-7", "additive", "shipped_in: v0.2.0\n", "A promise stamped later", ""))
	r.Commit("a sweep stamps itd-7 with the release that carried it")
	return r
}

func exists(t *testing.T, p string) bool {
	t.Helper()
	_, err := os.Lstat(p)
	return err == nil
}

func abs(r *gittest.Repo, rel string) string {
	return filepath.Join(r.Root(), filepath.FromSlash(rel))
}
