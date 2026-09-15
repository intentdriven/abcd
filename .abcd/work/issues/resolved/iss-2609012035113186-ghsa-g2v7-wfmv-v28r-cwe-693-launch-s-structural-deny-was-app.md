---
schema_version: 1
id: "iss-2609012035113186"
slug: "ghsa-g2v7-wfmv-v28r-cwe-693-launch-s-structural-deny-was-app"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/bundle.go"
resolution: "Fixed before this run by 30f62e82 (fix(launch): close nested/case-fold deny bypass and unscanned-payload gap), shipped in v0.6.7; recorded so the advisory has a ledger marker."
impact: internal
resolved_by:
  commit: "30f62e828f4c51f04de8589dd7722f29215ca69c"
---

GHSA-g2v7-wfmv-v28r (CWE-693): launch's structural deny was applied to the first path segment only, so a nested or case-varied denied namespace could enter the release payload. Fixed before this run by 30f62e82 (fix(launch): close nested/case-fold deny bypass and unscanned-payload gap), shipped in v0.6.7; the class record is resolved/iss-2608270735428304 (GitHub mirror #335), which does not name the GHSA id. This record binds the advisory id to its fixing commit so an advisory-keyed search hits. Evidence at v0.7.0: internal/core/launch/bundle.go segmentDenied and pathContainsDeniedSegment; tests TestNestedDeniedNamespaceExcluded and TestCaseFoldDeniedNamespaceExcluded in bundle_denyseg_test.go.

## Grounds

- pursued: the defect is closed on main; the record exists to bind the advisory to its fixing commit
