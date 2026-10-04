---
id: spc-2610031844142274
slug: when-you-type-abcd-in-a-terminal-or-ask-for-the-board-in-a
intent: itd-2610031214560142
origin: researcher-authored
production_mode: hand-written
---
# when-you-type-abcd-in-a-terminal-or-ask-for-the-board-in-a

## Summary

This spec delivers
[itd-2610031214560142](../../intents/planned/itd-2610031214560142-when-you-type-abcd-in-a-terminal-or-ask-for-the-board-in-a.md)
(the board shows a product thinker what waits on them and what comes next, in a
few plain lines). The file name and slug are the ones minted on filing; the
intent's title is the one that describes the work.

Bare `abcd` opens on the view for the product thinker: a first line saying whose
view it is, then what is being built, the next three things to build, and how
many more are ready or parked, each state said by a word and a symbol as well
as a colour, each title fitted to the window. One flag opens the view for the
facilitator, which keeps every row of today's board and adds each planned
intent's spec id and an in-flight marker. One core renderer draws both views in
two forms: text for a Terminal or a pipe, and a markdown list the plugin page
runs and pastes unchanged in the host session. The board writes nothing.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 14, its confirmed criteria A1 to A7, and its five scope
conditions. The menu under the board is
[spc-2610031844158884](spc-2610031844158884-under-the-board-a-short-what-next-menu-offers-the-few-steps.md)'s.

## Scope

In:

- `internal/core/statusblock`: each planned row's spec id, and an in-flight
  marker on a lane row whose branch exists (iss-2609201954342967 part 2, folded
  in by decision 7).
- `internal/core/implement/loop`: `StatusLanes` (`loop.go`, line 974) hands the
  lane's branch to the block.
- A new core package, `internal/core/board`: the one renderer, two views (the
  product thinker's and the facilitator's), two forms (text and markdown).
- `internal/textwidth`: a fit-to-width helper beside `Columns` (line 22) and
  `Wrap` (line 40), cutting by display width with an ellipsis.
- `internal/core/statusline`: one exported way to paint a text in a role's
  fixed pair, so the view label and the badge share `rolePairs` (`badge.go`,
  line 83) and `paint` (line 137) and no second copy of the hexes exists.
- `internal/surface/cli`: the bare command's two new flags, the piped default,
  the `view` field in `--json`, `--no-color`'s help widened from "the banner"
  to the board (`cli.go`, line 354); the regenerated
  `docs/reference/cli/commands.md` and the brief chapter's generated appendix.
- Pages and record: `commands/abcd.md` (run the markdown form and paste it
  unchanged in one fenced block, keeping `--json` for the fields the page
  reasons over); `README.md`'s "First run" board example (lines 119 to 132); the
  brief's `04-surfaces/08-abcd.md` ("What ships today" and "The status block").
- A4's dated receipt, in the local tier.

Builds on and reuses:

- The width defect,
  [iss-2610031207397996](../../../work/issues/resolved/iss-2610031207397996-the-bare-status-board-ignores-the-terminal-s-width-in-an.md),
  fixed first by its own record (step 1): the window width read through
  golang.org/x/term with 80 as the fallback, every row wrapped with a hanging
  indent through `textwidth.Wrap`, the first three rows printed as words, even
  indents, and the next-up intent listed once, with its edit to the brief
  chapter and the status-block test. This spec designs none of that; it draws
  the facilitator's view on top of it and reads the width through the same
  helper.
- The colour ladder and the TTY check: `term.ResolveColorMode` (`term.go`,
  line 35), `term.IsTerminal` (line 66), `term.UTF8Locale` (line 84), and the
  stdout seam `bannerTTY` (`banner.go`, line 26).
- The status block's one read, `statusblock.Read` (`statusblock.go`,
  line 131), and the readiness result's `SpecID` (`intent/ready.go`, line 53).
- The plugin page tests' pattern of reading a page from the repository root
  (`reading_definitions_page_test.go`, line 22).

Out:

- The width defect's own changes, named above; they land under
  iss-2610031207397996 with a `Resolves:` trailer.
- The "what next?" menu, the one write bare `abcd` makes in a Terminal, its
  off setting, and the edits to the read-only promise: spc-2610031844158884.
  This spec leaves bare `abcd` writing nothing, as decision 3 says.
- iss-2609201954342967's part 1 (the next-actions list), which joins the menu;
  the issue is resolved by the menu spec's last step, once both parts ship.
