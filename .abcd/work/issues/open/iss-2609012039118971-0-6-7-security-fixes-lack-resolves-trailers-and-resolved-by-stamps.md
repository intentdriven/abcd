---
schema_version: 1
id: "iss-2609012039118971"
slug: "0-6-7-security-fixes-lack-resolves-trailers-and-resolved-by-stamps"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/resolved"
---

Observation from the lane LA assessment: the two 0.6.7 security fixes 81f81f67 (GHSA-xrf8-4432-gw2f) and 15a31ea7 (GHSA-h2gm-w3hm-8xpq) were committed without `Resolves:` trailers and their records (iss-2608270735428527, iss-2608270735420161) moved to resolved/ without `resolved_by` stamps, so the RS001-RS003 gates either did not exist or did not fire on the security-release-2026-08-27 integration branch. Worth checking whether the other records from that pass share the gap, and whether the gates run on every integration branch today. Not fixed in this run; the two advisories are bound to their fixing commits by their own ledger markers instead.
