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
  source. `left_out` entries are registry kinds this run did not look at,
  taken from the code, and each carries one of four causes: `not_armed`
  (the configuration armed no such check), `nothing_to_read` (the check
  was armed and the run had nothing to read it against), `host_run` (the
  host's agents run it, not this command) and `other_run` (it is left to
  another way of running the command). `exclusion` entries are the
  command's exclusions, taken from the code. `listed` entries are the
  command's hand-written list. The last two stand in every run, and
  together they are what the command is known not to look at.
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
  check was armed, there was nothing to read or no kind could be read;
  otherwise `clear_except_gaps` when at least one check could not be made;
  otherwise `clear`. A command with no clear state (the drive of a drain)
  carries no `verdict` member.
- `depth`: present only where a model did the judging. One entry per
  judging role, keyed by lane where the judging is of a lane's change and
  by issue for the drain's judge (the launch preview's one role is the
  host's judging behind its receipts gate): the class each judging run ran
  at, the number of judging runs and the effort each was asked for. Each of
  the three is a value with where it came from (`run_state` or `receipt`),
  or is marked `not_told` (the host did the judging and no receipt says) or
  `none_asked` (abcd started the run and asked for none, so the runner's
  default applied).

A kind appears once: in `looked_at`, in `not_looked_at`, or as the kind of
a `could_not_check` entry whose check stopped the kind being read at all. A
check that failed on a kind the run did read (sweep's check of which
sessions are running) is a gap beside a `looked_at` entry. The encoder
refuses a statement with a kind in both `looked_at` and `not_looked_at`
(criterion 12).

The core returns the statement and writes nothing; each surface renders it.
One renderer in the package produces the lines every command prints:

- `Looked at:` and the labels, on every result; it says nothing where the
  run looked at no kind.
- `Left out in this run:` and the labels of the `left_out` entries, each
  with its cause, on every result that has one.
- `Known not looked at:` and the labels of the `exclusion` and `listed`
  entries, only when the verdict is `clear` or `clear_except_gaps`
  (decision 12), and so on neither `found` nor `none`.
- `Could not check:` and one count of checks per reason, naming the flag
  that lists them. No such line when there is no gap.
- `Judged by:` and one line for each depth entry: the role, the lane where
  there is one, the classes, the number of judging runs and the effort,
  with not told and none asked said in words.
- The summary line reads as clear except for the gaps when the verdict is
  `clear_except_gaps`, and says nothing was checked when it is `none`.

Detail on request is one flag, `--gaps`, on each declared command a person
types. It prints every gap with its reason and text, and is classified as
changing nothing looked at. Where a declared way is a flag on a wider
command, `--gaps` is accepted beside that flag. The sweep abcd runs on its
own has no flag, so its text lists every gap in full. The recorded result
carries every gap with or without the flag.

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

The unit of every walk below is a command path or a named way of running
one. Where a declared way is a flag on a wider command, the help block sits
in that command's long help under the flag's name, the flags classified are
the ones that apply with it, and the command's other ways, where it has
any, stay on the undeclared list under the path's own name: `drain` stays
listed while its drive is declared. A registry a repository names (banned
tokens, the confidential sources of a corpus) is named in help, explain and
the generated files by where it is read from.

Each flag a declared command registers itself and shows in its help is
classified one of three ways (decision 19): it adds named kinds to what is
looked at, it narrows or redirects what is looked at, or it changes nothing
looked at. Inherited and hidden flags are not classified.

Tests:

