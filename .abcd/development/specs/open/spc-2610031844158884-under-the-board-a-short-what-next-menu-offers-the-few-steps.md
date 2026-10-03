---
id: spc-2610031844158884
slug: under-the-board-a-short-what-next-menu-offers-the-few-steps
intent: itd-2610031215002409
origin: researcher-authored
production_mode: hand-written
---
# under-the-board-a-short-what-next-menu-offers-the-few-steps

## Summary

This spec delivers
[itd-2610031215002409](../../intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu-offers-the-few-steps.md)
(a "what next?" menu under the board offers the next step, in a Terminal and in
a host session). The file name and slug are the ones minted on filing; the
intent's title is the one that describes the work.

Under the board, a menu of at most four moves follows the state: start building
the next item, see everything that is ready (or answer what is waiting, when
something waits), open the other view, and "Nothing for now", always last. One
question object, built in the core, is drawn as an arrow-key or numbered menu
in a Terminal and handed unchanged to the host's question tool in a session.
In a Terminal, and only there, bare `abcd` sets the waiting-on status while the
menu waits and puts it back on every exit; this is the one write bare `abcd`
makes, and the brief chapter and the plugin page say so. Where nobody can
answer, the board prints alone, and one machine setting turns the menu off for
good.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 12, its confirmed criteria B1 to B7, and its five scope
conditions. The board it ends is
[spc-2610031844142274](spc-2610031844142274-when-you-type-abcd-in-a-terminal-or-ask-for-the-board-in-a.md)'s.

## Scope

In:

- `internal/core/board`: the menu's question object for each view and state,
  the moves behind its options, and the next-actions list folded in from
  iss-2609201954342967 part 1 (decision 5).
- `internal/core/question`: "Nothing for now" joins `Default.LaterLabels`
  (`limits.go`, line 48), so the menu's last option is its decide-later option
  (open question 2).
- `internal/core/mode`: one call that holds the waiting-on status for a
  question abcd asks itself, and puts back what it found.
- `internal/surface/cli`: the menu drawn and answered under the board in a
  Terminal; the conditions under which it is drawn; `--no-menu`; the off
  setting's read; "Start building" through the build's own start; refusals in
  plain words; the `menu` field in `--json`; `--show ready|actions`; the root's
  help and the regenerated `docs/reference/cli/commands.md`.
- `internal/core/surface/sentences.go`: the `abcd` sentence (line 24), which
  says "Writes nothing" today and becomes the one write, and the page
  descriptions generated from it.
- The read-only promise, edited where it is stated (decision 4):
  - the brief's `04-surfaces/08-abcd.md`: the opening paragraph ("strictly
    read-only", line 5), "What ships today" ("Two read-only forms", line 36),
    and the presence line's "the board reads it and never writes it"
    (line 112);
  - `commands/abcd.md`: "This command performs **zero writes**" (line 10) and
    "The board reads the state and never changes it" (lines 31 to 32);
  - the root command's help (`cli.go`, lines 219 to 231; line 225, "The bare
    and the id form are strictly read-only") and its doc comment (line 209,
    "bare invocation never mutates");
  - the bare-invocation section of the brief's `04-surfaces/README.md`
    (line 188), which lists where the convention bends.
- B6's dated receipt, in the local tier.

Builds on and reuses:

- The board: spc-2610031844142274's renderer, its two views, its `--view` and
  `--format` flags, and its markdown form in the session; this spec draws
  under them and changes none of them.
- The question type and its drawing, which exist only on the branch of PR #788
  (`feat/asking-pages`) and not yet at this spec's base: `question.Ask`,
  `question.Question`, `question.Option` and `Ask.Fields`
  (`internal/core/question/ask.go`), `ask.Prepare` and `ask.Draw`
  (`internal/surface/cli/ask/safe.go` and `layout.go`), and `term.Size`
  (`internal/term/size.go`, adopting golang.org/x/term). At this base
  `internal/core/question` holds the field view and the limits only
  (`Fields`, `Limits`, `CheckLimits` at `check.go`, line 118).
