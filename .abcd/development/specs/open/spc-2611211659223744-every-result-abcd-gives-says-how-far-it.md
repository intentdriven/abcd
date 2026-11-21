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
it can look at, and a run reports on every kind in it. What its code
leaves out by rule is listed apart, as its exclusions. What a command is
known not to look at beyond that is a short hand-written list beside the
command, labelled as known. A gap changes what a result says and records, and
changes no exit status. Every other command sits on a committed list that a
test lets only shrink.

This is the intent's first spec. It delivers the first set and closes with
a remainder, which leaves the intent planned: the intent ships when the
committed list is empty (decision 5, criterion 29).

Written by the technical facilitator on 2026-11-21, on auto, and reworked
the same day after a plan review and again after the final review before
landing; `## Plan review` and `## Final review` list the findings and what
was done about each. The intent's 29 criteria are numbered here in the
order the intent lists them.

## Approach

### The statement

A new package, `internal/core/extent`, holds the type every declared command
attaches to its result under one member, `extent`. The name is new because
several results already carry a member called `coverage` and the lifeboat probe
has a coverage type of its own (decision 8). The package imports no other
core, so every core can import it.

- `looked_at`: one entry per registry kind the run looked at, with its
  identifier, its label and how many it found. A kind in which nothing was
  found is an entry with a count of zero.
