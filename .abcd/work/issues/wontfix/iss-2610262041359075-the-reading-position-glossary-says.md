---
schema_version: 1
id: "iss-2610262041359075"
slug: "the-reading-position-glossary-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1011 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/reading-position.md:60"
remedy: "none (filed automatically)"
related_intents: [itd-2609020625407419]
finding_ref: cfk-5feaaf0b742b5373
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035439719. Same End A (comparative does not assemble) and the same later ruling, adr-2609021016272867, which the shipped comparative-reading intent records."
duplicates: [iss-2610262035439719]
---

The reading-position glossary says `comparative` does not assemble and refuses for want of a channel; the comparative-reading intent says it can now be commissioned and assembles a derived widening run

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/reading-position.md:60` — "`comparative` does not assemble and refuses: its object has no channel."
- End B: `.abcd/development/intents/shipped/itd-2609020625407419-a-comparative-reading-receives-the.md:21` — "The comparative reading can now be commissioned."

The glossary's worked example says the comparative position does not assemble and refuses, because its object has no channel. That was itd-199's ruling. The shipped comparative-reading intent withdraws that refusal under adr-2609021016272867: `reading assemble --position comparative` derives the one undispositioned widening run at the target and hands its items over as the candidate set. The comparative position cannot both refuse to assemble and assemble, so the canonical glossary teaches the superseded behaviour.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1011.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035439719. Same End A (comparative does not assemble) and the same later ruling, adr-2609021016272867, which the shipped comparative-reading intent records.
