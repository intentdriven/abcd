---
id: spc-2610030944505997
slug: asking-and-layout
intent: itd-2610030810350727
intents: [itd-2610030810350727, itd-201]
bundle: asking-and-layout
origin: researcher-authored
production_mode: hand-written
---
# asking-and-layout

## Summary

This spec delivers the bundle `asking-and-layout`:
[itd-2610030810350727](../../intents/planned/itd-2610030810350727-every-question-abcd-puts-to-a-person-reads-the-same-way-and.md)
(every question abcd asks reads the same way, on a narrow window and a wide
one) and
[itd-201](../../intents/planned/itd-201-every-question-abcd-s-agents-put-to-a-human-is-asked-one-at.md)
(every question is asked one thing at a time, in plain language, with options
that widen). Both ship at one moment, from one change set split into the steps
below.

Four things land. The field limits of a question are written once, in
`internal/core/question`, and everything that states them is produced from
that one place. The question check that already runs before every question on
the host's question tool (`abcd guard hook`) gains the field checks, refuses a
badly built question naming the part, the value, and the limit, and never
rewrites one; its mode gate is narrowed to abcd's own interviews. The GRILL
rule domain becomes a bundled default generated from the same Go source, as
SHELL is, so every managed repository receives it through the binary, and
abcd's own repository override is deleted. The interview pages carry the
amended asking rules in a generated block, and the knowledge floor moves to
one glossary page the rule points at.

The settled records are designed to here, never reopened: the layout intent's
decisions 1 to 19 (A6 struck by decision 11) and its confirmed criteria A1 to
A5, A7, and S1 to S4; itd-201's interview decisions 1 to 11, its confirmed
criteria R1 to R6, its reworded criteria, and its three scope conditions.

## Scope

In (this bundle):

- `internal/core/question`: The field limits (`Limits`, one value,
  `question.Default`), the field view every front door finally shows
  (`Fields`), the mapping from the companion's `Ask` onto it, the limits check
  (`CheckLimits`), the row estimate, and the asking-rule text the GRILL domain
  and the interview pages are generated from.
- `internal/surface/cli/guard_question.go`: The question check decodes the
  question tool's input, decides whether the question is abcd's, runs the
  limits check on abcd's questions anywhere, and keeps the mode gate for
  abcd's questions in a managed repository.
- `internal/core/rules`: GRILL generated into the bundled defaults
  (`grill.go`, beside `shell.go`), refused when declared by hand; abcd's own
  `.abcd/rules.json` loses its GRILL key.
- The plugin pages: `commands/intent.md`'s "How every question is asked"
  becomes a generated block; `commands/reflect.md`, `commands/embark.md`, and
  `commands/ahoy.md` name the same order (the thing first, the question last)
  for their interviews.
- The record and docs: a glossary page for the knowledge floor
  (`.abcd/development/brief/glossary/interview/knowledge-floor.md`); the
  layout rules in the brief's universal-patterns chapter (§ 1, invariant 1
  unchanged by decision 4); GRILL and its dormant escape in the default-domain
  list of `AGENTS.md` and of the managed block ahoy plants
  (`internal/core/ahoy/defaults/claude-md-marker-block.md`); the rules
  reference in `docs/reference/cli/commands.md`.
- Dated host evidence for A4 and A5: screenshots at 80 and 160 columns naming
  the Claude Code version, and the row estimate calibrated against them.

Out (the plain-Terminal spec,
[spc-2610030911534855](spc-2610030911534855-a-person-can-run-abcd-s-interviews-in-a-plain-terminal.md),
for itd-2610030810370060):

- The question type itself (`Ask`, `Question`, `Block`, `Option`, `List`),
  its JSON shape, and its structural check (`question.Check`). That spec
  defines the type first because its intent was planned first (decision 18);
  this bundle adds to it and redefines nothing.
- The drawing in a plain Terminal, the answer loop, the answers record and its
  "answered in" field, and the interviews abcd runs without a host.
- The lift of the banner's wrapper and display-width measure into a shared
  place (this bundle's row estimate uses that measure; open question 2 asks
  where it lives).

