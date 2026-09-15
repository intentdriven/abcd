---
id: itd-190
slug: the-claim-recording-gradient-an-intent-s-three-claim-kinds-c
spec_id: null
kind: discipline
kind_notes: "Cross-cutting gate over the intent readiness gate: an intent's three claim kinds — criterion, mechanism, context — carry three recording requirements, and the gate holds them. No user moment of its own; it imposes an acceptance gate every standalone intent inherits, sitting above itd-1 (which mandates the criterion claim) and extending the same shape to the other two. Enforcement is STAGED per the promotion ladder: the gradient and the nullity grammar are documented and the schema exists before the gate refuses, population is forward-only, and discipline-kind records are exempt."
suggested_kind: discipline
reclassification_history: []
builds_on: [itd-1]
severity: minor
impact: additive
---

# The claim recording gradient: an intent's three claim kinds carry three recording requirements, and the readiness gate holds them

## The rule

An intent carries up to three kinds of claim, each with its own recording
requirement at the readiness gate:

| Claim | Section | Requirement |
| --- | --- | --- |
| Criterion | `## Acceptance Criteria` (Given/When/Then) | Mandatory — exists today per itd-1 |
| Mechanism | `## Mechanism` | Prompted, nullable — declined with the nullity token; a present-but-empty section is a gate fault |
| Context | `## Scope Conditions` | Mandatory, with the nullity token as the explicit "none stated" — never left blank |

The nullity grammar is one exact token: `None stated.`, alone on its line
under the section heading — the same grammar for both sections, so the gate
distinguishes it from prose that happens to say "none" without semantics.

The distinction the gradient preserves: **an absent section is a claim not
carried; a recorded nullity is a claim considered and declined.** They are
never collapsed — three byte states, three semantics (absent; empty = gate
fault; nullity token = declined). An auditor-side flag for
mechanism-nullity intents lands with the record that owns verdicts, when
one does — no rule ships without an owner.

**Population** is forward-only, per the ruled population properties:
the gradient binds standalone intents at the readiness gate from adoption
on; existing records are untouched (sparseness is information, and an
absent stamp is never backfilled); discipline-kind records are exempt —
their template carries no claim sections.

Each scope condition carries a persistent identity that survives edits to
its text, so a later disposition (survived / narrowed / falsified /
untested) attaches to the condition, not to a string.

## The gate

- **Given** an intent without scope conditions and without an explicit
  nullity, **when** the readiness gate runs, **then** the gate exits
  non-zero and names the missing field.
- **Given** an intent whose `## Mechanism` carries the nullity token,
  **when** the readiness gate runs, **then** the gate passes and reports
  the recorded nullity.
- **Given** an intent whose `## Mechanism` is present but empty, **when**
  the readiness gate runs, **then** the gate exits non-zero and names the
  section.
- **Given** a scope condition whose text is edited, **when** the intent is
  re-read, **then** the condition's identity is unchanged.

## Staging

Lands staged per `loud-staging` and the itd-84/itd-81 pattern: the gradient
is documented and the schema exists before the gate refuses (schema before
commands); the refusal arrives calibrated, and any window in which the
format precedes its enforcement is disclosed, never papered over.

