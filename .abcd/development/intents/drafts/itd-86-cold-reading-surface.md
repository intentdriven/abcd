---
id: itd-86
slug: cold-reading-surface
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-55]
prd_path: null
severity: minor
---

# abcd Can Read Its Own Design Documents As A Stranger Would

## Press Release

> **abcd gains a cold reading: a pass that reads one design document as though it had never been seen before — knowing how designs of this kind work, but knowing nothing about how this one came to be framed — and surfaces the places where the document's own constraints contradict its framing.** Every other reviewer abcd has is warm: the fidelity reviewer knows the intent, the grill knows the acceptance criteria, the docs reviewer knows the code. Warmth is what lets them check correspondence, and it is also what makes them sympathetic — they fill in the gaps the author left, because they know what the author meant. The cold reading is denied all of it. It reports tensions, names the constraint each one violates, explains why it is a tension, and then stops: no fixes, no ranking, no verdict. What to do about them is the human's call.

> "The problem with reviewing our own briefs is that we already agree with them," said Alice, who maintains abcd's design record. "By the third read I'm not reading the document any more, I'm remembering the argument we had about it. I want a reader who doesn't know that argument happened — one that just sees a brief promising a thing on page two and quietly forbidding it on page nine, and says so without softening it into a suggestion."

## Why This Matters

abcd's design record is now large enough that no one comes to it cold. The brief, the principles, the intents and the ADRs cross-reference each other, and every reader arrives already knowing why things are shaped as they are. That knowledge is what makes an embedded reviewer fast — and it is exactly what makes them miss constraint divergence, because familiarity normalises the inconsistency. A reader who knows why the team framed things as they did will unconsciously smooth over what a stranger would trip on.

This is the Specification pass of the ABCD sensemaking method (see [the method note](../../research/notes/2026-07-13-abcd-sensemaking-method.md)), and abcd currently has no surface that performs it. The gap is not "another reviewer" — abcd has four. It is that all four are warm, and the method's value comes from a reader that is structurally denied context. abcd already believes this in the abstract: [`evaluator-outside-the-loop`](../../principles/evaluator-outside-the-loop.md) isolates the evaluator from the gate it judges. The cold reading extends that isolation from *what the evaluator may edit* to *what the evaluator may know*.

## What's In Scope

- A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone.
- The blindness contract, which is the substance of the intent and not a stylistic preference: the reading carries no project context, consults no prior reading, and is given no access to the ledger of what was previously dispositioned.
- The detection shape — **tension**, **constraint in play** (explicit or implicit, and what implies it), **why it is a tension** — with no fourth element proposing a remedy.
- The three prohibitions that define the output: propose no fixes, rank nothing (detections come in document order), pre-judge no intentionality (no "this may be deliberate, but…").
- Re-raising a previously surfaced tension that is still present in the document, per [`recurrence-is-signal`](../../principles/recurrence-is-signal.md) — the cold reading is the detector that principle is written for.
- A settled-enough check: a visibly mid-draft target (placeholders, TODOs, half-specified sections) is flagged in one line as likely to yield incompleteness rather than genuine constraint divergence, with the pass offered anyway.
- Each detection lands as a reading record in the issue tier via the cold-reading output contract — carrying a run-scoped identifier minted on the adr-45 mint, never content-derived, so a re-raised tension is countable as recurrence rather than collapsed into its first appearance.

## What's Out of Scope

- **The warm ledger and the disposition step.** These are operated by a human against a record the cold reading is deliberately kept blind to. This intent must not disposition, assign priority, or assess closure — doing so from inside the detector collapses the separation the method depends on.
- **Duplicating the grill (itd-27 / itd-42) or the foundations auditor (itd-55).** The grill interrogates acceptance criteria interactively; the auditor asks where a justification terminates; the cold reading asks whether the framing contradicts the document's own constraints. Three different questions, and the third is the only blind one.
- **Reviewing code.** The target is a design document. `ruthless-reviewer` owns diffs.
- **Auto-firing on a lifecycle transition.** Whether a cold reading is required before an intent is promoted is a policy question for later, not part of establishing the surface.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a design document containing a constraint stated in one section and breached by the framing in another, **when** the cold reading runs, **then** it reports that divergence as a detection carrying the tension, the constraint in play, and why it is a tension — and carrying no proposed remedy.
- **Given** a document whose surrounding project context would explain away a tension, **when** the cold reading runs with that context available in the environment, **then** the detection is still raised — the context does not suppress it.
- **Given** a tension that was surfaced by a previous cold reading and is still present in the document, **when** the cold reading runs again, **then** it is re-raised rather than suppressed as already-seen.
- **Given** a document with no constraint divergence, **when** the cold reading runs, **then** it reports the detections it has (possibly none) and adds the single closing line "No divergence beyond the detections above" — and does **not** declare the document closed or approved, since closure is a human judgement made against the ledger.
- **Given** a visibly mid-draft document, **when** the cold reading runs, **then** it says so in one line and flags that detections may be dominated by incompleteness, before proceeding.

## Open Questions

- ~~**Is the blindness enforceable, or only instructed?**~~ **Settled by design:** blindness is a property of what the input assembler passes, not an instruction to the reader. The reading runs only over an assembled input (positive inclusion, field projection, exclusion asserted by eval), invoked with no free-text argument, and its manifest makes what was passed auditable after the fact. Instructed-blindness is retired as the mechanism; the eval is what falsifies the contract rather than asserting it.
- ~~The two disposition axes~~ **Narrowed:** the disposition record reserves the two-axis hold field (frame-location crossed with MoSCoW priority), present in the schema and unpopulated — deferred by decision, reserved so retrofitting is never needed. The detection shape itself stays three-element; the axes belong to the human's disposition, never to the reading.
