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
  identifier, its label and one count: how many it found, by what the
  intent's table counts as a finding for the command. A kind in which
  nothing was found is an entry with a count of zero. For the readiness
  gate a check that can withhold readiness and fails counts one, and an
  advisory check that does not pass counts none and is reported as it is
  today. For the drive of a drain, which finds nothing, the count is how
  many it took or changed.
- `not_looked_at`: one entry per kind the run did not look at, each with a
  source. `left_out` entries are registry kinds this run did not look at,
  taken from the code, and each carries one of four causes: `not_armed`
  (the configuration armed no such check), `nothing_to_read` (the check
  was armed and the run had nothing to read it against), `host_run` (the
  host's agents run it, not this command) and `other_run` (it is left to
  another way of running the command). Where more than one cause applies,
  the entry carries the first of `host_run`, `other_run`, `not_armed` and
  `nothing_to_read`. `exclusion` entries are the
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
  otherwise `clear`. `none` is how the statement records that it gives no
  verdict. A command with no clear state (the drive of a drain) carries no
  `verdict` member at all.
- `depth`: present only where a model did the judging. One entry per
  judging role, keyed by lane where the judging is of a lane's change and
  by issue for the drain's judge (the launch preview's one role is the
  host's judging behind its receipts gate): the class each judging run ran
  at, the number of judging runs and the effort each was asked for. Each of
  the three is a value with where it came from (`run_state` or `receipt`),
  or is marked `not_told` (the host did the judging and no receipt says) or
  `none_asked` (abcd started the run and asked for none). An entry whose
  role ran no judging run gives the number, zero, and neither of the
  others.

A kind appears once: in `looked_at`, in `not_looked_at`, or as the kind of
a `could_not_check` entry whose check stopped the kind being read at all. A
check that failed on a kind the run did read (sweep's check of which
sessions are running) is a gap beside a `looked_at` entry. The encoder
refuses a statement with a kind in both `looked_at` and `not_looked_at`
(criterion 12).

The core returns the statement and writes nothing; each surface renders it.
One renderer in the package produces the lines every command prints:

- `Looked at:` and the labels, on every result. Where the run looked at no
  kind the line reads `Looked at: nothing`.
- `Left out in this run:` and the labels of the `left_out` entries, each
  with its cause, on every result that has one.
- `Known not looked at:` and the labels of the `exclusion` and `listed`
  entries, only when the verdict is `clear` or `clear_except_gaps`
  (decision 12), and so on neither `found` nor `none`, nor on a result
  with no `verdict` member.
- `Could not check:` and one count of checks per reason, naming the flag
  that lists them. No such line when there is no gap. Where no one can
  pass a flag (the sweep abcd runs on its own), the line names no flag and
  every gap follows it in full.
- `Judged by:` and one line for each depth entry: the role, the lane or the
  issue that keys it, the classes, the number of judging runs and the
  effort, with not told and none asked said in words. An entry with
  no run says so. A drive's lines end with one more, which says the
  numbers count the judging runs its state holds.
- The summary line reads as clear except for the gaps when the verdict is
  `clear_except_gaps`. When the run looked at no kind it says nothing was
  checked, and why: the causes of what was left out, the reasons nothing
  could be read, or that the registry held no kind, as a corpus with no
  confidential source gives the citation check. That holds for the verdict
  `none` and for a command with no `verdict` member.

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
stands beside them. Where the two differ, as when a launch gate could not
be measured and the report judges that as it does today, both are recorded
and shown, the command's own first, and the statement changes neither the
command's verdict nor its exit status. A command that refuses to run, or
that ends in an error of its own, gives that as it does today, and no
statement.

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
  so the two cannot differ. An exclusion is one entry for each rule. Where
  a repository sets what a rule leaves out (the docs lint's exempt paths,
  statuses and globs), the entry is the rule, named by where its values
  are read from, as a registry a repository names is;