- For each of the seven commands, a run on a fixture reports every registry
  kind exactly once, and every other entry is one of the command's
  exclusions or a kind on its hand-written list (criterion 2).
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
version moves once. Everything derived from the facts follows: the
per-command files under `docs/reference/cli/commands/` and the generated
block on every page under `commands/` (both rendered in
`internal/surface/cli/commandfacts.go`; the block prints the format
version and is therefore regenerated on every page in step 2), the answer
of `abcd explain` (rendered in `internal/surface/cli/explain.go`), and the
surface files generated from the command tree
(`.abcd/development/release/surface.json` and
`docs/reference/cli/commands.md`), which change again whenever a later
step adds the `--gaps` flag to a command.
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
| docs lint | `internal/core/lint` | the rules in `config.go` (`knownRules`), with the banned tokens a repository's configuration names | rules the configuration did not arm (`not_armed`); every armed rule when there is no document (`nothing_to_read`) | the paths it prunes, exempt paths and statuses, each rule's exempt globs, and fenced code |
| outbound lint | `internal/adapter/scanner`, `internal/surface/cli/lint_outbound.go` | the detectors it runs | none | the classes of finding it filters out |
| citation check | `internal/core/source` (`CiteCheck`) | the confidential sources of the corpus | none | a source's authors, unless its entry opts in (`ban_authors`) |
| readiness gate | `internal/core/intent` (`ReadyWithQuestions`) | its nine checks, some of them advisory | none | none |
| launch preview | `internal/core/launch` (`DryRun`) | its gates | gates the repository has not armed (`not_armed`); the deep-smoke and parity gates unless the run asks for them (`other_run`); the receipts gate, which the host runs (`host_run`, where the report spells its status `host-run`) | none |
| drive of a drain | `internal/surface/cli/drain_drive.go`, `internal/core/implement/loop` | two kinds fixed in code: the open issues it read to choose its lanes, and the change each lane made | none | none |

The launch preview's citation gate reports the status `not_implemented`
when the repository has not armed it. That status string stays as it is in
the report, and the statement gives the cause as `not_armed`: no gate is
unbuilt, so the statement has no cause for one. The readiness gate's verdict
follows its `ready` flag, and an advisory check that does not pass is
reported as it is today.

Two commands can look at no registry kind. A corpus with no confidential
source gives the citation check an empty registry. A docs lint with rules
armed and no document reports every armed rule as left out, with the cause
`nothing_to_read`. Both give the verdict `none`, which is what the ruling
on the docs lint asks (dcn-2609230000000020, item G2) and what
`nothing_checked` says today.

The drive's two kinds carry counts: how many open issues it read and how
many it took as lanes, and how many lanes made a change. What it did with
an issue it did not take is in its result as today, as a hand-back, and
the statement does not repeat it. How each lane was judged is the depth
entry, below.

Sweep already fails closed: a check it could not make holds every item and
is said once at the top. Those notes stay, and each also becomes a
`could_not_check` entry.

The sweep abcd runs on its own has no recorded result today:
`loop.AutoSweeper` returns one line of text, which the loop keeps as
`swept`, joins into its next-step text or adds to the diagnostics of a
resume. Step 3 has it return the statement with
the line. The loop records the statement on the record that holds the line,
and the line gains the renderer's text: the three lists it worked through,
the two it left to the sweep a person runs, and, when it found nothing,
what sweep is known not to look at. No one types it, so it has no help, no
explain answer and no flag: its kinds are named in the help and the explain
answer of `sweep`, as a second way of running it, and since no one can pass
it a flag its text lists every gap in full.

Depth. The drive of a drain reads what its state records as it hands out a
judging run, and not the review set a run was started with (decision 14).
The set is fixed at the start, and the state holds what ran. For each lane
the drive reads every round's validator records
(`ValidationRound.Validators`: the role, and the class and the effort each
reviewer was handed out at, after any move to another class) and the
round's judge where one was started (`ValidationRound.Judge`). The number
of runs counts those records: a reviewer handed out again within a round
reuses its record and is counted once, and a judge marked as not launched
is a `could_not_check` entry and not a run. A record with no class or
effort, as in a run opened with no review set, is marked `none_asked`.

The drain's judge (`RoleDrainJudge`) is recorded in the drain's own state
and not in a lane's, because it judges an issue's remedy before any lane
exists (`DrainState.Judgements`). Its entries are keyed by issue. A
judgement with a route recorded is one judging run of the runner that
answered; abcd asks that runner for no class and no effort, so both are
marked `none_asked`. A judgement with no route recorded is the host's
answer and is marked `not_told`. A runner that did not give an answer the
drain could take is kept as a fallback receipt
(`DrainState.JudgeFallbacks`, the latest 32, which name the role and the
reason and not the issue): each is a `could_not_check` entry and not a
run. No field is added to any state. The line gives classes and never a model's name, and
does not read the call record (itd-166). The drive's opening text shows the
set a run started then would use; each run reads the set again, so a result
can differ from that text.

