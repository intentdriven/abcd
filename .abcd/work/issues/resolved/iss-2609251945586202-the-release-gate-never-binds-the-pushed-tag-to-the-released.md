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
resolution: "A scaffolded semantic profile's verify job refuses a pushed tag that is not v plus the released tree's newest dated CHANGELOG version, read by record-lint --released-version (the receipts' strict reader), before the receipts gate; abcd's own profile was already closed by launch archive --tag."
impact: fix
resolved_by:
  commit: "3f613d3a4d39b7f83dcad31510ada0d83d37d2d1"
---

The release gate never binds the pushed tag to the released tree's CHANGELOG version. The content-commit derivation reads the version from the CHANGELOG only, and the release workflow's tag-shape check admits any vX.Y.Z tag, so a hand-pushed tag naming a version other than the tree's newest dated heading (for example v0.9.1 on a tree whose head is 0.9.0) is gated by the receipts for the CHANGELOG's version and publishes under a version nobody reviewed. abcd's own release has the archive-pin step, which names the tag and probably refuses first; a scaffolded repository has no such step. The fix shape is to pass TAG into record-lint --derive-content-sha on the tag path and require it to equal the bound version, which changes the pinned verify step in release.yml and its template, and needs a ruling on how the local launch receipts check, which has no tag, stays equal to the release job's verdict.

## Correction

(a) The hole was confined to scaffolded profiles. On abcd's own profile the verify job's plugin-archive step runs `launch archive --tag "$TAG"`, which refuses a tag that is not the version CHANGELOG.md dates newest (internal/surface/cli/archive.go, archiveRenderRequest) before the receipts step runs, so a mismatched tag never reached abcd's receipts gate; the "probably refuses first" above is confirmed. The scaffolded template renders that archive step only for abcd (`<% if .Abcd %>`), so a scaffolded repository had no such check. (b) The release job's tag-shape step is not the simpler place for the fix: it runs after verify has already passed the receipts gate and before Go is set up, so it could compare only through a second reader of the CHANGELOG heading in bash. The check lives in verify instead, before the receipts gate, through record-lint's strict reader; the tag-less `launch receipts` needs no ruling, because it and the derivation are unchanged.

## Grounds

- pursued: a hand-pushed tag naming another version (v0.1.1, v0.2.0, v0.1.0-rc.1 on a 0.1.0 tree) fails the rendered verify job at the binding with the receipts gate skipped and nothing published, while the tree's own tag publishes (TestScaffoldedGateRefusesATagNamingAnotherVersion); a scaffolded release published under a tag other than v plus the released tree's newest dated version would show it wrong
