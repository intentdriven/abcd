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
  not be pinned. The depth line reads that record and adds no field
  (decision 14).
- [itd-85](../drafts/itd-85-audit-verb.md) gave the audit command one serializer
  interface, with a SARIF writer named as a later implementation. The
  statement of a command that checks something is written through it
  (decision 13).
- [itd-2610301750162657](itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
  already holds the same stance for one verb: a clean spot-check states what
  it rules out and is never reported as no problems.

Three things show the gap.

- A clear result is read as complete when it is not. The help of `abcd sweep`
  says it lists everything the repository has left lying around, and it lists
  five kinds. Stashes and lock files are outside it and nothing says so
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
  and not in help text or nowhere. Shown wrong if, in the two releases after
  this ships, three or more issues are filed about something one of the
  first commands missed although its results already named that kind as not
  looked at; or if a reading test with both roles shows that the statement
  makes a clear result more trusted, not less.

_Proposed by the technical facilitator (2026-11-21); the product thinker has
not confirmed it. The last clause is there because the research found that an
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
see and five on what the work leaves alone, and let each stand.

What a result says:

- **Given** a command in the first set, **when** a run finishes, **then** its
  result names every kind of thing the run looked at, a kind in which it
  found nothing included.
- **Given** a command in the first set, **when** the kinds its code looks at
  change, **then** a test fails until the result's statement names the same
  kinds.
- **Given** a command in the first set, **when** its result is clear,
  **then** what is shown to a person names what was looked at beside the
  word for clear.
- **Given** a check that could not be made in this run (a store that could
  not be read, a forge that did not answer, a check the configuration did
  not arm), **when** the command reports, **then** the result lists it as
  could not check.
- **Given** an item listed as could not check, **when** it is recorded,
  **then** it carries one reason from a closed set of reasons and a line of
  text.
- **Given** a result with something it could not check, **when** it is shown
  to a person, **then** its summary reads as clear except for the gaps it
  names.
- **Given** a result with something it could not check, **when** it is
  recorded, **then** the record marks it as clear except for gaps, apart
  from clear.
- **Given** a run in which nothing was checked, **when** the command
  reports, **then** the result says nothing was checked and gives no
  verdict.
- **Given** a result with something it could not check, **when** it is shown
  to a person, **then** the gaps appear as one count for each reason.
- **Given** a result with something it could not check, **when** the person
  asks for the detail, **then** every gap is listed.

What a command is known not to look at:

- **Given** a command in the first set, **when** the command tree is built,
  **then** a test fails unless the command's declaration lists at least one
  kind of thing it is known not to look at.
- **Given** that list, **when** it is shown, in a result or in help,
  **then** it is labelled as known, never as everything that was left out.
- **Given** a command in the first set, **when** its result is clear,
  **then** what is shown to a person names what the command is known not to
  look at.
- **Given** a command in the first set, **when** its result lists findings,
  **then** what is shown leaves that list out.
- **Given** a command in the first set, **when** any run finishes, **then**
  the recorded result carries that list.

How hard a model looked:

- **Given** a command that starts a model itself to do the judging,
  **when** it reports, **then** the result names the class of model, the
  number of independent runs and the effort asked for.
- **Given** a result that rests on judging the host's own agents did,
  **when** it reports, **then** it says abcd was not told how hard they
  looked.

What the binary says about a command:

- **Given** a command with a declaration, **when** a person reads its help,
  **then** the help names the kinds the command can look at and the kinds it
  is known not to, and no word such as everything stands in their place.
- **Given** a command with a declaration, **when** an agent asks the binary
  to explain the command, **then** the answer names the same kinds as the
  help.
- **Given** a flag on a command with a declaration, **when** the command
  tree is built, **then** a test fails unless the declaration says of that
  flag either what it adds to what is looked at or that it changes nothing
  looked at.
- **Given** a kind of thing a command looks at, or a reason a check could
  not be made, **when** it appears in a recorded result, **then** it carries
  an identifier that is the same in every release.

How it is delivered:

- **Given** the commands the binary's help lists, **when** the command tree
  is built, **then** a test fails unless each carries a declaration or is
  named in one committed list of commands not yet declared.
- **Given** that committed list, **when** it gains a name, **then** a test
  fails.
- **Given** the release that first carries this, **when** it ships,
  **then** these carry a declaration: sweep, the docs lint, the outbound
  lint, the library's citation check, the readiness gate, the launch
  preview, and the drive of a drain.

## Decisions

Decisions 1 to 11, 13 and 14 are the technical facilitator's, each made
without a question because the record settles it, and all are open to the
product thinker at sign-off. Decisions 12 and 15 and the last sentence of
decision 9 are the product thinker's, from the interview of 2026-11-21.

1. No general switch for looking harder, and no result or help text claims
   completeness. The verdict's research found certainty unattainable, and both
   reviewers rejected the switch.
2. The statement is disclosure, and it adds no refusal and changes no exit
   status. Where a command is a control, invariant 16 already has it refuse
   an input it cannot examine, and a command that holds back on a failed
   check, as sweep does, keeps doing so. A gap changes what the result says
   and what it records, as auditing practice has an auditor who could not
   obtain the evidence modify the opinion and give the reason (ISA 705). The
   exit status stays as it is under the product thinker's ruling of
   2026-09-23 (dcn-2609230000000020, item G2), which kept the docs lint at
   exit 0 when it checked nothing and has the result say so. The cost is
   plain: a caller that reads only the exit status still cannot tell a clear
   run from a partial one, and has to read the result.
3. What a run looked at is derived from what the command's code ran in that
   run: for sweep the reads it made, whether or not a row came back, and for
   the readiness gate and the launch preview the checks and gates they ran.
   Only the list of what a command is known not to look at is written by
   hand, beside the command, and it is labelled as known. An entry is added
   whenever someone is surprised that a command did not cover a thing, unless
   the command is changed to cover it. The division is the one invariant 18
   draws for the brief's surface chapters, applied here by analogy: what the
   code can state is derived, and a claim about behaviour stays hand-written
   and labelled until the binary records it where a generator can read.
4. The fields commands carry today stay as they are and feed the statement,
   so the change is additive. The audit result's list of rules skipped as not
   applicable is one of them. Each field the statement then duplicates goes
   on a committed list of names to retire, and retiring them is a later,
   breaking change judged by itself.
5. Delivery is a ratchet, under the principle `ratchet-not-big-bang`: the
   first set, and a committed list of every other command that a test lets
   only shrink. The first set is the commands whose clear result stands
   between a run and a publish, a push or a delete (sweep, the docs lint,
   the outbound lint, the library's citation check, the readiness gate and
   the launch preview) and the drive of a drain, the command that starts
   models itself. The spec may add to the set and not take from it.
6. Widening what a command checks is not this intent. Under the verdict's
   reframing a cheap gap is fixed in the command's default run, as an issue of
   its own: sweep's is iss-2611211020368839, which has it list stashes and
   stale lock files and clear neither, and leaves the line naming what a
   sweep is known not to look at to this intent. A slow or risky check gets a flag named
   for what it adds. The stance is proposed as the principle
   `no-general-depth-switch`.
7. No mode for never assuming. Reading a source instead of recalling it
   (itd-197) and backing a claim about behaviour with something that runs
   (itd-195) are standing defaults with their own bounds, and no mechanical
   check tells a tested claim from a reasoned one. The statement therefore
   never labels anything verified on an agent's say-so.
8. The statement is one shared type in the core, under the principle
   `one-canonical-primitive`. The lifeboat probe's coverage answers a
   different question (which brief sections a repository can ground) and keeps
   its own shape, and the two results that already use the name coverage keep
   it, so the statement takes another name in the spec.
9. This intent is planned by itself and not as a bundle with itd-173 and
   itd-176. A bundle is for user moments that only make sense delivered
   together, and these make sense apart: saying how far a result looked is
   useful before anyone can choose a depth. The product thinker chose on
   2026-11-21 to plan the choice of depth (itd-173) as well, after this one,
   so itd-173 and itd-176 each build on this intent; decision 15 gives the
   order.
10. What could not be checked is shown as a count for each reason, with the
    list on request and nothing when there is none. Mature tools report
    skipped work that way.
11. A clear result always names what was looked at. That part is derived, so
    it cannot be left empty to make a result look clean, and a security
    standard asks for scope to be stated from what was included.
12. What a command is known not to look at is shown to a person only on a
    clear result, the moment a result could be read as complete (product
    thinker, 2026-11-21). The choice was between that and every result: a
    line that shows on every run may stop being read, and a line that shows
    only on a clear result is missing when findings are listed. The recorded
    result carries the list on every run, so when it is shown can change in a
    later release without changing what is recorded.

13. The way to a standard format stays open where one exists. Every kind of
    thing a command looks at and every reason a check could not be made
    carries an identifier that does not change between releases, and the
    statement of a command that checks something is written through the
    serializer interface the audit command already has (itd-85). A command
    that acts has no such format, and none is claimed.
14. The depth line mints no field. It reads the record every model call
    carries since itd-166: the model and the effort asked for, and the closed
    set of reasons a field could not be pinned. That abcd was not told is
    that record's not-reported reason, and an effort a runner could not be
    handed shows as not passed.
15. The draft that has a host's payload name its model and how many agents
    ran (itd-2609180517121254) is planned between this intent and the choice
    of depth, so that how hard a review looked is known before anyone chooses
    a depth (product thinker, 2026-11-21). The order is this intent, that
    draft, then itd-173.

The questions the pre-pass raised were settled without a question to the
product thinker, each for the reason given: the exit status (decision 2), the
duplicated fields (decision 4), the five neighbouring records (kept as they
are, under "What it touches"), the hand-written list (decision 3), and which
commands make the first set and what the mechanism counts (both for the
spec).

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
  closed set of reasons a field could not be pinned (itd-166), and the depth
  line reads that record.
- **The effect on over-trust is unproven.** Over-trust in a clear tool result
  is measured, and so is the risk that an explanation raises trust whatever
  the answer's quality. Hence the reading test in the mechanism.

**Path 2: a small native shape, with a seam that is real for checks and
absent for actions.** The statement is abcd's own type. For a command that
checks something (a lint, a gate), it maps onto SARIF: what was looked at
becomes results of the kinds pass and not applicable, what could not be
checked becomes notifications, and what the command is known not to look at,
with the depth of any judging, goes in the run's property bag. The audit
command already has the seam this needs, one serializer interface with a
SARIF writer named as a later implementation (itd-85), and the statement is
written through that interface, not beside it. The seam is for interchange
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
decision 12)._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a clear result is read as complete today (the help of sweep says everything and it lists five kinds), so each result is made to name what it looked at, what its command is known not to look at and what it could not check; expected: fewer clear results are acted on as if nothing were left; shown wrong if, in the two releases after it ships, three or more issues are filed about something one of the first commands missed although its results named that kind as not looked at, or if a reading test with both roles shows a clear result trusted more
