---
id: itd-201
slug: every-question-abcd-s-agents-put-to-a
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

<!-- abcd-review: INGESTED receipt=rcp-5796271104cf -->
Fidelity review — receipt rcp-5796271104cf (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:659dab7db2733d2ec5ff149ef80ed4bf67dca3209c5a43294def6e3310dc8628
Input attestations: diff:1b22bae3169447f0c773d703713c7f3b6a9261e3..4b9345575123ba47c2ee6fc1cd6dc7d27804812d (PR #784, spec steps 1 and 2; the request named no range, so it was read from the merge commit's parents; digest over git diff --no-color --no-ext-diff)@sha256:74de7e73c13db25c490cff764c4e7d4b22db849d5060b879ca83ab88752ddc63; diff:cc5121f6880b750dfda60f355d1beb6c37babddd..9392a45148241c28c7140b50660637184183ec40 (PR #788, spec steps 3 to 5 and the spec close 3fdb8a4ce; digest over git diff --no-color --no-ext-diff)@sha256:b3e33525369dc8b2e9de804c8d5e66e47de3d7ccb1fb1f476a187a8ae0816f21; repository:2e03c61e454bf09ecda8cee0ae2f377157b522f7 (origin/main, where every cited file:line was read; the local audit-notes commit 18170d03f above it touches none of them)@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 7 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: grill.go builds the GRILL domain at load from question.AskingRules(question.Default), the one Limits value in limits.go that the question check also reads; TestManagedRepositoryGetsGrillFromTheBinary gives a repository the empty rules skeleton ahoy writes, injects GRILL through the prompt hook on 'which option should we choose', and asserts 'abcd rules GRILL --json' reports source bundled with rules equal to the generated text (R1).
  evidence: internal/core/rules/grill.go:29 — "Rules: question.AskingRules(question.Default),"
  evidence: internal/core/question/limits.go:42 — "var Default = Limits{"
  evidence: internal/surface/cli/rules_grill_domain_test.go:49 — "skeleton := `{"schema_version":1,"disabled":false,"domains":{}}`"
  evidence: internal/surface/cli/rules_grill_domain_test.go:61 — "if got.Name != "GRILL" || got.Source != "bundled" {"
- ac-2 — MET_WITH_CONCERNS: The first GRILL rule tells every abcd interview to ask one thing at a time through the host's question tool, the parts of one thing as tabs and a dependent question alone after its answer (asserted by TestGrillTextAsksOneThingAtATime), and the check refuses more than four questions in one call with a remedy saying the same. Concern: only that count is enforced; whether one call holds the parts of one thing, whether a dependent question waited for its answer, and whether the agent used the tool rather than a message are rule text no hook can see.
  evidence: internal/core/question/asking.go:37 — "Ask one thing at a time, through the host's interactive question tool, never as a numbered list in a message."
  evidence: internal/core/question/asking.go:39 — "A question whose answer depends on an earlier one is asked alone, after that answer."
  evidence: internal/core/question/check.go:131 — "Remedy: "Ask one question, or up to four parts of one thing as tabs; ask a question that depends on an earlier answer alone, after that answer.","
  evidence: internal/core/rules/grill_test.go:103 — "r := ruleWith(grillRules(t), "one thing at a time", "as tabs", "up to four on a screen","
- ac-3 — MET_WITH_CONCERNS: The check refuses a question with no material before its closing question line (rule 11), an option with no description (rule 5) and a question whose last option is not a decide-later label (rule 6), each tested (TestOpenQuestionQuotesAndOffersDecideLater among them); the GRILL text states quoting in full in paragraphs and lists, one example in the question text, each option's meaning in its description, and each option's gain and cost. Concerns: 'in full', the one example and the gain and cost are rule text only, since rule 11 asks for one paragraph of material and nothing reads its content; and LaterLabels admits 'None of these' in place of 'Decide later', so a question passes the check without offering decide later.
  evidence: internal/core/question/asking.go:47 — "The thing being decided is quoted in full in the question itself, in paragraphs and lists, never referred to"
  evidence: internal/core/question/asking.go:68 — "Every question carries one example of the thing being decided, in the question text, and each option's description says what choosing that option means in practice."
  evidence: internal/core/question/asking.go:72 — "Each option's description names its gain and its cost, never one option's alone"
  evidence: internal/core/question/check.go:322 — "Quote the thing being decided in full first, in paragraphs and lists, then ask; never refer to it."
  evidence: internal/core/question/limits.go:49 — "LaterLabels: []string{"Decide later", "None of these"},"
  evidence: internal/core/question/limits_test.go:333 — "func TestOpenQuestionQuotesAndOffersDecideLater(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: The register rule gives the technical facilitator the mechanism, the ids and the trade-offs, the knowledge-floor page says leaving the ids out withholds the handle they act on, and the check's register rule never applies to a facilitator's question, so a Tech-chipped question naming a record number is admitted (the facilitator case of TestProductThinkerQuestionNamesNoRecordOrCommand). Concern: the mechanism and the ids are permitted and asked for, never required; nothing checks that a question put to the facilitator carries them.
  evidence: internal/core/question/asking.go:81 — "the technical facilitator gets the mechanism, the ids, and the trade-offs."
  evidence: .abcd/development/brief/glossary/interview/knowledge-floor.md:30 — "explaining it to them is padding, and leaving the ids out of their answer withholds the handle"
  evidence: internal/core/question/check.go:391 — "case Facilitator:"
  evidence: internal/surface/cli/guard_question_limits_test.go:338 — "t.Errorf("facilitator: the mechanism and the ids are theirs; code=%d stdout=%q stderr=%q", code, stdout, stderr)"
- ac-5 — MET_WITH_CONCERNS: One GRILL rule, scoped to abcd's own interviews and asserted by TestGrillTextScopesRegisterToAbcdInterviews, says the first question asks which role the person holds when that is not known. Concerns: it is rule text only, with no interview step or fixed question that asks the role and no check; the planning page sets the mode before the first question and the chip rule has the agent classify each question's addressee instead; and the rule's own claim that the mode records the answer is contradicted by the reset to managed on every answered question, so the answer survives only in the agent's context.
  evidence: internal/core/question/asking.go:83 — "In abcd's own interviews, when it is not known which role the person holds, the first question asks that, and the mode records the answer so the next question does not ask again."
  evidence: internal/core/rules/grill_test.go:171 — "{"role first", []string{"which role the person holds", "the first question asks that"}},"
  evidence: commands/intent.md:322 — "The sign-off is the product thinker's; set the mode before the first question"
  evidence: internal/core/mode/question.go:127 — "if err := SetAt(repoRoot, Managed); err != nil {"
- ac-6 — MET: withGrillDomain panics when the bundled rules.json declares GRILL, and mustParseDefaults calls it at load right after withShellDomain, which panics the same way on a hand-written SHELL; TestGrillDomainIsNotHandWritten asserts the embedded file declares no GRILL and TestHandWrittenGrillPanicsAtLoad asserts a hand-written GRILL panics naming the domain and 'by hand' (R2).
  evidence: internal/core/rules/grill.go:39 — "panic("rules: bundled defaults declare " + GrillDomain + " by hand; it is generated from internal/core/question")"
  evidence: internal/core/rules/rules.go:183 — "rs = withGrillDomain(rs)"
  evidence: internal/core/rules/shell.go:75 — "panic("rules: bundled defaults declare " + ShellDomain + " by hand; it is generated from the guard registry")"
  evidence: internal/core/rules/grill_test.go:82 — "func TestHandWrittenGrillPanicsAtLoad(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: The GRILL rule asks the parts of one thing as tabs, up to four on a screen and the rest on the next, a dependent question alone after that answer, and never a numbered list in a message (TestGrillTextAsksOneThingAtATime); the check refuses five questions in one call and admits the example's four tabs and then the fifth alone. Concerns: 'never a numbered list in a message' is prose no hook sees; and since iss-2610070637562567 the rows limit no longer refuses, so five criteria lumped into one question are shown with a note instead of refused.
  evidence: internal/core/question/asking.go:38 — "The parts of one thing (the criteria of one feature, the paragraphs of one text) are asked as tabs, up to %s on a screen, each short, and the rest on the next screen."
  evidence: internal/core/question/limits_test.go:107 — "{"five questions", RuleQuestionsPerCall, facilitator(), func(f *Fields) {"
  evidence: internal/core/question/limits_test.go:426 — "if fs := CheckLimits(four, Default, productThinker()); len(fs) != 0 {"
  evidence: internal/surface/cli/guard_question.go:199 — "// The rows limit alone does not refuse (iss-2610070637562567): a question"
- ac-8 — MET_WITH_CONCERNS: Rule 9 of the check refuses a record handle, a backtick span, go run, a /abcd: command or abcd followed by one of the binary's verbs in any field of a question for the product thinker, and TestProductThinkerQuestionNamesNoRecordOrCommand refuses each case under the product-thinker mode and, with no mode store, under a Product chip (R4). Concerns: the criterion's own example, 'the checkout', is vocabulary the check does not catch, held only by the register rule and the knowledge-floor page; and since iss-2610100626211810 a chipped question is checked against its chip rather than the mode, so a question for the product thinker that the agent chips Tech or Setup is never held to the register.
  evidence: internal/core/question/check.go:276 — "add(fl.part, RuleRegister, hit, "no record number and no command for the product thinker","
  evidence: internal/surface/cli/guard_question_limits_test.go:306 — "func TestProductThinkerQuestionNamesNoRecordOrCommand(t *testing.T) {"
  evidence: internal/surface/cli/guard_question.go:193 — "who = question.Unnamed"
  evidence: .abcd/development/brief/glossary/interview/knowledge-floor.md:52 — ""The question named the checkout, which is below the product thinker's knowledge floor; it"
- ac-9 — MET: Rule 8 refuses a label carrying '(Recommended)' in any case or opening or closing with a star, echoing the label, and its remedy RecommendedRemedy says the host's own instruction asks for a recommended first option and abcd's asking rule reverses it; TestRecommendedOrStarredOptionIsRefused asserts the deny names the label 'Number only *' and carries that remedy (R6).
  evidence: internal/core/question/check.go:77 — "const RecommendedRemedy = "The host's own instruction for this tool asks for a recommended first option; abcd's asking rule reverses it"
  evidence: internal/core/question/check.go:268 — "add(optionPart(i, "label"), RuleNeverRecommended, o.Label, "no (Recommended) label and no star mark", RecommendedRemedy)"
  evidence: internal/surface/cli/guard_question_limits_test.go:365 — "if !strings.Contains(lines[0], strconv.Quote("Number only *")) {"
  evidence: internal/surface/cli/guard_question_limits_test.go:368 — "if !strings.Contains(lines[0], question.RecommendedRemedy) {"
- ac-10 — MET_WITH_CONCERNS: The GRILL rule records deferral as an answer and never takes silence as consent (TestGrillTextRecordsDeferral), the check always requires a decide-later option, the planning page records a deferred question as deferred in the draft, the Terminal interview records the 'later' value, the build refuses a record that still asks an open question, and planning several drafts is refused without a bundle name, which holds the criterion's example (R5). Concerns: on the host path the deferral reaches the record only if the agent writes it; and planning stops mechanically only for the bundle name, elsewhere by page text limited to an open question that gates scope.
  evidence: internal/core/question/asking.go:78 — "Deferral is a real answer and is recorded as one; silence is never consent, and a step whose answer is missing is asked again rather than assumed."
  evidence: commands/intent.md:394 — "thinker defers is recorded as deferred in the draft; skipping it is not a"
  evidence: internal/core/intent/bundle.go:91 — "--bundle < name> is required to plan several intents as one (nothing moved)"
  evidence: internal/core/interview/written.go:581 — "Later: question.Option{Value: "later", Label: question.Default.LaterLabels[0],"
  evidence: internal/core/intent/startcheck.go:91 — "// startOpenQuestionsRow refuses a record that still asks a question."

Gap audit:
- honoured:
  - One thing at a time through the host's question tool, tabs for the parts of one thing, and a question that depends on an earlier answer asked alone
    evidence: internal/core/question/asking.go:37 — "Ask one thing at a time, through the host's interactive question tool, never as a numbered list in a message."
    evidence: internal/core/question/check.go:131 — "Ask one question, or up to four parts of one thing as tabs; ask a question that depends on an earlier answer alone, after that answer."
  - The thing being decided quoted in full, in paragraphs and lists, with one concrete example and what each answer means in practice
    evidence: internal/core/question/asking.go:47 — "The thing being decided is quoted in full in the question itself, in paragraphs and lists, never referred to"
    evidence: internal/core/question/asking.go:68 — "each option's description says what choosing that option means in practice"
  - Options that widen rather than recommend: no starred default and no recommended label, refused by the question check
    evidence: internal/core/question/check.go:268 — ""no (Recommended) label and no star mark", RecommendedRemedy"
  - The null answer always offered
    evidence: internal/core/question/check.go:246 — "as the last option; it is always offered."
  - Deferral is recorded as an answer; silence is never consent
    evidence: internal/core/question/asking.go:78 — "Deferral is a real answer and is recorded as one; silence is never consent"
    evidence: commands/intent.md:394 — "thinker defers is recorded as deferred in the draft"
  - The register follows the addressee: the product thinker sees no record id or command, the facilitator gets the mechanism and the ids
    evidence: internal/core/question/check.go:276 — "no record number and no command for the product thinker"
    evidence: internal/core/question/asking.go:81 — "the technical facilitator gets the mechanism, the ids, and the trade-offs."
  - The same text in the planning interview page as in the injected rule domain
    evidence: commands/intent.md:327 — "Written by `make asking-sync` from internal/core/question; edit asking.go or limits.go there, never this block."
    evidence: cmd/asking-sync/main_test.go:47 — "func TestIntentPageAskingBlockIsGenerated(t *testing.T) {"
  - The knowledge floor on one glossary page the rule points at (decision 5), and the dormant escape for a repository that does not want the domain (decision 9)
    evidence: internal/core/question/asking.go:85 — "the knowledge floor an explanation is measured against, is stated in full at"
    evidence: internal/core/ahoy/defaults/claude-md-marker-block.md:99 — "silences it with `{"GRILL": {"state": "dormant"}}` in `.abcd/rules.json`."
- diverged:
  - The rule ships as a rule domain that abcd installs into every repository it manages, ahoy install writing it into the managed repository's rules file: delivered instead through the binary's bundled defaults, generated at load, with ahoy writing only the empty skeleton (decision 8)
    evidence: .abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:56 — "`ahoy install` writes the domain into a"
    evidence: internal/core/rules/grill.go:44 — "rs.Domains[GrillDomain] = grillDomain()"
    evidence: internal/surface/cli/rules_grill_domain_test.go:47 — "The skeleton ahoy's stepRules writes: no domain declared, every bundled"
  - A session that never read the protocol still follows it from its first prompt: the domain is injected only on a prompt matching one of its recall words, and an agent that never received it writes no chip, so its question runs unchecked as another tool's while the mode reads managed
    evidence: internal/core/question/asking.go:19 — "var askingRecall = []string{"grill", "interview", "options", "decide", "decision", "plan", "choose", "which", "product thinker", "facilitator", "register"}"
    evidence: internal/surface/cli/guard_question.go:175 — "if !chip && st == mode.Managed {"
  - A check that the bundled default and this repository's copy do not drift: the copy was deleted and a hand-written GRILL panics at load, with a drift test against the generated text in its place (decision 8)
    evidence: internal/surface/cli/rules_grill_domain_test.go:84 — ".abcd/rules.json declares GRILL; the domain is generated into the binary and the override is deleted"
    evidence: internal/core/rules/grill_test.go:49 — "func TestGrillDomainIsGeneratedFromTheAskingRules(t *testing.T) {"
  - The mode records the role answer so the next question does not ask again: the mode is reset to managed on every answered question and the chip sets it per question, so the mode never holds the role
    evidence: internal/core/mode/question.go:127 — "if err := SetAt(repoRoot, Managed); err != nil {"
    evidence: internal/core/question/asking.go:90 — "the chip sets the mode when the question is asked, and the answer sets it back to managed"
  - Decide later always offered (the criterion's wording): the check also admits None of these as the one later option, so decide later itself can be absent
    evidence: internal/core/question/limits.go:49 — "LaterLabels: []string{"Decide later", "None of these"},"
- missing:
  - Whenever an agent needs a human decision, at any stop for a verdict: the intent's own open question 1 (whether the recall words fire on every stop for a verdict, or the mode verb should inject the domain when a stop is parked) is unanswered, and nothing injects GRILL at a parked stop, since neither the mode verb nor the mode store references the domain
    evidence: .abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:157 — "1. Whether the domain's recall words are wide enough to fire on every stop"
    evidence: internal/core/question/asking.go:19 — "var askingRecall = []string{"grill", "interview", "options""

Scope-condition dispositions:
- cond-2610030944504162 — untested: The degradation is written into the first GRILL rule every managed repository receives and asserted present by TestGrillTextAsksOneThingAtATime, but no interview ran on a host without a question tool, and the question check sees only calls to the host's question tool, so the order and wording of a question asked as a message are never checked.
  evidence: internal/core/question/asking.go:40 — "A host with no question tool asks one question per message, in the same order and the same words."
  evidence: internal/surface/cli/guard_question.go:45 — "// isQuestionTool reports whether a hook payload's tool is a question tool."
- cond-2610030944507807 — narrowed: The register, role-first and mode rules open with 'In abcd's own interviews', and a question without abcd's chip asked while the mode reads managed runs unchecked and unmarked (TestForeignQuestionIsNotRefused); but a question without the chip asked while the mode names somebody is treated as abcd's and refused on its rules (TestModeMadeQuestionAbcdsSaysSo), the trade-off the spec accepted in its open question 1, and since spc-2610180914469489 every question, another tool's included, is refused while the person is away. The delivered mode gate itself, an abcd chip refused while the mode reads managed, has since been replaced by the chip setting the mode.
  narrowing: Another tool's question in a managed repository escapes abcd's rules only while the mode reads managed and the person is present: while the mode names the product thinker or the facilitator, a question without abcd's chip is held to abcd's limits and register rule, and while the person is away every question is refused.
  evidence: internal/core/question/asking.go:80 — "In abcd's own interviews, address the person in their register"
  evidence: internal/surface/cli/guard_question.go:175 — "if !chip && st == mode.Managed {"
  evidence: internal/surface/cli/guard_question_limits_test.go:549 — "want := "treated as abcd's because the mode names " + who + "; another tool's question asked now is held to abcd's rules""
  evidence: internal/surface/cli/guard_question.go:92 — "return denyCall(awayQuestionRefusal())"
  evidence: internal/surface/cli/guard_question_away_test.go:58 — "{name: "another tool's", payload: func(t *testing.T, cwd string) string { return askPayload(t, cwd, foreign) }},"
- cond-2610030944507744 — narrowed: The delivered question check carries no casing rule among its fourteen (rule 12 is the em dash in a list item only), the GRILL text adds no casing check, and the writing-style guide labels both rules review permanently; but a machine check that predates this intent holds that casing in one fixed grammar: every verb's one-sentence summary is parsed under test, and the parser refuses a clause after the colon that does not open with 'Writes' and a clause after the semicolon that does not open with 'refuses', case-sensitively, the capital and the lower case chosen to follow the guide.
  narrowing: Holds for prose, for the asking rules and for every question abcd asks; it does not hold for each verb's one-sentence summary, whose fixed grammar is machine-checked for a capital after its colon and lower case after its semicolon.
  evidence: internal/core/question/check.go:33 — "RuleEmDashListItem Rule = "em-dash-in-list-item" // 12: no em dash in a list item"
  evidence: docs/reference/writing-style.md:63 — "| After a colon: A capital letter. | review, permanently"
  evidence: internal/core/surface/sentence.go:14 — "// "Writes", capitalised because the writing-style guide puts a capital after a"
  evidence: internal/core/surface/sentence.go:97 — "return Sentence{}, fmt.Errorf("the writing clause after the colon does not open with %q", writesOpening)"
  evidence: internal/core/surface/sentence.go:101 — "return Sentence{}, fmt.Errorf("the refusing clause after the semicolon does not open with %q or %q","
  evidence: internal/surface/cli/sentences_test.go:123 — "if _, err := surface.ParseSentence(sentence); err != nil {"
<!-- abcd-review-end receipt=rcp-5796271104cf -->
