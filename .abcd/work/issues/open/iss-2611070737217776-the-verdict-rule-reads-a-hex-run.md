---
schema_version: 1
id: "iss-2611070737217776"
slug: "the-verdict-rule-reads-a-hex-run"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces"
remedy: "Say in the brief and the request that such naming goes on a line of its own below the first, or stop reading hex runs that are not heads."
---

The verdict rule reads a hex run containing a digit on the result line as another head, so a reviewer who names a file's blob id or a similar path on that line is refused and asked again.

2026-11-07 and 2026-11-08: seen again in the autonomous run of 7 Nov with a bare short sha of another commit on the verdict line, each time costing a fresh reviewer or a rewritten verdict line. At 19:41 on 7 Nov a return was refused, twice that day, for quoting another commit on its verdict line inside prose; at 00:21 on 8 Nov a verdict line named the head and then the spec's landed short sha, and the reader took the second as the head. The sightings in the same run where a record id's digits were read as a commit are fixed under iss-2611101222229613.
