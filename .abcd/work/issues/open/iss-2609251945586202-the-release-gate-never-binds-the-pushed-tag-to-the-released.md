---
schema_version: 1
id: "iss-2609251945586202"
slug: "the-release-gate-never-binds-the-pushed-tag-to-the-released"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
---

The release gate never binds the pushed tag to the released tree's CHANGELOG version. The content-commit derivation reads the version from the CHANGELOG only, and the release workflow's tag-shape check admits any vX.Y.Z tag, so a hand-pushed tag naming a version other than the tree's newest dated heading (for example v0.9.1 on a tree whose head is 0.9.0) is gated by the receipts for the CHANGELOG's version and publishes under a version nobody reviewed. abcd's own release has the archive-pin step, which names the tag and probably refuses first; a scaffolded repository has no such step. The fix shape is to pass TAG into record-lint --derive-content-sha on the tag path and require it to equal the bound version, which changes the pinned verify step in release.yml and its template, and needs a ruling on how the local launch receipts check, which has no tag, stays equal to the release job's verdict.