- the hand-written list of kinds the command is known not to look at. What
  each list holds is written in its command's step and judged in that
  step's review, as the class of a flag is: the test checks only that the
  list of a command that looks at anything is not empty.

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
listed while its drive is declared. A path whose only way of running is
declared counts as declared and leaves the list: `launch` refuses without
its preview flag, so the preview declares the path, and its sub-verbs are
paths of their own. A registry a repository names (banned
tokens, the confidential sources of a corpus) is named in help, explain and
the generated files by where it is read from.

Each flag a declared command's help shows is classified one of three ways
(decision 19): it adds named kinds to what is looked at, it narrows or
redirects what is looked at, or it changes nothing looked at. A flag the
command inherits is classified too. One the root command registers is
classified once, in the surface, and that entry stands in every
declaration. One from any
other parent is classified in the declaration of each declared command
that shows it, as the citation check's corpus option is, which redirects
what is looked at. A flag hidden from help is not classified, because no
person is offered it.

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
  sees it (criteria 25 and 26). A change that declares one command and
  lists a new one keeps the length: the walk still makes it list the new
  name, and the reviewer of that change sees it.
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
JSON carries the whole declaration. Where a command's kinds do not fit,
the text names as many as fit and says how many more the JSON carries,
and the test of criterion 22 compares the help with the JSON. The long
help of a declared command
ends with a block generated from the same declaration, which labels the
exclusions and the hand-written list as known. A test asserts that help
and explain name the same kinds, that the block carries that label, and
that no help text of a declared command contains the word everything
(criteria 15, 21 and 22). That test is stricter than criterion 21 on
purpose: it refuses the word anywhere in a declared command's help, since
a test cannot tell which use is meant. The generated appendix of a
surface chapter stays flags and sub-verbs,
as invariant 18 has it; the chapters' prose and the command pages are
reworded in the same step as each command where they claim everything.

### The seven commands

| Command | Core | Registry | Registry kinds a run can leave out | Exclusions |
| --- | --- | --- | --- | --- |
| sweep | `internal/core/sweep` | worktrees, local branches, branches on origin, reservations, waiting reports (the five halves in `sweep.go`) | branches on origin and reservations, in a checkout with no origin (`nothing_to_read`) | the embargo store, a row kind the code names and never reads (`rows.go`) |
| the sweep abcd runs on its own | `internal/core/sweep` (`Automatic`), `internal/core/implement/loop` (`AutoSweeper`) | the same registry | reservations and waiting reports, left to the sweep a person runs (`other_run`) | the same |
| docs lint | `internal/core/lint` | the rules in `config.go` (`knownRules`), with the banned tokens a repository's configuration names | rules the configuration did not arm (`not_armed`); an armed rule the run gave nothing to read, as every rule fed only by the roots' documents is when the roots hold none (`nothing_to_read`) | the paths it prunes and fenced code; and, each named by where a repository sets it, every exemption the configuration carries, among them exempt paths, prefixes and edges, exempt statuses, each rule's exempt globs, ignored steps, allow lists, the commands listed as host-delegated or operator-internal, and the allowed contexts of a rule and of each banned token |
| outbound lint | `internal/adapter/scanner`, `internal/surface/cli/lint_outbound.go` | the two harness-leak kinds whose findings it keeps | none | every other class of finding its detectors raise, which it discards |
| citation check | `internal/core/source` (`CiteCheck`) | the confidential sources of the corpus, each named by its key alone, as the command reports them today, and never by title, alias or author | none | one rule: a source's authors are left out unless its entry opts in (`ban_authors`) |
| readiness gate | `internal/core/intent` (`ReadyWithQuestions`) | its nine checks, some of them advisory | a check that reports unchecked, as two do for an intent with no linked spec, or not applicable, as several do for a record of a bucket they do not apply to (`nothing_to_read`) | none |
| launch preview | `internal/core/launch` (`DryRun`) | the rows of its gates, and the lockstep, retention and bundle-rejection checks, which can refuse without a row | gates the repository has not armed (`not_armed`); the deep-smoke gate unless the run asks for it (`other_run`); the four gates that judge a plugin's payload (the installability smoke, the deep smoke when asked for, payload parity and hook compliance) on an artefact that is not a plugin (`nothing_to_read`, where the report spells their status `not_armed`); the receipts gate, which the host runs (`host_run`, where the report spells its status `host-run`) | what its scan never reads, because the bundle leaves it out by rule, one entry each: files no include pattern of the repository's names, named by where the patterns are set, files git ignores, and the denied namespaces |
| drive of a drain | `internal/surface/cli/drain_drive.go`, `internal/core/implement/loop` | two kinds fixed in code: the open issues it read to choose its lanes, and the change each lane made | the change each lane made, when it took no lane (`nothing_to_read`) | none |

