---
id: itd-2611211024360819
slug: every-result-abcd-gives-says-how-far-it
spec_id: spc-2611211659223744
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Every result abcd gives says how far it looked

## Press Release

> Every result abcd gives says how far it looked. A clear result never stands alone: it names what the command looked at, what it is known not to look at, and what it could not check this time. A sweep that finds nothing to clear says it looked at worktrees, branches, reservations and waiting reports, that it is known not to look at scratch folders or saved builds, and that it could not tell which sessions are still running, so it kept every worktree. Where abcd itself starts a model to do the judging, the result also says how hard it looked: the class of model, the number of runs and the effort it asked for. Where the host's own agents did the judging, the result says abcd was not told. The product thinker and the technical facilitator no longer take a clear result as proof that nothing is left. They read what it covers and decide whether that is enough. abcd never says it covered everything, and no switch promises that it will.

_Drafted by the technical facilitator on 2026-11-21 from the reframing of an
ideate verdict, and reworded the same day after two reviews. The product
thinker read the promise in short form at the planning interview that day
and let it stand. The criteria have since refined it, and the introduction
to the criteria lists each refinement: among them, where the host's agents
did the judging a result first names what a receipt on disk carries
(criterion 20), and where abcd starts a model and asks it for no class and
no effort the result says none was asked (criterion 19). The text above is
left as the product thinker read it._

## Why This Matters

Where this comes from. It carries the first part of the reframing recorded in
the ideate verdict
[`2026-11-21-ideate-thorough-flag-on-verbs.md`](../../research/ideas/488a0aa96ac5de805348635b27036addf15cddc2/2026-11-21-ideate-thorough-flag-on-verbs.md)
(decision
[dcn-2611211021077161](../../../work/decisions/dcn-2611211021077161-ideate-thorough-flag-on-verbs-verdict.md)).
The idea behind that verdict asked for a switch that makes a command look
harder, so that both roles could be certain nothing was missed. Two reviewers
who did not see each other's work turned the switch down and kept the need:
today neither role can see how far a result looked.

What it touches. No typed link (supersedes, reverses, duplicates, refines) is
asserted.

- It applies the rule of
  [adr-56](../../decisions/adrs/0056-an-exclusion-control-asserts-only-what.md),
  the brief's invariant 16, which was written for attestations, to every
  result a person acts on. That is new reach. The invariant's text does not
  change.
- [itd-173](../drafts/itd-173-verification-escalates-from-the-built-in.md) (the product
  thinker picks how hard verification looks, per project) and
  [itd-176](../drafts/itd-176-whatever-ships-says-how-hard-anyone.md) (whatever ships
  says how hard anyone looked) stay the carriers of
  [dcn-2609230000000008](../../../work/decisions/dcn-2609230000000008-delivered-work-states-the-verification.md),
  the ruling that delivered work states the rung it was checked at. Both are
  drafts with a state-of-the-art section and no criteria yet. This intent
  states the depth one run used; it does not choose a depth, and it does not
  compose what a shipped thing says. itd-176 can build its block from the
  statements defined here. Whether something not checked blocks a release is
  itd-176's open question and stays open; decision 2 below answers only the
  narrower one, that a check one run could not make blocks nothing.
- [itd-2609180517121254](../drafts/itd-2609180517121254-every-payload-a-host-hands-back-from-a.md)
  has a host's payload name its model and how many agents ran. The depth of
  work the host judged is that draft's to deliver. Until it does, a result
  here names what a receipt on disk carries and says abcd was not told for
  the rest, and this intent never refuses a payload. The
  product thinker chose to plan it straight after this intent (decision 15).
- [itd-166](../shipped/itd-166-each-run-should-record-the-model.md), shipped,
  already records what each model call was asked for and why a field could
  not be pinned, and keeps that record with the technical facilitator. The
  depth line does not read it. It reads what a drive's state records as it
  hands out each judging run: the class and the effort the review set names
  ([itd-2610240922461281](../shipped/itd-2610240922461281-a-person-sets-how-many-models-review.md),
  [itd-2611101419006280](../shipped/itd-2611101419006280-a-person-sets-the-whole-model-set-a-run.md)),
  after any move to another class, once for each round (decision 14).
- [itd-85](../drafts/itd-85-audit-verb.md) gave the lint command one serializer
  interface, with a SARIF writer named as a later implementation. The
  statement is not written through it, because the packages of four of the
  first commands cannot import its package. It keeps the way to that writer
  open with identifiers that do not change and a mapping written down in the
  spec (decision 13).
