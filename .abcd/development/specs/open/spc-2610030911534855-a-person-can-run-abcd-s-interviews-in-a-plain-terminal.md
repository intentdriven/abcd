---
id: spc-2610030911534855
slug: a-person-can-run-abcd-s-interviews-in-a-plain-terminal
intent: itd-2610030810370060
origin: researcher-authored
production_mode: hand-written
---
# a-person-can-run-abcd-s-interviews-in-a-plain-terminal

## Summary

This spec delivers
[itd-2610030810370060](../../intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a-plain-terminal.md):
abcd's interviews run in a plain Terminal with no host session, abcd draws each
question itself from the same parts the host's question view is given, a long
list of choices is searched by typing and moved through with the arrow keys,
and the answers land in the same records an interview inside Claude Code
writes, each answer marked with where it was given.

Four pieces land, in order: One question type and the drawing of it at the
window's width; the answer loop (the arrow-key list with typing to narrow, the
numbered fallback, and the guarantee that the Terminal is restored on every
exit); the interviews whose questions abcd fixes (setup and the routing
confirmation) run through the drawing, with the answers record and its
"answered in" field; and the interviews an AI writes (planning, retrospective)
run through the person's own route, the runner returning each question as the
shared parts for abcd to draw.

The intent's decisions 1 to 5 and its criteria B1 to B7 are settled and are
designed to here, not reopened: Both kinds of interview in the first version
(decision 1), the long list in this feature (decision 2), arrow keys by default
with the numbered list as a setting and as the mode wherever no keyboard mode is
possible (decision 3), the "answered in" field (decision 4), and
golang.org/x/term as a direct dependency (decision 5).

## Scope

In:

- `internal/core/question`, a new package: The question type both front doors
  use (its parts, its JSON shape, its structural check), and the long-list
  variant.
- `internal/term`: Raw mode, restore, and window size through golang.org/x/term;
  the word wrapper and display-width measure lifted out of
  `internal/surface/cli/banner.go` so the banner and the drawing share one copy.
- `internal/surface/cli/ask`, a new front-door package: The layout (question to
  lines at a width and a colour mode), the arrow-key list, the numbered reader,
  and the plain-text writer for a stream that is not a terminal.
- `internal/core/interview`, a new package: The answers record, the answers
  file, the "answered in" field, and the turn loop of an AI-written interview.
- `internal/core/ahoy`: The setup interview's value questions and approvals,
  and the routing confirmation, built as questions of the shared type from the
  words core already owns (`PromptHelp`, `machineRoutingQuestion`,
  `repoRoutingQuestion`).
- `internal/core/runner`: The question-returning output contract a role
  follows when it writes an interview's questions.
- `internal/core/oracle` and `agents/`: A planning-interviewer role in the
  roster, beside the retrospective's existing reflection-composer.
- `internal/surface/cli`: The prompter that draws, `--answers` on the
  interviews' verbs, and two sub-verbs, `abcd intent interview <itd-N>` and
  `abcd reflect interview <release-tag>`.
- The plugin pages `commands/ahoy.md`, `commands/intent.md` and
  `commands/reflect.md` (the host path writes the same answers record), the
  command reference, the surface snapshot, `go.mod` and `ACKNOWLEDGEMENTS.md`.

Out:

- The field limits (header length, words per label, sentences per meaning,
  and rows per question) and the question check on the host's question tool. Both
  belong to the companion bundle (itd-2610030810350727 with itd-201 and
  itd-2609151541116052); its limits come from one source the check reads
  (companion criterion A7), and this spec's drawing never enforces a limit of
  its own. Until the bundle lands, a question this spec draws is held only to
  the structural check below.
- The guided `ahoy connect` path and its model look-up
  (itd-2610030821294016); this spec delivers the long list it will use, not the
  connect questions.
- Windows (abcd is released for macOS and Linux; scope condition 2), terminals
  narrower than 80 columns or shorter than 24 rows (scope condition 1), and phones
  and the web console.