The launch preview's citation gate reports the status `not_implemented`
when the repository has not armed it. That status string stays as it is in
the report, and the statement gives the cause as `not_armed`: no gate is
unbuilt, so the statement has no cause for one. The readiness gate's verdict
follows its `ready` flag, and an advisory check that does not pass is
reported as it is today.

Two commands can look at no registry kind with no check failing. A corpus
with no confidential
source gives the citation check an empty registry, and its verdict is
`none`. A docs lint whose roots hold no document, or which has no root,
leaves out, with the cause `nothing_to_read`, every armed rule the run
gave nothing to read. A rule that also reads elsewhere (extra roots, name
roots, the roots' own directories, the repository's root, the agent
prompts, the records) still runs
there, as it does today, and is looked at. The verdict then follows the
order above: `found` when such a rule found something, `none` when no rule
read anything, and otherwise `clear`, with the left-out line naming the
rules that had nothing to read. `nothing_checked` keeps today's meaning
(no rule is configured, no root is configured, or the roots hold no
document, so no per-document rule read anything; the ruling of 2026-09-23,
dcn-2609230000000020, item G2), and its text and the exit status stay as
they are: a blocker from a rule that reads elsewhere still fails the run.
It can therefore be true beside a statement verdict of `clear` or `found`.

The drive's two kinds carry one count each, as every looked-at entry does:
how many of the open issues it read it took as lanes, and how many lanes
made a change. A drive that took no lane leaves the second kind out, with
the cause `nothing_to_read`. What it did with an issue it did not take is
in its result as today (passed, flagged, routed, parked, held back, passed
over or handed back), and the statement does not repeat it. How each lane
was judged is
the depth entry, below.

Sweep already fails closed: a check it could not make holds every item and
is said once at the top. Those notes stay, and each also becomes a
`could_not_check` entry. A checkout with no origin leaves the branches on
origin and the reservations out, with the cause `nothing_to_read`. It
stamps no fetch of the reservations, so the note that holds every
worktree stands as one gap beside the worktrees. The waiting reports are
read from abcd's inbox, with an origin or without.

The sweep abcd runs on its own has no recorded result today:
`loop.AutoSweeper` returns one line of text, which the loop keeps as
`swept`, joins into its next-step text or adds to the diagnostics of a
resume. Step 3 has it return the statement with
the line. The loop records the statement on the record that holds the line,
and the line gains the renderer's text: the lists it worked through,
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
round's judge where one was started (`ValidationRound.Judge`). A round
writes all its validator records as it opens, from the set, and a
reviewer is moved to another class only as it is handed out. So a
validator record is a judging run once it holds a verdict, which is how
the drain's own count of validators reads it (`drain.go`), and a judge's
record once it has ruled. A record that holds neither is no run and no
gap: one never handed out, one still out when the drive ended, one put
back when its reviewer was withdrawn, and a judge marked as not launched
because its agent did not clear the calibration gate. The lane's own
entry in the result says where the lane was left. A reviewer handed out
again within a round reuses its record and is counted once. A role with
no run in a lane has an entry that gives zero runs and neither a class
nor an effort. A judge whose record was dropped when a reviewer was
withdrawn is not counted either, because the state no longer holds it,
and the drive's hand-written list says so. A run's record with no class
or effort, as in a run opened with no review set, is marked
`none_asked`.

