---
schema_version: 1
id: "iss-145"
slug: "receipts-cannot-gate-maintainability"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "2026-07-27 external-source review"
found_at: ".abcd/development/principles/"
remedy: "Waits on ruling E (adopt 'receipts gate process integrity, not maintainability'): if adopted, add it as a Bounds line in `.abcd/development/principles/verifier-selects-gates-decide.md` (no receipt or deterministic gate certifies design quality; the human merge carries that verdict), or as its own principle file if the ruling asks for one, proven by record-lint's principle rules passing on the edited file; if declined, wontfix this record naming verifier-selects-gates-decide as the standing statement."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Adopt 'receipts gate process integrity, not maintainability' as a principle?"
---

receipts gate process integrity, not maintainability: candidate principle surfaced by the 2026-07 software-factories critique (primary observation: RL rewards fast oracles — tests pass in seconds — while bad architecture's cost function is measured in months, so maintainability has no fast oracle and models are not penalised for eroding it). Consequence for abcd: no receipt or deterministic gate can certify design quality; that verdict stays human (verifier-selects-gates-decide), and the maintainer merge on every PR is load-bearing, not ceremony. Candidate homes: a principle file or a brief mental-model line; adoption is the maintainer's call.

## Remedy grounds (2026-09-29)

- The candidate is the gate-side corollary of verifier-selects-gates-decide (gates admit, humans judge what no gate can), so a Bounds line there adds no new file, in line with less-but-better; a separate principle stays open to the ruling.
- No outside-practice check: the question is where an internal rule lives, not what the field does.
- Rejected: a receipt or lint that scores maintainability, which the record's own premise (no fast oracle for design quality) rules out.
