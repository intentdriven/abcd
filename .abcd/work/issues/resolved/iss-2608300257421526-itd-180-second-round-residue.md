---
schema_version: 1
id: "iss-2608300257421526"
slug: "itd-180-second-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-180 second-round reviews, 2026-08-30"
found_at: ".abcd/development/specs/open/spc-58, internal/core/lint/readingoutstanding.go, internal/core/lint/schema.go"
resolution: "spc-58 now states the shipped promote rule (every standing state but accepted refuses, rechecked under the write's lock); schema.go states the reading stores' missing required fields as a gap the writer and review cover; the reading package's header claim is corrected; lint.StandingDispositions went with the single-reader change."
impact: internal
---

itd-180 second-round residue: spc-58 still says promote refuses a reading item only when the disposition directory is absent while the build refuses any standing state but accepted; lint.StandingDispositions is exported with no production caller (a single reader in the issueschema leaf would retire it); the three reading stores declare no required fields so the committed-tree gate never judges a reading or disposition's content, contrary to the reading package's header claim that the reader refuses what the writer refuses.