- [itd-2610301750162657](itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
  already holds the same stance for one verb: a clean spot-check states what
  it rules out and is never reported as no problems.

Three things show the need.

- A clear result is read as complete when it is not. The help of `abcd sweep`
  says it lists everything the repository has left lying around, and it works
  through five lists: worktrees, local branches, branches on origin,
  reservations and waiting reports. Stashes and lock files are outside it and
  nothing says so
  ([iss-2611211020368839](../../../work/issues/open/iss-2611211020368839-sweep-help-says-everything-and-lists.md)).
  For the other half sweep is already the model: a check that could not be
  made holds every item it could have cleared and is said once, at the top.
- abcd already says, in many places, that something was skipped or could not
  be read, and says it differently each time. One search at commit 1ceb7adec
  finds 41 differently named fields for it in the results of 30 packages
  (`unreadable`, `skipped`, `redaction_degraded`, `liveness_unread`,
  `fork_unchecked` and the rest), and it misses others (`branches_not_read`,
  `not_measured`). None states what a command is known not to check, so the
  standing edge of an answer is in help text or nowhere. One command's code
  gives the reason in its own words: scope is not visible in a result, and a
  reader "cannot otherwise tell "nothing there" from "not looked at""
  (`internal/core/lifeboat/coverage.go`).
- Certainty is not on offer at any setting, so the honest thing to ship is the
  edge of the answer. Checking can show that faults are present and cannot
  show that none are left (Dijkstra, EWD340). In 1,642 annotated runs of
  multi-agent systems, 8.2 per cent had no or incomplete verification and 9.1
  per cent had incorrect verification (arXiv 2503.13657). A security standard
  asks an assessment report for the level attempted, the requirements checked
  and the reason for any left out, and calls running an automated tool alone
  insufficient (OWASP ASVS 5.0, Assessment and Certification).

The verdict lists each of those outside sources with what was found in it.

## Mechanism

- We expect fewer clear results to be acted on as complete, because the edges
  of each answer sit in the result itself, in one shape a person learns once,
  and not in help text or nowhere. No count exists from before, so the claim
  is held to a threshold and not to a comparison. Shown wrong if, in the two
  releases after the first one that carries the first set, three or more
  issues report a person or an agent acting on a clear result of a first-set
  command as if nothing were left, although the result named what it left
  out; or if a reading test with both roles shows that the statement makes a
  clear result more trusted, not less.

_Proposed by the technical facilitator (2026-11-21) and tightened after the
plan review of the same day; the product thinker has not confirmed it. An
issue is counted when it names this intent in its `related_intents` and
reports the misreading. An issue that only asks a command to cover more is
not counted, because decisions 3 and 6 invite those. The retrospective of
the second of those releases reads the count, runs the reading test and
records both. The last clause is there because the research found that an
explanation can raise trust in an answer whether or not the answer is right._

## Scope Conditions

- The commands the binary's help lists. A command hidden from help, which <!-- cond: cond-2611211659224781 -->
  the hooks call on their own behalf, is outside it.
- abcd states only what its own code knows. What a run looked at comes from <!-- cond: cond-2611211659221816 -->
  the lists the command's code works through. What a command is known not to
  look at is in part what its code leaves out by a rule it reads and in part
  a list written by people; it is never complete, and is labelled as known,
  not as the whole edge.
- abcd makes no statement about whether an agent tested a claim or reasoned <!-- cond: cond-2611211659223612 -->
  its way to it.
- For work a model judged, abcd knows the depth only where it started the <!-- cond: cond-2611211659222345 -->
  model itself, and then it knows the effort it asked for, not the effort
  spent.
- A person reads the result at a terminal, or an agent relays it, and the <!-- cond: cond-2611211659227854 -->
  statement reaches both from the same data.
- No setting, at any cost, makes a result complete. <!-- cond: cond-2611211659220977 -->
- No study was found of whether such a statement changes how people read a <!-- cond: cond-2611211659225165 -->
  clear result. The claim holds only as far as a reading test with both roles
  bears it out.

## Acceptance Criteria

Seeded by the technical facilitator on 2026-11-21 and reworked after its
reviews. At the planning interview that day the product thinker read
them as eleven plain lines, six on what a person will see and five on what
the work leaves alone, and let each stand. A plan review reported after the
sign-off and found bars that could not be tested as written, and the final
reviews before landing found more. The technical facilitator, on auto,
tightened them the same day. Some things the product thinker read or
answered are refined, none reversed. Each is flagged here for them and
queued for their return:

- a result in which nothing was checked gives no verdict whatever the
  reason, where their line 3 said nothing could be checked;
- a command that looks at nothing states that in place of a list (their
  line 4);
- part of what a command is known not to look at is taken from the code,
  where line 4 said people write that list: what the code leaves out by a
  rule it reads joins the list written by hand, which is still never
  empty. The line's label is known not looked at, the words of the example
  they read;
- work the host's agents judged shows what a receipt carries before it says
  abcd was not told (line 5);
- work judged by a model abcd started says none was asked where abcd asked
  for no class and no effort, as it does for the judge of a drain, where
  line 5 had every such result name both;
- an option may also narrow or redirect what is looked at (line 11);
- an option hidden from help is not classified, where line 11 said each
  option a command takes;
- the list of commands not yet declared is held by a test that fails when
  the list grows longer; a change that declares one command and lists
  another keeps its length and is left to its reviewer, where line 10 said
  the list can only get shorter;
- their answer that what a command is known not to look at shows only when
  nothing was found now covers two results, the clear one and the one that
  is clear except for gaps (decision 12);
- the drive of a drain has no clear state, so its result never reads as
  clear or as clear except for gaps, and its text never shows what it is
  known not to look at, which its help and its recorded result carry
  (lines 2 and 4 name no exception);
- the number of judging runs counts the runs that returned a verdict and
  that the run's state still holds, which can be lower than the number
  started (line 5);
- the sweep abcd runs on its own lists every gap without being asked,
  because no one can pass it a flag (line 2 has the full list on request);
- where a command already gives a verdict of its own, that verdict is
  shown first and keeps its meaning, so it can read as a pass above a
  statement that is clear except for gaps (line 2);
- the promise names three parts of a result, and a fourth is shown beside
  them: what this run left out of the command's own registry, and why (a
  check the configuration did not arm, a check the host runs). A kind left
  out this way does not make a result read as clear except for gaps, which
  is kept for a check that could not be made. The line shows on every
  result, so for a command whose configuration seldom changes it reads the
  same from run to run, which is the cost their answer on decision 12
  weighed for the other list.

