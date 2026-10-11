---
schema_version: 1
id: "iss-2610110941289527"
slug: "the-merge-queue-s-check-timeout-was"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "merge-pipeline evidence from abcd-0b, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
remedy: "once the macOS leg runs only in the queue, measure a queue entry's macOS wait plus run time and set check_response_timeout_minutes above its p90, with the measurement on this record; meanwhile keep max_entries_to_build within the macOS slots"
---

The merge queue's check timeout was shorter than one macOS queue entry takes: over 33 merge-group ci runs (2026-10-10 18:00Z to 2026-10-11 09:40Z) a macOS job waited a median 40 minutes for a runner (p90 68, max 81) and then ran a median 40 (p90 44), so an entry needed about 80 minutes against a 45-minute timeout. Every entry that had to wait was removed: #903 and #906 twice each, #908, #910, #913 and #915 once (#913 at 04:46:54Z with every check green but macOS, run 38110208736). Nothing merged from 03:36Z to 09:38Z with 15 entries queued. The person has since set 2 entries and 90 minutes (ruleset 19969675, mirror PR #926); 90 is still tight against 80.
