---
schema_version: 1
id: "iss-2608311127491949"
slug: "the-ship-ceremony-s-repoint-step-names-two-link-classes-ever"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "itd-184 ship ceremony, cold-reading cycle 1"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/CONTEXT.md"
---

The ship ceremony's repoint step names two link classes -- every intents/planned link to the shipping intent, and the closing spec's bare sibling links to still-open specs plus bare links to it from open specs. A third class exists and is not named: a link from an ALREADY-CLOSED spec pointing at ../open/<the spec being closed>. Closing spc-62 left spc-61, itself already closed, holding ../open/spc-62 and record-lint refused with a links_resolve BLOCKER. The survey greps the checklist implies (specs/open/ and bare siblings) do not match that shape, so it is invisible until the gate runs. Class (b) is already recorded as having broken two earlier ships; this is a third sibling of the same shape and the checklist should name all three, or the repoint should be mechanical rather than a hand survey.
