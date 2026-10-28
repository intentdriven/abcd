---
schema_version: 1
id: "iss-2610262043101146"
slug: "the-deferral-discipline-says-every"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1119 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-2610141956319695-a-deferred-finding-is-its-own-captured.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-2610141956319695, itd-2610161733316518]
finding_ref: cfk-5ed9f071ab1ea9c1
---

The deferral discipline says every confirmed finding is fixed, invalid or captured, with no exception; the model-matrix run keeps confirmed check findings out of the ledger.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-2610141956319695-a-deferred-finding-is-its-own-captured.md:20` — "A deferral is always a capture. There is no exception."
- End B: `.abcd/development/intents/planned/itd-2610161733316518-a-product-thinker-who-wants-to-know.md:323` — "copy's findings are kept in the run's own record, nothing reaches the issue ledger"

itd-2610141956319695 states that 'a deferral is always a capture. There is no exception': every finding a review confirms ends fixed, invalid, or captured as its own open issue. itd-2610161733316518 has a person judge each check copy's findings valid or not, then keeps them in 'the run's own record, nothing reaches the issue ledger'. A real defect a doc-fidelity copy finds and the person confirms is then neither fixed, invalid nor captured, which the discipline says cannot happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1119.