The words the criteria use:

- A declared command is a command path, or a named way of running one, that
  carries a declaration. Where the way is a flag on a wider command (the
  launch preview, the drive of a drain), the criteria about help, explain
  and flags are met on that command for that flag, and its other ways,
  where it has any, stay undeclared under the command's own name. A way of
  running a command that no one types (the sweep abcd runs on its own) has
  no help and no flag of its own: the criteria about help and explain are
  met on the command it is a way of running, and its text lists every gap
  in full.
- A command's registry is the list of the kinds of thing it can look at,
  which a run works through. It is fixed in the command's code, or, where
  the kinds are a repository's own (the banned tokens its configuration
  names, the confidential sources of its corpus), read by that code from
  the one place the repository names them. Help, explain and the files
  generated from them name such a registry by where it is read from, since
  the binary cannot list a repository's kinds.
- Left out in this run is a registry kind the run did not look at, with its
  cause from a closed set: the configuration did not arm it, there was
  nothing to read it against, the host runs it, or it is left to another
  way of running the command. It is a fact of one run, taken from the code.
- A command's exclusions are what its code leaves out by a rule the code
  itself reads: a path a lint prunes, a class of finding a scanner filters
  out. They are not kinds the command can look at, and each rule stands in
  every run. Where a repository sets what a rule leaves out (the paths a
  lint is told to exempt), the exclusion is the rule, named by where its
  values are read from.
- What a command is known not to look at is its exclusions, taken from the
  code, and a list written by hand beside the command. Both stand from run
  to run, and both are labelled as known.
- Nothing was checked means the run looked at no kind in its registry.
- A gap is a check that could not be made in this run. The criteria do not
  use the word for a kind a command is known not to look at.
- Recorded means the result's JSON form, and shown means its text.
- In the depth line the number of runs is the number of judging runs, as a
  drive's state or a receipt records them. Everywhere else a run is one
  execution of a command.

What counts as clear differs by command:

| Command | Clear | Findings | Nothing checked |
| --- | --- | --- | --- |
| sweep, and the sweep abcd runs on its own | the run would act on nothing | it lists something it would act on | does not arise: a sweep that cannot read the repository refuses |
| docs lint | no finding, and at least one armed rule had something to read | a finding | no rule armed, or no armed rule had anything to read |
| outbound lint | no finding in the text it scanned | a finding | does not arise: it refuses a text it cannot scan |
| citation check | no claim rests on a confidential source | one does | the corpus holds no confidential source |
| readiness gate | the intent is ready: every check that can withhold readiness passes | one of those checks fails | does not arise |
| launch preview | every gate it measured passes | a gate fails | does not arise |
| drive of a drain | none: it reports what its lanes did | none | no state of its own: a drive that looked at no kind says so |

