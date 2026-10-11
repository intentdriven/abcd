---
schema_version: 1
id: "iss-2608220750029986"
slug: "adrs-readme-md-index-omits-the-accepted"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "adr-47/adr-48 index rows are present in the decisions index (verified)"
impact: internal
---

adrs/README.md index omits the accepted adr-47 and adr-48 rows (site-build session) — fourth recurrence of the unindexed-ADR class after adr-45 (iss-38 wants these indexes gated); rows added in the same change that records this