- `not_looked_at`: one entry per kind the run did not look at, each with a
  source. `declared` entries are the command's hand-written list, copied on
  every run. `run` entries are taken from the code: a registry kind this
  run left out, or one of the command's exclusions. Each carries one of
  four causes: `not_armed` (the configuration armed no such check),
  `filtered` (the code leaves that class of input out, which is every
  exclusion), `host_run` (the host's agents run it, not this command) and
  `other_run` (it is left to another way of running the command).
- `could_not_check`: one entry per check that could not be made in this
  run, with the check's name, the kind it bears on where there is one, a
  reason and a line of text. The reasons are a closed set of five:
  `unreadable` (a store or file could not be read), `unreachable` (a remote
  or forge did not answer), `tool_missing` (a program the check needs is
  absent), `timed_out`, and `check_failed` (the check ran and returned an
  error of none of those kinds). A reason outside the set fails to encode.
- `verdict`: `clear`, `clear_except_gaps`, `found` or `none`, by the table
  in the intent, in this order: `found` whenever something was found;
  otherwise `none` when the run looked at no registry kind, whether no
  check was armed, there was nothing to read or every check failed;
  otherwise `clear_except_gaps` when at least one check could not be made;
  otherwise `clear`. A command with no clear state (the drive of a drain)
  carries no `verdict` member.
- `depth`: present only where a model did the judging. One entry per
  judging role, and per lane where the run has lanes: the class each
  judging run ran at, the number of judging runs and the effort each was
  asked for. Each of the three is a value with where it came from
  (`run_state` or `receipt`), or is marked `not_told` (the host did the
  judging and no receipt says) or `not_recorded` (abcd started the run and
  its state holds no such value).

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

Detail on request is one flag, `--gaps`, on each declared command a person
types. It prints every gap with its reason and text, and is classified as
changing nothing looked at. The sweep abcd runs on its own has no flag:
its text names `abcd sweep --gaps`. The recorded result carries every gap with or without
the flag.

Recorded means the JSON form and shown means the text (decision 17). A
verdict a command records today keeps its value: the launch report's
`verdict`, the readiness gate's `ready` and the docs lint's
`nothing_checked` say what they say today, and the statement's verdict
stands beside them.

### The registry and the declaration

Each declared command's core package gains one file, `extent.go`, holding:

- the registry: the identifiers and labels of the kinds the command can
  look at, as the list the run itself works through. Where a command builds
  its checks inline today (sweep's five halves, the readiness gate's check
  names, the launch preview's gates), the inline list becomes the registry,
  so the two cannot differ. Where the kinds are a repository's own (the
  banned tokens its lint configuration names, the confidential sources of
  its corpus), the registry is what the code reads from that one place;
- the exclusions: what the code leaves out by rule (the paths the docs lint
  prunes, the classes of finding the outbound lint filters out, the embargo
  store sweep names and never reads), as the value the code itself reads,
  so the two cannot differ;
- the hand-written list of kinds the command is known not to look at.

The command-line surface attaches all three to the command as it builds
the tree, together with the classification of each flag, which is the
surface's to know. A declaration belongs to a command path, or to a named
way of running one (decision 18): the launch preview and the drive of a
drain are a flag on a wider command, and the sweep abcd runs on its own is
typed by no one. A command that looks at nothing declares that and lists
nothing it is known not to look at.

Each flag a declared command registers itself and shows in its help is
classified one of three ways (decision 19): it adds named kinds to what is
looked at, it narrows or redirects what is looked at, or it changes nothing
looked at. Inherited and hidden flags are not classified.

Tests:

- In each core, a run on a fixture reports every registry kind exactly
  once, and every other entry is one of the command's exclusions or a kind
  on its hand-written list (criterion 2).
- In `internal/surface/cli`, a walk of every declared command fails on an
  empty hand-written list where the command looks at anything
  (criterion 13) and on a flag the declaration does not classify
  (criterion 23).
- A walk of every command path help shows fails unless the path is declared
  or named in `internal/surface/cli/testdata/extent_undeclared.txt`, and not
  both. A ceiling in the test equals the length of that file, so a new name
  fails unless the ceiling is raised in the same change, where a reviewer
  sees it (criteria 25 and 26).
- Each field of a declared command's recorded result that restates a member
  of the statement is named in
  `internal/surface/cli/testdata/extent_superseded_fields.txt`, one command
  path, field path and member to a line. Which fields those are is judged
  in review when each command's step lands (decision 4), and each of steps
  3 to 6 names the fields it adds in its pull request. The test proves what
  it can: an entry that names no field of the named command's result type
  fails, so the list cannot go stale (criterion 27).
- Two golden files hold the identifiers that may not change: the reasons
  and causes, in the statement's package, and the kinds and exclusions
  abcd defines, gathered by a walk of the declared commands into
  `internal/surface/cli/testdata/extent_identifiers.txt`. A test fails when
  one is renamed or removed, and additions pass (criterion 24). A kind a
  repository names itself, as a banned token of the docs lint is, is
  outside both.

### Help, explain and the facts

The command facts the binary exports (`internal/core/surface/facts.go`,
built by `CommandFacts` and `surfaceWalk` in
`internal/surface/cli/surface.go`) carry the declaration, and their format
version moves once. Everything derived from the facts follows, and
`internal/surface/cli/commandfacts.go` renders it: the per-command files
under `docs/reference/cli/commands/`, the generated block on every page
under `commands/`, which prints the format version and is therefore
regenerated on every page in step 2, and the answer of `abcd explain`.
Explain's text names the kinds inside its limit of forty lines, and its
JSON carries the whole declaration. The long help of a declared command
ends with a block generated from the same declaration, which labels the
exclusions and the hand-written list as known. A test asserts that help
and explain name the same kinds, that the block carries that label, and
that no help text of a declared command contains the word everything
(criteria 15, 21 and 22). The generated appendix of a surface chapter stays flags and sub-verbs,
as invariant 18 has it; the chapters' prose and the command pages are
reworded in the same step as each command where they claim everything.

### The seven commands

| Command | Core | Registry | Registry kinds a run can leave out | Exclusions |
| --- | --- | --- | --- | --- |
| sweep | `internal/core/sweep` | worktrees, local branches, branches on origin, reservations, waiting reports (the five halves in `sweep.go`) | none | the embargo store, a row kind the code names and never reads (`rows.go`) |
| the sweep abcd runs on its own | `internal/core/sweep` (`Automatic`), `internal/core/implement/loop` (`AutoSweeper`) | the same registry | reservations and waiting reports, left to the sweep a person runs (`other_run`) | the same |
| docs lint | `internal/core/lint` | the rules in `config.go` (`knownRules`), with the banned tokens a repository's configuration names | rules the configuration did not arm (`not_armed`) | the paths it prunes |
| outbound lint | `internal/adapter/scanner`, `internal/surface/cli/lint_outbound.go` | the detectors it runs | none | the classes of finding it filters out |
| citation check | `internal/core/source` (`CiteCheck`) | the confidential sources of the corpus | none | none |
| readiness gate | `internal/core/intent` (`ReadyWithQuestions`) | its nine checks, three of them advisory | none | none |
| launch preview | `internal/core/launch` (`DryRun`) | its gates | gates the repository has not armed (`not_armed`), and the receipts gate, which the host runs (`host_run`) | none |
| drive of a drain | `internal/surface/cli/drain_drive.go`, `internal/core/implement/loop` | the issues it took as lanes, and the judging of each: the drain's judge, the reviewers of each round and a round's judge | none | none |

The launch preview's citation gate reports the status `not_implemented`
when the repository has not armed it. That status string stays as it is in
the report, and the statement gives the cause as `not_armed`: no gate is
unbuilt, so the statement has no cause for one. The readiness gate's verdict
follows its `ready` flag, and an advisory check that does not pass is
reported as it is today.

Sweep already fails closed: a check it could not make holds every item and
is said once at the top. Those notes stay, and each also becomes a
`could_not_check` entry.

The sweep abcd runs on its own has no recorded result today:
`loop.AutoSweeper` returns one line of text, which the loop keeps as `swept`
or joins into its next-step text. Step 3 has it return the statement with
the line. The loop records the statement on the record that holds the line,
and the line gains the renderer's text: the three lists it worked through,
the two it left to the sweep a person runs, and, when it found nothing,
what sweep is known not to look at. No one types it, so it has no help, no
explain answer and no flag: its kinds are named in the help and the explain
answer of `sweep`, as a second way of running it, and where it counts a gap
its text names `abcd sweep --gaps` as the command that lists them.

Depth. The drive of a drain reads what each run's state records as it hands
out a judging run, and not the review set the run was started with
(decision 14). The set is fixed at the start, and the state holds what ran.
For each lane the drive reads every round's validator runs
(`ValidationRound.Validators`: the role, and the class and the effort each
reviewer was handed out at, after any move to another class), the round's
judge where one was started (`ValidationRound.Judge`), and the drain's
judge (`RoleDrainJudge`). The number of runs is the number of judging runs
those records hold for the role. The state records no class or effort for
the drain's judge today: step 6 has the run record both as it hands that
role out, and a run made before that is marked `not_recorded`. The line
gives classes and never a model's name, and does not read the call record
(itd-166). The drive's opening text shows the set a run started then would
use; each run reads the set again, so a result can differ from that text.

The launch preview's receipts gate is judged by the host's agents. It is a
registry kind the preview reports as not looked at, with the cause
`host_run`, and the statement's `depth` carries one entry for it: for each
of the three values, what a receipt on disk carries, marked `receipt`, and
`not_told` for the rest (criterion 20). The preview has no lanes, so the
entry is keyed by the gate alone.

### The way to a standard format

The statement is not written through `repolint.Serializer`. That interface
takes the lint's own result, and its package imports the docs lint's core
and the outbound scanner, and through them reaches the cores of the launch
preview and the readiness gate (decision 13). What this
spec delivers toward a SARIF writer under itd-85 is the stable identifiers
and this mapping, for the commands that check something: a `looked_at`
entry becomes results of the kind pass; a `not_looked_at` entry that is a
registry kind becomes a result of the kind not applicable; a
`could_not_check` entry becomes a notification whose descriptor is the
reason's identifier; the exclusions, the `declared` entries and the depth
go in the run's property bag. Sweep and the drive act, and
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
| 2 every registry kind is reported once, and every other entry is an exclusion or on the hand-written list | the fixture test in each core |
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
| 13 a declaration's hand-written list names a kind, or the command says it looks at nothing | the declaration walk |
| 14 an unarmed or left-out check is named, from the code | `not_looked_at` entries of source `run` |
| 15 the list is labelled as known | the renderer's label (step 1) and the help block's (step 2), asserted in both |
| 16 shown when nothing was found | the renderer, on `clear` and `clear_except_gaps` |
| 17 left out when findings are listed | the renderer, on `found` |
| 18 recorded on every run | `not_looked_at` is always encoded |
| 19 depth for each lane and role | `depth` on the drive's result, from each judging run the run's state records |
| 20 what a receipt carries, not told for the rest | `depth` on the launch preview's receipts gate |
| 21 help names the kinds and never everything | the generated help block and its test |
| 22 explain names the same kinds | the facts carry the declaration; the same test |
| 23 every flag is classified three ways | the declaration walk |
| 24 identifiers do not change | the two golden files of identifiers |
| 25 every command is declared or listed, not both | the walk over the command tree |
| 26 the list cannot grow unseen | the ceiling in that test |
| 27 fields that restate a member are on the retire list | named in review as each command lands; the test fails on an entry that names no field |
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

## Final review

Two reviewers read the planned intent, this spec and the records beside
them before landing, one for consistency between the records and one for
claims against the code. Both said fix first. Their major findings on this
plan, and what was done:

| Finding | Done |
| --- | --- |
| 1 three commands reported kinds outside the registry, which criterion 2 forbade | a command's exclusions are named and kept apart from its registry (decision 3); criterion 2 admits them and the hand-written list and nothing else; the table of commands has a column for each |
| 2 decision 12 was reworded after the sign-off and the record did not say so | the introductions to the decisions and to the criteria now say so, and it is the fourth refinement flagged for the product thinker |
| 3 no record said how a duplicated field is recognised | decision 4 defines a field that restates a member and leaves recognising one to review; criterion 27 and its test claim only what a test can prove |
| 4 the depth line could not be read from the review set: a reviewer can be moved to another class, a fix round runs the reviewers again, and two judging roles are not in the set | decision 14 and criterion 19 read each judging run from the run's state, which replaces what rows 3 and 4 of the plan review settled; the drain's judge gains a recorded class and effort in step 6 |

Smaller findings, each taken up: the order of the four verdicts when every
check failed; the receipts gate as a kind the host runs that also carries a
depth entry; one SARIF mapping in place of two; a test named for every
criterion a step lists; the sweep abcd runs on its own, which has no help,
no flag and no recorded result today; the cause `not_implemented`, dropped
because no gate is unbuilt; the readiness gate's advisory checks; where the
command facts are built and which generated copies a version move
rewrites; how many results use the name coverage; which packages the
serializer's package imports; the launch report as two files; the word gap
kept for a check that could not be made; a registry whose kinds a
repository names; and stale lines in the neighbouring records. This pass
added one finding of its own: steps 3 to 6 share the committed lists and
their test, so they are built one after another and not as parallel lanes.

## Open Questions

_All resolved at the spec build of 2026-11-21, by the technical facilitator
on auto: the member's name (`extent`), the closed sets of reasons and
causes, the flag that lists gaps, where the registry, the exclusions and
the declaration live, what the depth line reads, and which artefacts carry
the declaration are settled above. The
mechanism's reading test with both roles is run at the retrospective the
intent names, and is not a build step._

## Footprint

- packages: internal/core/extent, internal/core/surface, internal/core/sweep, internal/core/lint, internal/core/source, internal/core/intent, internal/core/launch, internal/core/implement/loop, internal/adapter/scanner, internal/surface/cli, commands, docs/reference/cli/commands, .abcd/development/brief/04-surfaces
- tests: internal/core/extent/extent_test.go, internal/core/extent/render_test.go, internal/surface/cli/extent_declaration_test.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/extent_help_test.go, internal/core/sweep/extent_test.go, internal/core/lint/extent_test.go, internal/core/source/extent_test.go, internal/core/intent/extent_test.go, internal/core/launch/extent_test.go, internal/surface/cli/lint_outbound_extent_test.go, internal/surface/cli/drain_drive_extent_test.go

## Steps

1. The statement and its renderer
   - criteria: 5, 6, 7, 8, 9, 10, 12, 15, 24
   - needs: none
   - packages: internal/core/extent
   - tests: internal/core/extent/extent_test.go: the four verdicts in their order from fixtures of findings and gaps, with `none` for a run in which every check failed, and no `verdict` member for a command with no clear state; a reason or a cause outside its set fails to encode; a kind in both lists fails to encode; the golden file of reasons and causes refuses a rename and admits an addition. internal/core/extent/render_test.go: the `Looked at:` line on every verdict; `Known not looked at:` with its label on the two clear verdicts and absent on `found`; one count of checks per reason and no line when there is no gap; the summary line for `clear_except_gaps` and for `none`
2. The declaration, the facts, help and the lists
   - criteria: 13, 15, 21, 22, 23, 24, 25, 26, 27
   - needs: 1
   - packages: internal/core/surface, internal/surface/cli, internal/surface/cli/testdata, docs/reference/cli/commands, commands, .abcd/development/brief/04-surfaces/41-explain.md
   - tests: internal/surface/cli/extent_declaration_test.go: on fixture declarations, an empty hand-written list fails unless the command declares it looks at nothing; a flag with no class fails, and an inherited or hidden flag is not asked for. internal/surface/cli/extent_ratchet_test.go: a command path that is neither declared nor in the file fails; one that is both fails; a file longer than the ceiling fails; a retire-list entry that names no field of a declared command's result fails; the golden file of kinds and exclusions refuses a rename and admits an addition. internal/surface/cli/extent_help_test.go: help and explain name the same kinds, the help block labels the exclusions and the hand-written list as known, explain's text stays inside its line limit, and a help text holding the word everything fails
3. Sweep, and the sweep abcd runs on its own
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/sweep, internal/core/implement/loop, internal/surface/cli/sweep.go, internal/surface/cli/drain_drive.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, commands/sweep.md, .abcd/development/brief/04-surfaces/40-sweep.md
   - tests: internal/core/sweep/extent_test.go: on a fixture with no reservation and no waiting report both kinds are still named; a clear run shows `Looked at:` and `Known not looked at:`, and a run that lists something leaves the second out and still records it; a failed process check, an unread reservation store and unread build runs each give one gap with its reason, beside the kind they bear on; the embargo store is named as an exclusion; the sweep abcd runs on its own names three lists as looked at and two as left to the other sweep, records its statement, and names `abcd sweep --gaps` where it counts a gap; the exit statuses are those of today on every fixture; `--gaps` lists each gap; the run reports the registry exactly once
4. The two lints and the citation check
   - criteria: 1, 2, 3, 4, 8, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/lint, internal/adapter/scanner, internal/core/source, internal/surface/cli/cli.go, internal/surface/cli/lint_outbound.go, internal/surface/cli/source.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, commands/lint.md, commands/library.md, .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/04-surfaces/07-library.md
   - tests: internal/core/lint/extent_test.go: a docs lint fixture with no rule armed gives the verdict `none` at exit 0 with `nothing_checked` unchanged; an unarmed rule is named as not looked at with the cause `not_armed` and a pruned path as an exclusion; a repository's own banned token keeps its name; a clear run shows both lines, and a run with a finding leaves the known list out and still records it; `--gaps` lists a check that failed; the run reports the registry exactly once. internal/core/source/extent_test.go: a clear check names the confidential sources read and shows both lines; a corpus with none gives the verdict `none` at exit 0; an unreadable corpus is one gap and the verdict `none`. internal/surface/cli/lint_outbound_extent_test.go: a clear scan names the detectors run, shows both lines and names the classes it filters as exclusions; a text it cannot scan is still refused
5. The readiness gate and the launch preview
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 20, 28
   - needs: 1, 2
   - packages: internal/core/intent, internal/core/launch, internal/surface/cli/cli.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, commands/intent.md, commands/launch.md, .abcd/development/brief/04-surfaces/05-intent.md, .abcd/development/brief/04-surfaces/04-launch.md
   - tests: internal/core/intent/extent_test.go: a ready intent names each of the nine checks, shows both lines, and `ready` is unchanged; an advisory check that does not pass changes neither `ready` nor the verdict; an intent that is not ready leaves the known list out and still records it. internal/core/launch/extent_test.go: a gate the repository has not armed is named as not looked at with the cause `not_armed`, the citation gate among them with its status string unchanged, and the receipts gate with the cause `host_run`; a gate that could not be measured is one gap, and `--gaps` lists it; the receipts gate's depth entry carries what a receipt on disk holds, marked `receipt`, and `not_told` for the rest; a preview whose every gate passes shows both lines; the report's `verdict` and the exit statuses are those of today; the run reports the registry exactly once
6. The drive of a drain
   - criteria: 1, 2, 4, 11, 18, 19, 28
   - needs: 1, 2
   - packages: internal/surface/cli/drain_drive.go, internal/core/implement/loop, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/surface/cli/drain_drive_extent_test.go: on a fake runner the result names each issue taken as a lane, a lane that changed nothing included, and reports the registry exactly once; for each lane and each judging role it gives the class each judging run ran at, the number of judging runs and the effort each was asked for, as the run's state records them; a reviewer moved to another class is named at the class it ran at; a lane with a fix round counts the runs of both rounds; a round's judge and the drain's judge are named as judging roles, and a state with no class for the drain's judge is marked `not_recorded`; a judging run that could not be started is one gap with its reason, and `--gaps` lists it; the statement carries no `verdict` member and records the hand-written list on every run; the drive's log follows the rules for what is shown

## Size

Six pull requests, built one after another. Steps 3 to 6 need only steps 1
and 2 and can be taken in any order, but they are not parallel lanes: each
takes its commands off the undeclared list and lowers the ceiling in the
same test file, and each adds its lines to the golden file of kinds and to
the retire list. Steps 3 and 6 also share
`internal/surface/cli/drain_drive.go` and `internal/core/implement/loop`,
and steps 4 and 5 share `internal/surface/cli/cli.go`. Step 4 is the
widest, because the docs lint's result type lives in the command-line
surface today and the statement has to be assembled where the core can
return it.
