---
id: itd-2610031348087517
slug: when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-3]
related_intents: [itd-2610030814013772]
related_issues: [iss-2609110944498549]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Setup asks whether abcd keeps parts of the README current, and which

## Press Release

> When abcd sets up a project, it asks the owner whether abcd should keep parts of the project's README.md current or leave the README entirely to them. If the owner says yes, abcd asks which parts it keeps, such as the title and the badges, including a badge saying abcd manages the project, and it keeps only those parts, inside marked places, never rewriting the owner's own text.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request, quoted verbatim: "when running /abcd:ahoy, ask the repo owner whether they want abcd to manage the repo's main README.md, or whether they want them to maintain it. If the answer is yes, ask what elements abcd should maintain (run SOTA), incl." and, asked what "incl." introduced: "incl. title, badges (incl. an abcd-managed badge) etc."; to be confirmed at the planning interview._

## Why This Matters

_Facilitator-written._

A README goes stale in the parts that can be read from the repository itself: the version, the licence, the build status, the command list. abcd already keeps one marked block current in a project's conventions file where the owner chose one (the default writes none, iss-2609110944498549; under itd-2610030814013772 that file is AGENTS.md alone); this extends the same arrangement to the README, but only where the owner asks and only for the parts they choose, because a project's files are the owner's (the principle the-users-directory-is-theirs: abcd writes only where it is handed a place, and removes only what it can prove it wrote).

Typed links: refines the setup interview of itd-3 (shipped; setup's questions); reuses the marked-block arrangement setup already plants in the conventions file. A state-of-the-art pass on which README elements tools keep current, and how they mark them, was started on 2026-10-03 at the product thinker's request; its report feeds this draft's reviews.

## Decisions

1. 2026-10-03, the product thinker at the planning interview, asked what keeping the parts current means (a rewrite each time setup runs; that plus a separate check reporting stale parts; decide later): both. Setup rewrites the chosen parts each time it runs, and a separate check that reports a stale or hand-edited part, writing nothing, is filed as its own draft that builds on this one.
2. 2026-10-03, the product thinker, told that a standing ruling (2026-09-23, ruling F) allows one mention of abcd in a project abcd sets up and that a visible badge would be a second: the badge is its own part, opt-in and off by default. This extends ruling F by the owner's explicit consent; it is recorded as the product thinker's decision, not the facilitator's.
3. 2026-10-03, the product thinker: the title is off the list; abcd never writes a title (nothing in a project derives one reliably).
4. 2026-10-03, the product thinker: the version appears as the forge's live release badge, never as a version written into the file (adr-19 keeps version numbers out of committed files); public projects only.
5. 2026-10-03, the product thinker, asked what happens in a project with no README (write nothing until the owner makes one; create a README holding only the chosen parts; decide later): abcd creates a README holding only the chosen parts. The facilitator records the tension with the-users-directory-is-theirs for the spec: the owner's yes to the parts is the hand-over, and the file is one abcd can prove it wrote.
6. 2026-10-03, decided without a question (both reviews and the principle agree): the owner's prose is never touched; a hand-edited part is reported and left; a duplicated or unclosed marker refuses the write and names the line.
7. 2026-10-03, decided without a question (the technical facilitator's): the question joins setup's fixed order after the conventions-file question; a non-interactive setup declines it and names the flag; a no is stored so it is not asked again; the build-status badge is offered only where the owner declared the project public; a badge address with a foreign query string or host is refused by name.

## Open Questions

None open for the product thinker: the interview of 2026-10-03 answered them (decisions 1 to 5). Owed at planning: the press release and criteria rewritten to the decisions, and the scope conditions.

## Mechanism

We expect the chosen README parts to stop going stale because setup rewrites them from the project itself, and the check reports any that drift between setups; a chosen part found stale after a setup run shows the claim wrong.

_Proposed by the facilitator; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._


## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: owners get help on their README only where they ask for it (the product thinker, 2026-10-03).
