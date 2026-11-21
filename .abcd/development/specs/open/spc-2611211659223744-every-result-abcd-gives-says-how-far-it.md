---
id: spc-2611211659223744
slug: every-result-abcd-gives-says-how-far-it
intent: itd-2611211024360819
origin: researcher-authored
production_mode: hand-written
---
# every-result-abcd-gives-says-how-far-it

## Summary

Seven commands gain one statement of how far a run looked: what it looked
at, what the command is known not to look at, and what it could not check
this time, with the depth of any judging abcd started itself. The statement
is one shared type with one renderer. What a run looked at is assembled by
the command's code as it runs; what the command is known not to look at is a
short hand-written list beside the command, labelled as known. A gap changes
what a result says and records, and changes no exit status. Every other
command sits on a committed list that a test lets only shrink.

Written by the technical facilitator on 2026-11-21, on auto, from the
intent's 24 acceptance criteria as they stood at sign-off. The criteria are
numbered here in the order the intent lists them.

## Approach

### The statement

A new package, `internal/core/extent`, holds the type every declared command
attaches to its result under one member, `extent`. The name is new because
two results already carry a member called `coverage` and the lifeboat probe
has a coverage type of its own (the intent's decision 8).

- `looked_at`: one entry per kind of thing the run looked at, with its
  identifier, its label and how many it found. A kind in which nothing was
  found is still an entry, with a count of zero.
- `known_not_looked_at`: the command's hand-written list, copied from its
  declaration on every run.
- `could_not_check`: one entry per check that could not be made, with a
  reason and a line of text. The reasons are a closed set of five:
  `unreadable` (a store or file could not be read), `unreachable` (a remote
  or forge did not answer), `not_armed` (the configuration armed no check),
  `tool_missing` (a program the check needs is absent) and `timed_out`. A
  reason outside the set fails to encode.
- `verdict`: `clear`, `clear_except_gaps`, `found` or `none`. It is `none`
  when the run checked nothing, `clear_except_gaps` when nothing was found
  and at least one check could not be made, and `found` whenever something
  was found, whether or not there were gaps.
- `depth`: present only where a model did the judging. It is filled from the
  call record itd-166 shipped (`internal/core/oracle`), never from new
  fields: the class of model, the number of independent runs, the effort
  asked for, and for each of those the existing reason it could not be
  pinned. Where the judging was the host's own agents, `depth` says so with
  that record's `not_reported` reason.

The core returns the statement and writes nothing; each surface renders it.
One renderer in the package produces the lines every command prints:

- `Looked at:` and the labels, on every result.
- `Known not looked at:` and the labels, only when the verdict is `clear`
  or `clear_except_gaps`. The product thinker chose that on 2026-11-21
  (decision 12).
- `Could not check:` and one count per reason, on every result that has a
  gap, naming the flag that lists them.
- The summary line reads as clear except for the gaps when the verdict is
  `clear_except_gaps`, and says nothing was checked when it is `none`.

Detail on request is one flag, `--gaps`, on each declared command. It
prints every gap with its reason and text, and it is declared as changing
nothing that is looked at. The recorded result carries every gap whether or
not the flag is given.

### The declaration

`internal/core/surface` already holds the hand-worded sentence of every
command, with a test that every command has one. The declaration sits beside
it, keyed the same way by command path. For each declared command it holds:

- the identifiers of the kinds the command can look at;
- the kinds it is known not to look at, each with a label, at least one;
- for every flag the command registers, either the kinds the flag adds or
  the statement that it changes nothing looked at.

Three tests hold it, all in `internal/surface/cli` where the command tree is
built.

- Each declared command's core package exports the kinds its code looks at.
  The test fails when that set and the declaration's differ (criterion 2).
- The test walks every flag of every declared command and fails on one the
  declaration does not mention (criterion 20), and on a declared command
  with an empty known-not-looked-at list (criterion 11).
- The test walks the commands help lists. Each is declared or named in
  `internal/surface/cli/testdata/extent_undeclared.txt`. A ceiling in the
  test equals the length of that file, so a new name fails the test unless
  the ceiling is raised in the same change, where a reviewer sees it, and a
  command that gains a declaration must leave the file (criteria 22 and 23). Commands hidden
  from help are outside the walk.

Kind and reason identifiers are constants. A golden file lists them, and a
test fails when one is renamed or removed; additions pass (criterion 21).

### Help and explain

The long help of a declared command ends with a block generated from its
declaration: the kinds it can look at, the kinds it is known not to, and
what each flag adds. `abcd explain` prints the same two lists inside its
line limit. A test asserts that both name the same kinds and that neither
help text contains the word everything (criteria 18 and 19). The brief's
surface chapters and the command pages are reworded in the same step as
each command, since a claim of everything there is the same defect.

### The seven commands

| Command | Core | What a run looked at comes from |
| --- | --- | --- |
| sweep | `internal/core/sweep` | the reads a sweep makes (worktrees and strays, local branches, branches on origin, reservations, inbox reports), recorded as each read is made, so a kind with no row is still named |
| docs lint | the docs lint behind `abcd lint docs` | the checks the configuration armed and the documents read; `nothing_checked` becomes the verdict `none` |
| outbound lint | `internal/surface/cli/lint_outbound.go` and its scanner | the text scanned and the detectors run |
| citation check | `internal/core/source` | the claims checked and the corpus read |
| readiness gate | `internal/core/intent` (`Ready`) | the checks the gate ran |
| launch preview | `internal/core/launch` (`DryRun`) | the gates it ran and the bundle it scanned |
| drive of a drain | `internal/surface/cli/drain_drive.go`, `internal/core/runner`, `internal/core/oracle` | the lanes it ran, with `depth` from each call's record |

Sweep already fails closed: a check it could not make holds every item and
is said once at the top. Those notes stay, and each also becomes a
`could_not_check` entry. Every field a command carries today stays, and
each field the statement now duplicates is added to
`internal/surface/cli/testdata/extent_superseded_fields.txt`, the list a
later, breaking change retires (decision 4).

The launch preview's gate on the documentation review rests on a review the
host's agents ran, so that row carries `depth` as not reported (criterion
17). The drive of a drain starts its models through the runner, so its
result carries the class, the number of runs and the effort asked for
(criterion 16).

