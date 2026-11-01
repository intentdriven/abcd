---
id: itd-90
slug: brief-interview-for-the-blanks
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-36, adr-35]
prd_path: null
severity: major
---

# The Product Thinker Is Handed Exactly The Questions Only They Can Answer — And Nothing Else

## Press Release

> **abcd turns a lifeboat's blanks into an interview.** When a repository is disembarked, the coverage report already names every brief section the repo could not fill and the question a human must answer. abcd now hands that list to the person who *can* answer it — the product thinker who owns the "why" — as a short, ordered interview they can take wherever they work, at whatever moment they have. They see only the open questions, each with what abcd already searched for it, so they are never asked to confirm something the repository already proved. What they write flows back into the brief marked as **theirs** — authored on a date by a named person, never disguised as something extracted from the code. The facilitator who ran the tool never had to know the answers; the product thinker never had to touch the tool.

> "Every handover doc I've ever been given was either a wall of stuff I already knew or a blank page," said Iris, a product lead inheriting an abandoned project. "This was neither. It was six questions, and every one of them was a thing only I could answer — and it told me what it had already looked at, so I wasn't retyping the README. I answered them on the train. That was the whole handover."

## Why This Matters

The coverage experiment (itd-88) proved that a repository's own record grounds most of the brief — but not all of it. Some sections are **human-owned by design**: who the users are, the mental model, why it was built this way. No repository will ever ground those, and the 2026-07-15 gate decision confirmed it for `product/personas`. The coverage report is honest about this: it produces the exact question. What it has never had is **a way to get that question to the one person who can answer it.**

That person is not the facilitator. Disembark is a technical operation — pointing abcd at a repository — and the facilitator mining a dead or inherited project has the questions and none of the answers. The product thinker has the answers and, today, no seat at the table: neither disembark nor embark is reliably a moment they are present for, and both assume a fluency with abcd's tooling they may not have. So the most valuable sections of a lifeboat — the ones a repository *cannot* supply — are precisely the ones with no path to being filled.

This intent closes that gap by making the answering step **its own thing**: asynchronous, environment-agnostic, driven entirely by the coverage JSON ([adr-36](../../decisions/adrs/0036-coverage-blanks-are-a-fillable-lifecycle.md)). The product thinker is met where they already are, handed only what is theirs to write, and their answers return to the brief with honest provenance. The lifeboat stops being a technical artefact the wrong person is handed and becomes a conversation with the right one.

## What's In Scope

- A capability that takes a coverage report and presents the product thinker with **only its open blanks**, in order, each carrying the question and what abcd already searched — never a grounded section they would only rubber-stamp.
- **Human-owned blanks are framed as prompts, not failures.** A `human-owned` section (personas, mental model) is presented as "this is yours to write"; an `extractable` blank still open is presented as "abcd could not find this — supply it or defer it."
- Answers return as JSON that abcd ingests back into the brief, each tagged with **authored-by provenance** (who, when, and whether an assistant helped) — structurally distinct from an extracted citation, per adr-36.
- **Deferral is a first-class answer.** A blank the product thinker cannot answer now can be marked deferred, and it re-surfaces at embark rather than being lost or silently treated as answered.
- Environment-agnosticism: the interface is the coverage JSON in and the answers JSON out, so the interview can run in whatever surface the product thinker reaches for, host-delegated per abcd's existing injected-function seam.

## What's Out of Scope

