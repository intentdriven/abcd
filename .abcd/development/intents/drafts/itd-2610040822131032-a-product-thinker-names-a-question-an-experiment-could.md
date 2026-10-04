---
id: itd-2610040822131032
slug: a-product-thinker-names-a-question-an-experiment-could
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212137128014]
related_issues: [iss-2610040758486433]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A lab runs end to end from a question the product thinker asks

## Press Release

> A product thinker names a question an experiment could answer, and abcd runs the lab end to end: it drafts the intention (the hypothesis, what would show it wrong, the measures and the stop conditions) for them to confirm, prepares the world, briefs the session that runs it, keeps the log as the experiment goes, and brings the harvest back to the project's record for them to adopt.

_Filed by the facilitator on 2026-10-04 at the product thinker's request, verbatim: "Do we need to file anything so that we can build a lab session automation once the manual run for pge is complete? In future, I want to use labs more often and ideally automated, too. We only need an intent or something so we won't forget what we already had to setup manually." To be shaped after the downstream brief lab's harvest._

## Why This Matters

_Facilitator-written._

abcd's lab verbs (itd-2609212137128014: mint, preflight, record, sweep, harvest) mechanise the lab's stages, but a person and an agent still drive every step, and a lab in a project other than the one asking, or in the real project rather than a throwaway copy, is set up entirely by hand. The product thinker wants labs used more often, and ideally run on their own. This draft keeps the record of what the first such run needed by hand, so the automation is built from what was actually done.

What the downstream brief lab of 2026-10-04 needed by hand, in order:

1. The question came from a design discussion here (the brief's tense), not from the lab: the product thinker's hypothesis was recorded verbatim in the decision log before anything ran.
2. A state-of-the-art pass ran first and fed the lab (research note 2026-10-04-brief-to-record-traceability-sota).
3. The facilitator wrote the intention by hand: the question, the hypothesis, four falsifiers, the measures, the stop conditions and the harvest's destination.
4. The world is the downstream project itself, not a snapshot, which the lab discipline does not yet cover (iss-2610040758486433); this was recorded as an operating constraint.
5. The abcd version was pinned once for the whole lab, chosen against a release cut the same day.
6. Mechanics abcd lacks were specified for hand-running (section markers, realises links with fingerprints, a coverage table, a suspect set per brief edit).
7. The session that runs the lab is another project's session, briefed by a cross-session message and a local brief file; the product thinker works through it with that session over several days.
8. The record rules (no project names, no local model server name, no absolute paths) were restated for the lab's filings.
9. The harvest returns to this project's record, where the drafts it tests (itd-2610040817105016, itd-61) are revisited.

Typed links: builds on itd-2609212137128014 (the lab verbs); held to the discipline itd-2609251624540864; the real-project lab gap is iss-2610040758486433.

## Decisions

1. 2026-10-04, the product thinker: file it now so the manual setup is not forgotten; shape it after the downstream brief lab completes.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Which of the nine hand steps abcd drafts for the person to confirm, and which it performs.
- How a lab in another project is started from this one, and how its harvest comes back.
- A lab in the real project: what replaces the snapshot's isolation, and what makes it safe to discard.
- What runs unattended, and what always waits for the product thinker (the intention, the adoption of findings).
- What the downstream brief lab's harvest says about each of the above.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
