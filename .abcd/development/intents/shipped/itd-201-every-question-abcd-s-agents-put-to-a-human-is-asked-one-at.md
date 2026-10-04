---
id: itd-201
slug: every-question-abcd-s-agents-put-to-a-human-is-asked-one-at
spec_id: spc-2610030944505997
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: [itd-200]
severity: major
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
related_intents: [itd-2610030810350727]
reverses: [itd-110]
bundle: asking-and-layout
---

# Every question abcd's agents put to a human is asked one at a time, in plain language, with options that widen

## Press Release

> **A question from an abcd agent is now something a person can answer
> without reading the record.** Whenever an agent needs a human decision, at
> a planning interview, a decomposition routing, a grill, or any stop for a
> verdict, it asks one thing at a time through the host's interactive
> question tool: tabs for the parts of one thing, and a question that depends
> on an earlier answer asked alone. Each question quotes the thing being
> decided in full, in paragraphs and lists, carries one concrete example of
> what each answer would mean in practice, and offers options that widen
> rather than recommend: no starred default, no recommended label, and the
> null answer always offered. Deferral is recorded as an answer; silence is
> never consent. The rule ships as a rule domain that abcd installs into
> every repository it manages, so a session that never read the protocol
> still follows it from its first prompt.

> "I was handed five design questions in one message, numbered, each three
> lines of record vocabulary," said a product thinker who answers for what
> their team ships. "I could not tell what any of them was asking me to
> choose between. Now I get one at a time, in words I use, with an example,
> and I answer in a click."

## Why This Matters

The record already holds the two halves that matter for options: the
widen-options rule and the three drafts that fix how options are offered.
What it does not hold is how a question reaches a person at all. On
2026-09-01 the product thinker received the outstanding design decisions as a
numbered list inside a long message and could not answer them; the same
session then asked the status-line interview one question at a time and
every answer landed. The difference was not the content but the delivery.

A rule that lives only in an interview page is followed by sessions that
read the page. A rule domain is injected by the loader whenever a prompt
matches its recall words, and re-injected after compaction, so it reaches
every session in this repository. Propagation is what makes it abcd's rule
rather than one repository's habit: `ahoy install` writes the domain into a
managed repository's rules file the way it writes the other conventions.

## Decisions (product thinker, 2026-09-01)

- **The register follows the addressee.** A product thinker is asked in
  product terms: outcomes and choices, no record ids, no code, no internals.
  A technical facilitator is asked in the mechanism's terms, with the ids and
  the trade-offs. When the agent does not know which hat the human wears, the
  first question asks that, and the itd-200 mode records the answer so the
  next question does not ask again.
- **Only real choices are asked** (ruled by the product thinker, 2026-09-20;
  iss-2609202055103741). A question is asked only where two or more answers
  are each defensible on the record; its options are exactly those answers
  plus the null answer. A decision with one defensible answer is recorded as a
  decision line naming why no question was put, not asked.
- **The question shows what it asks about** (ruled by the product thinker,
  2026-09-20; iss-2609202058058301). Text the human is asked to accept, edit
  or strike is quoted in the question itself, never referred to.
- **Each register has a knowledge floor** (iss-2609020716236446). The product
  thinker is not assumed to know version control, a shell, permissions, CI,
  hooks, environment variables, checksums, symbolic links or record ids; a
  concept below that floor is introduced in one product-terms sentence before
  it is used. The GRILL domain and the interview page carry the text.

## What's In Scope

- The GRILL rule domain in this repository's `.abcd/rules.json`, and the
  same text in the `/abcd:intent` interview page, so the protocol and the
  injected rule match.
- The domain in the binary's bundled defaults, so a managed repository
  receives it at install and update, with the usual per-repo override.
- A check that the bundled default and this repository's copy do not drift.

## What's Out of Scope

- The options rule's content (itd-167, itd-168, itd-169 on the design
  branch): this intent delivers the asking, not the widening.
- Any harness that offers no interactive question tool: there the rule
  degrades to one question per message, stated as such.

## Mechanism

We expect a person to answer more of the questions put to them, because a
single plain question with an example is answerable where a numbered list
of record vocabulary is not.

## Scope Conditions

