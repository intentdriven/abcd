---
id: itd-2610030810350727
slug: every-question-abcd-puts-to-a-person
spec_id: spc-2610030944505997
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
supersedes: [itd-110, itd-2609151541116052]
refines: [itd-201, itd-2609151541116052]
related_intents: [itd-2609292108373494]
bundle: asking-and-layout
impact: additive
---

# Every question abcd asks reads the same way, on a narrow window and a wide one

## Press Release

> Whenever abcd asks you something, you can tell at a glance who the question is for, where you are in the interview, what is being decided, and what each answer means.
>
> Every question is built the same way, from parts abcd controls:
>
> - a short label saying who it is for and which question this is ("Product Q2"; a total only when the interview's length is known);
> - the thing being decided, quoted in full, broken into paragraphs and lists rather than one block of text, then one plain question;
> - options, each with a label of a few words and one or two sentences on what choosing it means;
> - always one option that is the way to decide later.
>
> Nothing you need lives only in a side preview. A question that breaks these rules is refused before it reaches you, naming the part and the limit, so it is asked again properly. In Claude Code's terminal view such a question reads at 80 columns without scrolling.

_Confirmed by the product thinker at the planning interview on 2026-10-03 (answer: lead with the moment, then the rules), with the note "use paragraph breaks, and use lists instead of one big blob of text"._

## Why This Matters

Two screenshots the product thinker took on 2026-10-03 (an 81-column terminal, Claude Code's question view, during a routing interview) show what a question looks like today. The bullets below are the facilitator's reading of those screenshots; the images were not kept in the record:

- Who the question is for and where the person is in the interview are buried in a bracketed preamble inside the question prose ("[For you as product thinker; queued R2 of 5. On your last note: ...]"), not shown as a label.
- The question is one paragraph of 10 to 13 wrapped lines mixing context, a proposal, three numbered parts and the example, so the thing being decided has no visual edge.
- The options are short labels on the left; what each one means sits in a boxed preview on the right that is narrower than the question and cuts its content to a few lines.
- The previous answer is echoed above in grey as a long block with an arrow, so the history competes with the question on screen.
- Vocabulary drifts between questions (R1/R2 numbering, "builder", "facilitator", "full check"), so the person reads every question cold.

abcd controls only the text it hands the host's question view (a short header chip, the question text, up to four options with a label, a description and an optional preview). It does not control the frame. So this intent is a standard for what abcd puts in each of those fields, and how much of it, so the frame shows a readable question at 80 columns and uses the room at 160. The rules that already govern WHAT is asked (one at a time, plain words, widen and never recommend, quote the thing being decided) are other records: itd-201 and itd-2609151541116052 (drafts) and the GRILL rule domain; this intent is the layout they render through.

Typed links: supersedes itd-110 (its one-line idea, "the grill interview renders with clear structure and colour", routed here by the product thinker on 2026-10-03; one part of that idea is not carried, "the recommended answer distinctly styled", because the GRILL rule's widen-never-recommend reverses it); refines itd-201 and itd-2609151541116052 (their content rules, this the layout); interacts with itd-2609292108373494 (a question handed to an installed interviewing skill should keep the same layout); itd-2610030810370060 builds on this one (the plain-Terminal front door). Its layout rules go into the brief's universal-patterns chapter with the spec, by the routing the product thinker confirmed.

State of the art (2026-10-03, reports/sota-question-layout.md in the local tier), the findings that bear on this intent: the header chip (12 characters) can carry role and progress ("Product 2/5"); the question field renders all in bold and shows markdown asterisks literally; a preview switches the terminal view side by side, removes the free-text "Other" row, cuts tall previews with no scroll and cannot be combined with multi-select, so essential meaning belongs in the label and description, with the null answer an explicit option (three substantive options at most); the host's own tool description tells the model to mark a recommended first option, which contradicts the widen-never-recommend rule and could be refused by a hook on the question tool; a message written just before a question can vanish in the terminal, so the context belongs inside the question; how the web and mobile apps render the view is not documented.

## Mechanism

- We expect questions to read the same in every interview because each one is filled from the same few parts (the label, the quoted material, the question, the options with their meanings, the way to decide later).
- We expect every question to stay readable because a check refuses a badly built one before it reaches the person, naming the part and the limit, however the agent wrote it.

_Both claims confirmed by the product thinker at the planning interview, 2026-10-03; each is checked on its own._

## Scope Conditions

- Questions shown in Claude Code's terminal view, at least 80 characters wide. <!-- cond: cond-2610030944505175 -->
- Questions abcd's own agents ask, through Claude Code's question tool. <!-- cond: cond-2610030944502636 -->
- The same parts drawn by abcd itself in a plain Terminal (itd-2610030810370060). <!-- cond: cond-2610030944508679 -->
- Phones and the web console are outside the promise. <!-- cond: cond-2610030944500042 -->

_Confirmed by the product thinker at the planning interview, 2026-10-03._

## Acceptance Criteria

_Agent-seeded from the design review of 2026-10-03, unconfirmed: each is walked with its addressee at the planning interview. A1-A3 and A7 are the technical facilitator's; A4-A6 are the product thinker's._

- A1 (facilitator; CONFIRMED as written, 2026-10-03) Given the raw input of the 2026-10-03 routing question as a test fixture, when the question check runs, then it refuses with exit 2 naming each failing field.
- A2 (facilitator; CONFIRMED as written, 2026-10-03) Given a header such as "Product Q2", a question within budget, two to four options each with a label of at most five words and a description of at most two sentences, one option from the declared way-to-decide-later set, no bold markers and no "(Recommended)", when the question check runs, then it lets the question through.
- A3 (facilitator; CONFIRMED as written, 2026-10-03) Given an option labelled "(Recommended)" or starred, or a header over 12 characters, when the question check runs, then it refuses naming the rule, the value and the limit.
- A4 (product thinker; CONFIRMED with an addition, 2026-10-03) Given a question the check lets through, when it is shown in Claude Code's terminal view at 80 columns, then the label, the whole question, every option and what it means are visible without scrolling, and it fits within 24 rows; a longer one is split into tabs, each within 24 rows (a dated screenshot naming the Claude Code version).
- A5 (product thinker; CONFIRMED 2026-10-03, reworded by decision 20) Given a question abcd builds, when it reaches the host, then it carries no side preview, so every option's description shows; a go test asserts the check refuses a preview.
- ~~A6~~ (STRUCK, decision 11) Given the same question in the web console and the Claude app, then it reads as plain text in the same order (abcd cannot test this; a scope condition with a dated manual screenshot, or marked untested).
- A7 (facilitator; CONFIRMED in its one-source form, 2026-10-03) Given the field limits written once, in the place the question check reads them, when the GRILL rule text and the intent page are produced, then both state exactly those limits; changing a limit in that one place changes all three (example: raising the label limit from 5 to 6 words is one edit).
- S1 (product thinker; CONFIRMED 2026-10-03, folded from itd-2609151541116052) Given a criteria walk, when "does this criterion stand?" is asked, then the criterion's full text is the first paragraph of the question (example: a criterion's text, then "Does it stand?").
- S2 (product thinker; CONFIRMED 2026-10-03) Given a press-release confirmation, when the person is asked to confirm or change a paragraph, then that paragraph is quoted, never referred to by its number.
- S3 (product thinker; CONFIRMED 2026-10-03) Given an open question to resolve or defer, when it is asked, then its text is quoted and "decide later" is an option.
- S4 (product thinker; CONFIRMED 2026-10-03) Given the retrospective and the unpacking interviews, when any question is asked, then the same order holds: the thing first, the question last (example: the earlier answer to "what went well" shown before "is this answer complete?").

## Review findings (design and feasibility, 2026-10-03, reports/review-qlayout-design.md in the local tier)

Applied: the press release now promises only what abcd puts in each field (finding 1); the chip carries role and ordinal, with a total only when known (5); the long-material rule is one part per question, split at sentence boundaries, never a pointer to a file the product thinker cannot open (9).

Recorded, not settled (each an interview question): host-owned complaints from the screenshots (the grey echo of the previous answer, the preview box narrower than the question) that no field budget fixes (2); enforcement by extending the existing question check that already runs before every question (`abcd guard hook` on the question tool, `internal/surface/cli/guard_question.go`), refusing and never rewriting a question (3); brief invariant 1 asks every settings question to say how to change it later, which the layout has no slot for (4); the question object as one shared type the check validates, the plain Terminal draws and the plugin hands to the host (8); fixtures of the two screenshot questions so "would have prevented" is a test (10).

Flagged for the product thinker to confirm (record-discipline review, 2026-10-03), never classified by the facilitator: brief invariant 1 reads, verbatim, "every `AskUserQuestion` shows current state, consequences of each option, and how to change later"; this layout has no slot for the current state or for how to change later, so it either carries them or the invariant changes for interview questions. The press release restates two content rules other records own (one option is always the way to decide later, from itd-201; the thing being decided quoted in full, the whole of itd-2609151541116052); whether those stay here, and whether itd-2609151541116052 is kept, bundled, superseded or refined, is the product thinker's.

## Decisions

1. 2026-10-03, the product thinker at the planning interview: the press release leads with the moment (who a question is for, where you are, what is decided, what each answer means), then the rules; and a question's text uses paragraph breaks and lists, never one block of text.
2. 2026-10-03, the product thinker: this intent and itd-2609151541116052 (show the thing being decided before asking) both stay and are planned together as one bundle; the bundle's name is asked at planning.
3. 2026-10-03, the product thinker: itd-201 (one at a time, plain language, options that widen) joins the same bundle, so the bundle is three: the asking rules, show the thing first, and the layout.
4. 2026-10-03, the product thinker, on brief invariant 1 ("every `AskUserQuestion` shows current state, consequences of each option, and how to change later"): the layout carries both, a 'now:' line and a 'change later:' line wherever they apply, saying 'not applicable' where they do not (an interview question has no current setting); the invariant stands unchanged.
5. 2026-10-03, the product thinker, added to the bundle: every question's text follows abcd's Writing Style (docs/reference/writing-style.md: British English, a capital letter after a colon, lower case after a semicolon, a serial comma before the final item of three or more, no em dash inside a list item, present tense).
6. 2026-10-03, evidence the product thinker gave at the interview: the second question of this interview, shown in Claude Code's terminal view at 81 columns, "looks much better". It had a role-and-ordinal chip ("Product Q2"), the quoted material set off in its own paragraph, the question on its own line, four short option labels, and a side preview stating one concrete consequence per option.
7. 2026-10-03, observed during the interview: the existing question check refused a question asked while the mode read "managed" (the person's message had reset it), with the remedy naming `abcd mode product-thinker`. The check this intent extends already refuses at the point the field checks would.
8. 2026-10-03, the product thinker: "small" means a Terminal window 80 characters wide, tested automatically; phones (the Claude app, the web console on a phone) are not promised.
9. 2026-10-03, decided without a question (one defensible answer on the record): material too long for one question is put one part per question, as the GRILL rule already says; a side preview or a message before the question cannot carry it (the preview is cut with no scroll, and a message written just before a question can vanish in the terminal).
10. 2026-10-03, the product thinker, on height (tall questions get cut off): use tabs for parts of one thing, such as the criteria of one feature or the paragraphs of one text, up to four per screen, each short; a question whose answer depends on an earlier one is still asked alone. This narrows the GRILL rule's "one at a time" to dependent questions; the rule's text is amended in the change that builds this.
11. 2026-10-03, decided without a question: proposed criterion A6 (the web console and the Claude app) is struck, because decision 8 puts phones and the web console outside the promise.
12. 2026-10-03, the product thinker (note on the criteria walk): every question that asks whether a criterion stands carries a concrete example illustrating what the criterion means in practice, and a question that offers a choice between two forms explains the difference between them, so an answer is never given on wording alone. Added to the bundle.
13. 2026-10-03, the fixture for A1 is rebuilt from this session's own early routing questions (the screenshots were not kept), stored as test data.
14. 2026-10-03, the technical facilitator: the field limits have one source, read by the question check; the GRILL rule text and the intent page are produced from it (criterion A7).
16. 2026-10-03, the product thinker, widening decision 12: the example rule is part of every interaction with the product thinker (every question, every choice, every explanation), and of every interaction with the technical facilitator where feasible. Added to the bundle.
17. 2026-10-03, the product thinker (note on the facilitator's dependency question): a question's option previews also carry the trade-offs between the options, where that is possible. Added to the bundle.
18. 2026-10-03, the product thinker: impact additive (stamped at planning with the bundle). The companion itd-2610030810370060 was planned the same day, ahead of this bundle, on the product thinker's sign-off.
19. 2026-10-03, the product thinker: itd-2609151541116052 (show the thing first) is folded into this intent and superseded by it. This intent's scope gains its order and step list: in every abcd interview (planning, retrospective, unpacking and routing included), the thing itself is shown in full first and the question comes last: a criterion before "does it stand", a press-release paragraph before "confirm or change", an open question before "resolve or defer", mechanism and scope text before their questions. Its reason travels with it: on 2026-09-20 the product thinker was asked whether criteria were theirs while the criteria were not on screen (iss-2609202058058301). The bundle is now two records: this one and itd-201.
15. Planning waits on the bundle: itd-201 is being interviewed on 2026-10-03 (itd-2609151541116052 folded in, decision 19).
20. 2026-10-03, the product thinker, answering "How should abcd show what each option means?": "No side previews". The evidence, taken that day by the product thinker in Claude Code 2.1.288 in macOS Terminal (described in .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md; the images are kept in the local tier): with previews present the host shows the option labels only, every description hidden, and one side preview box for the focused option; at 80 by 24 that box showed 1 line and "2 lines hidden", and at 160 columns it stayed about 45 columns wide and showed 1 line with "5 lines hidden", so height, not width, cuts it; the orchestrator's own question was cut the same way; a sibling session, independently, saw the preview in full at about 200 columns but still no option descriptions, and at 80 by 24 a preview of 9 to 10 lines cut to its first line with "10 lines hidden". So abcd's questions carry no side preview: the option's meaning, gain and cost go in its description, which always shows, and the question check refuses a preview (rule 14). This replaces the preview as the home of the trade-offs in decision 17 and A5's preview layout, which is reworded to match.

## Open Questions

- Height budget: how many rows a question (or one tab) may take at 80 columns before it is split, so nothing is cut off (24 rows, the classic Terminal height, is the obvious candidate).

- Which field carries what: the addressee and progress in the header chip (12 characters) or on the question's first line; the material quoted in the question or in the preview.
- A length budget per field, and what happens to material longer than the budget (one part per question, as the GRILL rule already says, or a pointer to a file).
- What "small" means: a minimum terminal width (80 columns?), a phone-sized web or app view, or both; what "large" earns (side-by-side preview, more context).
- Whether the web console and the Claude app render the question view at all, and what plain-text fallback they get (a state-of-the-art check of each host's rendering is owed before planning).
- Colour and emphasis: whether markdown or colour survives in each field of each host.
- Where the standard is enforced: the GRILL rule domain, the brief's universal-patterns chapter, a lint over the questions an agent composes, or all three.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-7f60ee417b11 -->
Fidelity review — receipt rcp-7f60ee417b11 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:3b6ae0e6aa2e2bb15b5cf0839a7a20bfcfd698b45d4db62020da2d1968b37c19
Input attestations: diff:git diff 1b22bae3169447f0c773d703713c7f3b6a9261e3 4b9345575123ba47c2ee6fc1cd6dc7d27804812d (PR #784: spec steps 1 and 2)@sha256:74de7e73c13db25c490cff764c4e7d4b22db849d5060b879ca83ab88752ddc63; diff:git diff cc5121f6880b750dfda60f355d1beb6c37babddd 9392a45148241c28c7140b50660637184183ec40 (PR #788: spec steps 3 to 5 and the close 3fdb8a4ce04f0c3bd2877d8a24e3e301eef3f5b7)@sha256:b3e33525369dc8b2e9de804c8d5e66e47de3d7ccb1fb1f476a187a8ae0816f21; commit:2f81e44cca5b1723a76878e5d28b7577d49290a1 (after delivery, 2026-10-07: the guard hook blocks with the host's JSON deny)@-; commit:66e8d7ed04c1648deb8ba7ff5b3fb8841cee83a1 (after delivery, 2026-10-07: a question over the rows limit alone is shown with a note)@-; repo:40320db01e6277d61e58fbc8cc2988346c444e52 (origin/main 2e03c61e454bf09ecda8cee0ae2f377157b522f7 plus audit-note commits): the tree every unprefixed ref cites; a ref prefixed 9392a4514: cites the delivery merge@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 6 · NOT_MET 0 · INCONCLUSIVE 1

Audit owed (receipt rcp-7f60ee417b11): ac-6 INCONCLUSIVE. Remedy: re-run the audit. Carried by iss-2610241604049754.

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: TestRoutingQuestionOf20261003IsRefused runs the routing fixture through abcd guard hook with the mode at product-thinker, and at delivery (9392a4514) asserted exit 2, nothing on stdout, and exactly the parts its findings file lists. Concerns: the fixture is a rebuild of the routing question, not its raw input (decision 13); the refusal names ten of the eleven failing parts and only counts the eleventh (option 3's preview), because one refusal names at most ten; and since 2f81e44cc (2026-10-07) main refuses through the host's JSON deny with exit 0, so 'exit 2' no longer holds there.
  evidence: 9392a4514:internal/surface/cli/guard_question_limits_test.go:192 — "if code != 2 {"
  evidence: internal/surface/cli/testdata/questions/routing-2026-10-03.findings.json:2 — "Eleven parts break the rules; the refusal names the first ten and counts the eleventh (option 3 preview) as one more."
  evidence: internal/surface/cli/guard_question.go:380 — "const maxRefusalParts = 10"
  evidence: .abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:106 — "the fixture for A1 is rebuilt from this session's own early routing questions (the screenshots were not kept), stored as test data"
  evidence: internal/surface/cli/guard_hook_output.go:13 — "A block is the host's PreToolUse deny: exit 0 and one JSON object whose"
  evidence: internal/surface/cli/guard_question_limits_test.go:226 — "reason := mustDeny(t, stdout, stderr, code)"
- ac-2 — MET_WITH_CONCERNS: TestWellBuiltQuestionIsAdmitted, at the core and at the hook, admits a 'Product Q2' question with three options of short labels and two-sentence descriptions, 'Decide later' last, no bold and no '(Recommended)': exit 0 with nothing on either stream, managed and unmanaged. Concern: admission is narrower than A2's givens, because the check also requires a 'Now:' and a 'Change later:' line with values (decision 4), material before a last line ending in a question (rule 11), the decide-later option last (rule 6) and a non-empty description, so a question meeting only the givens A2 lists is refused.
  evidence: internal/surface/cli/guard_question_limits_test.go:248 — "if code != 0 || stdout != "" || stderr != "" {"
  evidence: internal/core/question/limits_test.go:79 — "func TestWellBuiltQuestionIsAdmitted(t *testing.T) {"
  evidence: internal/core/question/limits_test.go:30 — ""Now: not applicable","
  evidence: internal/core/question/limits_test.go:144 — "{"no Now: line", RuleNowAndChangeLater, facilitator(), func(f *Fields) {"
  evidence: internal/core/question/check.go:239 — "laterLimit := fmt.Sprintf("exactly one option labelled %s, the last", quoteList(l.LaterLabels))"
- ac-3 — MET: TestRecommendedStarredOrLongHeaderIsRefused, at the core and at the hook, refuses a 'Keep it (Recommended)' label, a '★ Keep it' label and a 'Product thinker Q2' header, each on a finding line that carries the rule, the quoted value and the limit ('no (Recommended) label and no star mark', '12 columns'); the refusal exited 2 at delivery and is the host's deny on main, and A3 names neither.
  evidence: internal/core/question/limits_test.go:188 — "func TestRecommendedStarredOrLongHeaderIsRefused(t *testing.T) {"
  evidence: internal/surface/cli/guard_question_limits_test.go:295 — "no line names the rule %q, the value %q and the limit %q"
  evidence: internal/core/question/check.go:268 — "add(optionPart(i, "label"), RuleNeverRecommended, o.Label, "no (Recommended) label and no star mark", RecommendedRemedy)"
  evidence: internal/core/question/check.go:186 — "if w := textwidth.Columns(t.Header); w > l.HeaderColumns {"
  evidence: internal/core/question/check.go:57 — "return fmt.Sprintf("%s, %s (%s): %q; limit: %s. %s", where, f.Part, f.Rule, f.Value, f.Limit, f.Remedy)"
  evidence: 9392a4514:internal/surface/cli/guard_question_limits_test.go:257 — "if code != 2 || stdout != "" {"
- ac-4 — MET_WITH_CONCERNS: At delivery every finding refused, the rows estimate's included: TestQuestionFitsTwentyFourRowsAt80 refuses five criteria in one question with the split remedy and admits them as four tabs and then one, and TestRowEstimateMatchesTheHostAt80By24 matches the 24 rows the host drew, from figures measured on Claude Code 2.1.288 at 80 by 24 and recorded in a dated note naming that version. Concerns: the screenshot exists only as that note's description (the spec's open question 6, decided (c)), and its images are in neither the record nor the main checkout's local tier; no screenshot shows a tabbed call; and since 66e8d7ed0 (2026-10-07) the hook admits a question whose only fault is its height, with a note to the agent, so on main a question the check lets through can exceed 24 rows and is not split.
  evidence: internal/core/question/limits_test.go:418 — "onlyRule(t, fs, RuleRows)"
  evidence: internal/core/question/limits_test.go:471 — "if got := estimateRows(tab, Default); got != 24 {"
  evidence: internal/core/question/limits.go:53 — "spc-2610030944505997 read them on 2026-10-03 from Claude Code 2.1.288's"
  evidence: 9392a4514:internal/surface/cli/guard_question.go:130 — "return &exitError{Code: 2}"
  evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:8 — "are kept in the local tier, under `.abcd/.work.local/receipts/`, and are not"
  evidence: internal/surface/cli/guard_question.go:203 — "if len(findings) > 0 && !rowsOnly(findings) {"
  evidence: internal/surface/cli/guard_question_rows_test.go:13 — "The rows limit is the one layout rule the hook does not refuse on"
- ac-5 — MET: Rule 14 refuses every preview on an abcd question, naming 'no side preview' and the decision-20 remedy; TestQuestionCarriesNoSidePreview asserts one finding per preview and the same question admitted without them; abcd's own question type maps no preview into the host's fields; and the dated calibration note records that the host hides every description while a preview shows and shows each one without.
  evidence: internal/core/question/check.go:346 — "add(optionPart(i, "preview"), RuleNoPreview, o.Preview, "no side preview", previewRemedy)"
  evidence: internal/core/question/limits_test.go:366 — "onlyRule(t, fs, RuleNoPreview)"
  evidence: internal/core/question/ask.go:238 — "choices = append(choices, Choice{Label: o.Label, Description: o.Meaning})"
  evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:17 — "With previews present, the host shows the option labels only: every"
  evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:39 — "options. The 24 rows: the chip (1), a blank (1), the question text (9), a blank"
- ac-6 — INCONCLUSIVE: A6 is struck by decision 11, because decision 8 puts phones and the web console outside the promise, and the spec lists them as out; the delivery neither tests nor claims how the web console or the Claude app render a question, so no delivered artefact decides the struck outcome, and it is recorded INCONCLUSIVE (the bullet's own fallback is 'marked untested') rather than MET.
  evidence: .abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:77 — "~~A6~~ (STRUCK, decision 11)"
  evidence: .abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:104 — "proposed criterion A6 (the web console and the Claude app) is struck, because decision 8 puts phones and the web console outside the promise"
  evidence: .abcd/development/specs/closed/spc-2610030944505997-asking-and-layout.md:100 — "- Phones and the web console (decision 8, scope condition 4)."
- ac-7 — MET: question.Default in internal/core/question/limits.go is the one value: CheckLimits reads it, the GRILL domain is generated from AskingRules(question.Default), and cmd/asking-sync renders the asking-rules block of commands/intent.md from it; TestOneEditMovesEveryStatementOfALimit raises LabelWords to 6 on a copy and sees the check admit a six-word label and both the rule text and the page block move from 'five words' to 'six words', and TestIntentPageAskingBlockIsGenerated fails while the committed block lags the rendering.
  evidence: internal/core/question/limits.go:41 — "// Default is the one statement of every limit."
  evidence: internal/core/rules/grill.go:29 — "Rules: question.AskingRules(question.Default),"
  evidence: cmd/asking-sync/main_test.go:88 — "after.LabelWords = 6"
  evidence: cmd/asking-sync/main_test.go:111 — "if !strings.Contains(pageBefore, "at most five words") || !strings.Contains(pageAfter, "at most six words") ||"
  evidence: cmd/asking-sync/main_test.go:49 — "if got, want := committedBlock(t, page), render(question.Default); got != want {"
  evidence: commands/intent.md:333 — "each label is at most five words, and each description at most two sentences"
- ac-8 — MET_WITH_CONCERNS: TestCriterionQuestionQuotesTheCriterionFirst admits a criterion's full text, the Now: and Change later: lines and 'Does it stand?', and refuses 'Does criterion A4 stand?' alone under rule 11; the generated rule text and step 8 of commands/intent.md tell the agent to quote the criterion in the question. Concern: the check proves only that some line of material precedes the question, not that it is the criterion's full text or its first paragraph, so a criterion referred to after any other text is admitted; the full-text half rests on the rule text.
  evidence: internal/core/question/limits_test.go:314 — "Does criterion A4 stand?"
  evidence: internal/core/question/check.go:321 — "at least one paragraph of material before the question"
  evidence: internal/core/question/check.go:315 — "if tl := strings.TrimSpace(ln); tl != "" && !hasPrefixFold(tl, l.NowPrefix) && !hasPrefixFold(tl, l.ChangeLaterPrefix) {"
  evidence: commands/intent.md:332 — "The criterion before "does it stand?""
  evidence: commands/intent.md:427 — "per criterion with the criterion quoted in the question; the human"
- ac-9 — MET_WITH_CONCERNS: TestPressReleaseQuestionQuotesTheParagraph refuses 'Confirm or change paragraph 2?' alone under rule 11 and admits it with the paragraph quoted first, and the rule text puts the paragraph before 'confirm or change?'. Concern: the check cannot tell a quoted paragraph from other material, so a paragraph referred to by its number after any other text is admitted; that the quote is the record's own paragraph is held by the rule text alone, as the spec says.
  evidence: internal/core/question/limits_test.go:322 — "Confirm or change paragraph 2?"
  evidence: .abcd/development/specs/closed/spc-2610030944505997-asking-and-layout.md:409 — "the record's own paragraph is judgement, held by the rule text."
  evidence: commands/intent.md:332 — "the paragraph before "confirm or change?""
- ac-10 — MET_WITH_CONCERNS: Rule 6 puts exactly one decide-later option, last, on every abcd question, and TestOpenQuestionQuotesAndOffersDecideLater refuses an open question asked with no material first and no 'Decide later', naming both; the rule text puts the open question before 'resolve or defer?'. Concern: as for S1 and S2, the check proves that some material precedes the question, not that it is the open question's own text; that half rests on the rule text.
  evidence: internal/core/question/limits_test.go:342 — "if !slices.Contains(got, RuleThingFirst) || !slices.Contains(got, RuleDecideLater) || len(got) != 2 {"
  evidence: internal/core/question/check.go:239 — "laterLimit := fmt.Sprintf("exactly one option labelled %s, the last", quoteList(l.LaterLabels))"
  evidence: commands/intent.md:332 — "the open question before "resolve or defer?""
- ac-11 — MET: commands/reflect.md and commands/embark.md bind their questions to the generated asking rules (the thing first, quoted in full, the question last) and at delivery set the mode to product-thinker before the first question, so the hook holds the interviews' questions to rule 11; TestInterviewPagesBindTheOrder checks both pages and admits the retrospective's 'Is this answer complete?' with the earlier answer quoted first, refusing it bare.
  evidence: cmd/asking-sync/main_test.go:214 — "bare.Tabs[0].Text = "Is this answer complete?""
  evidence: commands/reflect.md:61 — "section's earlier answer before "is this answer complete?"), with one concrete"
  evidence: commands/embark.md:86 — "marked `generated: asking-rules`): the lessons themselves first, quoted in"
  evidence: 9392a4514:commands/reflect.md:54 — "With `abcd mode product-thinker` set, dispatch the `reflection-composer` agent with the seed."
  evidence: internal/core/question/check.go:309 — "if !strings.HasSuffix(lastLine, "?") {"

Gap audit:
- honoured:
  - A short label saying whom the question is for and which it is, with a total only when the interview's length is known: the header must be the chip '< role> Q< n>' with an optional '/< total>', within twelve columns.
    evidence: internal/core/question/check.go:113 — "Q[1-9] [0-9]*(?:/[1-9] [0-9]*)?$"
    evidence: internal/core/question/limits_test.go:85 — "// A chip with a total, admitted though never required (review finding 5)."
  - Options of a few words, each with one or two sentences on what choosing it means: a label over five words, and a description that is empty or over two sentences, are refused.
    evidence: internal/core/question/check.go:219 — "add(optionPart(i, "label"), RuleLabelWords, o.Label, fmt.Sprintf("%d words", l.LabelWords),"
    evidence: internal/core/question/check.go:227 — "add(optionPart(i, "description"), RuleMeaningSentences, o.Description, fmt.Sprintf("%d sentences", l.MeaningSentences),"
  - Always one option that is the way to decide later: exactly one 'Decide later' or 'None of these' option, the last, on every abcd question.
    evidence: internal/core/question/check.go:239 — "laterLimit := fmt.Sprintf("exactly one option labelled %s, the last", quoteList(l.LaterLabels))"
  - Nothing you need lives only in a side preview: every preview on an abcd question is refused, and the generated rule text says why.
    evidence: internal/core/question/check.go:346 — "add(optionPart(i, "preview"), RuleNoPreview, o.Preview, "no side preview", previewRemedy)"
    evidence: internal/core/question/asking.go:69 — "An abcd question carries no side preview: While a preview shows, the host hides every option's description and cuts the preview to the rows it has"
  - A question that breaks a layout rule other than the rows limit is refused before it reaches the person, each finding naming the part, the value and the limit, and the question is never rewritten.
    evidence: internal/core/question/check.go:57 — "return fmt.Sprintf("%s, %s (%s): %q; limit: %s. %s", where, f.Part, f.Rule, f.Value, f.Limit, f.Remedy)"
    evidence: internal/surface/cli/guard_question.go:57 — "It never rewrites the question: it admits it or refuses it"
  - No option is marked, starred or labelled as recommended, so the options widen rather than recommend.
    evidence: internal/core/question/check.go:268 — "add(optionPart(i, "label"), RuleNeverRecommended, o.Label, "no (Recommended) label and no star mark", RecommendedRemedy)"
  - Every question is built from the same parts: abcd's question type fills the host's fields and abcd's own Terminal drawing alike.
    evidence: internal/core/question/ask.go:209 — "func (a Ask) Fields() Fields {"
    evidence: internal/surface/cli/ask/testdata/key-home-80.golden:15 — "3. Decide later"
- diverged:
  - 'Refused before it reaches you' and 'reads at 80 columns without scrolling': since 66e8d7ed0 (2026-10-07) a question whose only fault is its height is shown with a note to the agent instead of refused, so on main an admitted question can run past 24 rows; the intent's A4 was not amended.
    evidence: internal/surface/cli/guard_question.go:203 — "if len(findings) > 0 && !rowsOnly(findings) {"
    evidence: internal/core/question/asking.go:56 — "The rows limit is the one the check does not refuse on: a question over it is shown, and the agent is told afterwards to keep the next question within it."
  - 'Naming the part and the limit': one refusal names at most ten parts and counts the rest, so a question breaking more than ten rules learns the rest only on a retry (the A1 fixture's eleventh part among them).
    evidence: internal/surface/cli/guard_question.go:380 — "const maxRefusalParts = 10"
    evidence: internal/surface/cli/guard_question.go:427 — "... and %d more part(s); fix these and ask again to see the rest."
  - 'The thing being decided, quoted in full, broken into paragraphs and lists rather than one block of text, then one plain question': the check holds only the order (some line of material, then a last line ending in '?'); quoting in full, breaking into paragraphs and lists, and asking one plain question are rule text, so a single block, or a reference after unrelated text, is admitted.
    evidence: internal/core/question/check.go:315 — "if tl := strings.TrimSpace(ln); tl != "" && !hasPrefixFold(tl, l.NowPrefix) && !hasPrefixFold(tl, l.ChangeLaterPrefix) {"
    evidence: commands/intent.md:332 — "The thing being decided is quoted in full in the question itself, in paragraphs and lists, never referred to"
  - 'Whenever abcd asks you something': the check runs only on a question whose header is in abcd's chip grammar or that is asked while the mode names somebody; an abcd question with neither runs unchecked.
    evidence: internal/surface/cli/guard_question.go:175 — "if !chip && st == mode.Managed {"
    evidence: .abcd/development/specs/closed/spc-2610030944505997-asking-and-layout.md:484 — "required; but an abcd question that forgets both the chip and the mode runs"
- missing:
  - The dated screenshots that were to prove the 80-column reading (A4, and A5's host half): the record keeps a note describing them, and the images, said to be under .abcd/.work.local/receipts/, are not committed, while the main checkout's local tier has no such directory.
    evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:8 — "are kept in the local tier, under `.abcd/.work.local/receipts/`, and are not"
  - Host evidence that a longer question split into tabs reads within 24 rows per tab: every screenshot the note describes is a single-question call.
    evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:12 — "## What each screenshot shows"
  - 'On a narrow window and a wide one': nothing records a question without previews in the host's view wider than 80 columns; both 160-column screenshots, and the 200-column observation, carry previews, which abcd's questions now never do.
    evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:48 — "### 3. 160 by 24, with previews (`host-question-160x24-previews`)"
    evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:54 — "### 4. 160 by 24, the orchestrator's own question (`host-question-160x24-preview-cut-own-question`)"

Scope-condition dispositions:
- cond-2610030944505175 — narrowed: The rows estimate's host figures were measured once, on Claude Code 2.1.288 in macOS Terminal at 80 by 24, and the limits hold one question or tab to 24 rows at 80 columns, so reading without scrolling depends on the window's height and the host's frame as well as its width; the only wider observations (160 and about 200 columns) carried previews, which abcd's questions no longer carry.
  narrowing: holds for Claude Code's terminal view at least 80 columns wide and at least 24 rows tall, drawing the frame Claude Code 2.1.288 drew in macOS Terminal; a window of 80 or more columns but fewer than 24 rows, or a host version whose frame differs, is outside what the delivery verified
  evidence: internal/core/question/limits.go:28 — "24: one question, or one tab, at Columns"
  evidence: internal/core/question/limits.go:53 — "spc-2610030944505997 read them on 2026-10-03 from Claude Code 2.1.288's"
  evidence: .abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md:7 — "screenshots on 2026-10-03 in Claude Code 2.1.288, in macOS Terminal. The images"
- cond-2610030944502636 — narrowed: The hook judges the AskUserQuestion tool alone and takes a question to be abcd's by a header in the chip grammar or a mode naming somebody (the spec's open question 1, decided (a)): an unchipped question asked while the mode reads managed returns unchecked, and a question asked while the mode names somebody is held to abcd's rules whoever wrote it.
  narrowing: holds for the questions abcd's agents put through the AskUserQuestion tool that carry abcd's chip (Product, Tech or Setup, then Q< n>) or are asked while the mode names the product thinker or the facilitator; an abcd question with neither runs unchecked
  evidence: internal/surface/cli/guard_question.go:43 — "var questionTools = []string{"AskUserQuestion"}"
  evidence: internal/surface/cli/guard_question.go:175 — "if !chip && st == mode.Managed {"
  evidence: .abcd/development/specs/closed/spc-2610030944505997-asking-and-layout.md:484 — "required; but an abcd question that forgets both the chip and the mode runs"
- cond-2610030944508679 — survived: The plain-Terminal drawing merged in PR #788 lays out the same parts as the host's fields (the chip, the material in paragraphs and lists, the one question, each option with its meaning, Decide later, and the Now: and Change later: lines, which it places after the options as the spec allows), and every fixture question passes CheckLimits through Ask.Fields().
  evidence: internal/surface/cli/ask/testdata/key-home-80.golden:1 — "› Setup Q1"
  evidence: internal/surface/cli/ask/testdata/key-home-80.golden:18 — "Now: no key is stored"
  evidence: internal/surface/cli/ask/layout_test.go:97 — "for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {"
- cond-2610030944500042 — survived: The delivery keeps phones and the web console out of the promise: the spec lists them under 'Out of both', A6 is struck by decision 11, the limits and their calibration name only Claude Code's terminal view, and no delivered page or doc claims how a question renders on a phone or in the web console.
  evidence: .abcd/development/specs/closed/spc-2610030944505997-asking-and-layout.md:100 — "- Phones and the web console (decision 8, scope condition 4)."
  evidence: .abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:104 — "proposed criterion A6 (the web console and the Claude app) is struck, because decision 8 puts phones and the web console outside the promise"
  evidence: internal/core/question/limits.go:27 — "80: the narrow window promised"
<!-- abcd-review-end receipt=rcp-7f60ee417b11 -->

## Grounds

- pursued: Interviews are where your decisions are made; while questions arrive as walls of text, answers are rushed or misread, and we expect a readable layout to cut the follow-up questions and the answers you later reverse.