The drain's judge (`RoleDrainJudge`) is recorded in the drain's own state
and not in a lane's, because it judges an issue's remedy before any lane
exists (`DrainState.Judgements`). Its entries are keyed by issue. A
judgement with a route recorded is one judging run, of a runner or of the
decision adapter, which names a provider's model and no class; abcd asks
neither for a class nor for an effort, so both are marked `none_asked`.
A judgement with no route recorded is the host's answer: no receipt is
kept for it, so all three values are marked `not_told` (criterion 20). A
runner or an adapter that did not give an answer the drain could take
leaves a fallback receipt for most failures
(`DrainState.JudgeFallbacks`, the latest 32, which name the role and the
reason and not the issue): each is a `could_not_check` entry and not a
run, and step 6 maps its reason onto one of the five, with `check_failed`
for one that fits no other. A rate limit, an error that is not a
runner's failure and a judge stopped with the drain leave no receipt, so
such an attempt is in no result, and the drive's hand-written list says
so. The state keeps the latest 32 receipts and drops an older one, so a
failed attempt older than those is in no result either, and the list says
that too. Where the drain's state outlives one drive,
the statement carries the judgements and the receipts recorded since
this drive started, by their times. No field is added to any state. The
line gives classes and never a model's name, and
does not read the call record (itd-166). The drive's opening text shows the
set a run started then would use; each run reads the set again, so a result
can differ from that text.

The launch preview's receipts gate is judged by the host's agents. It is a
registry kind the preview reports as not looked at, with the cause
`host_run`, and the statement's `depth` carries one entry for it: for each
of the three values, what a receipt on disk carries, marked `receipt`, and
`not_told` for the rest (criterion 20). A receipt on disk today names the
judge's model and a tier, and no class, number of runs or effort, and the
preview reads only its detector policy. So step 5 marks all three
`not_told` and adds no reading of a receipt; the mark `receipt` waits for
a receipt that carries one of the three, which the draft on a host's
payload brings (itd-2609180517121254). The preview has no lanes, so the
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
after it moves them there from the hand-written list. Step 3's tests
count no kinds, so they hold either way. That issue also takes the word
everything out of sweep's help. If it has not landed when step 3 is built,
step 3 takes the word out, since its help test fails while the word
stands, and the issue keeps the rest.

## How the Acceptance Criteria are met

| Criterion | Met by |
| --- | --- |
| 1 a run names every registry kind it looked at | `looked_at` is filled as the run works through the registry; a fixture test per command asserts a zero-count kind is named |
| 2 every registry kind is reported once, and every other entry of what was not looked at is an exclusion or on the hand-written list | a fixture test for each of the seven commands |
| 3 a clear result is shown with what was looked at | the renderer's `Looked at:` line, asserted on a clear fixture per command |
| 4 a check that could not be made is recorded | each command turns its existing failed-check fields into `could_not_check` entries |
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
| 17 left out when findings are listed or no verdict is given | the renderer, on `found`, on `none` and on a result with no `verdict` member |
| 18 recorded on every run | `not_looked_at` is always encoded |
| 19 depth for each judging role, by lane or by issue | `depth` on the drive's result, from the records the drive's state holds, and the renderer's `Judged by:` line |
| 20 what a receipt carries, not told for the rest | `depth` on the launch preview's receipts gate, and on a drain's judgement the host answered |
| 21 help names the kinds and never everything | the generated help block and its test |
| 22 explain names the same kinds | the facts carry the declaration; the same test |
| 23 every flag help shows is classified three ways | the declaration walk, over inherited flags too |
| 24 identifiers do not change | the two golden files of identifiers |
| 25 every command is declared or listed, not both | the walk over the command tree |
| 26 the list cannot grow unseen | the ceiling in that test |
| 27 fields that restate a member are on the retire list | named in review as each command lands; the test fails on an entry that names no field |
| 28 the first set carries a declaration | steps 3 to 6, and a test in step 6 that none of the seven is on the undeclared list |
| 29 the list is empty when the intent ships | not this spec: it closes with a remainder, and later specs empty the list |