A run that looked at no kind in its registry has checked nothing, whatever
the reason (no check armed, nothing to read, or no kind could be read): it
gives no verdict and says why, also where the table says the case does not
arise. The readiness gate has advisory checks that never withhold
readiness; one that does not pass is reported as it is today and changes
neither the ready flag nor the verdict. The docs lint's own warning that
nothing was checked keeps the narrower meaning it has today, that no rule
fed by its roots read anything, and can stand beside a statement that is
clear or lists findings.

What a result says:

- **Given** a declared command, **when** a run finishes, **then** its result
  records and shows every kind in the command's registry that the run looked
  at, a kind in which it found nothing included.
- **Given** a declared command, **when** a run finishes on a fixture,
  **then** a test fails unless every kind in the command's registry is
  reported exactly once, as looked at, as left out in this run or as one
  it could not check, and unless every other entry of what was not looked
  at is one of the command's exclusions or a kind on its hand-written
  list.
- **Given** a declared command with a clear state, **when** its result is
  clear or clear except for gaps, **then** what is shown to a person names
  what was looked at beside the word for clear.
- **Given** a check that could not be made in this run (a store that could
  not be read, a forge that did not answer, a program that is missing, a
  check that ran out of time), **when** the command reports, **then** the
  result records it as could not check.
- **Given** a check recorded as could not check, **when** the result is
  read, **then** the entry carries one reason from a closed set of reasons
  and a line of text.
- **Given** a declared command with a clear state and a result with no
  finding, in which something was looked at and something could not be
  checked, **when** it is shown to a person, **then** its summary reads as
  clear except for the gaps it names.
- **Given** a declared command with a clear state and a result with no
  finding, in which something was looked at and something could not be
  checked, **when** it is recorded, **then** the statement marks it as
  clear except for gaps, apart from clear.
- **Given** a run in which nothing was checked, **when** the command
  reports, **then** the statement says nothing was checked and gives no
  verdict.
- **Given** a result with something it could not check, **when** it is shown
  to a person, **then** the gaps appear as one count of checks for each
  reason.
- **Given** a result with nothing it could not check, **when** it is shown,
  **then** no line about gaps appears.
- **Given** a result with something it could not check, **when** the person
  passes the flag that lists gaps, **then** every gap is listed.
- **Given** a kind of thing, **when** a result is recorded, **then** the
  kind is not both looked at and not looked at.

What a command does not look at:

- **Given** a declared command that looks at anything, **when** the command
  tree is built, **then** a test fails unless the hand-written list in its
  declaration names at least one kind of thing it is known not to look at; a
  command that looks at nothing declares that instead.
- **Given** a registry kind this run left out (a check the configuration
  did not arm, a check with nothing to read, a kind left to another way of
  running the command, a check the host runs), **when** the command reports,
  **then** the result records it and shows it as left out in this run, with
  its cause, taken from the code.
- **Given** the list of what a command is known not to look at, **when** it
  is shown, in a result or in help, **then** it is labelled as known, never
  as everything that was left out.
- **Given** a declared command with a clear state, **when** its result is
  clear or clear except for gaps, **then** what is shown to a person names
  what the command is known not to look at.
- **Given** a declared command, **when** its result lists findings or gives
  no verdict, **then** what is shown leaves that list out.
- **Given** a declared command, **when** any run finishes, **then** the
  recorded result carries that list.

How hard a model looked:

- **Given** a declared command that starts models itself to do the judging,
  **when** it reports, **then** for each judging role, by lane or by issue,
  the result records and shows the class each judging run ran at, the
  number of judging runs and the effort each was asked for, as the drive's
  state recorded them, and says so where abcd asked for none.
- **Given** a declared command whose result includes work the host's own
  agents judged, **when** it reports, **then** for the class of model, the
  number of runs and the effort it records and shows what a receipt on disk
  carries, and says abcd was not told for the rest.

What the binary says about a command:

- **Given** a declared command, **when** a person reads its help, **then**
  the help names the kinds in the command's registry, its exclusions and
  the kinds on its hand-written list, and the word everything does not
  stand in their place.
- **Given** a declared command, **when** an agent asks the binary to
  explain the command, **then** the answer names the same kinds as the
  help.
- **Given** a flag a declared command's help shows, one it inherits
  included, **when** the command tree is built, **then** a test fails unless
  the declaration says of that flag what it adds to what is looked at, how
  it narrows or redirects what is looked at, or that it changes nothing
  looked at.
