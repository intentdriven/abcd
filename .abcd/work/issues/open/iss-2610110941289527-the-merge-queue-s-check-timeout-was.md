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

**A second way an entry drops (2026-10-11).** The macOS job's own `timeout-minutes: 45` removes entries too: merge-group run 38131330623 (pr-914) had `check (macos-latest)` start at 10:18:34Z and cancelled at 11:03:56Z, 45m22s later, inside "Test (race, internal)", and the queue removed #914 at 11:04:27Z, counting the cancelled check as a failure.

**After #933 (one sample, 2026-10-11).** #933 moved the macOS suite to the merge queue only and made it one race-enabled pass. Its own queue run, 38134740530, ran `check (macos-latest)` in 24.7 minutes (the pass 23.6), against a median of 40 before, and waited 0.1 minutes for a runner, against a median of 40 (p90 68) before. With the person's ruling of 4 entries built at once and a 90-minute check timeout, one entry's wait plus run is now well inside the timeout. That is one run: the remedy's p90 over a day of queue runs is still owed before this record is resolved.

## Deferral 2026-10-11

Deferred past v0.13.4: mitigated, not yet proven: the person's ruling (4 entries built, a 90-minute check timeout) and #933 (macOS suite once, in the queue, one pass) brought one queue entry to 24.7 minutes with a 0.1-minute runner wait; the remedy's p90 over a day of queue runs is owed before it is resolved