## Plan review

A plan review reported after the intent was planned and this spec's first
draft was written. Its eighteen findings, and what was done:

| Finding | Done |
| --- | --- |
| 1 the result states were not closed, and clear was undefined for four commands | the intent's table of clear, findings and nothing checked per command; criteria 6 and 7 gained "no finding"; an unarmed check moved from could not check to not looked at in this run; the reasons are named above |
| 2 criterion 2 compared the statement with itself | the registry, and the fixture test against it; criterion 24 limited to kinds abcd defines |
| 3 the depth line could not be read from the call record, and the drive did not fit the clear-result lines | depth read per lane and role, at that time from the review set (replaced by row 4 of the final review); the drive has no clear state (decision 16); a declaration belongs to a path or a named way of running one (decision 18) |
| 4 reading the call record would surface what itd-166 keeps with the technical facilitator | the depth line does not read the call record (decision 14); what it read at that time, the review set, was replaced by row 4 of the final review |
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
| 2 decision 12 was reworded after the sign-off and the record did not say so | the introductions to the decisions and to the criteria now say so, and it is among the refinements flagged for the product thinker |
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
| 5 the verdict rule left a docs lint with rules armed and no document with no state it could report | a fourth cause, `nothing_to_read`, for an armed rule that had nothing to read; the third pair corrected which rules those are (row 8) |
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

A third pair read the second fix pass, and each reviewer said fix first on
one major finding:

| Finding | Done |
| --- | --- |
| 7 the drain's judge saying none was asked narrows a line the product thinker read, and the record did not flag it | flagged in the introduction to the criteria, which now lists every refinement of what the product thinker read or answered and carries no count |
| 8 a docs lint with no document still runs the rules that read outside its roots, and the second fix pass reported them as left out | only an armed rule the run gave nothing to read is left out with `nothing_to_read`; the others are looked at, the verdict follows the order, and `nothing_checked` keeps today's meaning; step 4 tests each case |

Smaller findings of the third pair, each taken up: criteria 6 and 7
limited to a command with a clear state; an exclusion a repository
configures, named by where it is set; one count on a looked-at entry; how
a result says why nothing was checked, an empty registry included;
`launch` as a path its preview declares; the shown lines for no kind
looked at, for a gap where no flag can be passed and for the key of a
depth entry; a drain judgement the host answered; a command's own verdict
and the statement's side by side; a test that the first set is declared; a
core test file for the drive's depth; the parity and deep-smoke gates, and
the checks that refuse without a row; what a receipt on disk carries
today; a sweep in a checkout with no origin, and its fourth failed-check
note; the outbound lint's registry; a round's judge that was not launched
or was withdrawn; the decision adapter as a way the drain's judge is
answered; the second spec that shares files with these steps; and the
order of this spec against the issue on sweep's help.

A fourth pair read the third fix pass. Both said fix first, on four major
findings between them:

| Finding | Done |
| --- | --- |
| 9 only the flags a command registers itself were classified, which narrowed a line the product thinker read without a flag and left the citation check's corpus option unclassified | every flag a declared command's help shows is classified, an inherited one included; a hidden flag stays exempt, and that is flagged in the introduction to the criteria |
| 10 a round writes its reviewers' records as it opens, so counting records would count judging runs that never ran, at the review set's class | a record is a judging run once it holds a verdict, and a judge's once it has ruled; a role with no run gives zero and neither a class nor an effort; step 6 tests a lane handed back before any hand-out and one left mid-round |
| 11 the first remedy for sessions that wait would refuse the questions the command pages reserve to the person, with nowhere to send them | the question gate is left as it is, and to the draft itd-2610240850336695, which designs what a session may answer alone; the remedy keeps the line every prompt carries |
| 12 the decision on those issues said the drain can take them, which the repository's own drain rule refuses | the decision says what the rule takes: a session works the three majors by hand, and a build can take the fourth |

