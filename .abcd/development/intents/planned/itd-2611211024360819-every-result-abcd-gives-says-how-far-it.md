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
and let it stand._

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
  statements defined here. Whether a gap blocks a release is itd-176's open
  question and stays open; decision 2 below answers only the narrower one,
  that a gap in one run's result blocks nothing.
- [itd-2609180517121254](../drafts/itd-2609180517121254-every-payload-a-host-hands-back-from-a.md)
  has a host's payload name its model and how many agents ran. The depth of
  work the host judged is that draft's to deliver. Until it does, a result
  here says abcd was not told, and this intent never refuses a payload. The
  product thinker chose to plan it straight after this intent (decision 15).
- [itd-166](../shipped/itd-166-each-run-should-record-the-model.md), shipped,
  already records what each model call was asked for and why a field could
  not be pinned, and keeps that record with the technical facilitator. The
  depth line does not read it. It shows the review set a run was started
  with
  ([itd-2610240922461281](../shipped/itd-2610240922461281-a-person-sets-how-many-models-review.md),
  [itd-2611101419006280](../shipped/itd-2611101419006280-a-person-sets-the-whole-model-set-a-run.md)),
  which the drive of a drain already shows a person before it starts, and
  adds no field (decision 14).
- [itd-85](../drafts/itd-85-audit-verb.md) gave the lint command one serializer
  interface, with a SARIF writer named as a later implementation. The
  statement is not written through it, because the cores of three of the
  first commands cannot import its package. It keeps the way to that writer
  open with identifiers that do not change and a mapping written down in the
  spec (decision 13).
