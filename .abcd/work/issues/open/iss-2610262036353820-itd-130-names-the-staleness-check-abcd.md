---
schema_version: 1
id: "iss-2610262036353820"
slug: "itd-130-names-the-staleness-check-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 659 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-130]
finding_ref: cfk-8b12a31c0e8a005b
---

itd-130 names the staleness check abcd version --check; the update chapter makes the check a flag of abcd update

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md:32` — "typed `abcd version --check` on the machine where they first installed the"
- End B: `.abcd/development/brief/04-surfaces/21-update.md:51` — "The check is a flag of this verb (itd-2609212130136102): it prints the"

itd-130 describes the user typing abcd version --check and abcd update completing what that check reports. 21-update says the check is a flag of the update verb itself, printing the version report. The one network staleness check is given two different command spellings.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 659.