Smaller findings of the fourth pair, each taken up: a confidential source
named by its key alone; refinements of lines 2, 4, 5 and 10 flagged; the
drive's rows in both tables; the docs lint's own warning beside the
statement's verdict; an entry with no run, the reason of a fallback and
the attempts that leave none; which judgements and receipts belong to one
drive; the order of causes where two apply; the launch preview's four
gates for a plugin's payload and what its bundle leaves out; the docs
lint's exemptions, and its rule that reads the roots' directories; a
readiness check that reports unchecked or not applicable; a sweep with no
origin; the soft failure the docs lint's gap test means; the other open
specs that share packages; step 3's packages and step 6's criteria; and
the tests of the help's wording, of the list's ceiling and of explain's
limit. Reading the record for finding 11 also showed that the same draft
would withdraw the route a second of those remedies builds on, which that
issue and the decision now say.

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
- tests: internal/core/extent/extent_test.go, internal/core/extent/render_test.go, internal/surface/cli/extent_declaration_test.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/extent_help_test.go, internal/core/sweep/extent_test.go, internal/surface/cli/sweep_extent_test.go, internal/core/implement/loop/autosweep_extent_test.go, internal/core/lint/extent_test.go, internal/surface/cli/lint_extent_test.go, internal/surface/cli/lint_outbound_extent_test.go, internal/core/source/extent_test.go, internal/surface/cli/source_extent_test.go, internal/core/intent/extent_test.go, internal/surface/cli/ready_extent_test.go, internal/core/launch/extent_test.go, internal/surface/cli/launch_extent_test.go, internal/core/implement/loop/extent_test.go, internal/surface/cli/drain_drive_extent_test.go

## Steps

1. The statement and its renderer
   - criteria: 5, 6, 7, 8, 9, 10, 12, 15, 24
   - needs: none
   - packages: internal/core/extent
   - tests: internal/core/extent/extent_test.go: the four verdicts in their order from fixtures of findings and gaps, with `none` for a run that looked at no registry kind (none armed, nothing to read, none readable), and no `verdict` member for a command with no clear state; a reason, a cause or a source outside its set fails to encode; a kind in both lists fails to encode; a depth entry encodes its three values with where each came from, and the marks `not_told` and `none_asked`; the golden file of reasons and causes refuses a rename and admits an addition. internal/core/extent/render_test.go: the `Looked at:` line on every verdict; `Left out in this run:` with each cause on every verdict that has such an entry; `Known not looked at:` with its label on the two clear verdicts and on neither `found` nor `none`; one count of checks per reason and no line when there is no gap; one `Judged by:` line for each depth entry; the summary line for `clear_except_gaps`; the line that says nothing was checked, and why, for `none`, for an empty registry and for a statement with no `verdict` member that looked at no kind; `Looked at: nothing`; a `Could not check:` line that names no flag and is followed by every gap
2. The declaration, the facts, help and the lists
   - criteria: 13, 15, 21, 22, 23, 24, 25, 26, 27
   - needs: 1
   - packages: internal/core/surface, internal/surface/cli, internal/surface/cli/testdata, docs/reference/cli/commands, commands, .abcd/development/brief/04-surfaces/41-explain.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/surface/cli/extent_declaration_test.go: on fixture declarations, an empty hand-written list fails unless the command declares it looks at nothing; a flag with no class fails, an inherited one included; a flag the root command registers is classified once; a hidden flag is not asked for; a way that is a flag on a wider command is a unit of its own. internal/surface/cli/extent_ratchet_test.go: a command path that is neither declared nor in the file fails; one that is both fails; a file longer than the ceiling fails; a retire-list entry that names no field of a declared command's result fails; the golden file of kinds and exclusions refuses a rename and admits an addition. internal/surface/cli/extent_help_test.go: help and explain name the same kinds, a registry a repository names is named by where it is read from, the help block labels the exclusions and the hand-written list as known, explain's text stays inside its line limit, and a help text holding the word everything fails
