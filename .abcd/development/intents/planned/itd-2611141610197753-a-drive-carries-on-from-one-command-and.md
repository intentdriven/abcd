---
id: itd-2611141610197753
slug: a-drive-carries-on-from-one-command-and
spec_id: spc-2611141617199772
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610222215485265]
severity: minor
related_issues: [iss-2611140902531015]
related_intents: [itd-2611141610192096, itd-2611141610201875, itd-2611141507299828, itd-2611150636498012]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A drive carries on from one command, and says in one line when it cannot

## Press Release

> **A drive carries on from one command, and says in one line when it cannot.** A person starts a drive with `abcd drain --drive`. After an interrupt, with or without a rebuild of abcd in between, the same command carries on with every run the interrupt left in the checkout, the lanes its drain lists included, and nothing else is typed for them. A lane the drive cannot take is set aside with its reason, and the drive opens the next one. While the drive runs, one line says whether it is working, waiting, stalled or stopped, since when, on what, and the one thing a person can do, or that there is nothing to do. `abcd implement status` and the board show the same line, so the answer to "is it working?" is the same wherever a person looks.

> "I pressed Ctrl-C, rebuilt, and typed the same command the next morning. It took up the three lanes it had left and said so. I did not have to learn what a take-over is." — Iris, a product thinker (they/them)

> "When I am asked whether the drive is working, I run the status command and read one line: stalled since 16:40 on an outage, the runner has not started. I no longer read the log for the last line that is not a wait." — Nia, a technical facilitator (they/them)

## Why This Matters

`abcd drain --drive` (itd-2610222215485265) runs through the ledger without input. Over three days on one machine twenty of its pull requests merged, and it also needed a second session to watch it, and a person needed six attempts over six hours to bring it back after a stop (`.abcd/development/research/notes/drive-operations-2026-11.md`). That intent made the drive run without input. This one makes it operable by someone who has not read its code.

The lab on drive operations (lab-261114145658-67e1f9b) reproduced each failure on the pinned build:

- A run taken over from a terminal is waited on without end by a drive started in that terminal, and no other lane opens: 0 lanes in eight hours (F-3, iss-2611140902531015).
- Whether an interrupted drive carries on at its next start depends on the schema version its run file states and on whether the stop's mark settled, neither of which a person can see. In the failing case the start ends on a paragraph of 706 characters that names two commands, and following it in the same terminal produces the stall above (F-4, F-15).
- One lane whose landing cannot find its record holds every other lane: 0 lanes opened in 23 hours, and 46 of 68 lines repeat one sentence (F-5). The one bound the drain has for a lane that keeps refusing never starts, because the refusal names no lane (F-14).
- Nothing the drive prints or stores says in one place whether it is working, waiting or stalled, since when and why. After a 23-hour stall `abcd implement status` lists 37 record lines without the cause, and the board says nothing is building (F-7, F-17).

The lab's two prototypes changed 121 lines between them, added no command, no flag and no stored state a person must clear, and removed the take-over, the closed terminal and both stalls from what a person has to know (F-8, F-9).

### Who it is for

- The product thinker (Iris), who starts a drive and leaves. They stop it with Ctrl-C, start it with the same command, and are never asked to know which process a run records as its driver.
- The technical facilitator (Nia), who is asked "is it working?". They read one line, at the drive's terminal or from a status surface they already use.

### What a person does

- Types `abcd drain --drive` and answers its one question.
- Looks at the terminal, or runs a status surface they already know, and reads one line.
- Presses Ctrl-C once to stop it. Types the same command to go on.

### What a person sees

- When something cannot go on: that the lane was set aside, why, and that the drive went on.
- When the drive itself waits on something with no named end, an outage or a limit that names no reset, for longer than the bound: `stalled`, since when, on what, and one act or that it needs a look.
- When every lane left is set aside: the drive does not end. It keeps waiting, tries the set-aside lanes again every hour until its end time, and the line reads `waiting` (decision 9).
- Where output is not a terminal: a `state:` line each time the state changes.

