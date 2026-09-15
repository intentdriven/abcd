---
id: itd-142
slug: the-brief-creation-interview-abcd-elicits-a-repository-s-bri
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-90]
severity: major
impact: additive
---

# The brief-creation interview: abcd elicits a repository's brief in a staged conversation — narrative first, then frontier rounds that ask only what the record cannot yet settle, options at conjectural questions (at most three candidate construals, a ceiling not a target, with a null answer always available), and a per-item confirmation — distilling committed products into the brief and a ledger while the transcript is discarded. Entry is one door for both modes: a brownfield repository arrives pre-populated from embark probe coverage, provenance-stamped and still requiring confirmation; a greenfield repository is the same interview run from the all-blank coverage state. Command-shaped, because it mutates state — abcd ships zero skills

Typed links: `refines` [itd-90](itd-90-brief-interview-for-the-blanks.md) —
itd-90 hands a lifeboat's coverage blanks to the product thinker as a
post-disembark interview; this intent generalises the same coverage-driven
entry into the brief-creation surface for any repository, greenfield included,
and adds the staged elicitation machinery (question regimes, holds, the
two-output rule). Consumes [adr-50](../../decisions/adrs/0050-framing-traces-never-enter-the-record.md)
(the two-output trust rule) and embodies the `widen-options-never-recommend`
principle. The interview's committed framing products land in the
[itd-143](itd-143-the-brief-gains-a-framing-chapter-under-01-product-the-macro.md)
framing chapter.

## Press Release

> **abcd interviews a repository's humans into a brief.** Creating a brief is
> a conversation, not a form: the interview opens with narrative — the story
> the humans tell about the project — then advances by frontier rounds, each
> round asking only what the record and the answers so far cannot settle. At
> a conjectural question it widens the space instead of narrowing it: at most
> three candidate construals, a ceiling not a target, with "none of these"
> always available — the tool never recommends. Every item is confirmed
> before it lands. What the interview commits is distilled product: the brief
> receives only confirmed content, a ledger records the grounds, and the
> transcript is discarded. A brownfield repository arrives with the interview
> pre-populated from `embark probe` coverage — provenance-stamped, still
> requiring confirmation; a greenfield repository runs the same interview
> from the all-blank coverage state. One door, both modes.

## Why This Matters

The brief is human-owned by design, and today its creation has no surface:
itd-90 reaches the product thinker only after a disembark, and a greenfield
project gets a skeleton and a blank page. The interview makes elicitation the
surface — and makes non-answers data: a "can't articulate yet" goes to a
**hold register** rather than being forced into a premature answer, holds
carry axes and exit by articulation. The two-output rule (adr-50) keeps the
elicitation honest: committed products are the only output that enters the
record; declined alternatives and framing traces stay in a local ledger side
no automated reviewer reads, so review pressure can never shape what a human
was willing to consider.

## What's In Scope

- The staged interview: narrative first, then frontier rounds, then options
  at conjectural questions, then per-item confirmation; distil to a ledger,
  discard the transcript.
- Four question regimes — narrative / options / entailment / defaults — with
  an escalation rule (a defaults question that turns out value-laden
  escalates to options). The escalation rule still needs explicit sign-off
  (see Open Questions).
- The hold register: non-articulation recorded as data, holds carrying two
  axes, exit by articulation.
- The two-output rule as shipped behaviour: brief and ledger receive
  committed products only; framing traces land in the local tier per adr-50.
- Common entry for both modes: brownfield pre-population from `embark probe`
  coverage (provenance-stamped, confirmation still required); greenfield as
  the all-blank coverage state running the same interview.
- Provenance stamps on pre-populated answers and the hold-register store —
  brief plumbing routed through this intent's spec.
- Command-shaped surface: the interview mutates state (brief, ledger, hold
  register), so it lands as a command per the skill/command boundary in
  [`05-internals/08-skills.md`](../../brief/05-internals/08-skills.md);
  abcd ships zero skills.

## What's Out of Scope

- **The spec.** Per the maintainer's 2026-08-22 sequencing decision, the
  planning interview for this draft waits until the collaborating prototype
  has run once; nothing here is specced ahead of that run.
- **The trust rule itself** — decided in adr-50 and brief invariant 14; this
  intent consumes it.
- **The recommendation stance** — the `widen-options-never-recommend`
  principle; this intent embodies it.
- **A hold route in capture's triage** — the neighbouring gap is the open
  ledger seed iss-2608220750029991; this intent's hold register is
  interview-scoped, not a triage route.
- **Auto-generating answers** — itd-90's rule carries over: the human is the
  author of record, and an unanswered item stays honestly open.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Initial set; the planning interview
> refines it after the prototype run._

- **Given** a conjectural question, **when** the interview presents options,
  **then** it presents at most three candidate construals with a null answer
  available, and recommends none of them.
- **Given** an interview item the human confirms, **when** the interview
  commits it, **then** the brief and the ledger receive the committed
  product, and the transcript is not retained.
- **Given** an item the human declines or a framing trace, **when** the
  session ends, **then** it exists only in the local ledger side, and no
  committed artefact or automated reviewer input carries it (adr-50).
- **Given** a "can't articulate yet" answer, **when** the interview records
  it, **then** it enters the hold register with its axes, remains visible as
  held, and exits only by articulation — never by silent expiry.
- **Given** a brownfield repository, **when** the interview opens, **then**
  every pre-populated answer names its `embark probe` provenance and still
  requires confirmation before it commits.
- **Given** a greenfield repository, **when** the interview opens, **then**
  it runs from the all-blank coverage state through the same rounds — no
  separate greenfield flow.

## Open Questions

- **The escalation rule — ratified (facilitator, 2026-08-28; decision log):** a
  defaults question that turns out conjectural escalates to the options
  regime. The recording obligation is adopted (2026-08-28) on top of the
  ratified rule: every escalation logged as data (which
  question, on what grounds), because a question that keeps escalating is
  a misclassified question and the log is the recalibration evidence.
- **One intent vs three at the final round — ratified (facilitator, 2026-08-28; decision log):** Round 5 articulates one intent, the exit noting
  candidates for subsequent articulation — one exercises the machinery
  without front-loading articulation ahead of crystallisation.
- **A held working-principle at the final round — ratified (facilitator ruling, 2026-08-28):** Round 5 proceeds with "mechanism open" recorded
  as a gap and flagged in the readiness summary. Implemented via the
  claim recording gradient: the mechanism claim recorded as an explicit
  nullity; an absent section and a recorded nullity are never collapsed.
- ~~**Where the hold register lives**~~ **Narrowed, not settled:** the
  disposition record reserves the two-axis hold field (frame-location ×
  MoSCoW), unpopulated; the interview's hold register adopts the same axes so
  the two taxonomies cannot fork. The home question — ledger state, brief
  section, or record family — stays with iss-2608220750029991 and the
  evidence chapter.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
