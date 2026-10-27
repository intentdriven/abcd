---
schema_version: 1
id: "iss-2610262043598765"
slug: "the-routing-intent-leaves-updating-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1175 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:129"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401, itd-2609221009495079]
finding_ref: cfk-ecec464ea12e13ee
---

The routing intent leaves updating the proposal from measurements to itd-17; the decision-adapter intent records itd-17 as superseded

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:129` — "Updating the proposal from measurements — itd-17."
- End B: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:52` — "A learned per-request router for lane models (itd-17, superseded)."

itd-2609170822093401 puts 'updating the proposal from measurements' out of its own scope and names itd-17 as the owner. itd-2609221009495079 lists itd-17 as superseded and puts the learned router out of its own scope too. The work is handed to a record that no longer carries it, and the record that superseded it does not take it on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1175.