### The way to a standard format

For the commands that check something (the two lints, the citation check,
the readiness gate and the launch preview), the statement is serialized
through the one serializer interface `internal/core/repolint` already has,
widened to carry the statement. A SARIF writer stays what that interface
names it: a later implementation. The mapping it would use is recorded here
so the shape cannot drift away from it: what was looked at becomes results
of the kinds pass and not applicable; each gap becomes a notification whose
descriptor is the reason's identifier; the known-not-looked-at list and the
depth go in the run's property bag. Sweep and the drive act, and have no
such format.

### What is not built

No general switch, no new refusal and no change of any exit status. No
SARIF writer and no schema check, which would need a validator the
repository does not have. No choice of depth (itd-173), no reported depth
for the host's agents (itd-2609180517121254) and no statement on what
ships (itd-176). Stashes and stale lock files in sweep's listing are
iss-2611211020368839, which lands by itself and changes what sweep's
declaration lists.

## How the Acceptance Criteria are met

| Criterion | Met by |
| --- | --- |
| 1 a run names every kind it looked at | `looked_at` is filled as each read or check runs; a fixture test per command asserts a zero-count kind is named |
| 2 a changed set of kinds fails a test | the comparison of the declaration with the kinds the core exports |
| 3 a clear result is shown with what was looked at | the renderer's `Looked at:` line, asserted on a clear fixture per command |
| 4 a check that could not be made is listed | each command turns its existing failed-check fields into `could_not_check` entries |
| 5 a gap carries a reason from a closed set and a text | the five reasons; an unknown one fails to encode |
| 6 shown as clear except for the gaps | the renderer's summary line for `clear_except_gaps` |
| 7 recorded as clear except for gaps | the `verdict` member |
| 8 nothing checked gives no verdict | the verdict `none`, and the summary line that says so |
| 9 gaps shown as one count per reason | the renderer's `Could not check:` line |
| 10 every gap listed on request | `--gaps` |
| 11 a declaration lists at least one known gap | the declaration test |
| 12 the list is labelled as known | the renderer's and the help block's label, asserted in both |
| 13 shown on a clear result | the renderer, on `clear` and `clear_except_gaps` |
| 14 left out when findings are listed | the renderer, on `found` |
| 15 recorded on every run | `known_not_looked_at` is always encoded |
| 16 depth where abcd starts the model | `depth` on the drive's result, from the oracle record |
| 17 not told where the host judged | `depth` as not reported on the launch preview's documentation-review row |
| 18 help names the kinds and never everything | the generated help block and its test |
| 19 explain names the same kinds | the explain block and the same test |
| 20 every flag is classified | the declaration test |
| 21 identifiers do not change | the golden file of identifiers |
| 22 every listed command is declared or on the list | the walk over the command tree |
| 23 the list cannot grow unseen | the ceiling in that test |
| 24 the seven carry a declaration | steps 3 to 6 |

## Open Questions

