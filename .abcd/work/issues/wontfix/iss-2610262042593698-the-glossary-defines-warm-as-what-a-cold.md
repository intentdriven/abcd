---
schema_version: 1
id: "iss-2610262042593698"
slug: "the-glossary-defines-warm-as-what-a-cold"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1107 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/warm.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-199]
finding_ref: cfk-f9c92a6f7f4af372
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-199's 'warm is cold plus a delta' preset no longer exists: the accepted ADR left one preset entry per position and no second name. The glossary definition (reserved reasoning and ledger material) is the live sense."
---

The glossary defines warm as what a cold reading must never see; itd-199 defines warmth as how much a reader is given.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/warm.md:17` — "names what a cold reading must not see and what the researcher keeps"
- End B: `.abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower.md:19` — "because warmth is how much a reader was given and nothing else"

warm.md defines warm as the researcher's reserved reasoning and the ledger material it works on (dispositions, admissions, manifests, the lapse log), and says warm 'names what a cold reading must not see'. itd-199 names a 'warm' preset that is 'cold' plus a delta of scope and states that 'warmth is how much a reader was given and nothing else', so a 'warm' reading is a wider reading of admissible material that still sees no warm material. The term means 'withheld from the reader' in one document and 'more of what the reader may see' in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1107.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-199's 'warm is cold plus a delta' preset no longer exists: the accepted ADR left one preset entry per position and no second name. The glossary definition (reserved reasoning and ledger material) is the live sense.