### What a person never has to know

- Which process a run records as its driver, and what a take-over is, for a drive they interrupted themselves.
- That an interrupt leaves a mark, whether it settled, and what lifts it.
- A run file's schema version.
- The difference between the loop's kinds of wait.

## Mechanism

We expect a person to bring a drive back with the start command alone, because every state the lab found a start stopping on is one the drive can read for itself: a recorded driver that is the drive's own terminal, a mark its own earlier interrupt left, and a lane that has waited a bounded time on something the drive cannot end. Shown wrong if, after this ships, a drive that a person interrupted needs any command but `abcd drain --drive` to take up the lanes its drain lists or any other run the interrupt left in the checkout, or if one lane's wait still stops other lanes opening.

We expect a person to stop asking whether the drive is working, because the loop already knows which wait it is in, since when, until when and the refusal's own remedy, and hands them to the surfaces as one value. Shown wrong if people still open the log, or ask a session to read it, to learn the drive's state.

## Scope Conditions

- The lab's own test of the whole idea: it is shown wrong by any new command, any flag a person must remember, or any stored state a person must clear. The one new stored thing, the drive's state, is overwritten by the next drive and cleared by nobody. A lane handed back as stuck leaves a mark, a worktree and a branch that a person clears by command, so decision 7 (the drive keeps opening lanes while hand-backs repeat) is an answer this condition counts against the design, and decision 8 (set aside, not handed back) is the answer that leaves nothing to clear. <!-- cond: cond-2611141617190715 -->
- One drive per working copy on one machine, as the drain's lock already enforces. The state line is stored in the checkout the drive runs from, and a status surface run in that checkout shows it. Which other checkouts and sessions show it is the status line's question (itd-2611141507299828). <!-- cond: cond-2611141617198456 -->
- Run files this build can read. The lab tried one earlier schema version (25); a file that states a version above this build's is refused by the read before any claim, as today. <!-- cond: cond-2611141617198101 -->
- A process table the drive can read. Telling the drive's own terminal from another live session reads process ancestry and the session markers; where it cannot be read the drive leaves the run, as today, and the lane is set aside at the bound. <!-- cond: cond-2611141617194898 -->
- A person's own stop stands until the person lifts it, and once it is lifted the next start takes up the runs it halted (decision 1). A start takes every interrupted run in the checkout, not only those its drain lists (decision 2). The single command is claimed for what the lab probed: an interrupt by signal, settled or not, and a run file at an earlier schema version written by the lab's test, which stands in for a rebuild. A closed terminal, a kill, a reboot and a file left by a real older build were not probed, and nothing here is evidence for them until a test or the live proof covers each. The lab did not build or test the start that takes every interrupted run in the checkout, or the one that takes up the runs a lifted stop halted, so the step that builds them must test both. <!-- cond: cond-2611141617192337 -->
- The lab measured on a fixture clock with a stand-in process table and a stand-in landing stage (F-3, F-5). No figure here comes from a live drive, and the live proof of the drive itself is still open under itd-2610222215485265. <!-- cond: cond-2611141617198387 -->

## What's Out of Scope

- The drive's terminal output: the start screen, one line per event, the region redrawn in place, plain output elsewhere and the full sentence one step away. That is itd-2611141610192096, which builds on this intent and opens with this intent's state line.
- Re-arming a pull request the merge queue removed, and fast-forwarding the drive's checkout between lanes. That is itd-2611141610201875, which builds on this intent.
- Keys, a terminal library, a full-screen interface and colour.
- The drive rebuilding itself from source. The lab did not probe it.
- A surface for what the drive's reservations cover, and a way to clear many handed-back lanes at once. Both are real needs from the incident note's event 10, and neither fits the state line (F-11).
- The subscription's budget share. The drive is not told it.
- Any new command, any flag a person must remember, and any stored state a person must clear.
- The host status line's row. itd-2611141507299828 reads the state this intent stores and defines none of it.