The launch preview's receipts gate is judged by the host's agents. It is a
registry kind the preview reports as not looked at, with the cause
`host_run`, and the statement's `depth` carries one entry for it: for each
of the three values, what a receipt on disk carries, marked `receipt`, and
`not_told` for the rest (criterion 20). The preview has no lanes, so the
entry is keyed by its one judging role, the host's judging behind that
gate.

### The way to a standard format

The statement is not written through `repolint.Serializer`. That interface
takes the lint's own result, and its package imports the docs lint's core,
the outbound scanner and the site's core, and through them reaches the
cores of the launch preview and the readiness gate (decision 13). What this
spec delivers toward a SARIF writer under itd-85 is the stable identifiers
and this mapping, for the commands that check something: a `looked_at`
entry becomes results of the kind pass; a `left_out` entry becomes a
result of the kind not applicable; a `could_not_check` entry becomes a
notification whose descriptor is the reason's identifier; the `exclusion`
and `listed` entries and the depth go in the run's property bag. Sweep and the drive act, and
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
| 2 every registry kind is reported once, and every other entry is an exclusion or on the hand-written list | a fixture test for each of the seven commands |
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
| 14 a registry kind a run left out is recorded and shown with its cause | `not_looked_at` entries of source `left_out`, and the renderer's `Left out in this run:` line |
| 15 the list is labelled as known | the renderer's label (step 1) and the help block's (step 2), asserted in both |
| 16 shown when nothing was found | the renderer, on `clear` and `clear_except_gaps` |
| 17 left out when findings are listed | the renderer, on `found` |
| 18 recorded on every run | `not_looked_at` is always encoded |
| 19 depth for each judging role, by lane or by issue | `depth` on the drive's result, from the records the drive's state holds, and the renderer's `Judged by:` line |
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
| 12 exclusions the code already states were treated as hand-written | they are entries of `not_looked_at` taken from the code (decision 3) |
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
| 4 the depth line could not be read from the review set: a reviewer can be moved to another class, a fix round runs the reviewers again, and the drain's judge is not in the set | decision 14 and criterion 19 read each judging run from the drive's state, which replaces what rows 3 and 4 of the plan review settled |

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

A second pair read that fix pass and again said fix first. Their major
findings on this plan, and what was done:

| Finding | Done |
| --- | --- |
| 5 the verdict rule left a docs lint with rules armed and no document with no state it could report | a fourth cause, `nothing_to_read`: every armed rule is then left out with it and the verdict is `none`, as the ruling on that lint asks; step 4 tests the case |
| 6 the drain's judge is launched with no class and no effort, and is recorded in the drain's state, not a lane's | its entries are keyed by issue, count the records that state keeps and mark class and effort `none_asked`; no field is added (decision 14) |

Smaller findings of the second pair, each taken up: what one run left out
is shown on every result, apart from what a command is known not to look
at; the three sources of a not-looked-at entry are named for what they
are; depth has a shown line and a test in step 1; a way that is a flag on a
wider command has a stated home for its help block and its flags; a
registry a repository names is named in help by where it is read from; the
drive's registry is two kinds fixed in code; assertions about flags and
exit statuses moved to surface test files; the steps' `needs` lines encode
their order; the generated files each step rewrites; the citation check's
refusal of an unreadable corpus and its handling of authors; the launch
preview's gates that run only when asked; a round's judge, which the review
set does list; and counts and import paths corrected against the code.

## Open Questions

_All resolved at the spec build of 2026-11-21, by the technical facilitator
on auto: the member's name (`extent`), the closed sets of reasons and
causes, the flag that lists gaps, where the registry, the exclusions and
the declaration live, what the depth line reads, and which artefacts carry
the declaration are settled above. The
mechanism's reading test with both roles is run at the retrospective the
intent names, and is not a build step._

## Footprint

