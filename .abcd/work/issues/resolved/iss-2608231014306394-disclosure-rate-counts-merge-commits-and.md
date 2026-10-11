---
schema_version: 1
id: "iss-2608231014306394"
slug: "disclosure-rate-counts-merge-commits-and"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/contributors.go"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

The contributors page's headline disclosure figure is wrong in a way that understates compliance badly: it publishes '71% commits disclose AI assistance (1037/1454)' when the honest figure is about 95%. Three compounding faults. (1) The numerator counts Assisted-by trailer occurrences, not commits: 8 commits carry two trailers. (2) The denominator counts all 1454 commits including 410 MERGE commits, which the forge creates and no human authors, so no trailer is expected of them: 375 of the 424 undeclared commits are merges. (3) The residue is then presented as a disclosure gap. Measured on 2026-08-23: 1043 authored (non-merge) commits, of which 48 carry no trailer (4.6%), 1 declares None, the rest disclose. The denominator for any disclosure rate is authored commits; merges belong in neither numerator nor denominator, and the split must be visible rather than assumed.