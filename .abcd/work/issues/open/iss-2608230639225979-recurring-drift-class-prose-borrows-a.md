---
schema_version: 1
id: "iss-2608230639225979"
slug: "recurring-drift-class-prose-borrows-a"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "adversarial review of SOTA research notes"
found_at: ".abcd/development/principles/"
remedy: "Waits on ruling E: if added, give `agents/ruthless-reviewer.md` (under priority 5, the project's own rules) and `agents/security-reviewer.md` one checklist item: every principle a diff or record cites must cover the cited behaviour by its letter, and a spirit-not-letter citation must say so, proven by a test that both pages carry the item; if declined, wontfix naming the SOTA protocol's repo-fidelity lens as the detector that stays."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Add a principle-citation check to the reviewer agents' checklists?"
---

Recurring drift class: prose borrows a principle's authority for behaviour adjacent to but outside its letter. Three instances caught by fresh-context reviewers across the 2026-08-22/23 research notes: loud-staging cited for run-outcome reporting, loud-staging cited for runtime endpoint fallback, evaluator-outside-the-loop cited for reviewer freshness (its letter governs gate ownership). Only detector today is the SOTA-protocol's repo-fidelity review lens, which is genre-scoped; candidate detector is a standing principle-citation check in the reviewer agents' checklists (semantic, so a reviewer item rather than a deterministic lint).

## Remedy grounds (2026-09-29)

- Three instances in two days of research notes meet recurrence-is-signal, and the check is semantic, so a reviewer item is the right rung rather than a lint.
- No outside-practice check: the question is whether an internal checklist grows by one line.
- Rejected: a deterministic lint, which cannot read a principle's letter against the behaviour it is cited for.
