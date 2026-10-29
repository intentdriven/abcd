---
schema_version: 1
id: "iss-2610262031503691"
slug: "the-mental-model-uses-brief-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 184 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:3"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does 'brief' name two things (the project brief and a lane's brief, kept apart in the glossary) or does the implementer's file get its own word? Recommended: keep both and record 'lane brief' in the glossary, because the shipped build chapter, the lane stage names and the loop code already say brief and lane brief, and a rename would touch code."
related_intents: [itd-2609201916151817]
finding_ref: cfk-065f03756dd5e916
---

The mental model uses 'brief' for the project's canvas, but itd-2609201916151817 uses 'the brief' for the file an implementer is handed for one lane.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:3` — "the **brief** (what is this project?)"
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:44` — "**The brief renderer**: Intent, spec, the conventions section of `AGENTS.md`, the decision lines the intent cites, into one file the implementer is handed."

03-mental-model makes the brief the top layer of the record, the one that answers 'what is this project?'. itd-2609201916151817's brief renderer combines the intent, spec, AGENTS.md conventions and decision lines into one file the implementer is handed, and `implement step` returns 'the brief'. One name covers two artefacts, in a record that also refers to the project brief.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 184.
