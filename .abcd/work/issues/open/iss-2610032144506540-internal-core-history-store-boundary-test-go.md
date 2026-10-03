---
schema_version: 1
id: "iss-2610032144506540"
slug: "internal-core-history-store-boundary-test-go"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "noindex step 2 review, 2026-10-03"
found_at: "internal/core/history/store_boundary_test.go"
origin: researcher-authored
production_mode: hand-written
remedy: "Write the hostile literals out explicitly in TestStorePathBoundaryScannerIsArmed, one per spelling the scanner claims (.abcd/transcripts, .abcd.noindex/transcripts, .work.local/transcripts, .abcd/local-transcript-roots, history/transcripts), as internal/abcdhome's TestSearchSettingsScannerIsArmed already does, so removing a needle from storePathNeedles fails the armed test by name. Grounds: an armed test drawing its cases from the list under test is tautological for that list; shown by removing one needle on a scratch copy and watching the test fail only after the change."
---

internal/core/history/store_boundary_test.go TestStorePathBoundaryScannerIsArmed builds its hostile literals by ranging over storePathNeedles, the scanner's own needle list, so a needle dropped from storePathNeedles shrinks the armed test instead of failing it: the proof that invariant 15's store-path scanner catches each spelling moves with the thing it is meant to check.
