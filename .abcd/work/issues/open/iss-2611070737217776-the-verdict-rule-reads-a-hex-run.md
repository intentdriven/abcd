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
