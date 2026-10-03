---
id: itd-2610030810350727
slug: every-question-abcd-puts-to-a-person-reads-the-same-way-and
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
- A5 (product thinker; CONFIRMED as written, 2026-10-03) Given the same question at 160 columns with a side preview, then the preview sits beside the options and the way-to-decide-later option stands in for the free-text row the preview removes (a dated screenshot; depends on the host).
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

## Open Questions

- Height budget: how many rows a question (or one tab) may take at 80 columns before it is split, so nothing is cut off (24 rows, the classic Terminal height, is the obvious candidate).

- Which field carries what: the addressee and progress in the header chip (12 characters) or on the question's first line; the material quoted in the question or in the preview.
- A length budget per field, and what happens to material longer than the budget (one part per question, as the GRILL rule already says, or a pointer to a file).
- What "small" means: a minimum terminal width (80 columns?), a phone-sized web or app view, or both; what "large" earns (side-by-side preview, more context).
- Whether the web console and the Claude app render the question view at all, and what plain-text fallback they get (a state-of-the-art check of each host's rendering is owed before planning).
- Colour and emphasis: whether markdown or colour survives in each field of each host.
- Where the standard is enforced: the GRILL rule domain, the brief's universal-patterns chapter, a lint over the questions an agent composes, or all three.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: Interviews are where your decisions are made; while questions arrive as walls of text, answers are rushed or misread, and we expect a readable layout to cut the follow-up questions and the answers you later reverse.