3. Sweep, and the sweep abcd runs on its own
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 28
   - needs: 1, 2
   - packages: internal/core/sweep, internal/core/implement/loop, internal/surface/cli/sweep.go, internal/surface/cli/sweep_auto_test.go, internal/surface/cli/drain_drive.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/sweep.md, .abcd/development/brief/04-surfaces/40-sweep.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/sweep/extent_test.go: on a fixture with no reservation and no waiting report both kinds are still named; a checkout with no origin leaves the branches on origin and the reservations out with the cause `nothing_to_read`, and carries the note that no fetch of the reservations is stamped as one gap beside the worktrees; a failed process check, an unread reservation store, unread build runs and a forge that did not answer each give one gap with its reason: a gap whose check stopped a kind being read at all carries that kind, which is then in neither list, and any other gap stands beside a kind the run read; the embargo store is named as an exclusion; the sweep abcd runs on its own names the lists it worked through as looked at, and reservations and waiting reports as left out with the cause `other_run`; the run reports the registry exactly once. internal/surface/cli/sweep_extent_test.go: a clear run shows `Looked at:` and `Known not looked at:`, and a run that lists something leaves the second out and still records it; `--gaps` lists each gap; the exit statuses are those of today on every fixture. internal/core/implement/loop/autosweep_extent_test.go: the loop records the statement of the sweep abcd runs on its own on the record that holds its line, and the line lists every gap in full; when it found nothing the line shows what sweep is known not to look at
4. The two lints and the citation check
   - criteria: 1, 2, 3, 4, 8, 11, 14, 16, 17, 18, 28
   - needs: 1, 2, 3
   - packages: internal/core/lint, internal/adapter/scanner, internal/core/source, internal/surface/cli/cli.go, internal/surface/cli/lint_outbound.go, internal/surface/cli/source.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/lint.md, commands/library.md, .abcd/development/brief/04-surfaces/16-lint.md, .abcd/development/brief/04-surfaces/07-library.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/lint/extent_test.go: a fixture with no rule armed gives the verdict `none` with every rule left out as `not_armed`; a fixture with only per-document rules armed and no document gives the verdict `none` with every armed rule left out as `nothing_to_read`; the same fixture with a rule armed that reads outside the roots names that rule as looked at, with the verdict `clear` when it finds nothing and `found` when it finds something; a fixture with no root behaves as one with no document; `nothing_checked` is unchanged in each; a pruned path is named as an exclusion; a repository's own banned token keeps its name; a directory the cross-store rule could not read, which that rule skips, is one gap, and a rule that fails outright ends the run with its error as today and no statement; the run reports the registry exactly once. internal/surface/cli/lint_extent_test.go: both `none` fixtures exit 0 and say nothing was checked, with what was left out and why; a blocker from a rule that reads outside the roots still fails the run when `nothing_checked` is true; a clear run shows `Looked at:` and `Known not looked at:`, and a run with a finding leaves the second out and still records it; `--gaps` lists the gap. internal/core/source/extent_test.go: a clear check names by key alone the confidential sources read; a corpus with none gives the verdict `none`; the rule that leaves a source's authors out unless its entry opts in is named as one exclusion; the run reports the registry exactly once. internal/surface/cli/source_extent_test.go: the corpus option the check inherits is classed as redirecting what is looked at; the citation check's exit statuses are those of today, an unreadable corpus is refused as today with no statement, a corpus with no confidential source says nothing was checked because the registry held no kind, and a clear check shows both lines. internal/surface/cli/lint_outbound_extent_test.go: a clear scan names the two harness-leak kinds exactly once, shows both lines and names every other class of finding, which it discards, as one exclusion; a text it cannot scan is still refused
