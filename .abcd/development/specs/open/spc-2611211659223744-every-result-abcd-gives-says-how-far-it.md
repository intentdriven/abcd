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
at, what it did not look at, and what it could not check this time, with
the depth of any judging abcd started itself. The statement is one shared
type with one renderer. Each command keeps a registry of the kinds of thing
it can look at, and a run reports on every kind in it. What a command is
known not to look at is a short hand-written list beside the command,
labelled as known. A gap changes what a result says and records, and
changes no exit status. Every other command sits on a committed list that a
test lets only shrink.

This is the intent's first spec. It delivers the first set and closes with
a remainder, which leaves the intent planned: the intent ships when the
committed list is empty (decision 5, criterion 29).

Written by the technical facilitator on 2026-11-21, on auto, and reworked
the same day after a plan review; `## Plan review` lists its findings and
what was done about each. The intent's 29 criteria are numbered here in the
order the intent lists them.

## Approach

### The statement

A new package, `internal/core/extent`, holds the type every declared command
attaches to its result under one member, `extent`. The name is new because
two results already carry a member called `coverage` and the lifeboat probe
has a coverage type of its own (decision 8). The package imports no other
core, so every core can import it.

- `looked_at`: one entry per registry kind the run looked at, with its
  identifier, its label and how many it found. A kind in which nothing was
  found is an entry with a count of zero.
- `not_looked_at`: one entry per kind the run did not look at, each with a
  source. `declared` entries are the command's hand-written list, copied on
  every run. `run` entries are taken from the code and carry one of five
  causes: `not_armed` (the configuration armed no such check),
  `not_implemented` (the check is not built yet), `filtered` (the code
  leaves that class of input out), `host_run` (the host's agents run it, not
  this command) and `other_run` (it is left to another way of running the
  command).
- `could_not_check`: one entry per check that could not be made in this
  run, with the check's name, the kind it bears on where there is one, a
  reason and a line of text. The reasons are a closed set of five:
  `unreadable` (a store or file could not be read), `unreachable` (a remote
  or forge did not answer), `tool_missing` (a program the check needs is
  absent), `timed_out`, and `check_failed` (the check ran and returned an
  error of none of those kinds). A reason outside the set fails to encode.
- `verdict`: `clear`, `clear_except_gaps`, `found` or `none`, by the table
  in the intent. It is `none` when the run checked nothing,
  `clear_except_gaps` when nothing was found and at least one check could
  not be made, and `found` whenever something was found. A command with no
  clear state (the drive of a drain) leaves it out.
- `depth`: present only where a model did the judging. One entry per lane
  and judging role, with the class of each reviewer, the number of runs and
  the effort asked for. Each of the three is a value with where it came
  from (`review_set` or `receipt`), or is marked not told.

A kind appears once: in `looked_at`, in `not_looked_at`, or as the kind of
a `could_not_check` entry whose check stopped the kind being read at all. A
check that failed on a kind the run did read (sweep's check of which
sessions are running) is a gap beside a `looked_at` entry. The encoder
refuses a statement with a kind in both `looked_at` and `not_looked_at`
(criterion 12).

The core returns the statement and writes nothing; each surface renders it.
One renderer in the package produces the lines every command prints:

- `Looked at:` and the labels, on every result.
- `Known not looked at:` and the labels of both sources, only when the
  verdict is `clear` or `clear_except_gaps` (decision 12).
- `Could not check:` and one count of checks per reason, naming the flag
  that lists them. No such line when there is no gap.
- The summary line reads as clear except for the gaps when the verdict is
  `clear_except_gaps`, and says nothing was checked when it is `none`.

Detail on request is one flag, `--gaps`, on each declared command. It
prints every gap with its reason and text, and is classified as changing
nothing looked at. The recorded result carries every gap with or without
the flag.

Recorded means the JSON form and shown means the text (decision 17). A
verdict a command records today keeps its value: the launch report's
`verdict`, the readiness gate's `ready` and the docs lint's
`nothing_checked` say what they say today, and the statement's verdict
stands beside them.

### The registry and the declaration

Each declared command's core package gains one file, `extent.go`, holding:

