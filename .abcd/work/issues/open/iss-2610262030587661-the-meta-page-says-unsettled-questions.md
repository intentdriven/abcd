---
schema_version: 1
id: "iss-2610262030587661"
slug: "the-meta-page-says-unsettled-questions"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 69 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:12"
remedy: "Mark the sentence at 23-reading.md:74 ('a reading of the rule rather than the rule itself ... ruling on it is owed') as an explicit 'Open question (adr-35)' note, or register it in 03-evidence/03-open-questions.md, so the owed product-thinker ruling on the comparative run's one-commit reach is recorded as a question."
finding_ref: cfk-5c08b09ce7807c77
---

The meta page says unsettled questions are recorded only as registered open questions or as explicit `Open question` notes; the reading chapter carries an owed product-thinker ruling inline, in neither form.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:12` — "Unsettled questions are recorded *as questions*, never as claims — registered in"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:74` — "That is a reading of the rule rather than the rule itself, and the product thinker's ruling on it is owed."

00-meta's truth rule says the brief records an unsettled question only in 03-open-questions.md or as an explicit `Open question (adr-35)` note. 23-reading says its reach for the comparative run is a reading of the rule rather than the rule itself, and that the product thinker's ruling on it is owed. That is an unsettled question, but it carries no Open question note and is not in the open-questions page. The brief's own record of its unsettled questions is therefore incomplete.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 69.