- A host without an interactive question tool degrades to one question per message, in the same order and wording. <!-- cond: cond-2610030944504162 -->
- The register and mode rules, and the question check's mode gate, hold for abcd's own interviews (its command pages), not for questions other tools ask in a managed repository. <!-- cond: cond-2610030944507807 -->
- The Writing Style rules on casing after a colon or semicolon stay review, never a machine check (adr-54). <!-- cond: cond-2610030944507744 -->

_Confirmed by the technical facilitator at the planning interview, 2026-10-03._

## Acceptance Criteria

- R1 (facilitator; CONFIRMED 2026-10-03) **Given** a managed repository whose rules file declares no GRILL key, **when** a prompt matches the recall words, **then** `abcd rules GRILL --json` renders the domain with `source: bundled`, generated from the one source that also holds the layout's field limits.
- **Given** an agent at an abcd interview, **when** it needs a decision, **then** it asks one thing at a time through the interactive question tool (tabs for the parts of one thing; a question that depends on an earlier answer asked alone, after that answer). _(Reworded by decision 2.)_
- **Given** a question is asked, **when** the person reads it, **then** it quotes the thing being decided in full, in paragraphs and lists, carries one example of the thing being decided in the question text and, in each option's description, what choosing it means, names each option's gain and cost, and offers "decide later". _(Reworded by decisions 2, 3 and 4.)_
- **Given** the person asked is the technical facilitator, **when** a question is asked, **then** it carries the mechanism and the record ids; for the product thinker see R4. _(Merged with R4.)_
- **Given** the addressee of an abcd interview is unknown, **when** its first question is asked, **then** it asks which role the person holds before anything else. _(Scoped to abcd's interviews by decision 10.)_
- R2 (facilitator; CONFIRMED 2026-10-03) **Given** the bundled defaults declare GRILL by hand, **when** the defaults load, **then** they refuse, as they refuse a hand-written SHELL. _(Replaces the drift check against a repository copy, which no longer exists, decision 8.)_

- R3 (product thinker; CONFIRMED 2026-10-03) **Given** five decisions outstanding, **when** the agent needs them, **then** it asks about one thing at a time, tabs for the parts of one thing, never a numbered list in a message (example: five open questions about one feature's criteria as four tabs on one screen and the fifth on the next, since a screen holds at most four; a question depending on their answers comes alone afterwards).
- R4 (product thinker; CONFIRMED 2026-10-03, checked by the question check) **Given** the person asked is the product thinker, **when** a question is asked, **then** it names no record number and no command (example: "the project's folder of work", never "the checkout").
- R6 (facilitator; CONFIRMED 2026-10-03) **Given** an option labelled "(Recommended)" or starred, **when** the question check runs, **then** it refuses naming the label, and the remedy says the host's own instruction asks for it and abcd's rule reverses it, so the agent does not loop.
- R5 (product thinker; CONFIRMED 2026-10-03) **Given** the person answers "decide later", **when** the interview moves on, **then** the record shows that answer, and nothing is taken as agreed from silence (example: the bundle name deferred is recorded as deferred, and planning stops until it is answered).

## Interview decisions (2026-10-03)

1. The product thinker: keep this as one record. The asking rules and their shipping to every project abcd looks after stay together in the bundle, with the cost (about a page of text in most sessions of every project) and the setup questions it brings.
2. The product thinker: the promise and criteria follow the layout decisions: "one thing at a time: tabs for the parts of one thing, and a question that depends on an earlier answer asked alone", and "the thing being decided quoted in full, in paragraphs and lists" in place of "one sentence of context". The rule text is amended to match when this is built.
3. The product thinker: an example lives in the question text (one example of the thing being decided) and in each option's description (what choosing it means); the side preview may repeat it with the trade-off, never hold it alone, since a preview can be cut off.
4. The product thinker: trade-offs stay neutral by the wording "each option's gain and cost, never one option's alone".
5. The product thinker (the knowledge-floor ruling owed since iss-2609020716236446 was resolved without a recorded answer): the floor lives on one glossary page, and the rule carries one line pointing to it. The earlier resolution's choice of a list in the rule text was the facilitator's, not the product thinker's; this answer replaces it when built.
6. The product thinker, on itd-110's "the recommended answer distinctly styled" (retired with itd-110): no option is ever styled or ordered as recommended; a recommendation appears only when the person asks for one, given in prose beside the question, never as an option. This reverses that part of itd-110.
12. Decided without a question (one defensible answer: the host and the layout's decision 10 allow at most four tabs per screen): R3's example reads four tabs, then the fifth on the next screen. Found by the spec writer, 2026-10-03.
8. The technical facilitator: GRILL is generated from one Go source, as SHELL is, with the layout's field limits; the generated text carries no record ids and no `go run`; abcd's own repository override is deleted in the same change.
9. The technical facilitator: the recall cost is accepted (about 1,100 tokens in most sessions of every managed repository); a repository silences it with {"GRILL": {"state": "dormant"}}.
10. The technical facilitator: the register and mode rules, and the question check's mode gate, apply to abcd's own interviews only; other tools' questions in a managed repository are not refused.
11. The technical facilitator: builds_on itd-200, reverses itd-110, and three scope conditions written.
7. The product thinker: itd-2609151541116052 (show the thing first) is folded into itd-2610030810350727 and superseded by it; the bundle is this record and the layout intent.
13. 2026-10-03, note: decision 20 of the layout intent (itd-2610030810350727, the product thinker: "No side previews") moves each option's gain and cost into its description. A side preview hides every option's description in the host and is cut to the rows the host leaves it, so abcd's questions carry none: where decision 3 says the side preview "may repeat it with the trade-off", and where the layout intent's decision 17 puts the trade-offs in the previews, the description now holds the meaning, the gain and the cost, and the question check refuses a preview.

## Review findings (design and feasibility, 2026-10-03, bundle with itd-2610030810350727 and itd-2609151541116052)

From the review recorded in the local tier (reports/review-bundle-design.md). Nothing here is settled; each item is put to its addressee at the interview.

- Blocking, mechanism: a managed repository receives a bundled rule domain through the abcd binary, never as a copy written into its rules file (setup writes the empty skeleton on purpose: the default domains live once in the binary). So the asking rules ship by joining abcd's bundled default domains, abcd's own repository drops its override in the same change, and criterion 4 becomes "the managed repository renders the rules with source: bundled", with criterion 7 (a drift check against a copy) replaced by deleting the copy.
- Blocking, the text: the rule domain as written names abcd-internal records and the `go run` form, which a managed repository cannot use. Proposed: generate it from one source in the binary, as the shell rules are, together with the layout's field limits (the layout's criterion A7, widened from the limits to the whole text).
- Should-fix, cost: the domain is about 1,100 tokens and recalls on everyday words (which, plan, decide, options, choose), so it would land in most sessions of every managed repository. Proposed: broad recall only for the rules that change what a person sees; the register, floor and mode rules behind narrower words; the per-repository escape stated.
- Should-fix, managed repositories: "the first question asks which hat the person wears", with the question check refusing every question while the mode reads managed, would make every managed repository's first question abcd's two-role question. Proposed: the register and mode rules cover abcd's own interviews only, and the rule text says so.
- Should-fix, staleness: criterion 1 counts four rules (the domain has twelve); three intents are named as being on a design branch (they are drafts).

Contradictions with the layout bundle's settled decisions, each for the product thinker, quoted from both sides: "one question at a time" here against the layout's decision 10 (tabs for parts of one thing, a dependent question alone); "one sentence of context" here against the layout's decision 1 (paragraphs and lists, the thing quoted in full); "options that widen rather than recommend" here against the layout's decision 17 (trade-offs in the previews, which stay neutral only if every option's cost and gain are named); and three homes for examples (the option's preview here, the question text in the layout's decisions 12 and 16, the option description in the layout's press release).

Proposed acceptance criteria (agent-seeded, unconfirmed; each walked with an example at the interview): R1 (facilitator) a managed repository with no GRILL key renders the rules from the binary with source bundled; R2 (facilitator) the bundled defaults refuse a hand-written GRILL, as they refuse SHELL; R3 (product thinker) five outstanding decisions are asked one thing at a time, tabs for the parts of one thing, never a numbered list in a message; R4 (product thinker) while the mode reads product thinker, a question names no record id and no command; R5 (product thinker) "decide later" is recorded as the answer, and nothing is taken as agreed from silence; R6 (facilitator) a "(Recommended)" or starred option is refused, and the remedy says the host's own instruction asks for it and abcd's rule reverses it.

## Open Questions

1. Whether the domain's recall words are wide enough to fire on every stop
   for a verdict, or whether the mode verb from itd-200 should inject it
   directly when the agent parks a stop.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-5796271104cf -->
Fidelity review OWED (receipt rcp-5796271104cf).
<!-- abcd-review-end receipt=rcp-5796271104cf -->