- A full-screen interface or an alternate screen: Nothing here takes over the
  window.

## Approach

### One question type, defined here first

The question type lives in `internal/core/question` and this spec defines it
first; the companion bundle consumes it and adds the field limits and the check
to it. The reason is order: This intent was planned ahead of the bundle on the
product thinker's sign-off (companion decision 18), the bundle waits on two
drafts that still need their own reviews (companion decision 15), and nothing
can be drawn without the type. Its shape is not this spec's invention: It is
the set of parts the companion's press release and decision 4 name, laid over
the host's question-tool input the state-of-the-art pass ranks first
(reports/sota-question-layout.md, recommendation 1), so the bundle inherits a
type it already described and only adds limits to it.

```go
// Ask is what one turn puts to the person: one question, or up to four
// parts of one thing shown as tabs (companion decision 10).
type Ask struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	ID          string   `json:"id"`           // stable: a setup key, or the turn's ordinal
	Chip        string   `json:"chip"`         // who it is for and which question: "Setup Q1"
	Material    []Block  `json:"material"`     // the thing being decided: paragraphs and lists
	Ask         string   `json:"ask"`          // the one plain question
	Options     []Option `json:"options"`      // the substantive answers
	Later       Option   `json:"later"`        // the way to decide later, always present
	Now         string   `json:"now,omitempty"`          // companion decision 4
	ChangeLater string   `json:"change_later,omitempty"` // companion decision 4
	List        *List    `json:"list,omitempty"`         // the long-list variant
}

type Block struct {
	Kind  string   `json:"kind"`  // "paragraph" or "list"
	Text  string   `json:"text,omitempty"`
	Items []string `json:"items,omitempty"`
}

type Option struct {
	Value   string `json:"value"`   // what is recorded
	Label   string `json:"label"`   // a few words
	Meaning string `json:"meaning"` // what choosing it means, with its trade-off
}

// List is a choice among many (300 models): Options stays empty and the
// choices live here; the Later option still applies.
type List struct {
	Choices []Option `json:"choices"`
}
```

`question.Check` is structural only: Every question has an id, a chip, an ask,
a `Later` option with a value, and either options or a list; values are unique
within a question; an `Ask` holds one to four questions. It refuses naming the
question and the part. The companion's limits are a second function on the
same type, added by the bundle; this spec leaves the seam (`Check` returns a
list of findings, so the bundle appends to it) and does not guess the limits.

`Later` is a field and not one option among the others so that no drawing and
no runner can leave it out: The layout always draws it last. The host path
maps the same type onto the host's question tool (chip to header, the material and the
ask to the question text, options plus `Later` to the options), which is how
the plugin hands the host the parts verbatim.

### The drawing at 80 and at 160 columns

`ask.Layout(a Ask, width int, mode term.ColorMode) []string` is pure: It takes
the window's width and the colour rung and returns the lines, so every layout
is a golden test with an injected width. The width is read from the terminal
the person answers at (`term.Size`, through golang.org/x/term) before each
question, so a resize between questions is honoured; `COLUMNS` is the fallback
and 80 the default.

Every block is laid out as follows, top to bottom:

1. The chip line: The chip, then the tab strip when the `Ask` has more than one
   part ("1 B1 · 2 B2 · 3 B3", the current tab marked by a glyph).
2. The material, each paragraph wrapped on its own and each list item with a
   hanging indent, one blank line between blocks.
3. The ask on its own line.
4. The options, numbered from 1, then `Later` as the last number.
5. The `now:` and `change later:` lines, each drawn when the question carries
   it.