- The plain-Terminal answer loop of
  [spc-2610030911534855](spc-2610030911534855-a-person-can-run-abcd-s-interviews-in-a-plain-terminal.md)
  step 2, which exists nowhere yet: the arrow-key list, choosing by number, the
  numbered reader chosen by `interview.list: numbered` or `ABCD_ACCESSIBLE` and
  forced by `TERM=dumb`, and `term.RawSession`, whose restore runs on every
  exit. The menu asks through it and holds no keyboard code of its own.
- The mode store and its question marker: `mode.ReadAt` and `mode.SetAt`
  (`store.go`, lines 70 and 109), `mode.MarkQuestionOpen`,
  `mode.ResetOnAnswer` and `mode.CanSet` (`question.go`, lines 57, 114
  and 165).
- The session's status rules, unchanged: the guard refuses a chip-headed
  question while the mode reads managed (`questionGate`,
  `guard_question.go`, line 82, naming `abcd mode` in `questionRefusal`,
  line 52), marks it open when it admits one (line 137), and the prompt hook
  resets it on the answer (`resetModeOnAnswer`, line 275).
- The build's start: `loop.Start` (`loop.go`, line 263), its runner read
  (`loadRunners`, `build.go`, line 927), and its refusal, `loop.Refusal`
  (`refusal.go`, line 15), whose `Check` names the pre-start check
  (`check.go`, lines 28 to 39).
- The layered configuration (`layered.Config`, `layered.go`, line 111), whose
  machine layer is `~/.abcd/config.json`, beside `~/.abcd/statusline.json`.

Out:

- The board itself, its views, its forms and its flags: spc-2610031844142274.
- The width defect, iss-2610031207397996, fixed first by its own record.
- Any change to the question layout or the answer loop
  (itd-2610030810350727 and itd-2610030810370060 are followed as they are).
- A write by the board in a pipe, under `--json`, under `--format markdown`, in
  a repository without a local tier, or in a session: there the board writes
  nothing, as today.
- The text of a waiting question, which nothing stores: "Answer what is
  waiting" says where it waits and cannot show it (open question 4).
- A verb or flag that writes the off setting: it is a key a person sets by
  hand (open question 6).

## Approach

### The moves follow the state

The core builds the menu from the status block, the stored status read before
the menu sets anything, and the view:

| View | Option 1 | Option 2 | Option 3 | Option 4 |
| --- | --- | --- | --- | --- |
| product thinker | Start building the next item | See everything that is ready, or Answer what is waiting | Open the facilitator's view | Nothing for now |
| facilitator | Start building the next item | See the next actions, or Answer what is waiting | Open the product thinker's view | Nothing for now |

- "Start building the next item" is offered only when the block has a head
  (`next_up`); its description names the item: "Starts building ‘<title>’ now,
  with no second question." (decision 7). Without a head the menu has three
  options, never fewer than two (B2).
- "Answer what is waiting" takes option 2's place when the stored status names
  the view's own role: `product-thinker` under the product thinker's view,
  `facilitator` under the facilitator's (decisions 6 and 9).
- "Nothing for now" is last, always, and is the question's `Later` option.

Each label is at most five words, as the limits require, so the facilitator's
view is opened by "Open the facilitator's view" (open question 1). The
question's chip is `Product Q<n>` under the product thinker's view and
`Tech Q<n>` under the facilitator's, `<n>` counting the menus asked in one run.
Its material is one paragraph saying what the board above shows; its `Now:`
line says what holds (`Now: one item being built, three ready to start,
nothing waiting on you`); its `Change later:` line says how to come back to it
("open the board again to choose another step"), and in the facilitator's view
also names the off setting. The product thinker's question carries no record
id and no command word, so it passes the register rule (`RuleRegister`,
`check.go`, line 29).

