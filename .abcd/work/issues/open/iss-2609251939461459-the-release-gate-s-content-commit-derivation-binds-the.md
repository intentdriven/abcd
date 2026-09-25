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
---

The release gate's content-commit derivation binds the receipts to the released version with a string equality that admits two empty versions: a released tree with no dated CHANGELOG heading, or with no CHANGELOG.md at all, derives the nearest receipts directory, and the previous release's PROMOTE receipts admit it. releaseVersionAt returns an empty version with no error for a missing file, against its own comment. Reachable on the hand-pushed-tag path, where the verify job's receipts step runs and nothing cross-checks the tag against the CHANGELOG.
