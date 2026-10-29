---
schema_version: 1
id: "iss-2610262041130402"
slug: "the-solo-mode-draft-runs-on-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 985 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:142"
remedy: "In itd-97 replace 'over the pluggable run seam (itd-29)' with the loop itd-2609201916151817 ships through abcd build; itd-29 is superseded and the seam does not ship."
related_intents: [itd-97]
finding_ref: cfk-6ce053e807600730
---

The solo-mode draft runs on the pluggable run seam via itd-29; the build sequence says itd-29's surface is superseded and the seam does not ship.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:142` — "The operator surface itd-29 drew over it is"
- End B: `.abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:60` — "over the pluggable run seam (itd-29)"

itd-97 builds solo mode's automated scheduling, reviews and run start on the pluggable run seam named by itd-29. 01-build-sequence.md says the run seam does not ship and that the surface itd-29 drew over it is superseded by itd-2609201916151817, whose loop runs without the seam. The draft depends on a seam and an intent the brief has withdrawn.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 985.
