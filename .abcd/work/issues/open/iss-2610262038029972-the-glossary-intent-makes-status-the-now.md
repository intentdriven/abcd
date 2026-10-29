---
schema_version: 1
id: "iss-2610262038029972"
slug: "the-glossary-intent-makes-status-the-now"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 767 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-announce.md:116"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does the Now / Next / Later view keep the word status now that /abcd:status names a different verb? Answer A: keep the word and add a disambiguating note to the record-families row and the status chapter (recommended: the status object in the board's JSON is a shipped surface, so renaming is breaking). Answer B: rename the view."
related_intents: [itd-2609211913453478]
finding_ref: cfk-4a22f2628632543f
---

The glossary intent makes status the Now / Next / Later record family; the announce chapter's abcd status is the reader of announcements.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/38-announce.md:116` — "the reader, `abcd status` and the prompt hook's block"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:39` — "one row per family, intent, spec, step, bundle, issue, release, status (Now / Next / Later)"

itd-2609211913453478 puts status on the record-families page as Now / Next / Later, the block the bare board renders. 38-announce.md names abcd status as the reader of the fixed-form announcements posted on the shared remote, and 37-reserve.md has it show the release holds as well. One word names the computed board block and also a separate verb that lists other sessions' announcements, so a reader who looks up status in the glossary runs a verb that does not show it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 767.