5. The readiness gate and the launch preview
   - criteria: 1, 2, 3, 4, 11, 14, 16, 17, 18, 20, 28
   - needs: 1, 2, 4
   - packages: internal/core/intent, internal/core/launch, internal/surface/cli/cli.go, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/intent.md, commands/launch.md, .abcd/development/brief/04-surfaces/05-intent.md, .abcd/development/brief/04-surfaces/04-launch.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/intent/extent_test.go: a ready intent names each of the nine checks exactly once and `ready` is unchanged; a check that reports unchecked or not applicable is left out with the cause `nothing_to_read`; an advisory check that does not pass changes neither `ready` nor the verdict. internal/surface/cli/ready_extent_test.go: a ready intent shows `Looked at:` and `Known not looked at:`; one that is not ready leaves the second out and still records it. internal/core/launch/extent_test.go: a gate the repository has not armed is left out with the cause `not_armed`, the citation gate among them with its status string unchanged; the deep-smoke gate is left out with the cause `other_run` unless the run asks for it, the four gates that judge a plugin's payload, the deep smoke among them when asked for, with the cause `nothing_to_read` on an artefact that is not a plugin and their status string unchanged, and the receipts gate with the cause `host_run`; what the bundle leaves out by rule is named as three exclusions; the lockstep, retention and bundle-rejection checks are each named exactly once; a gate that could not be measured is one gap; the receipts gate's depth entry marks all three values `not_told`, since a receipt on disk carries none of them today; the report's `verdict` is that of today; the run reports the registry exactly once. internal/surface/cli/launch_extent_test.go: a preview whose every gate passes shows both lines and the `Judged by:` line; `--gaps` lists the gap; the exit statuses are those of today
6. The drive of a drain
   - criteria: 1, 2, 4, 11, 14, 17, 18, 19, 20, 28
   - needs: 1, 2, 5
   - packages: internal/surface/cli/drain_drive.go, internal/surface/cli/drain.go, internal/core/implement/loop, internal/surface/cli/extent_ratchet_test.go, internal/surface/cli/testdata, docs/reference/cli/commands, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/core/implement/loop/extent_test.go: from a drive's state on fake runners, for each lane and each judging role the depth entry gives the class each judging run ran at, the number of judging runs and the effort each was asked for; a reviewer moved to another class is named at the class it ran at; a lane with a fix round counts the runs of both rounds; a reviewer handed out again within a round is counted once; a lane whose round was opened and handed back before any reviewer was handed out gives zero runs for each role, with no class and no effort; a lane left mid-round counts only the reviewers that returned a verdict; a round's judge is named as a judging role, counted once it has ruled, and one that was not launched gives zero runs and no gap; the drain's judge is named by issue: a judgement a runner or the decision adapter answered is one judging run with class and effort marked `none_asked`, a judgement the host answered has all three values marked `not_told`, and a fallback receipt the drain's state keeps is one gap with its reason and not a run; the statement names both kinds exactly once, with how many open issues were taken as lanes and how many lanes made a change, and a drive that took none leaves the second kind out with the cause `nothing_to_read`; the statement carries no `verdict` member. internal/surface/cli/drain_drive_extent_test.go: the drive's result records the statement the loop returns, with the hand-written list on every run, which names a judging run the state no longer holds, a failed attempt of the drain's judge that left no receipt and a fallback receipt older than the latest 32 the state keeps; its log follows the rules for what is shown, with one `Judged by:` line for each depth entry; `--gaps` lists each gap. internal/surface/cli/extent_ratchet_test.go: none of the seven first commands is on the undeclared list

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
spec, spc-2611141617199772, changes the same loop package,
`drain_drive.go`, `internal/surface/cli/cli.go`, `commands/drain.md` and
the drain chapter; whichever of it and steps 3 to 6 lands second is
rebased on the first. Four more open specs name packages these steps
change: spc-2611201455051897, spc-2611051628562376 and
spc-2611042207343783 the loop package and the drain's surface, and
spc-2611012047503720 the lint and launch packages. A build reserves its
footprint as it starts and backs off where another holds a path. Every step from 2 on also regenerates
`.abcd/development/release/surface.json` and `docs/reference/cli/commands.md`,
which is one more reason they are built in order.