_All resolved at the spec build of 2026-11-21, by the technical facilitator
on auto: the member's name (`extent`), the closed set of reasons, the flag
that lists gaps, and where the declaration lives are settled above. The
mechanism's reading test with both roles is run after the first release
that carries this, and is not a build step._

## Footprint

- packages: internal/core/extent, internal/core/surface, internal/core/sweep, internal/core/source, internal/core/intent, internal/core/launch, internal/core/oracle, internal/core/repolint, internal/surface/cli, commands, .abcd/development/brief/04-surfaces
- tests: internal/core/extent/extent_test.go, internal/core/extent/render_test.go, internal/surface/cli/extent_declaration_test.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/extent_help_test.go, internal/core/sweep/extent_test.go, internal/core/source/extent_test.go, internal/core/intent/extent_test.go, internal/core/launch/extent_test.go, internal/surface/cli/drain_drive_extent_test.go

## Steps

1. The statement and its renderer
   - criteria: 5, 6, 7, 8, 9, 12, 21
   - needs: none
   - packages: internal/core/extent
   - tests: internal/core/extent/extent_test.go: the four verdicts from fixtures of findings and gaps; a reason outside the five fails to encode; the golden file of identifiers refuses a rename and admits an addition. internal/core/extent/render_test.go: the `Looked at:` line on every verdict; `Known not looked at:` on the two clear verdicts and absent on `found`; one count per reason; the summary line for `clear_except_gaps` and for `none`
2. The declaration, the help block and the ratchet
   - criteria: 11, 18, 19, 20, 22, 23
   - needs: 1
   - packages: internal/core/surface, internal/surface/cli, internal/surface/cli/testdata, .abcd/development/brief/04-surfaces/41-explain.md, commands/explain.md
   - tests: internal/surface/cli/extent_declaration_test.go: an empty known-not-looked-at list and an unmentioned flag each fail, on a fixture declaration. internal/surface/cli/extent_ratchet_test.go: a listed command that is neither declared nor in the file fails; a file longer than the ceiling fails; a declared command still in the file fails. internal/surface/cli/extent_help_test.go: help and explain name the same kinds, and a help text holding the word everything fails
3. Sweep
   - criteria: 1, 2, 3, 4, 10, 13, 14, 15, 24
   - needs: 1, 2
   - packages: internal/core/sweep, internal/surface/cli/sweep.go, commands/sweep.md, .abcd/development/brief/04-surfaces/40-sweep.md
   - tests: internal/core/sweep/extent_test.go: on a fixture with no reservation and no inbox report both kinds are still named; a failed process check, an unread reservation store and unread build runs each give one gap with its reason; the exit statuses are those of today on every fixture; `--gaps` lists each gap; the declaration matches the kinds the package exports
4. The two lints and the citation check
   - criteria: 1, 2, 3, 4, 8, 10, 13, 14, 15, 24
   - needs: 1, 2
   - packages: internal/surface/cli/cli.go, internal/surface/cli/lint_outbound.go, internal/surface/cli/source.go, internal/core/source, internal/core/repolint, commands/lint.md, commands/library.md, .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/04-surfaces/07-library.md
   - tests: internal/core/source/extent_test.go: a clear check names the corpus and the claims read; an unreadable corpus is one gap. A docs lint fixture with no rule armed gives the verdict `none` at exit 0; one with a pruned path gives a gap. An outbound lint fixture names the detectors run. Each statement encodes through the widened serializer interface
5. The readiness gate and the launch preview
   - criteria: 1, 2, 3, 4, 10, 13, 14, 15, 17, 24
   - needs: 1, 2
   - packages: internal/core/intent, internal/core/launch, internal/surface/cli/cli.go, commands/intent.md, commands/launch.md, .abcd/development/brief/04-surfaces/05-intent.md, .abcd/development/brief/04-surfaces/04-launch.md
   - tests: internal/core/intent/extent_test.go: a ready intent names every check the gate ran. internal/core/launch/extent_test.go: a preview with a gate it could not measure gives a gap; the documentation-review row carries depth as not reported; exit statuses are those of today
6. The drive of a drain
   - criteria: 1, 2, 3, 4, 10, 13, 14, 15, 16, 24
   - needs: 1, 2
   - packages: internal/surface/cli/drain_drive.go, internal/core/runner, internal/core/oracle, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/surface/cli/drain_drive_extent_test.go: on a fake runner the result names the class, the number of runs and the effort asked for from the call records; an effort the runner could not be handed shows the record's own reason

## Size

Six pull requests. Steps 3 to 6 are independent of each other once 1 and 2
have landed, so they can run as four lanes. Step 4 is the widest, because
the docs lint's result type lives in the command-line surface today and the
statement has to be assembled where the core can return it.
