---
schema_version: 1
id: "iss-2609251939461459"
slug: "the-release-gate-s-content-commit-derivation-binds-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
resolution: "The derivation refuses a released tree with no CHANGELOG.md or no dated release heading instead of matching two empty versions."
impact: fix
resolved_by:
  commit: "2ccf16b73b3b870a493bf9a526dac30cbb330c9c"
---

The release gate's content-commit derivation binds the receipts to the released version with a string equality that admits two empty versions: a released tree with no dated CHANGELOG heading, or with no CHANGELOG.md at all, derives the nearest receipts directory, and the previous release's PROMOTE receipts admit it. releaseVersionAt returns an empty version with no error for a missing file, against its own comment. Reachable on the hand-pushed-tag path, where the verify job's receipts step runs and nothing cross-checks the tag against the CHANGELOG.

## Grounds

- pursued: a released tree naming no version refuses with a fail-closed message naming CHANGELOG (TestDeriveReleaseContentSha_RefusesAReleasedTreeWithNoVersion, review probes S6 and S6b); a derived sha for either shape would show it wrong.