- **Given** a kind of thing abcd defines, or a reason a check could not be
  made, **when** it appears in a recorded result, **then** it carries an
  identifier that is the same in every release.

How it is delivered:

- **Given** the commands the binary's help lists, **when** the command tree
  is built, **then** a test fails unless each is declared or named in one
  committed list of commands not yet declared, and not both.
- **Given** that committed list, **when** it grows longer, **then** a test
  fails.
- **Given** a declared command whose recorded result has a field that
  restates a member of the statement (what was looked at, what was left out
  or skipped, what could not be checked, or the verdict), **when** the
  statement lands on that command, **then** the field is named in one
  committed list of fields to retire, and a test fails on a name in that
  list that is no field of a declared command's result.
- **Given** the release that first carries this, **when** it ships,
  **then** these carry a declaration: sweep with the sweep abcd runs on its
  own, the docs lint, the outbound lint, the library's citation check, the
  readiness gate, the launch preview, and the drive of a drain.
- **Given** the commands the binary's help lists, **when** this intent is
  shipped, **then** the committed list is empty.

## Decisions

Decisions 1 to 11, 13, 14 and 16 to 19 are the technical facilitator's.
Where the record settles one, the decision names the record. Decisions 2,
4, 5 and 14 chose between answers the record left open, and each names the
other answer. Decisions 12 and 15 and the last sentence of decision 9 are
the product thinker's, from the interview of 2026-11-21. A plan review
reported after the intent was planned, and final reviews before landing.
Decisions 2 to 6, 8, 10 and 12 to 14 were reworded for them and 16 to 19
added, and the spec lists each major finding with what was done about it.
Decision 12 is the product thinker's and its rewording is flagged for them:
their answer said when nothing was found, and did not say which result
with no finding it meant. It now names the clear one and the one that is
clear except for gaps, and decision 16 settles the third, a result in
which nothing was checked.

1. No general switch for looking harder, and no result or help text claims
   completeness. The verdict's research found certainty unattainable, and both
   reviewers rejected the switch.
2. The statement is disclosure, and it adds no refusal and changes no exit
   status. Where a command is a control, invariant 16 already has it refuse
   an input it cannot examine, and a command that refuses or holds back
   today keeps doing so: the outbound lint refuses a text it cannot scan,
   and sweep holds every item a failed check could have cleared. A gap
   changes what the result says and what it records, as auditing practice
   has an auditor who could not obtain the evidence modify the opinion and
   give the reason (ISA 705). The record does not settle the exit status for
   every command. The product thinker's ruling of 2026-09-23
   (dcn-2609230000000020, item G2) kept the docs lint at exit 0 when it
   checked nothing and has the result say so, and it ruled on that lint
   alone. Treating the citation check the same way, which today reports
   clean at exit 0 against a corpus with no confidential source, is the
   technical facilitator's choice. The other answer was a failing exit
   whenever a control checked nothing. It would break the jobs that call
   these commands, and at the interview of 2026-11-21 the product thinker
   let stand that no exit status changes. The cost is plain: a caller that
   reads only the exit status still cannot tell a clear run from a partial
   one, and has to read the result.
3. Each declared command keeps one registry of the kinds of thing it can
   look at, in its own code or read by it from the one place a repository
   names them, and a run works through that registry. What a run looked at
   is reported from what the run did with each kind: looked at, left out in
   this run, or could not check. A check the configuration did not arm, a
   check with nothing to read, a check the host runs and a kind left to
   another way of running the command are registry kinds a run left out,
   each with its cause. A class of finding a scanner filters out and a path
   a lint prunes are exclusions: the code leaves them out by a rule it
   reads itself, they are not kinds the command can look at, and they are
   kept apart from the registry. The code states all of these already, and
   each is reported from the code. Only the standing list of what a command
   is known not to look at is written by hand, beside the command; that list
   and the exclusions are what the command is known not to look at, and are
   labelled as known. An entry joins the hand-written list whenever someone
   is surprised that a command did not cover a thing, unless the command is changed to
   cover it; a kind that enters the registry leaves the list. Invariant 18
   bears on this directly, since the declaration is the binary recording a
   claim about behaviour where a generator can read it: decision 18 says
   which artefacts carry it.