- packages: internal/core/extent, internal/core/surface, internal/core/sweep, internal/core/lint, internal/core/source, internal/core/intent, internal/core/launch, internal/core/implement/loop, internal/adapter/scanner, internal/surface/cli, commands, docs/reference/cli/commands, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces
- tests: internal/core/extent/extent_test.go, internal/core/extent/render_test.go, internal/surface/cli/extent_declaration_test.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/extent_help_test.go, internal/core/sweep/extent_test.go, internal/surface/cli/sweep_extent_test.go, internal/core/implement/loop/autosweep_extent_test.go, internal/core/lint/extent_test.go, internal/surface/cli/lint_extent_test.go, internal/surface/cli/lint_outbound_extent_test.go, internal/core/source/extent_test.go, internal/surface/cli/source_extent_test.go, internal/core/intent/extent_test.go, internal/surface/cli/ready_extent_test.go, internal/core/launch/extent_test.go, internal/surface/cli/launch_extent_test.go, internal/surface/cli/drain_drive_extent_test.go

## Steps

1. The statement and its renderer
   - criteria: 5, 6, 7, 8, 9, 10, 12, 15, 24
   - needs: none
   - packages: internal/core/extent
   - tests: internal/core/extent/extent_test.go: the four verdicts in their order from fixtures of findings and gaps, with `none` for a run that looked at no registry kind (none armed, nothing to read, none readable), and no `verdict` member for a command with no clear state; a reason, a cause or a source outside its set fails to encode; a kind in both lists fails to encode; a depth entry encodes its three values with where each came from, and the marks `not_told` and `none_asked`; the golden file of reasons and causes refuses a rename and admits an addition. internal/core/extent/render_test.go: the `Looked at:` line on every verdict; `Left out in this run:` with each cause on every verdict that has such an entry; `Known not looked at:` with its label on the two clear verdicts and on neither `found` nor `none`; one count of checks per reason and no line when there is no gap; one `Judged by:` line for each depth entry; the summary line for `clear_except_gaps` and for `none`
2. The declaration, the facts, help and the lists
   - criteria: 13, 15, 21, 22, 23, 24, 25, 26, 27
   - needs: 1
   - packages: internal/core/surface, internal/surface/cli, internal/surface/cli/testdata, docs/reference/cli/commands, commands, .abcd/development/brief/04-surfaces/41-explain.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/surface/cli/extent_declaration_test.go: on fixture declarations, an empty hand-written list fails unless the command declares it looks at nothing; a flag with no class fails, and an inherited or hidden flag is not asked for; a way that is a flag on a wider command is a unit of its own. internal/surface/cli/extent_ratchet_test.go: a command path that is neither declared nor in the file fails; one that is both fails; a file longer than the ceiling fails; a retire-list entry that names no field of a declared command's result fails; the golden file of kinds and exclusions refuses a rename and admits an addition. internal/surface/cli/extent_help_test.go: help and explain name the same kinds, a registry a repository names is named by where it is read from, the help block labels the exclusions and the hand-written list as known, explain's text stays inside its line limit, and a help text holding the word everything fails
3. Sweep, and the sweep abcd runs on its own
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/sweep, internal/core/implement/loop, internal/surface/cli/sweep.go, internal/surface/cli/drain_drive.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/sweep.md, .abcd/development/brief/04-surfaces/40-sweep.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/sweep/extent_test.go: on a fixture with no reservation and no waiting report both kinds are still named; a failed process check, an unread reservation store and unread build runs each give one gap with its reason, beside the kind they bear on; the embargo store is named as an exclusion; the sweep abcd runs on its own names three lists as looked at and two as left out with the cause `other_run`; the run reports the registry exactly once. internal/surface/cli/sweep_extent_test.go: a clear run shows `Looked at:` and `Known not looked at:`, and a run that lists something leaves the second out and still records it; `--gaps` lists each gap; the exit statuses are those of today on every fixture. internal/core/implement/loop/autosweep_extent_test.go: the loop records the statement of the sweep abcd runs on its own on the record that holds its line, and the line lists every gap in full
4. The two lints and the citation check
   - criteria: 1, 2, 3, 4, 8, 11, 14, 16, 17, 18, 28
   - needs: 1, 2, 3
   - packages: internal/core/lint, internal/adapter/scanner, internal/core/source, internal/surface/cli/cli.go, internal/surface/cli/lint_outbound.go, internal/surface/cli/source.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/lint.md, commands/library.md, .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/04-surfaces/07-library.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/lint/extent_test.go: a fixture with no rule armed gives the verdict `none` with every rule left out as `not_armed`; a fixture with rules armed and no document gives the verdict `none` with every armed rule left out as `nothing_to_read`, and `nothing_checked` is unchanged in both; a pruned path is named as an exclusion; a repository's own banned token keeps its name; a check that failed is one gap; the run reports the registry exactly once. internal/surface/cli/lint_extent_test.go: both `none` fixtures exit 0 and say nothing was checked, with what was left out and why; a clear run shows `Looked at:` and `Known not looked at:`, and a run with a finding leaves the second out and still records it; `--gaps` lists the gap. internal/core/source/extent_test.go: a clear check names the confidential sources read; a corpus with none gives the verdict `none`; a source's authors are named as an exclusion unless its entry opts in; the run reports the registry exactly once. internal/surface/cli/source_extent_test.go: the citation check's exit statuses are those of today, an unreadable corpus is refused as today with no statement, and a clear check shows both lines. internal/surface/cli/lint_outbound_extent_test.go: a clear scan names the detectors run exactly once, shows both lines and names the classes it filters as exclusions; a text it cannot scan is still refused