- **The extraction itself.** itd-88 owns probing the repository and producing the blanks; this intent begins where a blank already exists and asks who fills it.
- **The cold reading ([itd-86](itd-86-cold-reading-surface.md)).** That surface *reviews* a document for internal contradictions and is deliberately denied context. This one *answers* open questions and is fed the product thinker's knowledge. Opposite direction; they must not be merged.
- **Deciding which sections are human-owned.** That is the mapping's declaration (adr-36) and the coverage gate's call, not a runtime judgement of the interview.
- **Auto-generating answers.** An assistant may *help* the product thinker articulate an answer, but an unanswered `human-owned` blank is never filled by a model on its own — that would manufacture the exact fiction the coverage experiment exists to prevent. The human is the author of record.
- **The write path into the repository.** Embark (itd-88, M5) owns reconciling answers into a target; this intent produces the answers, it does not scaffold the destination.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a coverage report with grounded, partial, and blank sections, **when** the interview is presented, **then** it shows only the blanks (and any partials flagged for confirmation), never a grounded section — the product thinker is asked exactly what the repository could not answer and nothing it already did.
- **Given** a `human-owned` blank and an `extractable` blank both open, **when** each is presented, **then** the human-owned one is framed as the product thinker's to author and the extractable one as a coverage gap to supply or defer — the two are not presented as the same kind of failure.
- **Given** the product thinker answers a blank, **when** the answer returns to the brief, **then** it carries authored-by provenance (a person and a date, and an assistant flag if one helped) and is never recorded as extracted-from a file.
- **Given** a blank the product thinker cannot answer now, **when** they defer it, **then** it is marked deferred and re-surfaces at embark, rather than being dropped or recorded as answered.
- **Given** a `human-owned` blank left unanswered, **when** the interview completes, **then** abcd has not fabricated an answer for it — the section remains an open, honestly-empty prompt, not a model's guess presented as the human's.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- Asynchronous email interviews "garnered rich, insightful data that augmented
  the phone interviews", with email participants cited in proportion to their
  share of the sample (one sensitive-topic health study; 31 per cent chose
  email themselves), and ran 4 to 5 exchange cycles over 27 to 28 days on
  average (Hershberger and Kavanaugh, 2017); the export-answer-import loop is
  viable and must be budgeted as several rounds, not one pass.
- A static form is not the async interview: per-person yield from a survey was a
  sixth to an eighth of an interview's (Obaidi et al., 2025), and self-administered
  model interviews were acceptable but faulted for "insufficient depth" with no
  equivalence shown (Gheyi et al., 2026). The re-import runs the adaptive
  follow-up a form lacks.
- "Defer" is a distinct recorded state: a don't-know option changes what the
  other answers capture and its choosers are not neutral (Wetzelhütter, 2020).
- The stop record the autonomous-loops pass recommends (an MCP-elicitation
  envelope with addressee role, judgement-or-fact kind, a non-recommendation
  flag, accept / decline / cancel, and a version) is the shape of a blank: one
  record, answered live through the host's question tool or later from a file,
  with the waiting state durable data that resumes on the same id.
- PROV's `actedOnBehalfOf` names a facilitator relaying a product thinker's
  answer; the `agent-assisted` flag the intent specifies is justified by the
  co-creation evidence under itd-142.
- Partials are confirmable, by analogy with Obaidi et al.'s delayed-taxonomy
  condition (a classification taxonomy introduced after free elicitation
  yielded the most needs), which is not a direct test of showing what was
  found and asking for the top-up.

**Path 2**: one record with two renderings, synchronous and asynchronous, the
record canonical and the session incidental; in-progress answers keyed on the
source (the voyage log) and shared with itd-142's hold register rather than a
second store. No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

- **How synchronous must the loop be?** The strongest form is fully asynchronous — export the interview, answer it whenever, re-import — but a live assisted session may produce better answers. Whether both are supported, or one is canonical, is a design call.
- **Where does a deferred or partial answer live between disembark and embark?** The coverage report travels with the lifeboat, but a lifeboat is regenerable output (adr-35); a durable home for in-progress answers keyed on the source (the voyage log?) may be needed so an interview taken between the two operations is not lost on a re-pack.
- **Should partials be confirmable, or only blanks answerable?** A `partial` section is grounded-but-incomplete; whether the interview lets the product thinker top it up, or only touches true blanks, affects how much of the brief the human is asked to hold.
- **What is the minimum viable assistant?** The `agent-assisted` provenance flag implies a helper exists, but the deterministic (no-assistant) interview must stand on its own first — the assistant is an enhancement, not a dependency.
- (SOTA 2026-10-23) Whether the deferred-answer store and itd-142's hold register
  are one store with two entry points; the research supplies no reason for two.
