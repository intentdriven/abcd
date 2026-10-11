---
id: itd-2609151541116052
slug: every-question-an-abcd-interview-puts-to
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-03, from: standalone, to: superseded, reason: "the product thinker folded it into the layout intent on 2026-10-03; its order, step list and reason move into that intent's scope (decision 19)" }
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
related_intents: [itd-2610030810350727, itd-201]
kind_at_supersession: standalone
superseded_by: itd-2610030810350727
---

# During the peer-listing planning interview on 2026-09-20, the agent listed the rewritten acceptance criteria in prose and then asked, through the interactive question view, whether the criteria were the product thinker's. The prose was not on screen while the question showed, so the product thinker was asked to confirm text they could not see (iss-2609202058058301). A confirming question has to carry what it confirms.

> **Superseded by itd-2610030810350727** on 2026-10-03: the product thinker folded it into the layout intent on 2026-10-03; its order, step list and reason move into that intent's scope (decision 19)

What this record adds to the layout intent (itd-2610030810350727) and the asking rules (itd-201): the order (the thing itself first, the question last) and the list of steps it binds in every interview page, not only the planning interview (a criterion before "does it stand", a press-release paragraph before confirm or refine, an open question before resolve or defer, mechanism and scope text before their questions; the retrospective and the unpack interviews included).

## Press Release

> Whenever abcd asks you to confirm, change or set aside something, the thing itself is on the screen first: the criterion's wording, the paragraph, the open question, in full. The question comes last, after what it is about. You never answer on a title or a summary of text you cannot see, in any of abcd's interviews.

_Proposed by the design review of 2026-10-03; to be confirmed or rewritten by the product thinker at the planning interview._

## Why This Matters

Every question an abcd interview puts to a human shows the thing being decided before it asks: the acceptance criterion's full text before asking whether it stands, the press-release paragraph before asking to confirm or refine it, the open question before asking to resolve or defer it, the mechanism and scope-condition text before asking about them, at every step of every interview

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Review findings (design and feasibility, 2026-10-03, bundle with itd-2610030810350727 and itd-201)

From the review recorded in the local tier (reports/review-bundle-design.md); nothing here is settled.

- Should-fix, near-empty: the layout intent's press release already quotes the thing being decided in full, and the asking rules already say "quoted in the question itself". What this record alone holds is the order (the material first, the question last) and the list of steps it binds in every interview (a criterion before "does it stand", a press-release paragraph before confirm or refine, an open question before resolve or defer, mechanism and scope text before their questions), in every interview page, not only the planning one. The product thinker kept it in the bundle (layout decision 2); if that is too thin, the layout intent takes the list into its scope and supersedes this record. Flagged, not decided.
- Checkable by the question check: the question field has at least one paragraph before its last line, and its last line ends with a question mark. Judgement only: that the text shown is the record's own text.

Proposed acceptance criteria (agent-seeded, unconfirmed): S1 (product thinker) a criteria walk shows the criterion's full text as the first paragraph, then "Is this yours?"; S2 (product thinker) a press-release confirmation quotes the paragraph, never its number; S3 (product thinker) an open question is quoted before it is resolved or deferred, and "decide later" is an option; S4 (product thinker) the retrospective and the unpack interviews follow the same order; S5 (facilitator) a question whose field has no paragraph before its last line is refused, naming the field and the rule (fixture: "Are these criteria yours?" alone).

## Open Questions

- Keep or fold: does this record stay in the bundle as refining the asking rules (the order and the steps in every interview), or does the layout intent take the step list into its scope and supersede it? Asked at the interview.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