4. The fields commands carry today stay as they are and feed the statement,
   so the change is additive, which the product thinker let stand at the
   interview. The list of rules skipped as not applicable, on the result of
   the bare lint command, is one of them, and it waits for the spec that
   declares that command. A field that already gives a verdict keeps its
   value and its meaning: the launch report's verdict and the readiness gate's ready flag
   say what they say today, and the statement's own verdict stands beside
   them and is the one that marks a gap. A field restates a member of the
   statement when it gives the same fact under another name: what was looked
   at, what was left out or skipped, what could not be checked, or the
   verdict. Each such field goes on one committed list of names to retire
   when the statement lands on its command, and retiring them is a later,
   breaking change judged by itself. Which fields restate a member is judged
   by the reviewers of that change, as the truth of a flag's class is: no
   test can recognise it, so the test checks only that every name on the
   list is a field of a declared command's result. The other answer was to
   move the first set's fields into the statement now, which would break
   their readers in the same release.
5. Delivery is a ratchet, under the principle `ratchet-not-big-bang`: a
   first set, and a committed list of every other command that a test lets
   only shrink. Every name on that list traces to this intent, which ships
   when the list is empty (invariant 17). The first set is this intent's
   first spec, and later specs take the rest. The first set is the technical
   facilitator's pick and no rule selects it: six commands whose clear
   result is read just before something is published, pushed or deleted
   (sweep with the sweep abcd runs on its own, the docs lint, the outbound
   lint, the library's citation check, the readiness gate and the launch
   preview) and the drive of a drain, where abcd starts models itself. The
   release cut, the landing of a branch and the pruning of worktrees fit the
   same description and wait for a later spec. A spec may add to the first
   set and not take from it. The other answer was to ship this intent with
   the first set and file each later command as an intent of its own. It was
   not chosen because the promise says every result, and a promise met by
   seven commands would read as kept.
6. Widening what a command checks is not this intent. Under the verdict's
   reframing a kind a command could cheaply look at, and does not, is added
   to its default run as an issue of its own: sweep's is iss-2611211020368839, which has it list stashes and
   stale lock files, clear neither, and take the word everything out of its
   help. Naming the kinds in that help, and the line of what a sweep is
   known not to look at, are left to this intent, so that each is built
   once. A slow or risky check gets a flag named for what it adds. The
   stance is proposed as the principle `no-general-depth-switch`.
7. No mode for never assuming. Reading a source instead of recalling it
   (itd-197) and backing a claim about behaviour with something that runs
   (itd-195) are standing defaults with their own bounds, and no mechanical
   check tells a tested claim from a reasoned one. The statement therefore
   never labels anything verified on an agent's say-so.
8. The statement is one shared type in the core with one renderer, so that a
   person learns one shape once. That is the spirit of the principle
   `one-canonical-primitive`, whose letter covers infrastructure primitives
   and not a result type. The lifeboat probe's coverage answers a different
   question (which brief sections a repository can ground) and keeps its own
   shape, and the results that already use the name coverage keep it, so
   the statement takes another name in the spec.
9. This intent is planned by itself and not as a bundle with itd-173 and
   itd-176. A bundle is for user moments that only make sense delivered
   together, and these make sense apart: saying how far a result looked is
   useful before anyone can choose a depth. The product thinker chose on
   2026-11-21 to plan the choice of depth (itd-173) as well, after this one,
   so itd-173 and itd-176 each build on this intent; decision 15 gives the
   order.
10. What could not be checked is shown as one count of checks for each
    reason, with the list behind one flag and no line at all when there is
    none. Mature tools report skipped work that way.
11. A clear result always names what was looked at. That part is derived, so
    it cannot be left empty to make a result look clean, and a security
    standard asks for scope to be stated from what was included.
12. What a command is known not to look at is shown to a person only when
    nothing was found, on a result that is clear or clear except for gaps,
    the moment a result could be read as complete (product thinker,
    2026-11-21). The choice was between that and every result: a line that
    shows on every run may stop being read, and a line that shows only when
    nothing was found is missing when findings are listed. The recorded
    result carries the list on every run, so when it is shown can change in a
    later release without changing what is recorded.

13. The way to a standard format stays open where one exists, and is claimed
    no further than it reaches. Every kind of thing abcd defines and every
    reason a check could not be made carries an identifier that does not
    change between releases; a kind a repository names itself, as a banned
    token of the docs lint is, keeps the repository's name. The mapping onto
    SARIF is written down in the spec. The statement is not written through
    the serializer interface the lint command has (itd-85): that interface
    takes the lint's own result, and its package imports the docs lint's
    core, the outbound scanner and the site's core, and through them reaches
    the cores of the launch preview and the readiness gate, so none of those
    four can import it. A SARIF writer, when one is built under itd-85, reads
    the statement from the statement's own package. A command that acts has
    no such format, and none is claimed.