How the two specs split the work, in one line each: That spec owns the shape
of a question and how abcd draws it; this spec owns how much each part may
hold, who checks it, and the rule text agents follow. Its drawing never
enforces a limit of its own (its Scope, "Out"); its runner path calls this
bundle's `CheckLimits` once it exists (its Approach, step 4 of the AI-written
loop), so a question an AI writes for the Terminal is held to the same limits
as one an agent hands the host.

Out of both:

- Colon and semicolon casing and the serial comma: review by nature, never a
  machine check (adr-54; itd-201 scope condition 3). The rule text states them;
  nothing checks them.
- Host-owned rendering the screenshots showed (the grey echo of the previous
  answer, the preview box narrower than the question): no field limit reaches
  them.
- Phones and the web console (decision 8, scope condition 4).
- Other tools' questions in a managed repository (itd-201 decision 10).

## Approach

### The field limits, written once

`internal/core/question/limits.go` holds one value, and nothing else in the
tree states a limit:

```go
// Limits is every bound a question's fields are held to. question.Default is
// the one value the check, the GRILL domain, and the interview pages read
// (itd-2610030810350727 decision 14, criterion A7).
type Limits struct {
	HeaderColumns    int      // 12: the host's header chip
	ChipRoles        []string // "Product", "Tech", "Setup": the chip's first word
	QuestionsPerCall [2]int   // 1..4: one question, or up to four tabs
	OptionsPerQ      [2]int   // 2..4, the decide-later option included
	LabelWords       int      // 5
	MeaningSentences int      // 2: an option's description
	LaterLabels      []string // "Decide later", "None of these"
	Columns          int      // 80: the narrow window promised
	Rows             int      // 24: one question, or one tab, at Columns
	HostTextColumns  int      // the text measure inside the host's frame at Columns
	HostChromeRows   int      // rows the frame draws around a question
	NowPrefix        string   // "Now:"
	ChangeLaterPrefix string  // "Change later:"
	NotApplicable    string   // "not applicable"
}
```

`HostTextColumns` and `HostChromeRows` are measured, not chosen: step 5
calibrates them against the dated A4 screenshot, and until then they carry
provisional values (76 and 6) named as such in their comment.

### The field view, and why the check runs on it

The host's question tool takes four fields per question (a header, the
question text, and options with a label, a description, and an optional
preview), and the agent composes them itself: on the host path abcd's binary
never sees a question until the hook does. So the limits are checked on that
field view, `question.Fields`, and the companion's structured type is mapped
onto it before it is checked. One implementation then serves both front doors:

- the hook decodes the host's `tool_input.questions` into `Fields` (the JSON
  key names are the host's, so the decoding lives in the surface);
- `Ask.Fields()` maps the companion's type onto the same view: the chip to the
  header; the material's blocks, then the `Now:` and `Change later:` lines,
  then the ask, to the question text; the options and then `Later` to the
  options. The runner path and every fixed question abcd builds are checked
  through it.

In the question text the `Now:` and `Change later:` lines sit between the
material and the ask, because the text must end with the question (S4, and
the paragraph-before-question shape below). The companion's drawing places
the same two lines after the options; the words are the same and both satisfy
decision 4.

### The limits check

`question.CheckLimits(f Fields, l Limits, who Addressee) []Finding` returns
every finding at once, so one refusal names every part to fix and the agent
fixes them in one retry. A `Finding` carries the question's ordinal (its tab),
the part, the rule, the value (sanitised and capped, as the mode store's echo
is), the limit, and a remedy. The rules, each a separate finding:

