---
id: itd-2609292108373494
slug: when-the-person-has-installed-an
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-165]
related_adrs: [adr-22]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The grill hands the interview to an installed interviewing skill, and runs its own when there is none

Typed links: `related_issues` [iss-165](../../../work/issues/open/iss-165-the-grill-interview-capability-should.md) (the record this draft plans); `related_adrs` [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) (native default, an external tool as an opt-in adapter).

## Press Release

> When the person has installed an interviewing skill they prefer, abcd's grill hands the interview to it; when they have not, abcd runs its own full interview. The hand-over is visible and opt-in: abcd says which interviewer is asking, the installed skill is never required, and the record abcd keeps from the interview has the same shape whichever interviewer asked. "I already use an interviewing skill I trust everywhere else," said Iris, a product thinker. "Now abcd uses it too, and on a machine without it I still get abcd's own interview rather than none."

## Why This Matters

abcd's interviews (the grill of a draft and the planning interview) run on
abcd's own question rules, the GRILL rule domain. A person who already keeps an
interviewing skill they trust in their host gets a second, different interviewer
inside abcd. iss-165 asks for the pattern abcd uses elsewhere, basics built in
and the state of the art delegated: hand the interview to the installed skill
when it is present, and run abcd's own comprehensive interview when it is not.
The product thinker ruled on 2026-09-29 to plan it (ruling J11).

## What's In Scope

- **Detection** of an installed interviewing skill the person has named, at the
  moment an interview starts.
- **The hand-over**: the interview runs through that skill, and abcd says which
  interviewer is asking.
- **The fallback**: with no such skill, abcd's own interview runs unchanged.
- **One record shape**: what abcd keeps from the interview (the draft's edited
  sections, the answers it records) is the same whichever interviewer asked.

## What's Out of Scope

- Shipping or vendoring any third-party skill: the installed skill is an opt-in
  dependency, never a requirement.
- The sign-off act itself: `abcd intent plan` stays the product thinker's act,
  whichever interviewer ran the questions.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from iss-165 and the ruling, unconfirmed: the planning
interview walks each one.

- **Given** a host with the named interviewing skill installed, **when** an
  interview starts, **then** abcd hands the interview to that skill and says so
  before the first question.
- **Given** a host without it, **when** an interview starts, **then** abcd's own
  interview runs and says which interviewer is asking.
- **Given** an interview run by either interviewer, **when** it ends, **then**
  the draft's recorded sections and the planning gate's inputs have the same
  shape.
- **Given** a hand-over that fails part-way, **when** the interview resumes,
  **then** abcd's own interview continues from the last recorded answer and says
  that it took over.

## Open Questions

- **How the person names the skill**: a setting in the repository's
  configuration, a user-level setting, or detection by name.
- **Which of abcd's question rules bind the external interviewer**: one
  question at a time and the options that widen are abcd's stance; does the
  hand-over pass them to the skill, or accept the skill's own?
- **Which interviews it covers**: the grill of a draft only, or the planning
  interview too.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