5. The readiness gate and the launch preview
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 20, 28
   - needs: 1, 2, 4
   - packages: internal/core/intent, internal/core/launch, internal/surface/cli/cli.go, internal/surface/cli/receipts.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/intent.md, commands/launch.md, .abcd/development/brief/04-surfaces/05-intent.md, .abcd/development/brief/04-surfaces/04-launch.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/intent/extent_test.go: a ready intent names each of the nine checks exactly once and `ready` is unchanged; an advisory check that does not pass changes neither `ready` nor the verdict. internal/surface/cli/ready_extent_test.go: a ready intent shows `Looked at:` and `Known not looked at:`; one that is not ready leaves the second out and still records it. internal/core/launch/extent_test.go: a gate the repository has not armed is left out with the cause `not_armed`, the citation gate among them with its status string unchanged, the deep-smoke and parity gates with the cause `other_run` unless the run asks for them, and the receipts gate with the cause `host_run`; a gate that could not be measured is one gap; the receipts gate's depth entry carries what a receipt on disk holds, marked `receipt`, and `not_told` for the rest; the report's `verdict` is that of today; the run reports the registry exactly once. internal/surface/cli/launch_extent_test.go: a preview whose every gate passes shows both lines and the `Judged by:` line; `--gaps` lists the gap; the exit statuses are those of today
6. The drive of a drain
   - criteria: 1, 2, 4, 11, 18, 19, 28
   - needs: 1, 2, 5
   - packages: internal/surface/cli/drain_drive.go, internal/core/implement/loop, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/surface/cli/drain_drive_extent_test.go: on a fake runner the result names both kinds exactly once, with how many open issues were read, how many were taken as lanes and how many lanes made a change, a drive that took none included; for each lane and each judging role it gives the class each judging run ran at, the number of judging runs and the effort each was asked for, as the drive's state records them; a reviewer moved to another class is named at the class it ran at; a lane with a fix round counts the runs of both rounds; a reviewer handed out again within a round is counted once; a round's judge is named as a judging role, and one that could not be launched is one gap with its reason and not a run; the drain's judge is named by issue: a judgement a runner answered is one judging run with class and effort marked `none_asked`, a judgement the host answered is marked `not_told`, and a fallback receipt the drain's state keeps is one gap with its reason and not a run; `--gaps` lists each gap; the statement carries no `verdict` member and records the hand-written list on every run; the drive's log follows the rules for what is shown

## Size

Six pull requests, built in the order of their numbers, which the steps'
`needs` lines encode. Steps 3 to 6 are not parallel lanes: each takes its
commands off the undeclared list and lowers the ceiling in the same test
file, and each adds its lines to the golden file of kinds and to the retire
list. Steps 3 and 6 also share `internal/surface/cli/drain_drive.go` and
`internal/core/implement/loop`, and steps 4 and 5 share
`internal/surface/cli/cli.go`. Step 4 is the widest, because the docs
lint's result type lives in the command-line surface today and the
statement has to be assembled where the core can return it. Another open
spec, spc-2611141617199772, changes the same loop package and
`drain_drive.go`; whichever of it and steps 3 and 6 lands second is rebased
on the first. Every step from 2 on also regenerates
`.abcd/development/release/surface.json` and `docs/reference/cli/commands.md`,
which is one more reason they are built in order.