14. The depth line mints no field. It reads what the state of a drive
    records as it hands out each judging run, and not the review set a run
    was started with. The set (itd-2610240922461281, itd-2611101419006280)
    names a class and an effort for each reviewer of a role and for a
    round's judge, but it is fixed when the run starts: a reviewer whose
    class holds the model that wrote the work is moved to another class,
    and a lane with a fix round runs its reviewers again. So for each lane
    and each judging role the line gives the class each judging run ran at,
    the number of judging runs and the effort each was asked for, from the
    records the lane's rounds hold. A round writes its reviewers' records
    as it opens, from the set, so a record is a judging run only
    once it holds a verdict, and a judge's once it has ruled: a reviewer
    never handed out, one still out when the drive ended and a judge that
    was not launched are no run. A reviewer handed out again within a
    round reuses its record and is counted once, and a judge whose record
    was withdrawn with a reviewer's is not counted: the line counts what
    the state holds. The drain's judge is the one judging role the set
    does not list. It judges an issue's remedy before any lane exists,
    the drain's own state records
    it, and abcd asks it for no class and no effort: its line is keyed by
    issue, counts each judgement a runner or the decision adapter answered
    as one judging run and says of its class and effort that none was
    asked. A judgement the host answered is the
    host's judging, and the line says abcd was not told. A runner or an
    adapter that did not give an answer the drain could take is a gap and
    not a run where that state keeps a fallback for it; an attempt that
    leaves none is in no result, which the drive's hand-written list says.
    The other answer there was to start launching that judge at a class,
    which would change which model judges and is not this intent's to
    decide. The line gives the class and never the model's
    name. The opening text of a drive shows the set a run started then
    would use, and a result can differ from it. The record of what each
    call was asked for and served (itd-166) stays with the technical
    facilitator, as that intent decided, and the depth line does not read
    it. Where the host's agents did the judging, the result names what a
    receipt on disk carries and says abcd was not told for the rest; a
    receipt is the host's own account, relayed with its source and not
    vouched for, so the scope condition that abcd knows the depth only
    where it started the model itself stands. The depth member is defined
    once, here; the draft that has a host's payload name its model and its
    agents (itd-2609180517121254) fills it for the host's work and mints no
    second shape. The other answer on the source was to label the line as
    what the run was started with. It was not chosen because it would name
    a class a moved reviewer did not run at and undercount the runs of any
    lane with a fix round.
15. The draft that has a host's payload name its model and how many agents
    ran (itd-2609180517121254) is planned between this intent and the choice
    of depth, so that how hard a review looked is known before anyone chooses
    a depth (product thinker, 2026-11-21). The order is this intent, that
    draft, then itd-173.
16. What counts as clear is fixed for each declared command, in the table
    above the criteria. The drive of a drain has no clear state: it reports
    what its lanes did. The lines about a clear result therefore do not bind
    it, and its statement carries what it looked at, what it is known not to
    look at, the gaps and the depth, and no verdict. Its text does not show
    what it is known not to look at, which its help and its recorded result
    carry. A result in which nothing was checked does not show that list
    either: it says nothing was checked, and why.
17. Recorded means the result's JSON form and shown means its text. Where a
    command writes files, a JSON file is recorded and a text file is shown:
    the launch report is one of each, and a log a person reads, as the
    drive's is, follows the rules for what is shown.
18. A declaration belongs to a command path or to a named way of running
    one. Two of the first set are a flag on a wider command (the launch
    preview and the drive of a drain), and the sweep abcd runs on its own is
    typed by no one: it has no help and no flag of its own, so the criteria
    about help and explain are met on sweep, and its text lists every gap in
    full. For a way that is a flag on a wider command, those criteria are
    met on that command for that flag. The command facts the binary exports
    carry the declaration, so the explain command and the per-command files derived
    from those facts name the same kinds as help. Explain's text keeps to
    its line limit and its JSON carries the whole declaration. The generated
    appendix of a surface chapter stays flags and sub-verbs. A command that
    looks at nothing, as recording a decision does, declares that and lists
    nothing it is known not to look at.
19. A flag is classified one of three ways: it adds to what is looked at, it
    narrows or redirects what is looked at, or it changes nothing looked at.
    A dry run that fetches nothing and a flag that points a lint at another
    root are the second kind. Every flag a declared command's help shows is
    classified, one it inherits from a parent command included: the option
    that points the citation check at another corpus is one. A flag hidden
    from help is not classified, because no person is offered it. The
    product thinker read this line with two cases and with every option;
    the technical facilitator added the third case and the exemption of
    hidden options, and both are flagged for them.