- [itd-2610301750162657](itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
  already holds the same stance for one verb: a clean spot-check states what
  it rules out and is never reported as no problems.

Three things show the gap.

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
  look at is written by people, is never complete, and is labelled as known,
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

Seeded by the technical facilitator on 2026-11-21 and reworked after three
review passes, one bar to a line. At the planning interview that day the
product thinker read them as eleven plain lines, six on what a person will
see and five on what the work leaves alone, and let each stand. A plan
review reported after the sign-off and found bars that could not be tested
as written. The technical facilitator, on auto, tightened them the same day.
Three of the lines the product thinker read are refined, none reversed, and
each is flagged for them: a command that looks at nothing states that in
place of a list (their line 4), work the host's agents judged shows what a
receipt carries before it says abcd was not told (line 5), and an option may
also narrow or redirect what is looked at (line 11).

A declared command is a command path, or a named way of running one, that
carries a declaration. Its registry is the list, in its own code, of the
kinds of thing it can look at. Recorded means the result's JSON form, and
shown means its text. What counts as clear differs by command:

| Command | Clear | Findings | Nothing checked |
| --- | --- | --- | --- |
| sweep, and the sweep abcd runs on its own | the run would act on nothing | it lists something it would act on | does not arise: a sweep that cannot read the repository refuses |
| docs lint | no finding, with a check armed and a document read | a finding | no check armed, or no document |
| outbound lint | no finding in the text it scanned | a finding | does not arise: it refuses a text it cannot scan |
| citation check | no claim rests on a confidential source | one does | the corpus holds no confidential source |
| readiness gate | every check passes | a check fails | does not arise |
| launch preview | every gate it measured passes | a gate fails | does not arise |
| drive of a drain | none: it reports what its lanes did | none | none |

What a result says:

- **Given** a declared command, **when** a run finishes, **then** its result
  names every kind in the command's registry that the run looked at, a kind
  in which it found nothing included.
- **Given** a declared command, **when** a run finishes on a fixture,
  **then** a test fails unless every kind in the command's registry is
  reported, as looked at, as not looked at in this run or as one it could
  not check, and no other kind is.
- **Given** a declared command with a clear state, **when** its result is
  clear or clear except for gaps, **then** what is shown to a person names
  what was looked at beside the word for clear.
- **Given** a check that could not be made in this run (a store that could
  not be read, a forge that did not answer, a program that is missing, a
  check that ran out of time), **when** the command reports, **then** the
  result lists it as could not check.
- **Given** an item listed as could not check, **when** it is recorded,
  **then** it carries one reason from a closed set of reasons and a line of
  text.
- **Given** a result with no finding and something it could not check,
  **when** it is shown to a person, **then** its summary reads as clear
  except for the gaps it names.
- **Given** a result with no finding and something it could not check,
  **when** it is recorded, **then** the statement marks it as clear except
  for gaps, apart from clear.
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
  tree is built, **then** a test fails unless its declaration lists at least
  one kind of thing it is known not to look at; a command that looks at
  nothing declares that instead.
- **Given** a check the configuration did not arm, or one the code states
  it leaves out, **when** the command reports, **then** the result names it
  as not looked at in this run, taken from the code.
- **Given** the list of what a command is known not to look at, **when** it
  is shown, in a result or in help, **then** it is labelled as known, never
  as everything that was left out.
- **Given** a declared command with a clear state, **when** its result is
  clear or clear except for gaps, **then** what is shown to a person names
  what the command is known not to look at.
- **Given** a declared command, **when** its result lists findings,
  **then** what is shown leaves that list out.
- **Given** a declared command, **when** any run finishes, **then** the
  recorded result carries that list.

How hard a model looked:

- **Given** a declared command that starts models itself to do the judging,
  **when** it reports, **then** for each lane and each judging role the
  result names the class of model, the number of runs and the effort asked
  for, as the review set the run was started with gives them.
- **Given** a declared command whose result rests on judging the host's own
  agents did, **when** it reports, **then** for the class of model, the
  number of runs and the effort it names what a receipt on disk carries and
  says abcd was not told for the rest.

What the binary says about a command:

- **Given** a declared command, **when** a person reads its help, **then**
  the help names the kinds in the command's registry and the kinds it is
  known not to look at, and no word such as everything stands in their
  place.
- **Given** a declared command, **when** an agent asks the binary to
  explain the command, **then** the answer names the same kinds as the
  help.
- **Given** a flag a declared command registers itself and shows in its
  help, **when** the command tree is built, **then** a test fails unless the
  declaration says of that flag what it adds to what is looked at, how it
  narrows or redirects what is looked at, or that it changes nothing looked
  at.
- **Given** a kind of thing abcd defines, or a reason a check could not be
  made, **when** it appears in a recorded result, **then** it carries an
  identifier that is the same in every release.

How it is delivered:

- **Given** the commands the binary's help lists, **when** the command tree
  is built, **then** a test fails unless each is declared or named in one
  committed list of commands not yet declared, and not both.
- **Given** that committed list, **when** it gains a name, **then** a test
  fails.
- **Given** a field the statement duplicates on a declared command,
  **when** the statement lands on that command, **then** the field is named
  in one committed list of fields to retire.
- **Given** the release that first carries this, **when** it ships,
  **then** these carry a declaration: sweep with the sweep abcd runs on its
  own, the docs lint, the outbound lint, the library's citation check, the
  readiness gate, the launch preview, and the drive of a drain.
- **Given** the commands the binary's help lists, **when** this intent is
  shipped, **then** the committed list is empty.

## Decisions

Decisions 1 to 11, 13, 14 and 16 to 19 are the technical facilitator's.
Where the record settles one, the decision names the record; where it does
not, the decision names the other answer that was open. Decisions 12 and 15
and the last sentence of decision 9 are the product thinker's, from the
interview of 2026-11-21. A plan review reported after the intent was
planned. Decisions 2 to 6, 8, 10, 13 and 14 were reworded for it and 16 to
19 added, and the spec lists each finding with what was done about it.

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
   look at, in its own code, and a run works through that registry. What a
   run looked at is reported from what the run did with each kind: looked
   at, not looked at in this run, or could not check. A check the
   configuration did not arm, a gate not yet implemented, a class of text a
   scanner filters out and a path a lint prunes are stated by the code
   already, and each is reported as not looked at in this run, taken from
   the code. Only the standing list of what a command is known not to look
   at is written by hand, beside the command, and both lists are labelled as
   known. An entry joins the hand-written list whenever someone is surprised
   that a command did not cover a thing, unless the command is changed to
   cover it; a kind that enters the registry leaves the list. Invariant 18
   bears on this directly, since the declaration is the binary recording a
   claim about behaviour where a generator can read it: decision 18 says
   which artefacts carry it.
4. The fields commands carry today stay as they are and feed the statement,
   so the change is additive, which the product thinker let stand at the
   interview. The lint result's list of rules skipped as not applicable is
   one of them. A field that already gives a verdict keeps its value and its
   meaning: the launch report's verdict and the readiness gate's ready flag
   say what they say today, and the statement's own verdict stands beside
   them and is the one that marks a gap. Each field the statement duplicates
   goes on one committed list of names to retire when the statement lands on
   its command, and retiring them is a later, breaking change judged by
   itself. The other answer was to move the first set's fields into the
   statement now, which would break their readers in the same release.
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
   set and not take from it.
6. Widening what a command checks is not this intent. Under the verdict's
   reframing a cheap gap is fixed in the command's default run, as an issue of
   its own: sweep's is iss-2611211020368839, which has it list stashes and
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
   shape, and the two results that already use the name coverage keep it, so
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
    takes the lint's own result, and its package imports the cores of the
    docs lint, the launch preview and the outbound scanner, which therefore
    cannot import it. A SARIF writer, when one is built under itd-85, reads
    the statement from the statement's own package. A command that acts has
    no such format, and none is claimed.
14. The depth line mints no field. It reads the review set a run was
    started with (itd-2610240922461281, itd-2611101419006280): for each
    judging role, the class of each reviewer, how many reviewers there are
    and the effort asked for. The number of runs is the number of reviewers
    the set lists for the role. The drive already shows that set to the
    person before it starts, so nothing reaches a result that a person is
    not shown today. The record of what each call was asked for and served
    (itd-166) stays with the technical facilitator, as that intent decided,
    and the depth line does not read it. Where the host's agents did the
    judging, the result names what a receipt on disk carries and says abcd
    was not told for the rest. The depth member is defined once, here; the
    draft that has a host's payload name its model and its agents
    (itd-2609180517121254) fills it for the host's work and mints no second
    shape.
15. The draft that has a host's payload name its model and how many agents
    ran (itd-2609180517121254) is planned between this intent and the choice
    of depth, so that how hard a review looked is known before anyone chooses
    a depth (product thinker, 2026-11-21). The order is this intent, that
    draft, then itd-173.
16. What counts as clear is fixed for each declared command, in the table
    above the criteria. The drive of a drain has no clear state: it reports
    what its lanes did. The lines about a clear result therefore do not bind
    it, and its statement carries what it looked at, the gaps and the depth.
17. Recorded means the result's JSON form and shown means its text. A file
    that is both, as the launch report and the drive's log are, follows the
    rules for what is shown.
18. A declaration belongs to a command path or to a named way of running
    one. Two of the first set are a flag on a wider command (the launch
    preview and the drive of a drain), and the sweep abcd runs on its own is
    typed by no one. The command facts the binary exports carry the
    declaration, so the explain command and the per-command files derived
    from those facts name the same kinds as help. Explain's text keeps to
    its line limit and its JSON carries the whole declaration. The generated
    appendix of a surface chapter stays flags and sub-verbs. A command that
    looks at nothing, as recording a decision does, declares that and lists
    no gap.
19. A flag is classified one of three ways: it adds to what is looked at, it
    narrows or redirects what is looked at, or it changes nothing looked at.
    A dry run that fetches nothing and a flag that points a lint at another
    root are the second kind. Only the flags a command registers itself and
    shows in its help are classified. The product thinker read this line
    with two cases; the technical facilitator added the third, and it is
    flagged for them.

The questions the pre-pass raised were settled without a question to the
product thinker, each for the reason given: the exit status (decision 2), the
duplicated fields (decision 4), the five neighbouring records (kept as they
are, under "What it touches"), the hand-written list (decision 3), which
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
  was found. It is why decision 12 shows the line only on a clear result.
- **Narrow flags are the established form.** Between them, the manuals of
  restic, git fsck and rsync say what a default check does not verify, what
  one flag adds or narrows, and what that costs.
- **Fields for model-judged work are partial and still moving.**
  OpenTelemetry's conventions for generative models name the model asked for
  and are marked as under development; no field for effort was found. abcd
  already records the model and the effort asked for on every call, with a
  closed set of reasons a field could not be pinned (itd-166), and a run's
  review set names the class and the effort of each reviewer
  (itd-2611101419006280). The depth line reads the review set (decision 14).
- **The effect on over-trust is unproven.** Over-trust in a clear tool result
  is measured, and so is the risk that an explanation raises trust whatever
  the answer's quality. Hence the reading test in the mechanism.

**Path 2: a small native shape, with a seam that is real for checks and
absent for actions.** The statement is abcd's own type. For a command that
checks something (a lint, a gate), it maps onto SARIF: what was looked at
becomes results of the kinds pass and not applicable, what could not be
checked becomes notifications, and what the command is known not to look at,
with the depth of any judging, goes in the run's property bag. The lint
command has one serializer interface with a SARIF writer named as a later
implementation (itd-85). The statement is not written through it: that
interface takes the lint's own result, and three of the first commands'
cores cannot import its package (decision 13). What keeps the way open is
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
question for the product thinker. What it changed is in decisions 2 to 6, 8,
10, 13, 14 and 16 to 19, and three refinements are flagged in the
introduction to the criteria._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a clear result is read as complete today (the help of sweep says everything and it works through five lists), so each result is made to name what it looked at, what its command is known not to look at and what it could not check; expected: fewer clear results are acted on as if nothing were left; shown wrong if, in the two releases after the first that carries the first set, three or more issues report a person or an agent acting on a clear result of a first-set command as if nothing were left although the result named what it left out, or if a reading test with both roles shows a clear result trusted more
