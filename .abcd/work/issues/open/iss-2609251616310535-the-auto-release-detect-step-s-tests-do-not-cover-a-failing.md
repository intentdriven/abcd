---
schema_version: 1
id: "iss-2609251616310535"
slug: "the-auto-release-detect-step-s-tests-do-not-cover-a-failing"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The auto-release detect step's tests do not cover a failing gh run view or an in-progress verify job (read to heal); both paths are read correctly but a regression in either would pass (autoreleasedetect_test.go:169-217; review2-workflows R2).
