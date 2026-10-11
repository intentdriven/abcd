---
id: spc-18
slug: abcd-gates-a-new-idea-before-it-becomes
intent: itd-104
---
# abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea

## Summary

spc-18 delivers itd-104's idea-admission protocol: `/abcd:ideate`, an
optional three-leg gauntlet — primary-source research, a record grill, and an
independent adversarial review — whose outcome is a registered verdict with
rejected alternatives, recorded whether the idea survives or dies. Only a
survivor graduates to a draft intent. The design decisions below were settled
by the 2026-07-27 grill; this spec records them together with the mechanism —
it does not reopen them.

## Settled constraints (from the grill)

- **Optional verb, never a gate.** Capture friction stays at one line: no
  ideate step is required or suggested as blocking by the intent or capture
  verbs. The routing help carries the nudge — ideate is named for big,
  unproven ideas.
- **The adversarial leg codifies fresh-context, off-policy, unknown
  authorship** — the evaluator did not do the research and receives the idea
  as an artefact of unknown authorship. This is the only measured debiasing
  effect, per the salvaged 2026-07-14 research record, and it is the
  evaluator-outside-the-loop principle applied to ideas.
- **Two load-bearing steps survive automation** (from the proven manual run):
  check the record first, and open the primary source.

## Mechanism

### Division of labour

Ideate follows the host-delegated pattern the disembark family proved: the
host runs the judgement legs as agents; the abcd binary is the deterministic
frame that scaffolds the run, validates the legs' outputs, and writes the
record. The binary never does LLM work; the host never writes the record
directly.

### The three legs, in order

1. **Primary-source research.** A host agent researches the idea's
   load-bearing claims, each checked against its primary source — a claim
   carries its primary URL or document reference, never only a secondary
   citation. Output: a claims table (claim, primary source, verified /
   falsified / unverifiable).
2. **Record grill.** A host agent reads the existing record — brief, intents
   (all buckets), ADRs, principles — and answers: does an entry cover,
   contradict, or supersede this idea? Every hit is cited by record id. The
   binary validates at ingest that every cited id resolves in the repository
   — a verdict citing a non-existent record is refused.
3. **Adversarial review.** A fresh-context agent, off-policy, receiving the
   idea and the two earlier legs' outputs as artefacts of unknown authorship,
   tries to kill the idea. It did not conduct the research (evaluator outside
   the loop).

### Verdict record (spec-time decision)

The verdict is a dated research record:
`.abcd/development/research/YYYY-MM-DD-ideate-<idea-slug>.md`, written by the
binary from a structured verdict the host supplies
(`abcd ideate record <idea-slug> --verdict-json <file>`). It contains the
idea as captured, the three legs' findings, the verdict
(survives / killed / reframed), the rejected alternatives, and — for record
grill hits — the cited record ids. A one-line dated entry in
`.abcd/work/DECISIONS.md` points at it. No new record family: killed ideas
are research outcomes, and the research directory is where a future session
looks before re-proposing one. A surviving idea graduates through the
existing quoted-text intent create — ideate mints no intents itself.

### Wiring

Both planes at delivery: the `abcd ideate` verb family (CLI) and the
`/abcd:ideate` skill (`commands/abcd/ideate.md`) that orchestrates the three
legs and feeds the verdict to the binary. The routing help in the intent and
capture surfaces names ideate as the optional path for big, unproven ideas —
a pointer, never a precondition.

## Acceptance-criteria mapping

- AC 1 (never blocking; routing help names it) → Settled constraints +
  Wiring.
- AC 2 (three legs in order; adversarial leg fresh-context, off-policy,
  unknown authorship) → The three legs.
- AC 3 (decision recorded with rejected alternatives either way; only
  survivors graduate) → Verdict record.
- AC 4 (record-grill hits cited by id, validated) → The three legs + Verdict
  record.

## Out of scope

- Consuming a lifeboat or external document as the idea source (open
  question retained; the idea arrives as quoted text in this spec).
- Any change to capture or intent-create flow beyond the routing-help text.
- Automation of the legs inside the binary — the legs are host agents by
  boundary; the binary validates and records.
