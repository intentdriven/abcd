---
schema_version: 1
id: "iss-2610160608400185"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Drop the conventional-commit fallback at 04-launch.md:496-501 and say the version derives from record impact, with an empty cut refused."
resolution: "04-launch.md's bump-tier rule drops the conventional-commit fallback: the version derives from the impact of the records entering or leaving shipped/ and resolved/, and a cut with nothing user-facing is refused as empty-cut"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/04-launch.md:496 and :501 says the patch tier covers 'a release with no intent-tied change' and that a change not tied to any intent falls back to conventional-commit derivation. Nothing in the shipped tree derives a version from conventional commits: the version comes from the records' declared impact (internal/core/changelog/impact.go), and a cut with nothing user-facing is refused as empty-cut (internal/core/release/emit.go:120).
