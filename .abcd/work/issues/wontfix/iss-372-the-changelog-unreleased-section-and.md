---
schema_version: 1
id: "iss-372"
slug: "the-changelog-unreleased-section-and"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "manual-capture"
found_at: "CHANGELOG.md"
wontfix_reason: "duplicate of iss-2608220150157510: same Unreleased-block merge-contention fix (per-record entries over the shared tail)"
---

The CHANGELOG Unreleased section and DECISIONS.md tail are structural merge-conflict magnets under parallel PRs: every concurrent branch appends to the same two files, and the maintainer hand-resolved conflicts across the 2026-08-20 merge sweep with bughunt round 3 DIRTY on exactly these files as the third occurrence. The derived-cut model points at the fix: entries could live in the records themselves (per-record files never conflict) and compose at cut time, retiring the shared Unreleased append entirely — itd-93's surface should weigh it