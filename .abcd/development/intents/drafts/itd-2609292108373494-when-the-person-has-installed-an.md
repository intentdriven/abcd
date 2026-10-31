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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- A skill has a small, stable on-disk shape: `SKILL.md` with `name` (which "Must
  match the parent directory name") and `description`, optional `compatibility`,
  and progressive disclosure (Agent Skills specification). Detection reads
  those two fields and one more: a skill with `disable-model-invocation: true`
  cannot be invoked by the agent on its own (its description is not even in
  context), so such a skill is detected but cannot be handed over, and abcd
  tells the person to type the skill's command or runs its own interview.
- Where a host looks differs per host: the host abcd ships for reads personal,
  project, nested, additional-directory and plugin skill directories, and other
  hosts use their own (one 2026 survey lists six). The path table is the part a
  new host replaces.
- abcd can start a skill only through the host (its CLI or SDK loads a leading
  `/skill-name`), and a headless start cannot hold an interview: there the
  host's question tool auto-resolves with an empty answer (anthropics/claude-code
  issue 50728, closed "not planned"), and even interactively it has an open
  rendering bug (February 2026).
  A hand-over is therefore only meaningful inside a live interactive session,
  started by abcd's command page telling the agent to invoke the named skill.
- Public interviewing skills disagree on abcd's core rule: the brainstorming
  skill forbids option menus ("Options steer"), a grill-me skill gives "your
  recommended answer" with each question. abcd's rules travel to the external
  interviewer as a stated brief, abcd records which interviewer asked, and the
  record shape is unaffected because abcd writes it from the answers.
- The skill ecosystem is unvetted (a vendor scan put critical issues in 13.4 per
  cent of 3,984 skills; the vendor sells the remedy): detect one skill the person
  named, never scan for "any interviewing skill".

**Path 2**: the person names one skill (user-level, overridable per repository);
abcd checks the host-specific path for a `SKILL.md` whose `name` matches its
directory and that does not set `disable-model-invocation`; the command page
hands over in a live session and prints the asking
discipline it expects; otherwise, and always when headless, abcd's own interview
runs and says so. Covers the grill and the planning interview alike. No Go
dependency.

The independent fit-challenge runs at plan time.

## Open Questions

- **How the person names the skill**: a setting in the repository's
  configuration, a user-level setting, or detection by name.
- **Which of abcd's question rules bind the external interviewer**: one
  question at a time and the options that widen are abcd's stance; does the
  hand-over pass them to the skill, or accept the skill's own?
- **Which interviews it covers**: the grill of a draft only, or the planning
  interview too.
- (SOTA 2026-10-23) Whether answers taken by an interviewer that recommends
  (grill-me's shape) need a marker in the record, given
  `widen-options-never-recommend` binds abcd's own questions only.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