1. **Header.** Present, at most `HeaderColumns` display columns, and in the
   chip grammar `<role> Q<n>` with an optional `/<total>`, the role from
   `ChipRoles` ("Product Q2", "Tech Q3", "Setup Q1/4"). A total is admitted,
   never required (the layout intent's applied review finding 5: a total only
   when the interview's length is known).
2. **Questions per call.** One to four; more than one is tabs, and each tab is
   checked on its own.
3. **Options.** Two to four, the decide-later option counted.
4. **Label words.** At most `LabelWords` words, split on white space.
5. **Meaning sentences.** Each description present and at most
   `MeaningSentences` sentences, counted at `.`, `?`, or `!` before white space
   or the end, with "e.g.", "i.e.", and "etc." masked. The description holds
   the meaning, the gain and the cost; two sentences hold them, and no preview
   carries any of it (rule 14, the layout intent's decision 20).
6. **Decide later.** Exactly one option whose label is in `LaterLabels`, and
   it is the last; it is always offered.
7. **No bold markers.** No `**` or `__` in the header, the question text, a
   label, or a description (the host shows them literally). Previews are
   rendered as markdown by the host and are not held to this.
8. **Never recommended.** No label carrying "(Recommended)" in any case, and
   no label starting or ending with a star (`*`, `★`, `☆`, `⭐`). The remedy
   names the cause, so the agent does not loop (R6): "The host's own
   instruction for this tool asks for a recommended first option; abcd's
   asking rule reverses it: no option is marked, styled, or ordered as
   recommended. Remove the mark and ask again; if the person asks for a
   recommendation, give it in prose beside the question."
9. **The product thinker's register (R4).** When the addressee is the product
   thinker, no field (previews included) carries a record handle, found by
   `recordid.HandleInText`, or a command: a backtick span, `go run`, a `/abcd:`
   slash command, or `abcd` followed by one of the binary's verbs, which the
   surface passes in from the command tree so the core holds no copy of it.
   The addressee is the mode's when a mode store exists; where none does (a
   repository abcd does not manage), the chip's role word stands in, since the
   chip names whom the question is for.
10. **Now and change later.** The question text carries a line starting
    `Now:` and a line starting `Change later:` (case-insensitive), each with a
    value, `not applicable` included (decision 4; invariant 1 stands).
11. **The thing first, the question last.** The text's last non-blank line ends
    in `?`, and at least one paragraph of material precedes it, the `Now:` and
    `Change later:` lines not counted (S1 to S4; the S5 shape the bundle review
    proposed).
12. **No em dash in a list item.** No line of the question text that opens a
    list item carries an em dash (Writing Style, decision 5). The pattern is
    the docs-lint token `punctuation/em-dash-in-list-item`; a test holds the
    two equal, since the token lives in ahoy's JSON seed. The other Writing
    Style rules stay review (scope condition 3).
13. **Rows.** At `Columns`, the question (or each tab) fits `Rows`: the
    question text wrapped at `HostTextColumns`, each option's label row and its
    description wrapped beneath it at `HostOptionColumns`, and `HostChromeRows`,
    the chip's row included (calibrated in step 5; a preview is refused by rule
    14, so its layout is not estimated). Over budget, the remedy says to split the material into parts as tabs
    (up to four) or into successive questions, one part per question, and
    never to move it into a message before the question or into a preview
    (decision 9).
14. **No side preview.** No option carries a preview, refused with the remedy
    "abcd's questions carry no side preview (decision 20): put the option's
    meaning, gain and cost in its description." Added in step 5 on the layout
    intent's decision 20: with a preview the host hides every option's
    description and cuts the preview to the rows it has, whatever the width.

The check refuses and never rewrites. The host lets a pre-tool hook replace a
tool's input; this check does not use that, because a rewritten question puts
words in the agent's mouth that neither the agent nor the person chose
(invariant 1's transparency; the design review's finding 3).

### The question check in the guard hook

`questionGate` today decodes only the tool name and the working directory.
`guardHookInput.ToolInput` gains `Questions json.RawMessage`, and the gate runs
in this order:

1. Decode the questions into `[]question.Fields`. A payload whose questions
   cannot be read is not a decision: the gate fails open loudly through
   `questionFailOpen` (exit 1, the question runs, the warning shows), the
   guard's existing contract.
