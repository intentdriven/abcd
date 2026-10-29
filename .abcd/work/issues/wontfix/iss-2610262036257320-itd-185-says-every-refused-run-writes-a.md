---
schema_version: 1
id: "iss-2610262036257320"
slug: "itd-185-says-every-refused-run-writes-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 646 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold.md:85"
remedy: "none (filed automatically)"
related_intents: [itd-185]
finding_ref: cfk-d93a259fdae28947
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The delivery draws the line at proven identity; the intent's criteria never stated it, which its audit notes say."
---

itd-185 says every refused run writes a refusal record; the reading chapter says refusals made before the run's identity is proven, a hash disagreement among them, leave none

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-185-one-ingest-verb-validates-every-cold.md:85` — "A refused run still writes a refusal record — run metadata and the named reason, no items"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:276` — "A malformed envelope, a run id nothing is parked under, a hash disagreement, and a manifest naming another run or another position each refuse before that point and leave no refusal record at all."

itd-185 makes a list-level refusal always durable through a refusal record, and counts a manifest reference whose hash disagrees as one of the refusals of the run. 23-reading says a malformed envelope, an unparked run id, a hash disagreement or a manifest naming another run refuses before identity is proven and leaves no refusal record at all. The two give opposite answers for what a hash-mismatched run leaves behind.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 646.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The delivery draws the line at proven identity; the intent's criteria never stated it, which its audit notes say.
