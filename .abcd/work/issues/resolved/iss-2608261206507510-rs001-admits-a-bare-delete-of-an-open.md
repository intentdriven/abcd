---
schema_version: 1
id: "iss-2608261206507510"
slug: "rs001-admits-a-bare-delete-of-an-open"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-a/round-8"
found_at: "scripts/check-issue-resolution.sh"
resolution: "RS001 tests that the id enters a terminal folder, so a bare delete no longer satisfies a Resolves trailer; fresh-capture-and-resolve still passes."
impact: internal
resolved_by:
  commit: "872d261b"
---

RS001 admits a bare delete of an open record as a declared resolution