- The question a waiting answer holds: nothing stores its text (the mode store
  holds one word, `mode/store.go`; the marker holds its presence,
  `mode/question.go`), so no view shows it. Decision 6 keeps the owed answer on
  the status line alone.
- The status line, the badge and their colours: unchanged. The role pairs stay
  provisional until itd-200's third open question lands, and the board follows
  whatever the pairs become.
- Any record held only in another working copy (scope condition 2).

## Approach

### One renderer, two views, two forms

`internal/core/board` takes the board's data and returns lines; it never writes
to a stream and never reads the environment, so every view and form is a
golden test with an injected width and colour rung:

```go
type View int // Product (the default) or Facilitator

type Form int // Text or Markdown

type Frame struct {
	View  View
	Form  Form
	Width int            // the window's columns; ignored by Markdown
	Mode  term.ColorMode // Mono in a pipe, under NO_COLOR and --no-color
	ASCII bool           // no UTF-8 locale: the plain-text symbols
}

// Input is everything the views draw from: the status block, and the rows
// today's board carries, each already read by the front door.
type Input struct {
	Dir    string
	Status *statusblock.Block
	Rows   []Row // the facilitator's rows: presence, peers, inbox, oracle, reviews, receipts
}

func Render(in Input, f Frame) []string
```

The facilitator's rows are handed in already worded (the defect's step makes
them words), so the core holds no knowledge of the inbox, the oracle or the
reviews readers; the surface reads them as it does today
(`boardPresence`, `boardPeers`, `boardInbox`, `boardOracle`, `boardReviews`)
and passes each as a labelled row. The status block is the one structured
input, because both views lay it out differently.

Every title and directory name passes `termsafe.Sanitize` before it is
measured, so widths are of what is drawn, and a title can never forge a line.

### The view for the product thinker

```text
view for the product thinker
● building: The board shows a product thinker what waits on them and what…
○ next: A 'what next?' menu under the board offers the next step, in a…
○ next: When abcd sets up a project it asks the owner whether abcd keeps…
○ next: Connecting a model service works out of the box
• 11 more ready, 40 parked
```

- Line 1 is the label, and only the label (decision 2). Its words carry the
  meaning; colour only repeats them.
- `building:` lines: one per intent the build's state file shows in a lane,
  however many lanes it has (open question 5). No lane: one line,
  `● building: nothing right now`.
- `next:` lines: the head (`next_up`) first, then Next in pick order, three in
  all, wherever the defect's fix lists the head. Fewer than three READY: as
  many as there are; none: `○ next: nothing is ready`.
- The count line: the READY intents not shown, then every Later row (planned
  intents the gate refuses, then drafts), as numbers: `• 11 more ready, 40
  parked`, `• no more ready, 40 parked`.
- No record id, no lane name, no target release, no command word, and no
  owed-answer line, whatever the stored status names (A1; decision 6).
- Each title is fitted to one line by display width, with `…` (`...` without
  UTF-8), through the new `textwidth.Fit(s string, limit int, ellipsis string)
  string`, at the measure of open question 4.

### The view for the facilitator

Its first line is `view for the facilitator`; every line after it is today's
board as the defect's step leaves it, in today's order (decision 11):

```text
view for the facilitator
abcd — ~/code/abcd
  git repo:   yes
  ...
  status:     Now 2 · Next 3 · Later 40
    Now:
      itd-2610031214560142  spc-2610031844142274  The board shows a
          product thinker what waits on them…  [lane-1: implement (run-…);
          in flight]
    Next:
      itd-…  spc-…  A 'what next?' menu under the board…  [next up]
    Later: 40 intents
```

Each Now and Next row gains the spec id the readiness gate judged
(`ReadyResult.SpecID`) after the intent id, and a lane row whose branch exists
gains `in flight` in its brackets (A5). Later stays a count (ruling BV1); its
rows carry `spec_id` in `--json` alone. Rows wrap at the window width with a
hanging indent, through the defect's wrap, rather than being cut: this reader
wants the whole row.

### Spec ids and the in-flight marker on the block