## Acceptance Criteria

- **Given** a run that was taken over from a terminal, **when** `abcd drain --drive` is typed in that terminal, **then** the drive takes the run and its lanes go on, with nothing else typed.
  - Evidence: lab-261114145658-67e1f9b, F-3 and F-8.
- **Given** a drive that was interrupted, whether or not its stop settled and whether or not abcd was rebuilt since, **when** `abcd drain --drive` is typed again, **then** the drive carries on with every lane its drain still lists that the interrupt left, lanes parked on a pull request included, and its output names no other command for them. **Given** a run an interrupt parked that the drain no longer lists, **then** the drive takes it up too, because at its start it takes every interrupted run in the checkout and not only those its drain's state lists (decision 2).
  - Evidence: lab-261114145658-67e1f9b, F-4, F-8 and F-15.
- **Given** a run whose recorded driver is alive and is not the starting drive's own terminal (another terminal, or a host session), **when** the drive starts, **then** it does not take the run, says who holds it, and goes on to other work. **Given** a run a person's own stop halted, **then** the drive does not take it while the stop stands and names it, and once the person has lifted the stop the next start takes it up (decision 1).
  - Evidence: lab-261114145658-67e1f9b, F-8 (the three refutation fixtures) and F-9.
- **Given** a lane whose run the drive cannot claim or take up, or whose wait the stuck bounds do not count, for a reason the drive cannot end, **when** that has lasted the stuck bound's minutes, **then** the drive says the lane is set aside and why, opens the next lane, and, when every lane left is set aside, does not end: it keeps waiting and retries the set-aside lanes every hour until its end time (decisions 3 and 9). The lane's reservation is left to lapse (decision 4), and the same drive tries the lane again after the other lanes are done (decision 5).
  - Evidence: lab-261114145658-67e1f9b, F-5 and F-9.