2. Decide whether the question is abcd's (itd-201 decision 10). It is abcd's
   when either holds:
   - its header (any tab's) is in abcd's chip grammar, which only abcd's
     interview pages are taught to write; or
   - the mode names somebody (`product-thinker` or `facilitator`), which only
     abcd's mode verb sets, and which the prompt hook resets on the next human
     message, so the window is one question long.

   Anything else is another tool's question and runs, unchecked and unmarked,
   exit 0, wherever it is asked. Open question 1 weighs this against two other
   signals.
3. For abcd's question, run `CheckLimits` wherever the hook runs, managed or
   not: the setup interview asks before a repository is managed, and the
   limits need no store.
4. Where the existing preconditions hold (a checkout abcd manages, with the
   local tier), run the mode gate as today: an abcd chip while the mode reads
   managed is refused with the existing refusal and its `abcd mode` remedy.
5. Any finding from steps 3 and 4 refuses with exit 2: one head line, "Blocked
   by the abcd guard (question tool): N part(s) of this question break abcd's
   asking rules; fix each and ask again.", then one line per finding naming the
   tab, the part, the value, the limit, and the remedy. Every echoed value
   passes `termsafe.Sanitize` (invariant 13), and the output is plain text on
   stderr, which the host replays to the agent.
6. An admitted abcd question is marked open, as today, so the next human
   message resets the mode.

`hooks/hooks.json` needs no change: its PreToolUse matcher already names the
question tool beside the shell tool.

### GRILL generated from one Go source

The rule text lives beside the limits, in `internal/core/question/asking.go`:
`AskingRules(l Limits) []string` returns the domain's rules with every limit
filled from `l` (numbers written as words), and `AskingRecall` its recall
terms. `internal/core/rules/grill.go` mirrors `shell.go`:
`withGrillDomain` adds `GRILL` to the parsed bundled defaults from
`AskingRules(question.Default)` and panics if `defaults/rules.json` declares
`GRILL` by hand (R2). It is then an ordinary bundled domain to every loader
contract: per-field user and repository overrides, `dormant`, the kill switch,
`*GRILL`, dedup, and provenance (`source: bundled`, R1). abcd's own
`.abcd/rules.json` drops its GRILL key in the same change (decision 8), so this
repository runs the text every managed repository runs.

The generated text carries no record handle and no `go run` (decision 8). The
amendments to the twelve rules in today's repository override:

- **One thing at a time, with tabs** (layout decision 10; itd-201 decision 2):
  the parts of one thing (the criteria of one feature, the paragraphs of one
  text) are asked as tabs, up to four on a screen; a question whose answer
  depends on an earlier one is asked alone, after that answer; never a
  numbered list in a message. A host with no question tool degrades to one
  question per message, in the same order and words (scope condition 1).
- **Quoted in full**: the thing being decided is quoted in the question itself,
  in paragraphs and lists, never referred to; "one sentence of context" goes.
  Material too long for one question is put one part per question.
- **The layout**: one rule states the fields with their limits, filled from
  `Limits` (the chip, the material, the `Now:` and `Change later:` lines, the
  question last, two to four options, each label and meaning within budget,
  "Decide later" always offered, no bold, 24 rows at 80 columns), and says the
  question check refuses a question that breaks them.
- **Examples** (itd-201 decision 3; layout decisions 12 and 16): one example
  of the thing being decided in the question text, and what choosing each
  option means in its description; a preview may repeat it with the
  trade-off and never holds it alone.
- **Gain and cost** (itd-201 decision 4): each option's gain and cost, never
  one option's alone; the previews carry the trade-offs where they can
  (decision 17), in the same neutral form.
- **Recommendation only on request** (itd-201 decision 6): no option is ever
  styled, marked, or ordered as recommended; a recommendation appears only
  when the person asks for one, in prose beside the question.
- **The knowledge floor** (itd-201 decision 5): the two floor rules collapse to
  one line pointing at the glossary page, which holds the floor in full.
- **Register and mode, scoped** (itd-201 decision 10): the register rule, the
  "ask the role first" rule, and the mode rules open with "In abcd's own
  interviews", and the mode rule names `abcd mode facilitator` and
  `abcd mode product-thinker` with no checkout-specific form.
- **Kept as they are, ids stripped**: only real choices are asked; deferral is
  recorded as an answer and silence is never consent.

The recall terms are today's override's, and the cost is accepted (itd-201
decision 9: about 1,100 tokens in most sessions of every managed repository).
The escape, `{"GRILL": {"state": "dormant"}}` in the repository's
`.abcd/rules.json`, is stated in the default-domain list of `AGENTS.md` and of
the managed block, and in the rules reference.

### The interview pages and the glossary floor

`commands/intent.md`'s "How every question is asked" paragraph and its
"What each register is assumed to know" paragraph are replaced by one block
between `<!-- generated: asking-rules -->` and `<!-- /generated -->`, rendered
from `AskingRules(question.Default)` and the glossary pointer. The paragraph's
own reason for stating the floor in full ("the rule domain ... is declared in
abcd's own repository alone") stops being true and goes with it. A generator,
`cmd/asking-sync` (with `-check`, mirroring `cmd/scaffold-sync`), writes the
block; a test under preflight fails when the committed block differs from the
rendering, so a limit changed in `Limits` without regenerating the page fails
the gate (A7). Open question 3 weighs the generator's home.

`commands/reflect.md` (the retrospective), `commands/embark.md` (the unpacking
interview), and `commands/ahoy.md` (the routing confirmation) each gain one
sentence binding their questions to the same order and pointing at the
generated block, and `commands/intent.md` steps 3, 4, 5, 6, and 8 already
quote the thing in the question; step 8 gains the example rule (decision 12).

The knowledge floor moves, nearly verbatim from today's paragraph, to
`.abcd/development/brief/glossary/interview/knowledge-floor.md`, a glossary
entry in the existing `interview` context. Open question 4 asks how the
one-line pointer resolves in a managed repository.

## How each acceptance criterion is met

**A1.** The fixture is the 2026-10-03 routing question rebuilt from this
session's own early routing questions (decision 13), stored as
`internal/surface/cli/testdata/questions/routing-2026-10-03.json` with the
list of findings it must produce beside it (no chip header, one block with no
paragraph before the question, meaning only in previews, no decide-later
option). `TestRoutingQuestionOf20261003IsRefused` feeds it to `abcd guard hook`
with the mode at `product-thinker` and asserts exit 2 and exactly the listed
parts named.

**A2.** `TestWellBuiltQuestionIsAdmitted` feeds a question headed
"Product Q2", within the row budget, with three options of at most five-word
labels and two-sentence descriptions, the last "Decide later", no `**`, and no
"(Recommended)", and asserts exit 0, nothing on stderr, and the question
marked open.

**A3.** `TestRecommendedStarredOrLongHeaderIsRefused` runs three cases (a label
"Keep it (Recommended)", a label "★ Keep it", a header "Product thinker Q2")
and asserts exit 2 and a line naming the rule, the value, and the limit in
each.

**A4.** Two halves. The automatic half, `TestQuestionFitsTwentyFourRowsAt80`,
holds the row estimate over golden questions: A2's question fits; a criteria
walk of five criteria in one question is refused with the split remedy; the
same five as four tabs and then one question fit, each tab on its own. The
host half is a dated screenshot of A2's question in Claude Code's terminal view
at 80 by 24, naming the version, taken in step 5, which also calibrates
`HostTextColumns` and `HostChromeRows` so the estimate matches what the host
drew.

**A5.** Reworded by the layout intent's decision 20: a question abcd builds
carries no side preview. `TestQuestionCarriesNoSidePreview` asserts every
preview is refused under rule 14, one finding per option naming it, and the
same question without previews admitted. The host evidence is the dated
screenshots at 80 and 160 columns with previews, described in
`.abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md`:
with previews the host hides every description, and the preview box is cut by
height at either width.

**A7.** `TestOneEditMovesEveryStatementOfALimit` copies `question.Default`,
raises `LabelWords` from 5 to 6, and asserts that `CheckLimits` admits a
six-word label it refused before, that `AskingRules` says "six words" where it
said "five", and that the intent-page block rendered from the copy says the
same. `TestIntentPageAskingBlockIsGenerated` asserts the committed block in
`commands/intent.md` equals the rendering from `question.Default`, and the
GRILL domain is generated from the same call at init, so the three cannot
part.

**S1.** `TestCriterionQuestionQuotesTheCriterionFirst`: a criteria-walk
question whose text is a criterion's full text, a blank line, the `Now:` and
`Change later:` lines, and "Does it stand?" is admitted; "Does criterion A4
stand?" alone is refused under the paragraph-first rule. The rule text and
step 8 of `commands/intent.md` carry the example.

**S2.** `TestPressReleaseQuestionQuotesTheParagraph`: "Confirm or change
paragraph 2?" alone is refused (no paragraph before the question); the same
question with the paragraph quoted first is admitted. That the quoted text is
the record's own paragraph is judgement, held by the rule text.

**S3.** `TestOpenQuestionQuotesAndOffersDecideLater`: an open question asked
without its text first, or without "Decide later", is refused naming each
part.

**S4.** The check applies to every question with an abcd chip, whichever page
asks it, so the retrospective's and the unpacking interview's questions are
held to the same order. `TestInterviewPagesBindTheOrder` asserts that
`commands/reflect.md`, `commands/embark.md`, and `commands/ahoy.md` each point
at the generated block, and a fixture of the retrospective's "is this answer
complete?" with the earlier answer quoted first is admitted, without it
refused.

**R1.** `TestManagedRepositoryGetsGrillFromTheBinary` builds a managed
repository whose `.abcd/rules.json` is the skeleton ahoy writes, matches the
prompt "which option should we choose", and asserts the domain is injected and
`abcd rules GRILL --json` reports `source: bundled`, its rules equal to
`AskingRules(question.Default)`.

**R2.** `TestGrillDomainIsNotHandWritten` asserts the embedded
`defaults/rules.json` declares no GRILL, and
`TestHandWrittenGrillPanicsAtLoad` feeds `withGrillDomain` a set that
declares one and asserts the panic, as the SHELL pair does.

**R3.** One thing at a time is rule text, since a numbered list in a message
is prose no hook sees; `TestGrillTextAsksOneThingAtATime` asserts the tabs
rule is in the generated domain. The check bounds what it can: more than four
questions in one call is refused, and each tab is checked alone. The
criterion's example ("five open questions ... as tabs on one screen") meets
the four-per-screen limit of decision 10 and the host's own maximum of four;
the design asks them as four tabs and then one, and open question 5 flags the
wording.

**R4.** `TestProductThinkerQuestionNamesNoRecordOrCommand`, with the mode at
`product-thinker`: a description naming `iss-2609202058058301`, a text naming
`abcd mode`, and a label in backticks are each refused; the same question with
the mode at `facilitator` is admitted. A case with no mode store and a
"Product" chip is refused the same way.

**R5.** The check guarantees "Decide later" is always offered (rule 6). The
record side is the rule text (deferral recorded as an answer, silence never
consent), asserted present by `TestGrillTextRecordsDeferral`, and the prompt
hook's existing reset; in the plain Terminal the companion's answers record
writes the `later` value.

**R6.** `TestRecommendedOrStarredOptionIsRefused` asserts the refusal names the
label, and that its remedy names the host's own instruction and abcd's rule
reversing it.

**The reworded criteria.** One thing at a time through the question tool (R3's
test and rule 2); quoted in full in paragraphs and lists, one example in the
question, each option's meaning in its description, gain and cost, "decide
later" (rules 5, 6, and 11, and `TestGrillTextCarriesTheAmendments`, which
asserts each amended rule's presence in the generated text); the technical
facilitator gets the mechanism and the ids (rule 9 does not apply under the
`facilitator` mode, R4's second case); an unknown addressee is asked the role
first in abcd's interviews (rule text, scoped by decision 10, asserted by
`TestGrillTextScopesRegisterToAbcdInterviews`).

**The mode gate's scope (itd-201 decision 10; scope condition 2).**
`TestForeignQuestionIsNotRefused`: a header "Pick a theme" while the mode reads
managed runs with exit 0 and is not marked open; `TestAbcdChipWhileManagedIsRefused`:
"Product Q1" while the mode reads managed is refused with the existing
`abcd mode` remedy.

## Open design questions

These were for the technical facilitator; the records did not settle them.
Each was decided on 2026-10-03 without a question, under the person's ruling
that an obvious answer is decided rather than asked; the reason is given
beneath each.

1. **How the check knows a question is abcd's.** (a) The chip grammar or a mode
   naming somebody (designed to): no new state, and the chip is already
   required; but an abcd question that forgets both the chip and the mode runs
   unchecked, and another tool's question asked in the one-question window
   after abcd sets the mode is held to abcd's limits. (b) An interview marker:
   the interview pages run a verb that writes "an abcd interview is open" to
   the local tier, cleared at its end: exact, but a page that forgets to clear
   it holds other tools' questions to abcd's rules until it is cleared, and it
   needs the local tier, so the setup interview before install has none. (c)
   Read the session transcript the hook payload names and look for an abcd
   command page: no agent discipline needed, but host-specific, slow on a long
   session, and a transcript format abcd does not own.
   - Decided: (a). It adds no state and rests on the chip every abcd question
     already carries; (b) fails closed on other tools' questions when a page
     forgets to clear it and has no home before install, and (c) reads a
     transcript format abcd does not own.
2. **Where the display-width measure lives.** The row estimate needs the
   wrapper and measure the companion lifts out of the banner. (a) A pure leaf
   package (`internal/textwidth`) that both `internal/term` and
   `internal/core/question` import (designed to): the core stays clear of
   terminal I/O, at the cost of one more package and a change to the
   companion's step 1. (b) `internal/core/question` imports `internal/term`:
   no new package, but core then depends on a package holding raw-mode code.
   (c) The estimate stays in the surface and the core checks everything but
   rows: no import question, but the runner path then needs its own copy of
   the estimate, a second statement of a limit.
   - Decided: (a). The core stays clear of terminal I/O, as the
     transport-agnostic boundary requires, and the limit is stated once; (b)
     breaks the boundary and (c) states a limit twice.
3. **Who regenerates the intent page's block.** (a) `cmd/asking-sync` with
   `-check` (designed to): mirrors scaffold-sync, explicit. (b) Fold it into
   `cmd/scaffold-sync`: one command fewer, but that command is about workflow
   pins. (c) A test-only `-update` flag on the parity test: no command at all,
   but an unfamiliar fix for a contributor who meets the failure.
   - Decided: (a). It follows the existing scaffold-sync pattern a contributor
     already knows, and keeps that command about workflow pins.
4. **How the knowledge-floor pointer resolves in a managed repository.** The
   glossary lives in abcd's development record. (a) The page's path under the
   plugin root, since every marketplace install carries `.abcd/` (designed
   to): local and versioned with the binary, but a path an agent must join
   itself. (b) The page's address on the public forge: one click, but it can
   run ahead of the installed version. (c) `abcd rules GRILL` prints the
   floor's one-line definition beneath the pointer, read from the page at
   build time: self-contained, but the released binary excludes `.abcd/`, so
   the definition would need a second home in the Go source.
   - Decided: (a). The page travels with the installed version; (b) can
     describe a newer abcd than the one installed, and (c) states the
     definition twice.
5. **R3's example against the four-per-screen limit.** R3's example asks five
   open questions "as tabs on one screen"; decision 10 allows four per screen
   and the host allows four per call. The design asks four tabs and then one.
   R3 is the product thinker's confirmed criterion, so the wording is theirs
   to amend or confirm; the facilitator raises it, the design does not.
   - Decided: four tabs, then one. R3's example was corrected to match,
     recorded as itd-201 decision 12; the criterion's rule (tabs for parts of
     one thing) is unchanged, only its example.
6. **Where the A4 and A5 screenshots are kept.** The audit reads them. (a)
   Committed under `.abcd/development/research/` with a dated note: durable and
   reachable by the auditor, at the cost of binary files in the record. (b)
   The local tier: no binaries committed, but the audit on another checkout
   cannot see them. (c) Committed as a dated note that describes them, the
   images kept locally, as the layout intent did with the original
   screenshots.
   - Decided: (c). It follows the layout intent's precedent, and the tests
     read the measured figures (`HostTextColumns`, `HostChromeRows`), which
     the note records; images can be promoted to the record later, while a
     committed binary cannot be taken back out of the history.

## Footprint

- packages: internal/core/question, internal/core/rules, internal/core/ahoy, internal/surface/cli, cmd/asking-sync, commands/, .abcd/rules.json, .abcd/development/brief, AGENTS.md, docs/reference
- tests: TestRoutingQuestionOf20261003IsRefused, TestWellBuiltQuestionIsAdmitted, TestRecommendedStarredOrLongHeaderIsRefused, TestQuestionFitsTwentyFourRowsAt80, TestQuestionCarriesNoSidePreview, TestRowEstimateMatchesTheHostAt80By24, TestOneEditMovesEveryStatementOfALimit, TestIntentPageAskingBlockIsGenerated, TestCriterionQuestionQuotesTheCriterionFirst, TestPressReleaseQuestionQuotesTheParagraph, TestOpenQuestionQuotesAndOffersDecideLater, TestInterviewPagesBindTheOrder, TestManagedRepositoryGetsGrillFromTheBinary, TestGrillDomainIsNotHandWritten, TestHandWrittenGrillPanicsAtLoad, TestGrillTextAsksOneThingAtATime, TestProductThinkerQuestionNamesNoRecordOrCommand, TestGrillTextRecordsDeferral, TestRecommendedOrStarredOptionIsRefused, TestGrillTextCarriesTheAmendments, TestGrillTextScopesRegisterToAbcdInterviews, TestForeignQuestionIsNotRefused, TestAbcdChipWhileManagedIsRefused; the em-dash pattern held equal to the docs-lint token; every fixed question abcd builds passing CheckLimits

## Steps

1. The field limits and the limits check
   - criteria: A2, A3, R4, R6, and S1 to S3 at the core; the core half of A7
   - packages: internal/core/question, internal/core/ahoy (test only)
   - tests: `CheckLimits` over golden field views, one refusal case per rule and an admitted case; the row estimate over five criteria as one question (refused) and as four tabs then one (admitted); the em-dash pattern equal to the docs-lint token; `Ask.Fields()` places the `Now:` and `Change later:` lines before the ask and `Later` last
   - lands after the companion's step 1 when that lands first, adding `Ask.Fields()`; if this step lands first, it creates the package with `Fields` and `Limits` alone and the companion adds the mapping
   - landed: #784
2. The question check in the guard hook
   - criteria: A1, A2, A3, R4, R6, and the mode gate's scope
   - packages: internal/surface/cli
   - tests: TestRoutingQuestionOf20261003IsRefused (the fixture and its expected findings in testdata), TestWellBuiltQuestionIsAdmitted, TestRecommendedStarredOrLongHeaderIsRefused, TestProductThinkerQuestionNamesNoRecordOrCommand, TestRecommendedOrStarredOptionIsRefused, TestForeignQuestionIsNotRefused, TestAbcdChipWhileManagedIsRefused; an unreadable questions field failing open loudly with exit 1; every echoed value sanitised; the existing gate tests unchanged
   - landed: #784
3. GRILL generated from the one source
   - criteria: R1, R2, R3, R5, the reworded criteria; the GRILL half of A7
   - packages: internal/core/question, internal/core/rules, .abcd/rules.json, .abcd/development/brief/glossary
   - tests: TestManagedRepositoryGetsGrillFromTheBinary, TestGrillDomainIsNotHandWritten, TestHandWrittenGrillPanicsAtLoad, TestGrillTextAsksOneThingAtATime, TestGrillTextRecordsDeferral, TestGrillTextCarriesTheAmendments, TestGrillTextScopesRegisterToAbcdInterviews; the generated text carries no record handle and no `go run`; the loader contracts (override, dormant, kill switch, `*GRILL`) hold for GRILL as for SHELL; abcd's own `abcd rules GRILL --json` reports `source: bundled` once its override is gone
   - landed: feat/asking-pages
4. The interview pages, the generated block, and the docs
   - criteria: A7, S4, and the page side of S1 to S3
   - packages: cmd/asking-sync, commands/, internal/core/ahoy, AGENTS.md, .abcd/development/brief, docs/reference
   - tests: TestOneEditMovesEveryStatementOfALimit, TestIntentPageAskingBlockIsGenerated, TestInterviewPagesBindTheOrder; the managed block's default-domain list names GRILL and its dormant escape, with abcd's own `AGENTS.md` matching it; every fixed question abcd builds (setup, routing) passing `CheckLimits`; docs-lint clean over the glossary page and the universal-patterns section
   - landed: feat/asking-pages
5. Host evidence and calibration
   - criteria: A4 and A5 (host halves)
   - packages: internal/core/question, .abcd/development/research
   - tests: TestQuestionFitsTwentyFourRowsAt80, TestRowEstimateMatchesTheHostAt80By24 and TestQuestionCarriesNoSidePreview (A5 as decision 20 rewords it) with `HostTextColumns`, `HostOptionColumns` and `HostChromeRows` set from the screenshots; the dated screenshots at 80 by 24 and at 160 columns, each naming the Claude Code version, kept where open question 6 rules
   - landed: feat/asking-pages
