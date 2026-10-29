---
schema_version: 1
id: "iss-2610262033033258"
slug: "itd-90-relies-on-embark-to-write-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 354 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-90-brief-interview-for-the-blanks.md:43"
remedy: "In itd-90 strike 'Embark (itd-88, M5) owns reconciling answers into a target' (itd-88 carries no such milestone and embark writes only its five record families and the marker block) and name the real owner of the write into the brief: itd-142, which refines itd-90 and commits the interview's confirmed products to the brief."
related_intents: [itd-90]
finding_ref: cfk-94400eb0e0738111
---

itd-90 relies on embark to write the product thinker's answers into the target; embark writes only its record families and never writes anything else from the lifeboat.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-90-brief-interview-for-the-blanks.md:43` — "Embark (itd-88, M5) owns reconciling answers into a target; this intent produces the answers, it does not scaffold the destination."
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:204` — "and everything else in the lifeboat informs the report but is never written"

itd-90 produces answers that are ingested back into the brief, and it leaves the write path to embark: 'Embark (itd-88, M5) owns reconciling answers into a target'. The embark page's unpack writes ADRs, issues, intents, specs and release retrospectives, plus the marker block. Everything else in the lifeboat, the brief included, informs the report but is never written, and no interview answers enter its plan. As written, embark's scope cannot carry the dependency itd-90 hands it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 354.