Prose never runs past 80 columns, at any width (WCAG 1.4.8; the
state-of-the-art pass, recommendation 7): The measure is `min(width, 80)` less
the indent. A wide window earns columns, not longer lines. At a narrow window
each option's label sits on its numbered line and its meaning is indented
beneath it. When the window holds the label column (the longest label plus the
number and a gap) and a meaning column of at least 60 columns, the label and
its meaning sit side by side, the meaning wrapped in its own column at a
measure of at most 80. At 80 columns the stacked form is drawn; at 160 the
side-by-side form always is, since a label of a few words fits the label
column with room to spare.

The wrapper is the banner's balanced wrap, lifted to `term.Wrap` and
`term.Columns` (East Asian wide runes count two) so the banner and the drawing
measure the same way; the banner's tests stay as the proof the lift changed
nothing.

Nothing is cut off. A question taller than the window is drawn whole and the
Terminal scrolls; keeping questions within 24 rows is the companion's limit,
not a truncation here. The arrow-key list alone is bounded: Its visible window
is the rows left beneath the question, at least five, with "12 more above" and
"40 more below" lines.

### Colour, NO_COLOR and TERM=dumb

Colour comes from `term.ResolveColorMode`, unchanged: `--no-color`, NO_COLOR
(present and non-empty) and a TERM of `dumb` or unset give `Mono`. The drawing
uses the 16 basic colours only, whatever higher rung the terminal offers, and
nothing is carried by colour alone: The chip, the current option, and the
current tab are each marked by a glyph (`›`, or `>` without a UTF-8 locale per
`term.UTF8Locale`) and colour only repeats what the glyph says. In `Mono` the
layout is the same lines with no SGR sequence.

TERM=dumb also says the terminal cannot move its cursor, so it forces the
numbered mode as well as `Mono`: An arrow-key list there would print cursor
codes as garbage.

### Sanitising before drawing

Every string in a question is runtime-read: A runner wrote it, a record
supplied it, or a model service listed it. Before layout, `ask.Safe(a)` passes
every single-line part (chip, ask, labels, values, meanings, list choices,
`now`, `change later`) through `termsafe.Sanitize` and every material block
through `termsafe.SanitizeBlock`, so an injected escape, a C1 control, a bidi
override, or a zero-width rune reaches the screen as a visible `?` and a line
break inside a label cannot forge a line. Measuring happens after sanitising,
so widths are of what is drawn. The only escape sequences the drawing emits are
the ones it composes itself (colour, cursor up, and erase line), which adr-49
decision 2 admits as trusted.

### The answer loop: Arrow keys first, typing to narrow

The default on a terminal is the arrow-key list (decision 3). Raw mode comes
from golang.org/x/term (`term.MakeRaw` on the input descriptor), wrapped in one
`term.RawSession` type in `internal/term` so no call site handles termios
itself:

- Up and Down move the current option; Page Up and Page Down move a window;
  Home and End go to the ends; Enter chooses; a digit followed by Enter chooses
  by number, so B1's "or by its number" holds in the arrow-key mode too.
- A printable character narrows the list: The filter is the typed text matched
  case-insensitively as a substring of each choice's label and value, shown on
  a "filter:" line; Backspace widens it; Escape clears it. The filter applies to
  the long-list variant and to ordinary options alike. When nothing matches,
  the list says so and keeps `Later` visible.
- Left, Right, and Tab move between the parts of a tabbed `Ask`.
- The list is redrawn in place with cursor-up and erase-line only, never the
  alternate screen, so the Terminal's scrollback keeps the whole interview and
  a person can copy from it. On a choice the list collapses to one plain line,
  "› Setup Q1: Keep it in the system keychain", which is what stays on screen.
- A window resize (SIGWINCH) redraws at the new width.

The Terminal is restored on every exit. `RawSession.Restore` is idempotent
(`sync.Once`) and is reached from four places: The deferred call on every
return, error included; a recovered panic, which restores and re-panics;
Ctrl-C, which arrives as the byte 0x03 because raw mode clears `ISIG`, and
restores, prints a newline, records nothing for the open question, keeps the
answers already given, and exits 130; and a `signal.Notify` handler for
SIGINT, SIGTERM, and SIGHUP sent from outside, which restores before the
process exits. Ctrl-Z restores, stops the process with SIGTSTP, and on SIGCONT
re-enters raw mode and redraws. A write that fails (the terminal closed) ends
the session through the deferred restore.

