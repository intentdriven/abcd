---
schema_version: 1
id: "iss-2610262032212691"
slug: "launch-says-intents-carry-no-version"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 257 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:491"
remedy: "Reword 04-launch.md line 491 to say the brief and the intents carry no version label except a planned intent's optional target_release (adr-2609212115255771 decision 3), which the preview and the cut report and never refuse on; drop 'roadmap'. Same sentence block as the retired-phase-vocabulary edit, so make both in one change."
finding_ref: cfk-749716a32adc8c98
---

Launch says intents carry no version label; the intent chapter says a planned intent may name the release vX.Y.Z it must land by

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:491` — "The brief, intents and roadmap carry no version label."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:75` — "A planned intent may name the release it must land by, `vX.Y.Z` or `next`"

Launch states that versions are never a sequencing input on the design record and that the brief, intents and roadmap carry no version label. The intent chapter defines target_release, a planned-intent field naming the release the intent must land by as vX.Y.Z or next, which the preview and the cut report. A release tag carried on an intent as a land-by deadline is a version label used as a sequencing input, so the two statements cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 257.