- the registry: the identifiers and labels of the kinds the command can
  look at, as constants the run itself works through. Where a command
  builds its checks inline today (sweep's five halves, the readiness gate's
  check names, the launch preview's gates), the inline list becomes the
  registry, so the two cannot differ;
- the hand-written list of kinds the command is known not to look at.

The command-line surface attaches both to the command as it builds the
tree, together with the classification of each flag, which is the surface's
to know. A declaration belongs to a command path, or to a named way of
running one (decision 18): the launch preview and the drive of a drain are
a flag on a wider command, and the sweep abcd runs on its own is typed by no
one. A command that looks at nothing declares that and lists no gap.

Each flag a declared command registers itself and shows in its help is
classified one of three ways (decision 19): it adds named kinds to what is
looked at, it narrows or redirects what is looked at, or it changes nothing
looked at. Inherited and hidden flags are not classified.

Tests:

- In each core, a run on a fixture reports every registry kind exactly
  once, and no other (criterion 2).
- In `internal/surface/cli`, a walk of every declared command fails on an
  empty known-not-looked-at list where the command looks at anything
  (criterion 13) and on a flag the declaration does not classify
  (criterion 23).
- A walk of every command path help shows fails unless the path is declared
  or named in `internal/surface/cli/testdata/extent_undeclared.txt`, and not
  both. A ceiling in the test equals the length of that file, so a new name
  fails unless the ceiling is raised in the same change, where a reviewer
  sees it (criteria 25 and 26).
- Each field the statement duplicates on a declared command is named in
  `internal/surface/cli/testdata/extent_superseded_fields.txt`; the test
  reads the declared commands' result types and fails on a duplicate the
  file does not name (criterion 27).
- A golden file lists the identifiers of kinds abcd defines and of the
  reasons and causes; a test fails when one is renamed or removed, and
  additions pass (criterion 24). A kind a repository names itself, as a
  banned token of the docs lint is, is outside the file.

### Help, explain and the facts

The command facts the binary exports (`internal/core/surface/facts.go`,
built in `internal/surface/cli/commandfacts.go`) carry the declaration, and
their format version moves once. Everything derived from the facts follows:
the per-command files under `docs/reference/cli/commands/` and the answer
of `abcd explain`. Explain's text names the kinds inside its limit of forty
lines, and its JSON carries the whole declaration. The long help of a
declared command ends with a block generated from the same declaration. A
test asserts that help and explain name the same kinds and that no help
text of a declared command contains the word everything (criteria 21 and
22). The generated appendix of a surface chapter stays flags and sub-verbs,
as invariant 18 has it; the chapters' prose and the command pages are
reworded in the same step as each command where they claim everything.

### The seven commands

| Command | Core | Registry | Not looked at in this run, from the code |
| --- | --- | --- | --- |
| sweep | `internal/core/sweep` | worktrees, local branches, branches on origin, reservations, waiting reports (the five halves in `sweep.go`) | the embargo store, which it never reads |
| the sweep abcd runs on its own | `internal/core/sweep` (`Automatic`) | the same registry | reservations and waiting reports, left to the sweep a person runs |
| docs lint | `internal/core/lint` | the rules in `config.go` | rules the configuration did not arm; paths it prunes |
| outbound lint | `internal/adapter/scanner`, `internal/surface/cli/lint_outbound.go` | the detectors it runs | the classes of finding it filters out |
| citation check | `internal/core/source` (`CiteCheck`) | the confidential sources of the corpus | none |
| readiness gate | `internal/core/intent` (`Ready`) | its checks | none |
| launch preview | `internal/core/launch` (`DryRun`) | its gates | gates not armed, gates not implemented, and the receipts gate, which the host runs |
| drive of a drain | `internal/surface/cli/drain_drive.go`, `internal/core/implement/loop` | the issues it took as lanes and the review of each lane | none |

Sweep already fails closed: a check it could not make holds every item and
is said once at the top. Those notes stay, and each also becomes a
`could_not_check` entry. The sweep abcd runs on its own records the
statement in its result, and its one line names the three lists it worked
through.

Depth. The drive of a drain reads the review set each run was started with
(`loop.ReviewSet`): for each judging role, the class of each reviewer, how
many reviewers the set lists, which is the number of runs, and the effort
asked for. That is what the drive shows the person before it starts, and
nothing else is read (decision 14). The launch preview rests on judging the
host's agents did: for each of the three it names what a receipt on disk
carries and marks the rest not told (criterion 20).

### The way to a standard format

The statement is not written through `repolint.Serializer`. That interface
takes the lint's own result, and its package imports the cores of the docs
lint, the launch preview and the outbound scanner (decision 13). What this
spec delivers toward a SARIF writer under itd-85 is the stable identifiers
and this mapping, for the commands that check something: a `looked_at`
entry becomes results of the kind pass; a `not_looked_at` entry becomes a
result of the kind not applicable; a `could_not_check` entry becomes a
notification whose descriptor is the reason's identifier; the declared list
and the depth go in the run's property bag. Sweep and the drive act, and
have no such format.

### What is not built

No general switch, no new refusal and no change of any exit status. No
SARIF writer and no schema check, which would need a validator the
repository does not have. No choice of depth (itd-173), no reported depth
for the host's agents beyond what a receipt carries (itd-2609180517121254)
and no statement on what ships (itd-176). No declaration for a command
outside the first set, bare `drain` included. Stashes and stale lock files
in sweep's listing are iss-2611211020368839, which lands by itself: landing
before step 3 it adds two kinds to the registry step 3 builds, and landing
after it moves them there from the hand-written list.

## How the Acceptance Criteria are met

| Criterion | Met by |
| --- | --- |
| 1 a run names every registry kind it looked at | `looked_at` is filled as the run works through the registry; a fixture test per command asserts a zero-count kind is named |
| 2 every registry kind is reported, and no other | the fixture test in each core |
| 3 a clear result is shown with what was looked at | the renderer's `Looked at:` line, asserted on a clear fixture per command |
| 4 a check that could not be made is listed | each command turns its existing failed-check fields into `could_not_check` entries |
| 5 a gap carries a reason from a closed set and a text | the five reasons; an unknown one fails to encode |
| 6 shown as clear except for the gaps | the renderer's summary line for `clear_except_gaps` |
| 7 recorded as clear except for gaps | the `verdict` member |
| 8 nothing checked gives no verdict | the verdict `none`, and the summary line that says so |
| 9 one count of checks per reason | the renderer's `Could not check:` line |
| 10 no line about gaps when there is none | the renderer, asserted on a fixture with no gap |
| 11 every gap listed on request | `--gaps` |
| 12 a kind is not both looked at and not looked at | the encoder refuses it |
| 13 a declaration lists a known gap, or says it looks at nothing | the declaration walk |
| 14 an unarmed or left-out check is named, from the code | `not_looked_at` entries of source `run` |
| 15 the list is labelled as known | the renderer's and the help block's label, asserted in both |
| 16 shown when nothing was found | the renderer, on `clear` and `clear_except_gaps` |
| 17 left out when findings are listed | the renderer, on `found` |
| 18 recorded on every run | `not_looked_at` is always encoded |
| 19 depth for each lane and role | `depth` on the drive's result, from the review set |
| 20 what a receipt carries, not told for the rest | `depth` on the launch preview's receipts gate |
| 21 help names the kinds and never everything | the generated help block and its test |
| 22 explain names the same kinds | the facts carry the declaration; the same test |
| 23 every flag is classified three ways | the declaration walk |
| 24 identifiers do not change | the golden file of identifiers |
| 25 every command is declared or listed, not both | the walk over the command tree |
| 26 the list cannot grow unseen | the ceiling in that test |
| 27 duplicated fields are on the retire list | the test over `extent_superseded_fields.txt` |
| 28 the first set carries a declaration | steps 3 to 6 |
| 29 the list is empty when the intent ships | not this spec: it closes with a remainder, and later specs empty the list |

## Plan review

A plan review reported after the intent was planned and this spec's first
draft was written. Its eighteen findings, and what was done:

| Finding | Done |
| --- | --- |
| 1 the result states were not closed, and clear was undefined for four commands | the intent's table of clear, findings and nothing checked per command; criteria 6 and 7 gained "no finding"; an unarmed check moved from could not check to not looked at in this run; the reasons are named above |
| 2 criterion 2 compared the statement with itself | the registry, and the fixture test against it; criterion 24 limited to kinds abcd defines |
| 3 the depth line could not be read from the call record, and the drive did not fit the clear-result lines | depth read from the review set, per lane and role; the drive has no clear state (decision 16); a declaration belongs to a path or a named way of running one (decision 18) |
| 4 reading the call record would surface what itd-166 keeps with the technical facilitator | the depth line reads only the review set, already shown before a drive starts (decision 14) |
| 5 the serializer interface could not carry the statement | not written through it; identifiers and a written mapping instead (decision 13) |
| 6 an existing verdict field against the additive rule, and shown against recorded | existing fields keep their value (decision 4); recorded and shown defined (decision 17) |
| 7 the promise says every result and the first release carries seven | the intent ships when the list is empty (decision 5, criterion 29); this spec closes with a remainder |
| 8 the sweep abcd runs on its own fell outside | it is in the first set with sweep |
| 9 two flag classes missed flags that narrow or redirect | a third class (decision 19) |
| 10 the mechanism could not be counted | it counts issues that report acting on a clear result and name the intent, and names the retrospective that reads the count |
| 11 the reason named for host-judged work was the wrong one | criterion 20 names what a receipt carries and marks the rest not told |
| 12 exclusions the code already states were treated as hand-written | they are `run` entries of `not_looked_at` (decision 3); criterion 12 keeps the two lists apart |
| 13 sweep's help would be reworded twice | the issue's remedy now only removes the word everything; naming the kinds is this spec's |
| 14 invariant 18 is touched, not an analogy | the facts carry the declaration; the chapter appendix stays as it is (decision 18) |
| 15 overlap with the draft on a host's payload | the depth member is defined here once and that draft fills it (decision 14) |
| 16 no means of asking for the gaps, no unit for the count | `--gaps`; a count of checks |
| 17 three decisions claimed the record settled what it did not | decisions 2, 4 and 5 now name the other answer and whose choice it was |
| 18 small points | the principle cited for its spirit (decision 8); the lint command called by its name; declared and listed is refused; no gap line when there is none |

Not taken up: the review asked whether a control that checked nothing
should fail. That would change an exit status, which the product thinker
let stand as unchanged, so decision 2 records it as the answer not chosen.

## Open Questions

_All resolved at the spec build of 2026-11-21, by the technical facilitator
on auto: the member's name (`extent`), the closed sets of reasons and
causes, the flag that lists gaps, where the registry and the declaration
live, and which artefacts carry the declaration are settled above. The
mechanism's reading test with both roles is run at the retrospective the
intent names, and is not a build step._

## Footprint

- packages: internal/core/extent, internal/core/surface, internal/core/sweep, internal/core/lint, internal/core/source, internal/core/intent, internal/core/launch, internal/core/implement/loop, internal/adapter/scanner, internal/surface/cli, commands, docs/reference/cli/commands, .abcd/development/brief/04-surfaces
- tests: internal/core/extent/extent_test.go, internal/core/extent/render_test.go, internal/surface/cli/extent_declaration_test.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/extent_help_test.go, internal/core/sweep/extent_test.go, internal/core/lint/extent_test.go, internal/core/source/extent_test.go, internal/core/intent/extent_test.go, internal/core/launch/extent_test.go, internal/surface/cli/drain_drive_extent_test.go

## Steps

1. The statement and its renderer
   - criteria: 5, 6, 7, 8, 9, 10, 12, 15, 24
   - needs: none
   - packages: internal/core/extent
   - tests: internal/core/extent/extent_test.go: the four verdicts from fixtures of findings and gaps, and none for a command with no clear state; a reason or a cause outside its set fails to encode; a kind in both lists fails to encode; the golden file of identifiers refuses a rename and admits an addition. internal/core/extent/render_test.go: the `Looked at:` line on every verdict; `Known not looked at:` on the two clear verdicts and absent on `found`; one count of checks per reason and no line when there is no gap; the summary line for `clear_except_gaps` and for `none`
2. The declaration, the facts, help and the two lists
   - criteria: 13, 21, 22, 23, 25, 26, 27
   - needs: 1
   - packages: internal/core/surface, internal/surface/cli, internal/surface/cli/testdata, docs/reference/cli/commands, .abcd/development/brief/04-surfaces/41-explain.md, commands/explain.md
   - tests: internal/surface/cli/extent_declaration_test.go: on fixture declarations, an empty known-not-looked-at list fails unless the command declares it looks at nothing; a flag with no class fails, and an inherited or hidden flag is not asked for. internal/surface/cli/extent_ratchet_test.go: a command path that is neither declared nor in the file fails; one that is both fails; a file longer than the ceiling fails; a duplicated field missing from the retire list fails. internal/surface/cli/extent_help_test.go: help and explain name the same kinds, explain's text stays inside its line limit, and a help text holding the word everything fails
3. Sweep, and the sweep abcd runs on its own
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/sweep, internal/surface/cli/sweep.go, internal/surface/cli/drain_drive.go, internal/core/implement/loop/terminal.go, commands/sweep.md, .abcd/development/brief/04-surfaces/40-sweep.md
   - tests: internal/core/sweep/extent_test.go: on a fixture with no reservation and no waiting report both kinds are still named; a failed process check, an unread reservation store and unread build runs each give one gap with its reason, beside the kind they bear on; the embargo store is named as not looked at; the sweep abcd runs on its own names three lists as looked at and two as left to the other sweep; the exit statuses are those of today on every fixture; `--gaps` lists each gap; the run reports the registry exactly
4. The two lints and the citation check
   - criteria: 1, 2, 3, 4, 8, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/lint, internal/adapter/scanner, internal/core/source, internal/surface/cli/cli.go, internal/surface/cli/lint_outbound.go, internal/surface/cli/source.go, commands/lint.md, commands/library.md, .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/04-surfaces/07-library.md
   - tests: internal/core/lint/extent_test.go: a docs lint fixture with no rule armed gives the verdict `none` at exit 0 with `nothing_checked` unchanged; an unarmed rule and a pruned path are named as not looked at; a repository's own banned token keeps its name. internal/core/source/extent_test.go: a clear check names the confidential sources read; a corpus with none gives the verdict `none` at exit 0; an unreadable corpus is one gap. An outbound lint fixture names the detectors run and the classes it filters, and still refuses a text it cannot scan
5. The readiness gate and the launch preview
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 20, 28
   - needs: 1, 2
   - packages: internal/core/intent, internal/core/launch, internal/surface/cli/cli.go, commands/intent.md, commands/launch.md, .abcd/development/brief/04-surfaces/05-intent.md, .abcd/development/brief/04-surfaces/04-launch.md
   - tests: internal/core/intent/extent_test.go: a ready intent names every check the gate ran, and `ready` is unchanged. internal/core/launch/extent_test.go: a gate not armed, a gate not implemented and the receipts gate are each named as not looked at with their cause; a gate that could not be measured is one gap; the receipts gate carries depth from a receipt on disk and not told for the rest; the report's `verdict` and the exit statuses are those of today
6. The drive of a drain
   - criteria: 1, 2, 4, 11, 14, 18, 19, 28
   - needs: 1, 2
   - packages: internal/surface/cli/drain_drive.go, internal/core/implement/loop, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/surface/cli/drain_drive_extent_test.go: on a fake runner the result names, for each lane and each judging role, the class of each reviewer, the number of runs and the effort asked for, as the review set of that run gives them; a run whose review set changed between lanes shows each lane's own; the statement carries no verdict; the drive's log follows the rules for what is shown

## Size

Six pull requests. Steps 3 to 6 are independent of each other once 1 and 2
have landed, so they can run as four lanes; steps 3 and 6 both edit
`internal/surface/cli/drain_drive.go` and take turns. Step 4 is the widest,
because the docs lint's result type lives in the command-line surface today
and the statement has to be assembled where the core can return it.