`statusblock.Row` (`statusblock.go`, line 70) gains `SpecID string
json:"spec_id,omitempty"`, set for every planned row from the readiness result
already in hand. `statusblock.Lane` (line 89) gains `Branch string
json:"branch,omitempty"` and `InFlight bool json:"in_flight,omitempty"`.
`loop.StatusLanes` fills `Branch` from the lane's recorded branch
(`state.go`, line 376, `build/<run>-<lane>`), and `Read` sets `InFlight` when
the intent's spec is in `specs/open/` and `gitutil.ResolveCommit(root,
"refs/heads/"+branch)` resolves (open question 3). A lane still at its
`worktree` stage, before its branch is cut, is not in flight.

### The label's colour, per decision 10

The label is the one painted element that is not one of the 16 colours.

- At `term.TrueColor` the label is painted in the role's fixed pair, dark text
  on the light fill, as the badge paints it: `statusline.PaintRole(state,
  text string) string`, a new exported wrapper over `rolePairs` and `paint`,
  with `mode.ProductThinker` for the product thinker's view and
  `mode.Facilitator` for the facilitator's. The padding spaces are the fill's,
  as `renderBadge` adds them (line 115).
- At `Ansi256` and `Ansi16` the label is its words, unpainted (scope
  condition 4). No 4-bit stand-in is drawn, so the role colour means one pair
  wherever it shows.
- At `Mono` (a pipe, `NO_COLOR`, `--no-color`, `TERM` dumb or unset), and in
  the markdown form always, nothing is painted.

The other states keep to the 16 colours, each repeating its word and symbol:
`building` in green (32), `next` in cyan (36), the count line unpainted. Red
and yellow stay for problems, as the research ranks it; the facilitator's
view paints nothing it does not paint today.

### Symbols

`●` building, `○` next, `•` the count line, from the set every common terminal
font carries. Without a UTF-8 locale (`term.UTF8Locale` false) they are `*`,
`o` and `-`, and the ellipsis is `...` (A6). The markdown form always uses
the UTF-8 symbols: the host session renders UTF-8.

### The markdown form

The markdown form is a list, never a table, with no escape byte and no fitted
title (there is no window):

```markdown
view for the product thinker

