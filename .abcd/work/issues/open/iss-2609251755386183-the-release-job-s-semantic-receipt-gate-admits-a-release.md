---
schema_version: 1
id: "iss-2609251755386183"
slug: "the-release-job-s-semantic-receipt-gate-admits-a-release"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
---

The release job's semantic receipt gate admits a release that carries no receipts of its own, whenever an earlier release's receipts are in the tree. record-lint --derive-content-sha (lint.DeriveReleaseContentSha) arms the gate against the NEAREST commit on the released lineage that a .abcd/work/reviews/<sha>/ directory names, and never checks that the commit belongs to this release. A roll to a new version with no new receipts derives the previous release's content commit, whose PROMOTE receipts are valid, so receipt_gate passes and the release publishes unreviewed. Reproduced on a scratch clone of this repository at the v0.10.0 tip: a commit rolling CHANGELOG to 0.11.0 derived 64ea8f62 (the v0.10.0 cut) and the armed gate printed no finding. Every release after the first that ever recorded receipts is exposed; the itd-93 scaffolded gate and abcd launch receipts inherit it through the same reader. Fix direction: bind the derived commit to the release — its newest dated CHANGELOG version must equal the released tree's, else fail closed naming both.