- **Given** a landing's reservation refusal that names no lane, because the issue's record is in no status folder of the checkout, **when** the drive meets it, **then** the lane no longer holds the drive without bound: it is set aside, not handed back as stuck (decision 8). A refusal built without a lane that is the machine's and not the lane's (the review set, the configuration, the remote's identity) is not counted against any lane.
  - Evidence: lab-261114145658-67e1f9b, F-9 and F-14.
- **Given** a drive in any state, **when** a person looks at its terminal, runs `abcd implement status` or reads the board in the drive's checkout, **then** one line says `working`, `waiting`, `stalled` or `stopped`; since when; on what; and one act or that there is nothing to do; and the three places show the same line, composed once from one stored value (decision 6).
  - Evidence: lab-261114145658-67e1f9b, F-7, F-11 and F-17.
- **Given** a stored state whose drive ended without writing a last line, **when** a status surface reads it, **then** the line says `stopped`, not heard from since the time it was last written, and never `working` (decision 6).
  - Evidence: lab-261114145658-67e1f9b, F-11 and F-17.
- **Given** a wait with no named end, **when** it has lasted the stuck bound's minutes with no lane moving, **then** the line says `stalled` and its `since` is when anything last moved, whatever reasons the waits gave in between. A wait with a named end, which is the pace window's next eligible time or a rate limit's named reset, is never `stalled` before that end. An outage's next try, a look at origin, a limit that names no reset and a look-again time are not ends, so an outage reads `stalled` once it has lasted the bound. The one look-again time that is an end is the hourly retry of set-aside lanes while every lane left is set aside: a drive waiting on it reads `waiting` and never `stalled` (decisions 3, 6 and 9).
  - Evidence: lab-261114145658-67e1f9b, F-11.
- **Given** lanes handed back one after another for the same reason, **when** the count reaches the bound, **then** the line says `stalled` and names the reason, instead of `working`. The drive then keeps opening lanes (decision 7). The count that makes the line read `stalled` is the run's to propose, with its reasons, when the step is planned, so this criterion cannot be written more exactly until then.
  - Evidence: lab-261114145658-67e1f9b, F-11 (events 1 and 2 of the incident note).
- **Given** an interrupt, **when** the drive ends, **then** every lane it held either carries the stop's mark or keeps its reservation, whatever schema version the run's file states.
  - Evidence: lab-261114145658-67e1f9b, F-16.

## Open Questions

All resolved by the product thinker on 2026-11-15 (decisions 4, 5, 7, 8 and 9).

## Decisions

Recorded 2026-11-14 within an autonomous run, on the product thinker's delegation (dcn-2611141613199819). The product thinker asked for this intent to be planned without them and for every design decision the run takes to be recorded for their review. Each question is quoted as the lab met it. Each answer is the one the lab's evidence best supports and that changes least for a person. Each is decided by the run for owner review and is open to reversal at that review. A design review on the same day found three of the six answers were not the run's to give, and the product thinker answered them on 2026-11-15, together with the lab's first two decisions: decisions 1 and 2 are reversed by them, decision 3 is kept, decisions 4, 5, 7, 8 and 9 are theirs, and decision 6 stands as the run's, for review.

1. **After a person's own stop is lifted, the next drive start takes up the runs it halted** (product thinker, 2026-11-15, reversing the run's answer of 2026-11-14).
   - Question (DQ-1): "After a person's own stop is lifted, does the next start take up the runs it halted?"
   - Answer: yes. While a stop stands the drive leaves its run and names it. Once the person has lifted it, the next start takes it up, with nothing typed. The run's first answer was no, a typed take-over as now; it is reversed.
   - What it costs and departs from: the lift becomes the go, so the person's stop has no second deliberate act. adr-2611050740453233 states that a run that took a stop needs a take-over before it steps again, so nothing resumes by itself when a stop lifts. This answer departs from that consequence for a drive's start. The lab did not build or test it (its prototype leaves a lifted stop alone on purpose), so its step must test it.
   - Alternatives and what each costs: the run's first answer, which is the rejected one: the start leaves a lifted stop's run alone and a typed take-over follows the lift. That keeps the person's second deliberate act, and costs two commands, with the take-over typed in the drive's terminal to be safe. The lift taking the runs over itself is also rejected: it is one command and keeps the act the person's, and `abcd implement stop --lift` gains a second effect.
   - Evidence: F-8 and the probe p1-claim-earlier-runs-2, where the prototype leaves a lifted person's stop alone on purpose and the fixture passes, so the answer taken has no lab fixture of its own.
   - Reversed if: people who lift a stop are seen to be surprised by a drive that takes the run up at its next start.