- ● building: The board shows a product thinker what waits on them and what comes next, in a few plain lines
- ○ next: A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session
- ○ next: …
- ○ next: …
- • 11 more ready, 40 parked
```

The facilitator's markdown form is the same rows as nested list items, one
level per today's indent. Every line begins with the label, a blank, or `- `,
so a title can never open or close the page's fence; a test holds that over a
title made of backticks and tildes.

### The bare command

- `--view product|facilitator` picks the view; omitted, the product thinker's,
  always (decision 8), never the stored status's role and never the last one
  chosen.
- `--format text|markdown` picks the form; omitted, text.
- `--json` is unchanged and additive: today's fields, the block's new
  `spec_id`, `branch` and `in_flight`, and `view` naming the view asked for.
  `--json` beside `--format` is refused, exit 2, naming both flags, and so is
  either new flag beside a record id, whose answer has one form.
- In a pipe (`bannerTTY` false) the text form is drawn at `Mono` and 80
  columns, the product thinker's view unless `--view` says otherwise (A2). This
  changes what a pipe receives by default; the facilitator's view keeps
  every row a script read before, behind `--view facilitator`.
- On a Terminal the banner is drawn above the board as today (open question 2),
  then the view; `--no-color` and `NO_COLOR` reach both through
  `term.ResolveColorMode`.
- The surface reads the width through the defect's helper (the window, else
  `COLUMNS`, else 80), and the locale through `term.UTF8Locale`.

### In the host session

`commands/abcd.md` runs `"${CLAUDE_PLUGIN_ROOT}/abcd" --format markdown` and
tells the agent to paste its output unchanged, in one fenced block, adding
nothing inside the fence and retelling none of it outside (decision 12; A3). The
page keeps `--json` for the fields it reasons over: the record-id dispatch, and
the menu's fields once spc-2610031844158884 adds them. Its sections on the
presence, peers, inbox, oracle and reviews rows move under "the view for the
facilitator", which `--view facilitator --format markdown` draws.

Whether the agent passes it on unchanged is measured, not assumed (scope
condition 5): A4's eval below.

## How each acceptance criterion is met

**A1.** `TestProductViewGolden`, in `internal/core/board`, at 80 columns and
`Mono`, on two fixtures: one intent in a lane, three READY and 40 parked
(`… no more ready, 40 parked`), and one in a lane, fourteen READY and 40 parked
(`• 11 more ready, 40 parked`). Line 1 is `view for the product thinker`; then
one `building:` line, three `next:` lines and the count line. In
`internal/surface/cli`, `TestBareBoardOpensOnTheProductView` runs bare `abcd`
with the TTY seam on, the stored status set to `product-thinker`, then to
`facilitator`: the label is the product thinker's both times, and no line holds
a record handle (`recordid.HandleInText`), the word `abcd` followed by a verb,
or `waiting on`.

**A2.** `TestPipedBoardCarriesNoEscape`: bare `abcd` with stdout a buffer and
`COLORTERM=truecolor`, then with `--json`. No output holds the byte 0x1b; the
text form's first line is `view for the product thinker`, and the JSON's
`view` is `product-thinker`.

**A3.** `TestMarkdownFormGolden` (both views) byte-compares with its golden and
asserts no `|`-delimited table row and no 0x1b. `TestFenceSafeMarkdownTitles`
holds the fence rule. `TestAbcdPageRelaysTheMarkdownForm` reads
`commands/abcd.md` and asserts it runs exactly
`"${CLAUDE_PLUGIN_ROOT}/abcd" --format markdown` and tells the agent to paste
the output unchanged in one fenced block.

**A4.** A dated receipt,
`.abcd/.work.local/logs/board-session-relay-<yyyy-mm-dd>.md`: `/abcd` asked in
three host sessions across two models in this checkout. For each session the
receipt records the model, the fenced block copied from the reply, the output
of the markdown form run in the same checkout at the same commit, and the empty
diff between them. The product thinker then names the next item from one of the
replies, and the receipt quotes them.

**A5.** `TestFacilitatorViewGolden`: today's board fixture plus one run whose
lane branch exists and whose spec is open. Line 1 reads `view for the
facilitator`; the directory, git, record, tiers, presence, peers, inbox, oracle,
reviews and receipts rows are all present; the lane row carries its spec id and
`in flight`. In `internal/core/statusblock`, `TestPlannedRowsCarryTheirSpec`
and `TestInFlightNeedsAnOpenSpecAndABranch` (a lane at `worktree` before its
branch exists, and one whose spec is closed, are not in flight).

**A6.** `TestBoardWithoutColourKeepsWordsAndSymbols`: with the TTY seam on and
`COLORTERM=truecolor`, under `NO_COLOR=1`, then under `--no-color`, the lines
equal the `Mono` golden, with no 0x1b, each state word beside its symbol
(`● building:`, `○ next:`, `•`); with `LANG=C` and no `LC_*`,
`* building:` replaces `● building:`. `TestLabelPaintsOnlyAtTrueColor` holds the
label's three rungs.

**A7.** `TestBoardFitsTheWindow`: a 200-character title and a title of
East Asian wide glyphs, drawn at 80 columns in both views and both colour
extremes. Every line's `textwidth.Columns`, measured with escapes stripped, is
at most 80. `TestFitCutsByDisplayWidth`, in `internal/textwidth`, holds the
helper on wide glyphs, a cut that would split one, and the ASCII ellipsis.

## Open design questions

These are the technical facilitator's; the records do not settle them. Each is
designed to the marked option, and the reason is given beneath it.

1. **How a pipe and the page ask for a view and a form.** (a) Two flags,
   `--view product|facilitator` and `--format text|markdown` (designed to).
   (b) Two booleans, `--facilitator` and `--markdown`. (c) A sub-verb.
   - (a): one flag per axis reads as what it chooses, and the menu's
     "open the view for the facilitator" maps onto one value. (c) is a
     `<verb> view`, which the naming discipline forbids
     (`02-constraints/04-naming.md`).
   - Decided: as the spec is designed to, on 2026-10-03, without a question (the person's ruling that an obvious answer is decided; no product choice is involved).
2. **The banner above the board on a Terminal.** (a) Keep it: itd-112's banner
   is drawn first, as today, and the board's first line is the label (designed
   to). (b) Drop it whenever the product thinker's view is drawn, so the label
   is the first line on the screen.
   - (a): the banner is a shipped intent's promise, and removing it is the
     product thinker's call, not this spec's, and the view's six lines still
     fit one 80 by 24 screen beneath it. If the
     receipt for A4 or B6 shows the product thinker losing the board above the
     menu, (b) goes to them as a question.
   - Decided: the product thinker, 2026-10-03, asked whether the banner stays above the board (keep it; drop it from the board, the banner staying on the help; decide later): keep the banner.
3. **What "its branch exists" reads.** (a) The lane branch this checkout's run
   state records, resolving as a local branch (designed to). (b) Any local
   branch whose name holds the spec's id. (c) The peers listing.
   - (a): it is the one branch the record ties to the spec; (b) guesses from a
     name, and (c) reports another working copy, which scope condition 2 leaves
     out.
   - Decided: as the spec is designed to, on 2026-10-03, without a question (the person's ruling that an obvious answer is decided; no product choice is involved).
4. **How wide a title runs in the product thinker's view.** (a) The window, up
   to 100 columns (designed to). (b) The whole window.
   - (a): the research caps a glance at about 100 columns; a wider window shows
     more of a title, never a wall.
   - Decided: as the spec is designed to, on 2026-10-03, without a question (the person's ruling that an obvious answer is decided; no product choice is involved).
5. **One `building:` line per intent, or per lane.** (a) Per intent, however
   many lanes its run has (designed to). (b) Per lane, as the facilitator's
   view lists them.
   - (a): the product thinker asks what is being built, not how many lanes
     build it; the lanes stay in the facilitator's view.
   - Decided: as the spec is designed to, on 2026-10-03, without a question (the person's ruling that an obvious answer is decided; no product choice is involved).

## Footprint

- packages: internal/core/statusblock, internal/core/implement/loop, internal/core/board, internal/textwidth, internal/core/statusline, internal/surface/cli, commands/abcd.md, README.md, docs/reference, .abcd/development/brief/04-surfaces
- tests: TestPlannedRowsCarryTheirSpec, TestInFlightNeedsAnOpenSpecAndABranch, TestFitCutsByDisplayWidth, TestProductViewGolden, TestFacilitatorViewGolden, TestMarkdownFormGolden, TestFenceSafeMarkdownTitles, TestLabelPaintsOnlyAtTrueColor, TestBoardWithoutColourKeepsWordsAndSymbols, TestBoardFitsTheWindow, TestBareBoardOpensOnTheProductView, TestPipedBoardCarriesNoEscape, TestFormatAndJSONAreRefusedTogether, TestAbcdPageRelaysTheMarkdownForm, the existing board tests moved behind --view facilitator; the command reference's drift test and the brief appendix after regeneration; docs-lint and record-lint clean; A4's dated receipt in the local tier

## Steps

1. The width defect, under its own record
   - criteria: none of this spec's; iss-2610031207397996's remedy, which the facilitator's view and the title fit build on
   - packages: internal/surface/cli, internal/core/statusblock, internal/term, go.mod, .abcd/development/brief/04-surfaces
   - tests: the defect's golden board at 80 columns with a long title and the receipts line, and the non-TTY no-escape assertion, each watched fail first; lands with `Resolves: iss-2610031207397996`, and adopts golang.org/x/term unless spc-2610030911534855 step 1 (PR #788) has already
2. Spec ids and the in-flight marker on the block
   - criteria: the data half of A5
   - packages: internal/core/statusblock, internal/core/implement/loop
   - tests: TestPlannedRowsCarryTheirSpec, TestInFlightNeedsAnOpenSpecAndABranch, each watched fail first; the existing block tests pass unchanged
   - lands after step 1, which edits the same block and its test
3. The renderer: two views, two forms, the label's colour
   - criteria: A1, A3 (the form), A5 (the render), A6, A7
   - packages: internal/core/board, internal/textwidth, internal/core/statusline
   - tests: TestFitCutsByDisplayWidth, TestProductViewGolden, TestFacilitatorViewGolden, TestMarkdownFormGolden, TestFenceSafeMarkdownTitles, TestLabelPaintsOnlyAtTrueColor, TestBoardWithoutColourKeepsWordsAndSymbols, TestBoardFitsTheWindow, each watched fail first; the badge tests pass unchanged
   - lands after step 2
4. The bare command draws the views
   - criteria: A1, A2, A5 and A6 end to end
   - packages: internal/surface/cli, README.md, docs/reference, .abcd/development/brief/04-surfaces
   - tests: TestBareBoardOpensOnTheProductView, TestPipedBoardCarriesNoEscape, TestFormatAndJSONAreRefusedTogether, each watched fail first; the existing board tests pass with `--view facilitator`; the command reference and the brief appendix regenerated with `go generate ./internal/surface/cli`
   - lands after step 3
5. The page relays the markdown form; the receipt and the close
   - criteria: A3 (the page), A4; the intent's close
   - packages: commands/abcd.md
   - tests: TestAbcdPageRelaysTheMarkdownForm, watched fail first; A4's receipt taken at the step's branch tip before the pull request; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031844142274` (the intent already declares `impact: additive`) with a `Delivers: itd-2610031214560142` trailer
   - lands after step 4