A title is runtime-read text. Every composed menu is held to
`question.CheckLimits` before it is drawn or emitted, and a title that makes a
description fail (a sentence end inside it, a record handle in the product
thinker's register) is dropped from the description, which then reads "Starts
building the next item now, with no second question.", the title staying in
the board above. A menu therefore never fails the limits at run time.

### What each move does

| Move | In a Terminal | In a session |
| --- | --- | --- |
| Start building the next item | starts the build at once, then ends | the page runs the build through `/abcd:build` |
| See everything that is ready | prints every ready title in the view, then asks again | the page runs `--show ready --format markdown` and pastes it |
| See the next actions | prints the next-actions list, then asks again | the page runs `--show actions --view facilitator --format markdown` |
| Answer what is waiting | says where the question was asked, then asks again | as in a Terminal |
| Open the other view | draws the other view and its menu | the page runs that view's markdown form, then asks its menu |
| Nothing for now | ends | ends |

A move that prints asks the menu again under what it printed, with the next
ordinal on the chip; the status stays held until the menu ends. The Terminal
never redraws over the board: everything stays in the scrollback.

### Start building at once

The menu calls the same start `abcd build <itd-N>` makes, factored out of
`newBuildCommand` (`build.go`, line 93) into one function both call, so the
runner read, the pace and the refusals are the build's own. There is no second
question (decision 7; B7). On a start the product thinker's view prints
"Started building ‘<title>’."; the facilitator's view adds the run id and the
build's own `next:` line. A refusal is printed under the board in plain words,
from the refusal's check:

| Refusal | Printed |
| --- | --- |
| `peers` (contention) | Not started: another working copy is building ‘<title>’. |
| `open_questions` | Not started: ‘<title>’ still has an open question. |
| `claim_sections` | Not started: ‘<title>’ has a section still to answer. |
| `hold` | Not started: ‘<title>’ is on hold. |
| `blocked` | Not started: ‘<title>’ waits on other work first. |
| `steps` | Not started: ‘<title>’ has nothing left to build. |
| any other | Not started: the build could not begin. |

The facilitator's view adds the refusal's own line (`loop.Refusal.Error`). The
head already passes these checks when the board is read, so a refusal here
means something changed in between.

### The one write, in a Terminal

The menu is drawn only when all of these hold; otherwise the board prints
alone, writes nothing, and emits no escape byte (B3; decision 10):

- stdin, stdout and stderr are each a Terminal (`term.IsTerminal`); the menu is
  drawn on stderr, as the answer loop draws every question;
- neither `--json` nor `--format markdown` nor `--no-menu` is passed;
- the off setting does not say off;
- the repository is managed (`ahoy.Managed`) and its local tier can take the
  write (`mode.CanSet`).

A new call in `internal/core/mode` holds the status for a question abcd asks
itself:

```go
// Hold records that abcd itself is asking s a question: it reads the status
// and the marker as they are, sets s, and marks the question open. The
// returned release puts the status back as it found it, and removes the
// marker only when Hold wrote it. Release is idempotent.
func Hold(repoRoot string, s State) (release func() error, err error)
```

The menu calls `Hold` with the view's role before it draws, and calls `Hold`
again when the other view is opened, so the status follows the view shown
(decision 9; B4). The release runs from every exit: an answer, end of input,
Ctrl-C, a signal from outside, a panic, and a failed write. It is handed to the
answer loop's restore, so it runs in the same once-only path that restores the
Terminal (spc-2610030911534855 step 2's four exits). If that step lands without
a way to hand it a function, this spec's step 3 adds one to `term.RawSession`;
the numbered reader, which holds no raw mode, runs the release from the same
deferred call and signal handler.

The release puts back what it found (open question 3): a status that read
`managed` reads `managed` again and the marker is gone, which is B4; a status
that already named someone is left naming them. If the process is killed with
SIGKILL, the status stays held: the next answer in a session of this checkout
resets it through the prompt hook, as a stale marker is reset today, and
`abcd mode managed` resets it by hand.

### In the host session

In a session the host's question tool asks, and abcd writes nothing itself
(scope condition 3). `--json` gains a `menu` object, present where a menu
could be asked (a managed repository):

```json
{
  "menu": {
    "addressee": "product-thinker",
    "question": {
      "header": "Product Q1",
      "question": "The board above shows …\n\nNow: …\nChange later: …\n\nWhat would you like to do next?",
      "options": [
        {"label": "Start building the next item", "description": "Starts building ‘…’ now, with no second question."},
        {"label": "See everything that is ready", "description": "…"},
        {"label": "Open the facilitator's view", "description": "…"},
        {"label": "Nothing for now", "description": "…"}
      ]
    },
    "moves": [
      {"label": "Start building the next item", "build": "itd-…"},
      {"label": "See everything that is ready", "args": ["--show", "ready", "--format", "markdown"]},
      {"label": "Open the facilitator's view", "args": ["--view", "facilitator", "--format", "markdown"]},
      {"label": "Nothing for now"}
    ]
  }
}
```

`question` is the host question tool's own input, made by `Ask.Fields` and
encoded with the key names `hostQuestionInput` already reads
(`guard_question.go`, line 149), so the page hands it over verbatim and the
labels and descriptions are the Terminal's (B6). `moves` says what each answer
does, so the page holds no mapping of its own.

`commands/abcd.md` then says, after pasting the board: run
`"${CLAUDE_PLUGIN_ROOT}/abcd" mode <addressee>` (today's rule, which the guard
enforces), ask `menu.question` with the host's question tool unchanged, and do
the chosen move. The guard marks the question open and the answer resets the
status, as for every question.

### The next-actions list

The facilitator's "See the next actions" prints the list iss-2609201954342967
part 1 asks for, derived the way `abcd <record-id>` derives one record's next
move (`record.Describe`, `record.go`, line 115), for:

- each shipped intent whose fidelity review is owed (`intent.Reviews`,
  `owed.go`, line 81);
- each draft whose Open Questions are all answered, ready for its planning
  interview;
- each lapsed claim in the shared run state (`internal/core/implement`,
  `claim.go`).

Each line is the record's id, its title fitted to the window, and its move with
its command. At most ten are listed, then "and N more". The product thinker's
view never shows this list.

### The off setting and the per-run form

The setting is `board.menu` in the machine's `~/.abcd/config.json`, `"on"` (the
bundled default) or `"off"`, read through `layered` with a claimed `board`
namespace (decision 8; open question 6). A `board.menu` key in the
repository's `.abcd/config.json` is never read as the answer: it is skipped
with one line on stderr naming both files, as a runner route set in the
repository is skipped today. `--no-menu` turns the menu off for one run.

### The read-only promise, restated

The edits named under Scope all say the same thing, in each place's own words:
bare `abcd` writes nothing, except that in a Terminal, while its menu waits for
an answer, it sets the waiting-on status and puts it back when the menu ends;
in a pipe, under `--json` or `--format markdown`, and in a session it writes
nothing. The `abcd` sentence becomes:

> Render the board, or say what one record id is and its next move: Writes only
> the waiting-on status, in a Terminal; refuses any other positional argument.

It is 154 characters, under `SentenceCap` (160), and the page descriptions are
regenerated from it.

## How each acceptance criterion is met

**B1.** `TestMenuOffersFourMoves`, in `internal/core/board`: a block with a
head and nothing waiting gives `Product Q1` and four options, in order, the
last "Nothing for now". `TestTerminalMenuAnswersByArrowsAndByNumber`, in
`internal/surface/cli`, runs bare `abcd` in a pseudo-terminal through the
answer loop's test seam: Down, Enter chooses option 2; `3` then Enter chooses
option 3.

**B2.** `TestMenuMovesFollowTheState`, a table over both views: something
waiting puts "Answer what is waiting" in option 2 with four options and
"Nothing for now" last; no READY intent drops "Start building", leaving three;
both together leave "Answer what is waiting", the other view and "Nothing for
now".

**B3.** `TestNoMenuWhereNobodyCanAnswer`, a table: stdin a pipe, stdout a pipe,
stderr a pipe, `board.menu: "off"` in the machine file, `--no-menu`, no local
tier, and a repository file setting `board.menu` (skipped, so the menu shows,
with its stderr line). In each no-menu case the board prints alone, the output
holds no 0x1b, and `.abcd/.work.local/mode` and `question_open` are
byte-identical (or equally absent) before and after.

**B4.** `TestMenuHoldsTheStatusWhileItWaits`: an injected answer reader reads
the store while the menu waits. Under the product thinker's view it reads
`product-thinker` with the marker present; after "Open the facilitator's
view" the chip is `Tech Q2` and the status `facilitator`. After an answer, end
of input, and an injected Ctrl-C, the status reads `managed`, the marker is
gone, and the Terminal's restore has run. In `internal/core/mode`,
`TestHoldPutsBackWhatItFound` holds the found status and a marker Hold did not
write.

**B5.** `TestMenuQuestionPassesTheLimits`: every combination of view, waiting
and head, and titles carrying a record handle, a sentence end and wide glyphs,
held to `question.CheckLimits` with the `Product` and the `Tech` addressee and
the binary's verbs: no finding.

**B6.** `TestAbcdPageAsksTheMenuThroughTheHostTool` reads `commands/abcd.md`
and asserts it sets the status with `abcd mode` from `menu.addressee`, asks
`menu.question` unchanged, and follows `menu.moves`. The dated receipt,
`.abcd/.work.local/logs/board-menu-session-<yyyy-mm-dd>.md`, taken through
`/abcd` in a host session, records the date and host version, the question as
asked beside `menu.question` from `--json` in the same checkout, the status
line reading "waiting on the product thinker" while it waits and
"abcd-managed" after, and the product thinker's own account of the step they
wanted and took.

**B7.** `TestStartBuildingStartsAtOnce`: choosing option 1 creates the run in
the local tier and the answer reader is called once. `TestStartRefusalInPlainWords`:
with a peer holding the intent, the board is followed by "Not started: another
working copy is building ‘<title>’." and no run is created; each other row of
the refusal table is held.

## Open design questions

These are the technical facilitator's; the records do not settle them. Each is
designed to the marked option, and the reason is given beneath it.

1. **The labels within five words.** B1's example reads "Open the view for the
   facilitator" (six words) and "Start building <title>". (a) "Open the
   facilitator's view" and "Start building the next item", the title in the
   description (designed to). (b) Raise `LabelWords` to six.
   - (a): B5 holds every label to the limits as they stand, and decision 7
     already puts the item in the description.
2. **"Nothing for now" as the decide-later option.** (a) Add it to
   `question.Default.LaterLabels` (designed to). (b) Label the option "Decide
   later".
   - (a): the press release names "Nothing for now", and the limits check
     needs exactly one decide-later option, last. On #788's branch the asking
     rules text is generated from `LaterLabels`, so its golden changes in the
     same step.
3. **What the status reads after the menu, when something was already
   waiting.** (a) Put back what it found, removing only a marker it wrote
   (designed to). (b) Reset to `managed`, as the prompt hook does.
   - (a): the menu did not create the waiting answer, so ending the menu must
     not erase it; B4's fixture starts at `managed`, so (a) meets it exactly.
4. **What "Answer what is waiting" does in a Terminal.** (a) Say in plain
   words that the question was asked in the agent session that stopped for
   it, and ask the menu again, changing nothing (designed to). (b) Mark the
   waiting answer as answered. (c) Leave the move out in a Terminal.
   - (a): nothing stores the question's text, so a Terminal cannot show it;
     (b) would clear a stop nobody answered, and (c) contradicts B2.
5. **The facilitator's moves and the next actions.** (a) Option 2 is "See the
   next actions", displaced by "Answer what is waiting" as the product
   thinker's is (designed to). (b) The first derived action as its own move.
   (c) The list as rows of the facilitator's view, the moves unchanged.
   - (a): it keeps both views one shape, and decision 5 puts the list in the
     menu, not the board.
6. **Where the off setting lives, and how it is set.** (a) `board.menu` in
   `~/.abcd/config.json`, set by hand, a repository key skipped with a warning
   (designed to). (b) A key in `~/.abcd/statusline.json`. (c) A flag on
   `abcd ahoy install` that writes it.
   - (a): the configuration file already has a machine layer and the
     repository-key refusal; the status-line file is the status line's own.
     (c) can follow if people ask for it.
7. **How the session gets the menu.** (a) A `menu` object in `--json` holding
   the host tool's input and the moves (designed to). (b) The page composes
   the question from the status fields.
   - (a): one builder feeds both places, as the research ranks first; (b)
     drifts with the model.

## Footprint

- packages: internal/core/board, internal/core/question, internal/core/mode, internal/core/record, internal/core/surface, internal/surface/cli, internal/term, commands/abcd.md, docs/reference, .abcd/development/brief/04-surfaces
- tests: TestMenuOffersFourMoves, TestMenuMovesFollowTheState, TestMenuQuestionPassesTheLimits, TestNextActionsList, TestHoldPutsBackWhatItFound, TestHoldRefusesWithoutATier, TestTerminalMenuAnswersByArrowsAndByNumber, TestNoMenuWhereNobodyCanAnswer, TestMenuHoldsTheStatusWhileItWaits, TestStartBuildingStartsAtOnce, TestStartRefusalInPlainWords, TestBoardMenuSettingIsMachineOnly, TestMenuJSONIsTheHostToolInput, TestShowListsReadyAndActions, TestAbcdPageAsksTheMenuThroughTheHostTool, TestAbcdPageStatesTheOneWrite; the limits and asking-rules tests updated for the new later label; the sentence and command-reference drift tests after regeneration; docs-lint and record-lint clean; B6's dated receipt in the local tier

## Steps

1. Hold the waiting-on status for a question abcd asks
   - criteria: the store half of B4
   - packages: internal/core/mode
   - tests: TestHoldPutsBackWhatItFound, TestHoldRefusesWithoutATier, each watched fail first; the mode and guard tests pass unchanged
   - waits on nothing; may land before the board
2. The menu's question and its moves
   - criteria: B2, B5, and the core half of B1
   - packages: internal/core/board, internal/core/question, internal/core/record
   - tests: TestMenuOffersFourMoves, TestMenuMovesFollowTheState, TestMenuQuestionPassesTheLimits, TestNextActionsList, each watched fail first; the limits test and the asking-rules golden updated for "Nothing for now"
   - lands after spc-2610031844142274 step 3, and waits on PR #788's question type reaching the base
3. The menu in a Terminal, and the one write stated where the promise is made
   - criteria: B1, B3, B4, B7
   - packages: internal/surface/cli, internal/term, internal/core/surface, docs/reference, .abcd/development/brief/04-surfaces, commands/abcd.md
   - tests: TestTerminalMenuAnswersByArrowsAndByNumber, TestNoMenuWhereNobodyCanAnswer, TestMenuHoldsTheStatusWhileItWaits, TestStartBuildingStartsAtOnce, TestStartRefusalInPlainWords, TestBoardMenuSettingIsMachineOnly, TestAbcdPageStatesTheOneWrite, each watched fail first; the sentence and the command reference regenerated with `go generate ./internal/surface/cli`
   - lands after steps 1 and 2, spc-2610031844142274 step 4, and spc-2610030911534855 step 2 (the answer loop)
4. The menu in the host session
   - criteria: the page half of B6
   - packages: internal/surface/cli, commands/abcd.md, docs/reference
   - tests: TestMenuJSONIsTheHostToolInput, TestShowListsReadyAndActions, TestAbcdPageAsksTheMenuThroughTheHostTool, each watched fail first
   - lands after step 3 and spc-2610031844142274 step 5
5. The receipt and the close
   - criteria: B6's receipt; the intent's close; iss-2609201954342967 resolved, both its parts now shipped
   - packages: .abcd/work/issues
   - tests: B6's receipt taken at the step's branch tip before the pull request; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031844158884` (the intent already declares `impact: additive`) with `Delivers: itd-2610031215002409` and `Resolves: iss-2609201954342967` trailers
   - lands after step 4
