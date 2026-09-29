---
schema_version: 1
id: "iss-2608221310142705"
slug: "new-top-level-verbs-face-no-design-time-test-and-the-ideate"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "ideate-cli-verb-taxonomy-verdict"
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Waits on ruling E (home of the design-time verb test): if the naming register, add one line to `.abcd/development/brief/02-constraints/04-naming.md` beside the metaphor-versus-exempt criterion asking 'does an existing record-owning verb already own this act?', answered in the Decisions of any intent that proposes a top-level verb; if a principle, the same question as a new file under `.abcd/development/principles/` passing record-lint's principle rules; if the verdict note, wontfix this record citing the 2026-08-22 note."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Where does the design-time verb test live: a principle, a naming-register line, or the verdict note?"
---

New top-level verbs face no design-time test outside the assessment family, and the ideate verdict on the verb taxonomy left one unclaimed. adr-40 already imposes a design-time test, but only on assessment surfaces and only on one axis: what is compared to what, with a surface that would perform two acts split at design time rather than documented as a bundle. It says nothing about whether a proposed verb belongs at the top level at all, so a non-assessment verb faces no test. The 2026-08-22 ideate verdict (research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md) killed an open-ended 'restructure when confusable pairs appear' deferral because a condition with no detector is the phantom gate enforcement-claims-are-facts forbids. What it put in that place is a design-time question asked when a verb is proposed: does an existing record-owning verb already own this act? The question is answerable by the human designing the verb, which a trigger conditioned on future confusion is not. Five unbuilt verb names are already in play across three different statuses, so the test has live subjects: dredge, loot and audit sit in the reserved table of brief/02-constraints/04-naming.md; audit and reflect sit on its metaphor-exemptions list; and oracle is in neither, appearing only as a design-target sub-verb under a note recording that no oracle verb is registered. It currently exists only inside a research note, which is not where a design-time rule can be found by the person who needs it. Decide its home: a principle under development/principles/, a line in the naming register at brief/02-constraints/04-naming.md beside the existing metaphor-vs-exempt criterion and the bare-command-as-render discipline, or an explicit decision to leave it in the verdict record. itd-146 carries the same question as an open question; this capture exists so the test survives that intent not shipping.

## Remedy grounds (2026-09-29)

- The naming register is where a person designing a verb already reads the naming rules, and itd-146 shipped without settling the question, so this record is the only carrier left.
- No outside-practice check: the question is placement of an internal rule.
- Rejected: a trigger conditioned on future confusable pairs, which the 2026-08-22 verdict killed as a phantom gate.
