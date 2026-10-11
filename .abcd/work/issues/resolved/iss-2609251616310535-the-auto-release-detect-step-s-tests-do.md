---
schema_version: 1
id: "iss-2609251616310535"
slug: "the-auto-release-detect-step-s-tests-do"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "autoreleasedetect_test.go now runs a failed gh run view (refuses, names the run) and an in-progress verify job (heals) against the committed workflow and both template profiles."
impact: internal
resolved_by:
  commit: "19b64d720af68e89ad8f4649f8cb2c2b44f0c7b0"
---

The auto-release detect step's tests do not cover a failing gh run view or an in-progress verify job (read to heal); both paths are read correctly but a regression in either would pass (autoreleasedetect_test.go:169-217; review2-workflows R2).

## Grounds

- pursued: a regression that swallows a failed run view or reads an unconcluded verify as a refusal now fails the detect table; either mutation passing it would show it wrong