2. **A start takes every interrupted run in the checkout** (product thinker, 2026-11-15, reversing the run's answer of 2026-11-14).
   - Question (DQ-2): "Which runs are the drive's to take at its start?"
   - Answer: every run in the checkout that an interrupt halted, not only those its drain's state lists. A run an interrupt parked on its pull request whose drain has since been replaced is taken up with the rest. The run's first answer, the lanes the drain lists, which is what the prototype does, is reversed.
   - Not built or tested in the lab: its prototype takes only the drain's lanes, so the step that builds this must test it against the same fixtures and the replaced-drain case.
   - Alternatives and what each costs: the run's first answer, which is the rejected one: the runs its drain's state lists, which is what the prototype does. It leaves one case, a run an interrupt parked on its pull request whose drain has since been replaced, which waits for a typed take-over that the drive names once. The answer taken has the start act on runs the current drain never opened, which is its cost.
   - Evidence: F-8 and the probes p1-claim-earlier-runs-2 and p1-existing-tests-2. The second form of the prototype passes all six fixtures taking only the drain's lanes, and the repository's own test `TestANewDrainNamesARunASignalParked` states the case the answer taken now takes up, so the test is rewritten and the wider start is built and tested against it.
   - Reversed if: a start is seen to act on a run no interrupt of a drive halted.
3. **A lane may keep the drive waiting for the stuck bound's minutes, and the same figure is the stall bound.**
   - Question (DQ-3): "How long may one lane keep the drive waiting before it is set aside?"
   - Answer: the stuck bound's minutes, read from the existing `pace.stuck_minutes` (60 as bundled). It is one figure for three things: a lane handed back as stuck, a lane set aside, and a wait that reads `stalled`.
   - Alternatives and what each costs: 30 minutes, the prototype's constant, which halves the idle time per blocked lane and adds a second figure to explain beside the stuck bound. A new `pace` key, which is a setting to document. The answer taken costs up to an hour idle per blocked lane, and it is already a `pace` key, so a person who wants less sets the one key they would set for a stuck lane.
   - Evidence: F-9 and the probe p2-set-aside, where the lane is set aside at +30m0s and +36m21s and both other lanes then land. The mechanism is the prototype's and only the figure differs. The lab measured nothing that depends on 30 against 60 except the idle time. This departs from the prototype's figure, which the lab itself marks as the product thinker's to decide.
   - Reversed if: an hour idle per blocked lane is seen to cost a drive more than a second figure costs a reader.
   - Kept by the product thinker on 2026-11-15.
4. **A set-aside lane's reservation is left to lapse** (product thinker, 2026-11-15; formerly open question 1).
   - Question (DQ-4): "What becomes of a set-aside lane's reservation?"
   - Answer: it is left to lapse. There is no push, and the issue stays hidden from other sessions until the lease lapses, at most two hours.
   - What it needs: itd-2610222215485265 must be amended. Its press release says the drive never leaves an issue held while it waits for someone who is not coming, and its mechanism is shown wrong if a drive ends with an issue still reserved. Both are to say that a set-aside lane's reservation may stand until its lease lapses. That intent is not edited here; the amendment is filed as iss-2611150635298371.
5. **A set-aside lane gets another try in the same drive, after the other lanes are done** (product thinker, 2026-11-15; formerly open question 2).
   - Question (DQ-5): "Does a set-aside lane get another try in the same drive?"
   - Answer: yes, once the other lanes are done. A block that clears in a minute need not wait for a restart.
   - What it costs: a second timer, and a lane that flaps. The step that builds it plans both.
6. **The state is one small file in the local tier, and a wait with no named end is stalled at the stuck bound's minutes.**
   - Question (DQ-7, its first part): "Where is the state line kept, what counts as stalled, and what does the drive do when it is?"
   - Answer, where it is kept: in a file in the checkout's local tier beside the drive's log, written whole by the drive each time a field changes. It is not part of a run's state, and nobody clears it. It carries the drive's process id and start time and the time it was written, so a reader tells a drive that ended without a last line and shows it as `stopped`. How a reader tells is the spec's to say, and it is not the lab's: the lab's design reads "the drive's process gone", and nothing was stored in the lab.
   - Answer, what counts as stalled by time: a wait with no named end that has lasted the stuck bound's minutes with no lane moving. This follows decision 3's one figure. The named ends are the pace window's next eligible time and a rate limit's named reset; an outage has none.
   - The count of lanes handed back in a row that reads `stalled`, and what the drive does then, are decision 7.
   - Alternatives and what each costs: a stall bound of 30 minutes, the prototype's, which is a second figure beside decision 3's. The lab gives no other place to keep the line.
   - Evidence: F-11 and the probes p4-state-line-mapping and p4-pinned-lines-with-times for the line's fields and the time rule, with the prototype's first error (a stall that never registered while the wait's reason alternated) written into the criterion. F-17 for the surfaces. The stored form rests on no measurement: the prototype stored nothing.
   - Reversed if: a reader cannot tell a dead drive from a working one with what the file carries, or the product thinker wants a shorter stall bound than the stuck bound.
   - Decided by the run for owner review. It was not put to the product thinker, because the record leaves one defensible answer.
7. **When the drive is stalled on lanes handed back one after another for one reason, it keeps opening lanes** (product thinker, 2026-11-15; formerly the second and third parts of DQ-7).
   - Question: "What counts as stalled when lanes are handed back one after another, and what does the drive do when it is?"
   - Answer: the drive keeps opening lanes. The count that makes `stalled` is the run's to propose, with its reasons, when the step is planned.
   - What it costs: agent runs spent on lanes that will fail, and each lane handed back as stuck leaves a mark, a worktree and a branch that a person clears by command, which the first scope condition counts against the design.
8. **A lane whose landing cannot find its record is set aside** (product thinker, 2026-11-15; formerly open question 4).
   - Question: "Is a lane whose landing cannot find its record handed back as stuck, or set aside?"
   - Answer: set aside, not handed back as stuck. The lab's prototype measured this lane set aside with both other lanes landing and nothing left to clear (F-9), so step 3 of the spec is dropped and the set-aside gains the fourth call site.
   - What it costs: the lane holds the drive for the bound, an hour under decision 3, and a drain that a host session steps has no set-aside, so the refusal still goes uncounted there (F-14).
9. **When every lane left is set aside, the drive does not end** (product thinker, 2026-11-15; formerly open question 5, "What is the one act when a drive ends because every lane left is set aside?").
   - Answer: the drive keeps waiting. It tries the set-aside lanes again every hour until its end time (`--until`, or its other end conditions), and the line reads `waiting`, not `stopped`. It does not read `stalled` while those hourly retries are its named next try.
   - Reconciling with the stall criterion: that criterion says a look-again time is not a named end, and the stall rule by time of decision 6 treats only the pace window's next eligible time and a rate limit's named reset as ends. This decision makes the hourly retry of set-aside lanes a named end, so a drive waiting on it reads `waiting` and not `stalled`. The planning step must reconcile the two, and says in the stall rule which look-again times are ends.
   - What it costs, which the product thinker accepted: a drive process stays running, mostly idle, until its end time. A lane whose block still stands is tried again every hour and set aside again.

## Typed Links

- builds_on: itd-2610222215485265 (`abcd drain --drive` works through the ledger from a plain terminal). That intent made the drive run without input, and this one changes how its stop, its start and its waits read to a person. Its eighth criterion and its decision 8 are kept for an interrupt: the drive still releases what it can, and this intent adds that a lane whose stop could not be applied keeps its reservation. Its press release ("it never leaves an issue held while it waits for someone who isn't coming") and its mechanism's test need amending, since decision 4 leaves a set-aside lane's reservation to lapse; the amendment is filed as iss-2611150635298371.
- built on by: itd-2611141610192096 (the terminal output, whose first live line is this intent's state line) and itd-2611141610201875 (the merge queue and the checkout).
- read by: itd-2611141507299828 (the status line), which shows the stored state and is blocked by this intent.
- related: spc-2611051628562376, the drain intent's open spec, whose step 3 keeps for the product thinker the same kind of choice that decision 5 answers for the drive.
- related: iss-2611140902531015 (the take-over from the drive's own terminal), which the first criterion answers. adr-2611050740453233 (a stop is recorded state a person sets), whose consequence that nothing resumes by itself when a stop lifts decision 1 departs from for a drive's start.
- evidence: lab-261114145658-67e1f9b (findings F-3 to F-9, F-11 and F-14 to F-17) and `.abcd/development/research/notes/drive-operations-2026-11.md`.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a person is expected to bring an interrupted drive back with the start command alone, and to learn its state from one line, because every state the lab found a start stopping on is one the drive can read for itself and the loop already knows its own wait; shown wrong if a drive a person interrupted needs any command but abcd drain --drive to take up every run the interrupt left in the checkout, if one lane's wait still stops other lanes opening, if people still read the log to learn the drive's state, or if the design needs a new command, a flag a person must remember or a stored state a person must clear