The questions the pre-pass raised were settled without a question to the
product thinker, each for the reason given: the exit status (decision 2), the
duplicated fields (decision 4), the neighbouring records under "What it
touches" (none is superseded; two drafts gained a dated note and three a
`builds_on` edge to this intent), the hand-written list (decision 3), which
commands make the first set (decision 5) and what the mechanism counts
(under `## Mechanism`).

## SOTA

Surveyed on 2026-11-21, and challenged the same day by an independent pass
whose corrections are folded in below. Sources, evidence tiers and what could
not be verified are in the
[result-coverage SOTA note](../../research/notes/2026-11-21-result-coverage-statement-sota.md).

- **No existing format carries all three parts.** SARIF 2.1.0 carries two of
  them: what was looked at, as results of the kinds pass and not applicable,
  and what a run could not do, as its notifications and a flag for whether
  the run succeeded. Outside its property bags it has no place for what a
  tool is known not to look at, or for how hard a model looked. OSCAL's
  assessment results come closest on the first two parts, with lists of what
  a review includes and excludes; nothing was found in them for the third.
  Test formats carry one reason per skip and nothing that stands from run to
  run.
- **A gap changes the verdict's wording.** ISA 705 has an auditor who could
  not obtain the evidence modify the opinion ("except for the possible
  effects of the matter(s)") and give the reason; SARIF says a consumer
  "SHALL NOT assume that a failed run contains a complete set of analysis
  results".
- **Scope is stated from what was included.** ASVS asks for a report written
  "from the perspective of what was included rather than what was not
  included".
- **Counts by default, detail on request, nothing when empty.** pytest keeps
  skip detail out of its default output "to avoid cluttering" it; Semgrep
  prints a count per reason and points at a flag for the list, which was read
  from its source and not from its manual.
- **A line that never changes stops being seen.** The evidence comes from
  warnings, not from statements of scope, and no study of this kind of line
  was found. It is why decision 12 shows the line only when nothing was
  found.
- **Narrow flags are the established form.** Between them, the manuals of
  restic, git fsck and rsync say what a default check does not verify, what
  one flag adds or narrows, and what that costs.
- **Fields for model-judged work are partial and still moving.**
  OpenTelemetry's conventions for generative models name the model asked for
  and are marked as under development; no field for effort was found. abcd
  already records the model and the effort asked for on every call, with a
  closed set of reasons a field could not be pinned (itd-166), and a run's
  review set names the class and the effort of each reviewer
  (itd-2611101419006280). The depth line reads what a run recorded of it
  for each judging run (decision 14).
- **The effect on over-trust is unproven.** Over-trust in a clear tool result
  is measured, and so is the risk that an explanation raises trust whatever
  the answer's quality. Hence the reading test in the mechanism.

**Path 2: a small native shape, with a seam that is real for checks and
absent for actions.** The statement is abcd's own type. For a command that
checks something (a lint, a gate), it maps onto SARIF: what was looked at
becomes results of the kind pass, a registry kind a run left out becomes a
result of the kind not applicable, what could not be checked becomes
notifications, and the exclusions and the hand-written list, with the depth
of any judging, go in the run's property bag. The lint
command has one serializer interface with a SARIF writer named as a later
implementation (itd-85). The statement is not written through it: that
interface takes the lint's own result, and the packages of four of the
first commands cannot import its package (decision 13). What keeps the way open is
narrower and real: identifiers that do not change, and that mapping written
down in the spec for a later writer to read. The seam is for interchange
only: the forge's code scanning reads none of those three places, so an
export shown there would carry nothing of the statement. A command that acts,
as sweep does when it removes things, has no format to be exchanged in, and
none is claimed for it. No dependency. A SARIF export checked against the
published schema is left out, because that check needs a validator the
repository does not have, and a new dependency needs the person's word.

## Open Questions

_All resolved at the planning interview of 2026-11-21, in `## Decisions`
(whether the choice of depth is planned as well, in decisions 9 and 15, and
when a result shows what its command is known not to look at, in
decision 12). The plan review that reported after the sign-off raised no
question for the product thinker, and neither did the final reviews before
landing. What they changed is in decisions 2 to 6, 8, 10, 12 to 14 and 16
to 19, and the refinements of what the product thinker read or answered
are flagged in the introduction to the criteria._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a clear result is read as complete today (the help of sweep says everything and it works through five lists), so each result is made to name what it looked at, what its command is known not to look at and what it could not check; expected: fewer clear results are acted on as if nothing were left; shown wrong if, in the two releases after the first that carries the first set, three or more issues report a person or an agent acting on a clear result of a first-set command as if nothing were left although the result named what it left out, or if a reading test with both roles shows a clear result trusted more