### The numbered fallback

The numbered mode reads whole lines and never touches termios: The same layout,
then "Type a number, or part of a name to narrow the list, then Enter." A typed
number chooses; typed text narrows a long list and redraws it numbered, 20 at a
time, "n" and "p" paging. It is chosen:

- by the setting `interview.list` set to `numbered` in the layered
  configuration (machine layer; `arrows` is the bundled default), or by the
  environment variable `ABCD_ACCESSIBLE` non-empty for one session, which is
  how a screen-reader user who cannot set a file first gets it;
- always when the terminal cannot take a keyboard mode: TERM=dumb, or raw mode
  refused by the terminal (`MakeRaw` returning an error drops to numbered with
  one line on stderr saying so, never a failed interview).

No terminal signals a screen reader reliably, which is why the setting and the
variable exist (open question 2 asks about the variable's name).

### When output or input is not a terminal

A question is drawn only when stdin, stdout, and stderr are all terminals (the
letter of adr-49 decision 1 and of scope condition 4; the question is written to
stderr, as the install's prompter writes it today, so a `--json` result on
stdout stays clean). Otherwise the interview is in plain-text mode:

- each question is written as plain text, the same lines as `Mono` at 80
  columns, numbered, with no colour, no cursor code, and no other escape byte;
- each answer comes from a flag (the setup questions already name theirs,
  `PromptHelp.Flag`) or from the answers file named by `--answers`; the setup
  interview's existing line-per-answer stdin stream (iss-167) is kept as it is;
- a question that neither supplies refuses with exit 2, naming the question's
  id, the flag, and the answers-file key that would answer it, and writes
  nothing; it never blocks on a pipe and never takes a default silently.

The answers file is one JSON shape for every interview:

```json
{
  "schema_version": 1,
  "interview": "setup",
  "answers": [
    {"id": "visibility", "value": "private"},
    {"id": "Q2", "value": "later", "note": "ask again after the bundle", "answered_in": "Claude Code"}
  ]
}
```

A fixed interview matches answers by question id; an AI-written interview,
whose questions are not known ahead, matches them in order by ordinal (`Q1`,
`Q2`, …) and refuses when the file runs out, naming the question it could not
answer. `jsonstrict` reads the file, so an unknown key or a duplicate refuses.
The retrospective's existing `reflect write --answers` file keeps its own shape;
the retrospective interview produces it at its end (below).

### The answers record and the "answered in" field

Every interview writes one answers record through one writer,
`interview.Write`: The interview's name and target, and per question the
question as asked (the sanitised `Ask`), the value chosen, the note if any, and
`answered_in`, which is `Terminal` or `Claude Code` (decision 4) and nothing
else. The record is written to the repository's local tier,
`.abcd/.work.local/interviews/<interview>-<stamp>.json`, or for the
machine-wide part of setup to `~/.abcd/interviews/`, each level made through
the guarded store maker the other `~/.abcd` stores use.

`answered_in` is stamped by the front door that collected the answer: The
drawing and the numbered reader stamp `Terminal`; an answers file entry carries
its own `answered_in`, and an entry without one takes the verb's
`--answered-in` flag, which defaults to `Terminal`. The plugin pages pass
`--answered-in "Claude Code"` on the host path. So a scripted run in a plain
shell records `Terminal`, the same as the drawn run it replays (B3), and the
host path records `Claude Code` (B5). The value is asserted by the plugin page,
not proven: Open question 3 asks whether a hook should stamp it instead.

The interview's own outputs (the setup's configuration, the routing table, the
retrospective's README, and the planning record) are produced from the answers
alone, through the writers that already exist, so two runs given the same
answers write the same outputs, whichever front door asked.

### The interviews whose questions abcd fixes

The setup interview asks through `ahoy.Prompter`. A new front-door prompter,
`drawnPrompter` in `internal/surface/cli`, implements `ahoy.Prompter` and
`ahoy.TerminalPrompter` by building an `Ask` and handing it to the answer loop:

- a value question (`Prompt(key, choices, def)`) becomes a question whose id is
  the key, whose material is `PromptHelp.About` of the help `ahoy.HelpIn`
  gives for the install's repository (iss-2610031236155833), whose options are
  `PromptHelp.Choices` with their meanings, whose `change_later` line is
  `PromptHelp.ChangeLaterLine` (the flag hint, or, for a question no flag
  answers, where its answer is changed later, iss-2610031236155833), and whose
  `Later` option leaves the value unset so the gap stays listed by `abcd ahoy`
  (the core's own words, so no door invents its own, iss-163);
- an approval (`Confirm(question)`) becomes a question with the options "Yes,
  make the change" and "No, leave it", and `Later` declining, as the refusing
  prompter does;
- the routing confirmation's two offers become questions whose material is the
  proposal said in counts (how many agents, how many at each tier, their
  fan-out bounds; the product thinker's 2026-10-03 ruling on
  iss-2610031236155833), from `machineRoutingQuestion` and
  `repoRoutingQuestion`.

The chip is "Setup Q<n>", numbered in the fixed prompt order
(`categoryPromptOrder`), with a total only where the order fixes one. The
existing `stdinPrompter` stays the prompter off a terminal, so piped installs
behave as before; `newPrompter` picks `drawnPrompter` only when all three
streams are terminals.

### The interviews an AI writes, through the person's own route

Planning and the retrospective run in a plain Terminal through
`abcd intent interview <itd-N>` and `abcd reflect interview <release-tag>`. An
AI writes their questions; with no host session, the AI is the runner on the
person's own route (adr-25's opt-in adapter; itd-2609201916056194, rulings RN2
and OC2), and abcd draws every question itself.

The turn loop in `internal/core/interview`:

1. Load the runner configuration and resolve the role's route: The
   retrospective's role is `reflection-composer`, already in the roster;
   planning gains a `planning-interviewer` role (its brief in `agents/`, a
   frontier row in the proposal table) that follows `commands/intent.md`'s
   planning interview.
2. Refuse before anything runs when no route reaches a runner: The role's
   route is the host, there is no host session in a plain Terminal, and no
   `runner.fallback_host` is set. The refusal is one paragraph, exit 2, nothing
   written:
   "abcd intent interview: the planning interview's questions are written by
   an AI, and no route of yours reaches one. Set roles.planning-interviewer.runner
   to claude or opencode, with that runner enabled under runner.<name>, in
   ~/.abcd/config.json; it runs on your own paid key, so a repository's setting
   cannot do it. The setup and routing interviews need no route:
   `abcd ahoy install`."
3. Each turn writes a brief to the local tier (the seed, the answers so far,
   and the output contract) and dispatches the role through `runner.Dispatcher`
   with `HostSession: false`. The output contract is the receipt as exactly one
   of `{"ask": <Ask>}` or `{"done": <outcome>}`.
4. The dispatcher's `Validate` decodes the receipt strictly and runs
   `question.Check` (and the bundle's limits once they exist) on an `ask`. A
   receipt that fails is `ReasonInvalid`: The existing fallback receipt is
   written, the configured host is tried, and with none the interview stops
   with the dispatcher's own refusal naming the runner and the reason, the
   answers so far kept in the record.
5. A valid `ask` is sanitised, drawn, answered, and appended to the answers
   record with `answered_in: Terminal`; the next turn is dispatched with it.
6. `done` ends the loop. The retrospective's outcome is the `reflect write`
   answers object, written through `reflect write`'s own path with its floors
   and refusals unchanged. The planning interview's outcome is the record
   changes the role made with the tools its contract grants, as the host agent
   makes them today (open question 4 asks whether that should become a
   structured outcome abcd writes).

Off a terminal the same loop takes its answers from the answers file in order,
so a scripted run against a stub runner replays a drawn run exactly.

## How each acceptance criterion is met

**B1.** The setup interview's value questions are drawn through
`drawnPrompter`. At 80 columns the layout draws the chip ("Setup Q1"), the
material, the ask, each option numbered with its meaning beneath it, and
`Later` as "decide later"; the answer is taken by the arrow keys and Enter or
by a typed number. `TestSetupQuestionDrawsAt80Columns` holds the layout of a
setup question built from `PromptHelp` against a golden file at width 80;
`TestSetupQuestionAnswersByArrowsAndByNumber` feeds the answer loop two key
streams (Down, Enter; then "2", Enter) and asserts both choose the same option.
When the guided connect path adds the key-home question, the intent's example
is the same test with that question's fixture.

**B2.** At 160 columns the label column and the meaning column sit side by
side, and the meaning's measure is capped at 80.
`TestQuestionAt160ColumnsSitsSideBySide` holds the golden layout at width 160
and asserts that no run of prose (any line with its leading label column and
indent removed) is wider than 80 display columns; a companion case at width
100 asserts the breakpoint rule (side by side only when a 60-column meaning
column fits).

**B3.** With stdout or stdin not a terminal the interview is in plain-text
mode, answers come from flags or `--answers`, and the record is written by the
same writer with `answered_in` defaulting to `Terminal`.
`TestInterviewOffATerminalIsPlainAndMatchesTheTerminalRecord` runs the setup
interview twice, once with the answer loop driven by an injected key stream
and once with stdout a pipe and an answers file carrying the same choices, and
asserts that the piped output holds no ESC byte, no C1 control, and no cursor
sequence (adr-49's machine-stream assertion), and that both answers records
and both written configurations are byte-identical. A case with a question the
file does not answer asserts exit 2, the question's id and flag named, and
nothing written.

**B4.** NO_COLOR and TERM=dumb give `Mono`, and TERM=dumb also gives the
numbered mode; the current option is marked by a glyph in every mode.
`TestNoColorAndDumbTerminalDrawNoColour` draws a question under NO_COLOR=1,
TERM=dumb, and TERM unset and asserts no SGR sequence in any, no cursor sequence
under TERM=dumb, and that the current option's line differs from the others by
its glyph alone when the colour is removed.

**B5.** Both front doors feed the same answers through the same writers, and
`answered_in` is the only field stamped by the door.
`TestRecordsDifferOnlyInWhereAnswered` runs the setup interview through the
drawn path and through the host path (an answers file with
`--answered-in "Claude Code"`, as the plugin page runs it), then compares the
answers records field by field and the written configuration byte for byte:
equal except `answered_in`. A second case runs the retrospective interview
against a stub runner returning the same two questions and then `done`, once
drawn and once from an answers file marked `Claude Code`, and asserts the same
of its answers record and the retrospective it writes.

**B6.** The long-list variant narrows as the person types and the Terminal is
restored on every exit. `TestLongListNarrowsAsThePersonTypes` drives the list's
state machine (a pure function of key events, no terminal) over a fixture of
300 model names: Typing "claude" leaves only the names containing it, Down and
Up move within what is left, Backspace widens, and Enter records the current
one. `TestLongListRestoresTerminalOnInterrupt` opens a pseudo-terminal with the
standard library (`/dev/ptmx` and the platform's grant and unlock requests, a
test helper, no new module), runs the test binary as a child on its other end,
types "claude" and then Ctrl-C, and asserts the child exited 130 and that the
terminal's attributes after the exit equal those before it (`ICANON` and `ECHO`
set). Further cases send SIGTERM from outside and force a panic inside the
loop, with the same assertion.

**B7.** Every part of a question is sanitised before it is measured or drawn.
`TestDrawnMaterialIsSanitisedFirst` builds a question whose material, label,
meaning, and list choices carry ESC sequences, U+009B, a bidi override, a
zero-width space, and a bare carriage return, draws it in colour, and asserts
that the only escape sequences in the output are the drawing's own SGR
sequences and that each injected rune appears as `?`. A runner case sends the
same question as a stub runner's `ask` receipt and asserts the same.

## Open design questions

These were for the technical facilitator; the record did not settle them.
Each was decided on 2026-10-03 without a question, under the person's ruling
that an obvious answer is decided rather than asked; the reason is given
beneath each.

1. **One dispatch per question, or several.** Each turn of an AI-written
   interview starts the harness again and re-reads the brief, which costs
   seconds and tokens per question. Options: (a) one dispatch per turn, the
   simplest and the most faithful to "a dependent question is asked alone",
   slow on a long planning interview; (b) a turn returns up to four
   independent parts as one tabbed `Ask` (companion decision 10), fewer
   dispatches, at the cost of a role that must judge independence; (c) resume
   the harness's own session between turns, the cheapest, but the runner
   adapters do not resume today and each harness's resume differs. The spec
   designs to (a) with the `Ask` already able to carry (b).
   - Decided: (a). It is the only option faithful to "a dependent question is
     asked alone" without a new judgement, (c) has no adapter to stand on, and
     the `Ask` keeps (b) open for later.
2. **The name of the accessibility variable.** `ABCD_ACCESSIBLE` is abcd's own;
   `ACCESSIBLE` is what huh reads and GitHub CLI has its own. Options: abcd's
   own only (no surprise from another tool's variable), or also honour
   `ACCESSIBLE` (a screen-reader user who set it once gets every tool's
   numbered mode), at the cost of a variable abcd does not own changing its
   behaviour.
   - Decided: honour both, `ABCD_ACCESSIBLE` taking precedence when set. The
     only behaviour `ACCESSIBLE` can change is a switch to the numbered list,
     which is always safe to give, so the person who set it once is served and
     nothing they did not ask for can happen.
3. **Who stamps "answered in: Claude Code".** The plugin page passes
   `--answered-in`, which the agent could omit or get wrong. Options: (a) keep
   it on the page, simple, an assertion the record cannot prove; (b) add a
   PostToolUse hook on the question tool that records the answer the host
   returned, stamped by the binary, which proves the field but depends on the
   host's hook payload carrying the answer, which the state-of-the-art pass did
   not verify; (c) both, the hook when the payload carries the answer and the
   flag otherwise.
   - Decided: (a). (b) and (c) rest on a host payload nobody has verified; the
     page's flag ships now, and a binary-stamped field is a later change once
     the payload is checked.
4. **How the planning interview's outcome reaches the intent record in a
   Terminal.** Options: (a) the planning-interviewer role edits the record with
   the tools its contract grants, as the host agent does today, parity with the
   host path and no new writer, but the record changes are an AI's edits abcd
   only validates after; (b) the role returns a structured outcome (decisions,
   confirmed criteria) in `done` and abcd writes it, which makes the outcome
   checkable and B5 provable for planning, but needs record writers that do not
   exist yet for decisions and criteria. The spec designs to (a).
   - Decided: (a). It keeps parity with the host path and needs no new writer;
     abcd's validation after the edit is the same gate the host path has.
5. **Drawing on stderr while stdout is piped.** The spec draws only when all
   three streams are terminals, the letter of adr-49 decision 1, so
   `abcd intent interview itd-N --json | jq` falls to plain-text mode and needs
   an answers file. Options: Keep the letter; or draw on stderr whenever stdin
   and stderr are terminals, which keeps the interview interactive under a
   piped `--json` and needs adr-49 amended to name the stream the person
   reads.
   - Decided: keep the letter of adr-49. A piped `--json` run is a machine
     consumer, the answers file serves it, and amending an accepted ADR for a
     convenience is not warranted.

## Footprint

- packages: internal/core/question, internal/core/interview, internal/core/ahoy, internal/core/runner, internal/core/oracle, internal/core/reflect, internal/core/layered, internal/term, internal/surface/cli, internal/surface/cli/ask, agents/, commands/, docs/reference, go.mod, ACKNOWLEDGEMENTS.md
- tests: TestSetupQuestionDrawsAt80Columns, TestSetupQuestionAnswersByArrowsAndByNumber, TestQuestionAt160ColumnsSitsSideBySide, TestInterviewOffATerminalIsPlainAndMatchesTheTerminalRecord, TestNoColorAndDumbTerminalDrawNoColour, TestRecordsDifferOnlyInWhereAnswered, TestLongListNarrowsAsThePersonTypes, TestLongListRestoresTerminalOnInterrupt, TestDrawnMaterialIsSanitisedFirst; the structural check's refusals; the banner's tests unchanged over the lifted wrapper; the no-route refusal; an invalid runner question falling back with its receipt

## Steps

1. The question type and the drawing
   - criteria: B2, B4, B7, and the layout half of B1
   - packages: internal/core/question, internal/term, internal/surface/cli/ask, internal/surface/cli, go.mod, ACKNOWLEDGEMENTS.md
   - tests: the structural check refuses a question without `Later`, an empty ask, and duplicate values, naming the part; golden layouts at 80, 100, and 160 columns with every prose run at most 80; NO_COLOR, TERM=dumb, and TERM unset draw no SGR sequence and mark the current option by glyph; injected escapes, C1, bidi, zero-width, and bare carriage returns drawn as `?`; the banner's tests pass unchanged over `term.Wrap`; `term.Size` falls back to `COLUMNS`, then 80
2. The answer loop: Arrow keys, typing to narrow, the numbered fallback, restore on every exit
   - criteria: B6, and the answer half of B1
   - packages: internal/term, internal/surface/cli/ask, internal/core/layered
   - tests: the list state machine over 300 choices (narrow, widen, clear, page, and choose by arrows and by number, nothing matching keeps `Later`); the pseudo-terminal child restored after Ctrl-C, SIGTERM, and a panic, exit 130 on the interrupt; `interview.list: numbered` and `ABCD_ACCESSIBLE` select the numbered reader; TERM=dumb and a refused raw mode fall to it with one stderr line
3. The fixed interviews, the answers record and "answered in"
   - criteria: B1, B3, B5 (setup and routing)
   - packages: internal/core/interview, internal/core/ahoy, internal/surface/cli, commands/
   - tests: TestSetupQuestionDrawsAt80Columns and TestSetupQuestionAnswersByArrowsAndByNumber through `drawnPrompter`; the routing confirmation drawn with the proposal in counts; the piped run plain, byte-identical in record and configuration to the drawn run; an unanswered question refused with its id and flag; the host path's answers file marked `Claude Code` differing from the drawn run in `answered_in` alone; the answers file refusing unknown and duplicate keys; the existing piped-install tests unchanged
4. The AI-written interviews through the person's own route
   - criteria: B5 (retrospective), B7 (runner text), decision 1
   - packages: internal/core/interview, internal/core/runner, internal/core/oracle, internal/core/reflect, agents/, internal/surface/cli, commands/, docs/reference
   - tests: a stub runner's two `ask` receipts drawn and recorded, then `done` writing the retrospective through `reflect write`'s path; the no-route refusal naming the role key, the machine file, and `abcd ahoy install`, exit 2, nothing written; an `ask` failing the check recorded as an invalid fallback and refused with no fallback host; the answers-file replay by ordinal matching the drawn run; the planning-interviewer row in the roster and its agent page present; the surface snapshot and command reference regenerated with the two sub-verbs
