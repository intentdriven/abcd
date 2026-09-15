---
term: loop
bounded_context: core
definition: The record loop — brief to intent to spec to shipped work to audited verdict and back onto the brief — which shipping closes twice, once by grading the acceptance criteria and once by rewriting the brief passage. Two other loops carry the word and are always qualified: the autonomous-run loop and the lifeboat round-trip.
aliases: ["the loop", "closed loop", "record loop"]
forbidden_synonyms: ["workflow", "pipeline", "cycle", "process"]
status: draft
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/phase
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# loop

The **loop** is abcd's shape: a brief states what the project is, an [intent](intent.md) states
one user-facing change, `plan` mints a [spec](spec.md), the work ships, and the verdict is
written back. Shipping **closes the loop twice** — the intent auditor grades each acceptance
bullet against the repository and writes the verdict onto the intent, and the same change
rewrites the brief passage to describe what actually shipped. The second half is what keeps the
brief a true description rather than a wish list. The user-facing account is
[`docs/explanation/process.md`](../../../../../docs/explanation/process.md).

The loop has a last step and nothing runs it automatically: `abcd intent plan` moves a draft to
`planned/`, and only the spec store's `close` moves a planned intent to `shipped/`
([`commands/intent.md`](../../../../../commands/intent.md)).

## Senses

| Sense | The one spelling | Where it lives |
|---|---|---|
| Brief to intent to spec to shipped work to verdict and back | **the loop**, or "the record loop" where a run loop is also in view | [`docs/explanation/process.md`](../../../../../docs/explanation/process.md), [`commands/intent.md`](../../../../../commands/intent.md) |
| The iteration an autonomous-run adapter supplies: pick up ready work, gate each step on a receipt, apply the safety guard | **the run loop** | [adr-27](../../../decisions/adrs/0027-autonomous-run-pluggable-seam.md), [`roadmap/phases/phase-5-run-seam.md`](../../../roadmap/phases/phase-5-run-seam.md) |
| Pack a lifeboat out of a repository and unpack it into a fresh one | **the lifeboat round-trip** | [`roadmap/phases/phase-6-lifeboat.md`](../../../roadmap/phases/phase-6-lifeboat.md), [disembark](disembark.md) |

**abcd never owns the run loop.** It defines the contract — iterate over ready work, gate on a
receipt, guard for safety — and a pluggable adapter supplies the loop itself. That is the whole
reason the second sense needs its own name.

**Where the record disagrees.** The record has no settled noun. It writes "closes the loop
twice", "Phase 6 closes abcd's loop with the lifeboat round-trip", and "the loop has a last
step" — three different loops, one bare article, no definition outside the page that coins each
(iss-2609012245352480). This entry is `draft` because it fixes a spelling the prose has not
yet adopted: "closed loop" is recorded here as an alias, not as the form to write.

## When to use

Use "the loop" for the record loop, and only where no run loop or round-trip is in view. Name
the other two in full every time.

## When NOT to use

Do not use "the loop" for a [phase](phase.md) — a phase is a stretch of the sequence, and the
loop is the cycle each change runs through, whatever phase it lands in. Do not use it for the
agent loop of the wider literature; the terminology crosswalk records abcd's position on that
term separately.

## Examples

- "Shipping closes the loop twice: the verdict lands on the intent and the brief passage is rewritten."
- "The run loop gates each iteration on a receipt, whichever adapter provides it."
- "The lifeboat round-trip — disembark on a corpus repo, embark into an empty target — is the integration milestone."

## Related terms

- [intent](intent.md), [spec](spec.md), [brief](brief.md) — the record loop's three record kinds
- [phase](phase.md) — the sequencing unit, not a stage of the loop
- [disembark](disembark.md), [lifeboat](lifeboat.md) — the round-trip's two halves
