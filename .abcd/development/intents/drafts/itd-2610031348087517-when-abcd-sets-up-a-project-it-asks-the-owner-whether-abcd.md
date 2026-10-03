---
id: itd-2610031348087517
slug: when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Setup asks whether abcd keeps parts of the README current, and which

## Press Release

> When abcd sets up a project, it asks the owner whether abcd should keep parts of the project's README.md current or leave the README entirely to them. If the owner says yes, abcd asks which parts it keeps, such as the title and the badges, including a badge saying abcd manages the project, and it keeps only those parts, inside marked places, never rewriting the owner's own text.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request, quoted verbatim: "when running /abcd:ahoy, ask the repo owner whether they want abcd to manage the repo's main README.md, or whether they want them to maintain it. If the answer is yes, ask what elements abcd should maintain (run SOTA), incl." and, asked what "incl." introduced: "incl. title, badges (incl. an abcd-managed badge) etc."; to be confirmed at the planning interview._

## Why This Matters

A README goes stale in the parts that can be read from the repository itself: the version, the licence, the build status, the command list. abcd already keeps one marked block current in a project's conventions file; this extends the same arrangement to the README, but only where the owner asks and only for the parts they choose, because a project's files are the owner's (the principle the-users-directory-is-theirs: abcd writes only where it is handed a place, and removes only what it can prove it wrote).

Typed links: refines the setup interview of itd-3 (shipped; setup's questions); reuses the marked-block arrangement setup already plants in the conventions file. A state-of-the-art pass on which README elements tools keep current, and how they mark them, was started on 2026-10-03 at the product thinker's request; its report feeds this draft's reviews.

## Open Questions

_Facilitator-written, for the reviews and the planning interview._

- Which elements abcd offers, from the research: derivable ones (title, version, licence, badges, command list, table of contents) as against the owner's prose, which abcd never writes.
- The abcd-managed badge: what it says and links to, and whether it is offered or always added when the owner says yes. abcd's own convention confines naming a tool to credit (a README badge is the sanctioned place).
- How abcd keeps the parts current (at setup only, on every setup run, or checked by a lint that reports drift), and what it does when the owner has edited inside a marked part.
- A project with no README: create one with only the chosen parts, or leave it.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._


## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
