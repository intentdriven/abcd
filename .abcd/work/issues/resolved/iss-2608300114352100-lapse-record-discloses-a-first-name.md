---
schema_version: 1
id: "iss-2608300114352100"
slug: "lapse-record-discloses-a-first-name"
severity: "major"
category: "security"
source: "impl-review"
found_during: "cold-reading Phase 1 adversarial review, 2026-08-30"
found_at: ".abcd/work/issues/open/iss-2608300002557321-lapse-inferred-go-ahead.md"
resolution: "Homonym parenthetical replaced with a name-free wording; branch history still carries the original until the PR is squash-merged."
impact: internal
---

The session-1 lapse record explains the misread go-ahead as a homonym and thereby discloses a participant's first name; the filing's own 2026-08-29 decision line rules that roles, never names, appear in any record or commit. Found by the Phase 1 adversarial review; the text also reached the pushed filing branch, so the branch history carries it until the PR is squash-merged or the branch rewritten.
