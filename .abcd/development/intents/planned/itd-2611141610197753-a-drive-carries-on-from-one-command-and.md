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
related_intents: [itd-2611141610192096, itd-2611141610201875, itd-2611141507299828]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A drive carries on from one command, and says in one line when it cannot

## Press Release

> **A drive carries on from one command, and says in one line when it cannot.** A person starts a drive with `abcd drain --drive`. After an interrupt, with or without a rebuild of abcd in between, the same command carries on with the lanes its drain lists, and nothing else is typed for them. A lane the drive cannot take is set aside with its reason, and the drive opens the next one. While the drive runs, one line says whether it is working, waiting, stalled or stopped, since when, on what, and the one thing a person can do, or that there is nothing to do. `abcd implement status` and the board show the same line, so the answer to "is it working?" is the same wherever a person looks.

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
- When every lane left is set aside: the drive ends and the line reads `stopped`. What the one act is then is open question 5.
- Where output is not a terminal: a `state:` line each time the state changes.

### What a person never has to know

- Which process a run records as its driver, and what a take-over is, for a drive they interrupted themselves.
- That an interrupt leaves a mark, whether it settled, and what lifts it.
- A run file's schema version.
- The difference between the loop's kinds of wait.

## Mechanism

We expect a person to bring a drive back with the start command alone, because every state the lab found a start stopping on is one the drive can read for itself: a recorded driver that is the drive's own terminal, a mark its own earlier interrupt left, and a lane that has waited a bounded time on something the drive cannot end. Shown wrong if, after this ships, a drive that a person interrupted needs any command but `abcd drain --drive` to take up the lanes its drain lists, or if one lane's wait still stops other lanes opening.

We expect a person to stop asking whether the drive is working, because the loop already knows which wait it is in, since when, until when and the refusal's own remedy, and hands them to the surfaces as one value. Shown wrong if people still open the log, or ask a session to read it, to learn the drive's state.

## Scope Conditions

- The lab's own test of the whole idea: it is shown wrong by any new command, any flag a person must remember, or any stored state a person must clear. The one new stored thing, the drive's state, is overwritten by the next drive and cleared by nobody. A lane handed back as stuck leaves a mark, a worktree and a branch that a person clears by command, so open questions 3 and 4 each have an answer this condition counts against the design. <!-- cond: cond-2611141617190715 -->
- One drive per working copy on one machine, as the drain's lock already enforces. The state line is stored in the checkout the drive runs from, and a status surface run in that checkout shows it. Which other checkouts and sessions show it is the status line's question (itd-2611141507299828). <!-- cond: cond-2611141617198456 -->
- Run files this build can read. The lab tried one earlier schema version (25); a file that states a version above this build's is refused by the read before any claim, as today. <!-- cond: cond-2611141617198101 -->
- A process table the drive can read. Telling the drive's own terminal from another live session reads process ancestry and the session markers; where it cannot be read the drive leaves the run, as today, and the lane is set aside at the bound. <!-- cond: cond-2611141617194898 -->
- A person's own stop is outside what a start lifts or takes over (decision 1). The single command is claimed for what the lab probed: an interrupt by signal, settled or not, and a run file at an earlier schema version written by the lab's test, which stands in for a rebuild. A closed terminal, a kill, a reboot and a file left by a real older build were not probed, and nothing here is evidence for them until a test or the live proof covers each. <!-- cond: cond-2611141617192337 -->
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
- **Given** a drive that was interrupted, whether or not its stop settled and whether or not abcd was rebuilt since, **when** `abcd drain --drive` is typed again, **then** the drive carries on with every lane its drain still lists that the interrupt left, lanes parked on a pull request included, and its output names no other command for them. **Given** a run an interrupt parked that the drain no longer lists, **then** the drive leaves it and names it once with the one command that takes it over, which is safe to type in the drive's own terminal (decision 2).
  - Evidence: lab-261114145658-67e1f9b, F-4, F-8 and F-15.
- **Given** a run whose recorded driver is alive and is not the starting drive's own terminal (another terminal, or a host session), **when** the drive starts, **then** it does not take the run, says who holds it, and goes on to other work. **Given** a run a person's own stop halted, **then** the drive does not take it either, whether or not the stop has been lifted, and names the take-over that is the person's to type (decision 1).
  - Evidence: lab-261114145658-67e1f9b, F-8 (the three refutation fixtures) and F-9.
- **Given** a lane whose run the drive cannot claim or take up, or whose wait the stuck bounds do not count, for a reason the drive cannot end, **when** that has lasted the stuck bound's minutes, **then** the drive says the lane is set aside and why, opens the next lane, and ends only when every lane left is set aside (decision 3). What becomes of the lane's reservation waits on open question 1, whether the same drive tries the lane again on open question 2, and what the line gives as the one act when the drive ends on set-aside lanes on open question 5.
  - Evidence: lab-261114145658-67e1f9b, F-5 and F-9.
- **Given** a landing's reservation refusal that names no lane, because the issue's record is in no status folder of the checkout, **when** the drive meets it, **then** the lane no longer holds the drive without bound: it is either counted as any other refusal of that lane and handed back as stuck, or set aside, as open question 4 is answered. A refusal built without a lane that is the machine's and not the lane's (the review set, the configuration, the remote's identity) is not counted against any lane.
  - Evidence: lab-261114145658-67e1f9b, F-9 and F-14.
- **Given** a drive in any state, **when** a person looks at its terminal, runs `abcd implement status` or reads the board in the drive's checkout, **then** one line says `working`, `waiting`, `stalled` or `stopped`; since when; on what; and one act or that there is nothing to do; and the three places show the same line, composed once from one stored value (decision 6).
  - Evidence: lab-261114145658-67e1f9b, F-7, F-11 and F-17.
- **Given** a stored state whose drive ended without writing a last line, **when** a status surface reads it, **then** the line says `stopped`, not heard from since the time it was last written, and never `working` (decision 6).
  - Evidence: lab-261114145658-67e1f9b, F-11 and F-17.
- **Given** a wait with no named end, **when** it has lasted the stuck bound's minutes with no lane moving, **then** the line says `stalled` and its `since` is when anything last moved, whatever reasons the waits gave in between. A wait with a named end, which is the pace window's next eligible time or a rate limit's named reset, is never `stalled` before that end. An outage's next try, a look at origin, a limit that names no reset and a look-again time are not ends, so an outage reads `stalled` once it has lasted the bound (decisions 3 and 6).
  - Evidence: lab-261114145658-67e1f9b, F-11.
- **Given** lanes handed back one after another for the same reason, **when** the count reaches the bound, **then** the line says `stalled` and names the reason, instead of `working`. The count, and whether the drive then goes on opening lanes or pauses, wait on open question 3; this criterion cannot be written more exactly until it is answered.
  - Evidence: lab-261114145658-67e1f9b, F-11 (events 1 and 2 of the incident note).
- **Given** an interrupt, **when** the drive ends, **then** every lane it held either carries the stop's mark or keeps its reservation, whatever schema version the run's file states.
  - Evidence: lab-261114145658-67e1f9b, F-16.

## Open Questions

Each question below is the product thinker's to decide, and none is answered here. Questions 1 to 3 were first answered by the run and were withdrawn after the design review of 2026-11-14, because each answer differs in what a person sees or must do and no evidence settles it. Questions 1 to 3 are quoted as the lab met them, with the answers it can defend and what each costs. Questions 4 and 5 came out of that review.

1. **DQ-4. "What becomes of a set-aside lane's reservation?"** (F-9)
   - Left to lapse (the prototype): no push. Cost: the issue is hidden from every other session for up to two hours. Amendment it needs: itd-2610222215485265's press release says the drive never leaves an issue held while it waits for someone who is not coming, and its mechanism is shown wrong if a drive ends with an issue still reserved and no lane working on it. Both would have to say that a set-aside lane's reservation may stand until its lease lapses.
   - Released at once with a note: others may take the issue. Cost: a push; the next drive must reserve again and may find it taken, with the lane's branch left behind. Amendment it needs: none to that press release or mechanism. That intent's decision 8, which releases a lane's reservation on a stop, would be extended to a set-aside, and would have to say what happens where the run is recorded as another live session's and the reservation may not be the drive's to release.
   - Renewed while the drive runs: the lane is safe to resume. Cost: the issue is hidden for as long as the drive lives, which is the complaint of the incident note's event 10. Amendment it needs: the press release sentence and the mechanism's test above would both be reversed for a set-aside lane.

2. **DQ-5. "Does a set-aside lane get another try in the same drive?"** (F-9)
   - No, the next start tries it (the prototype). Cost: a block that clears in a minute still waits for a restart.
   - Yes, once the other lanes are done or after a longer wait. Cost: a second timer, and a lane that flaps.
   - It is handed back as stuck instead. Cost: a tombstone a person must clear before any drain takes the issue again, which is twenty commands for twenty lanes.

   This is the same kind of choice that step 3 of spc-2611051628562376 keeps for the product thinker for a lane still paused when an outage runs out: hand it back as stuck, or leave it for the next drive. Neither record settles the other.

3. **DQ-7, its second and third parts. "What counts as stalled when lanes are handed back one after another, and what does the drive do when it is?"** (F-11)
   - Stalled after N lanes in a row handed back for one reason. Cost: this rule is the only way the incident note's events 1 and 2 show at all, and N is a guess: the lab measured no count, and the note says that eleven of the twenty stuck lanes fit one cause and eight another, not that they came one after another.
   - When stalled on repeated hand-backs, keep opening lanes. Cost: agent runs spent on lanes that will fail, and each lane handed back as stuck leaves a mark, a worktree and a branch that a person clears by command, which this intent's first scope condition counts against the design.
   - When stalled on repeated hand-backs, pause. Cost: a drive that stops itself on a coincidence, and a pause that something has to lift.

4. **"Is a lane whose landing cannot find its record handed back as stuck, or set aside?"** (F-9, F-14)
   - Counted as any other refusal and handed back as stuck, which is the lab brief's fifth criterion and the first step of its build order. Cost: the lane's finished and passed work is discarded with the hand-back, and the lane leaves a mark, a worktree and a branch that a person clears by command, which this intent's first scope condition counts against the design. The lab did not build this.
   - Set aside, which is what the lab's prototype did and measured for this lane: set aside at +36m21s under `--until` and at +30m0s without it, both other lanes landed, and nothing was left to clear (F-9). Cost: the lane holds the drive for the bound, an hour under decision 3, where the stuck count would act within three looks; a drain that a host session steps has no set-aside, so the refusal still goes uncounted there (F-14); and questions 1 and 2 then apply to this lane.

5. **"What is the one act when a drive ends because every lane left is set aside?"** (F-9; the lab did not try a second start over a block that still stands)
   - What is known: the drive ends, the line reads `stopped`, and today's end for passed lanes names `abcd drain --drive` as the way on. A restart tries each set-aside lane again, and where the block still stands it waits the bound again and ends again. The line has room for one act, and no record says what that act is when the same command would only repeat the wait. No answer is proposed here.

## Decisions

Recorded 2026-11-14 within an autonomous run, on the product thinker's delegation (dcn-2611141613199819). The product thinker asked for this intent to be planned without them and for every design decision the run takes to be recorded for their review. Each question is quoted as the lab met it. Each answer is the one the lab's evidence best supports and that changes least for a person. Each is decided by the run for owner review and is open to reversal at that review. A design review on the same day found three of the six answers were not the run's to give: decisions 4 and 5 and part of decision 6 are withdrawn and are open questions 1 to 3. Decisions 1, 2 and 3 and the rest of decision 6 remain.

1. **A person's own stop stays the person's to take over.**
   - Question (DQ-1): "After a person's own stop is lifted, does the next start take up the runs it halted?"
   - Answer: no. It stays a typed take-over, as now. A start takes over only what an interrupt to a drive left.
   - Alternatives and what each costs: the start takes them, which is one command after the lift, and the lift then becomes the go, so the person's stop loses its second deliberate act. The lift takes them over itself, which is one command and keeps the act the person's, and `abcd implement stop --lift` gains a second effect. The answer taken costs two commands, and needs a take-over typed in the drive's terminal to be safe, which the first criterion delivers.
   - Evidence: F-8 and the probe p1-claim-earlier-runs-2, where the prototype leaves a lifted person's stop alone on purpose and the fixture passes. adr-2611050740453233 states the same consequence: a run that took a stop needs a take-over before it steps again, so nothing resumes by itself when a stop lifts.
   - Reversed if: the product thinker wants the lift to be the go, or people who lift a stop are seen to start the drive and wait on a run that needs the take-over.
   - Decided by the run for owner review.
2. **A start takes the runs its drain's state lists.**
   - Question (DQ-2): "Which runs are the drive's to take at its start?"
   - Answer: those its drain's state lists, which is what the prototype does.
   - Alternatives and what each costs: every run in the checkout whose mark was set by a signal to a drive, which closes the gap below, and has the start act on runs the current drain never opened; the lab did not build or test it. The answer taken leaves one case: a run an interrupt parked on its pull request, whose drain has since been replaced by a new one, still waits for a typed take-over. The drive names it once with that command, and the first criterion makes the command safe in the drive's terminal.
   - Evidence: F-8 and the probes p1-claim-earlier-runs-2 and p1-existing-tests-2. The second form of the prototype passes all six fixtures, and the repository's own test `TestANewDrainNamesARunASignalParked` shows the case it leaves.
   - Reversed if: a person has to type a take-over for a run their own interrupt parked after this ships. That is the lab's own test of the claim, and the wider answer is then built and tested against the same fixtures.
   - Decided by the run for owner review.
3. **A lane may keep the drive waiting for the stuck bound's minutes, and the same figure is the stall bound.**
   - Question (DQ-3): "How long may one lane keep the drive waiting before it is set aside?"
   - Answer: the stuck bound's minutes, read from the existing `pace.stuck_minutes` (60 as bundled). It is one figure for three things: a lane handed back as stuck, a lane set aside, and a wait that reads `stalled`.
   - Alternatives and what each costs: 30 minutes, the prototype's constant, which halves the idle time per blocked lane and adds a second figure to explain beside the stuck bound. A new `pace` key, which is a setting to document. The answer taken costs up to an hour idle per blocked lane, and it is already a `pace` key, so a person who wants less sets the one key they would set for a stuck lane.
   - Evidence: F-9 and the probe p2-set-aside, where the lane is set aside at +30m0s and +36m21s and both other lanes then land. The mechanism is the prototype's and only the figure differs. The lab measured nothing that depends on 30 against 60 except the idle time. This departs from the prototype's figure, which the lab itself marks as the product thinker's to decide.
   - Reversed if: an hour idle per blocked lane is seen to cost a drive more than a second figure costs a reader.
   - Decided by the run for owner review.
4. **Withdrawn 2026-11-14 after the design review.** The run had answered DQ-4 by leaving a set-aside lane's reservation as it stands. That contradicts the press release and the mechanism of itd-2610222215485265, which are the product thinker's, so it is open question 1.
5. **Withdrawn 2026-11-14 after the design review.** The run had answered DQ-5 by leaving a set-aside lane to the next start. That is the same kind of choice spc-2611051628562376 keeps for the product thinker in its step 3, so it is open question 2.
6. **The state is one small file in the local tier, and a wait with no named end is stalled at the stuck bound's minutes.**
   - Question (DQ-7, its first part): "Where is the state line kept, what counts as stalled, and what does the drive do when it is?"
   - Answer, where it is kept: in a file in the checkout's local tier beside the drive's log, written whole by the drive each time a field changes. It is not part of a run's state, and nobody clears it. It carries the drive's process id and start time and the time it was written, so a reader tells a drive that ended without a last line and shows it as `stopped`. How a reader tells is the spec's to say, and it is not the lab's: the lab's design reads "the drive's process gone", and nothing was stored in the lab.
   - Answer, what counts as stalled by time: a wait with no named end that has lasted the stuck bound's minutes with no lane moving. This follows decision 3's one figure. The named ends are the pace window's next eligible time and a rate limit's named reset; an outage has none.
   - Withdrawn 2026-11-14 after the design review: the count of lanes handed back in a row that reads `stalled`, and the answer that a stalled drive keeps opening lanes. Both rest on no measurement and are open question 3.
   - Alternatives and what each costs: a stall bound of 30 minutes, the prototype's, which is a second figure beside decision 3's. The lab gives no other place to keep the line.
   - Evidence: F-11 and the probes p4-state-line-mapping and p4-pinned-lines-with-times for the line's fields and the time rule, with the prototype's first error (a stall that never registered while the wait's reason alternated) written into the criterion. F-17 for the surfaces. The stored form rests on no measurement: the prototype stored nothing.
   - Reversed if: a reader cannot tell a dead drive from a working one with what the file carries, or the product thinker wants a shorter stall bound than the stuck bound.
   - Decided by the run for owner review.

## Typed Links

- builds_on: itd-2610222215485265 (`abcd drain --drive` works through the ledger from a plain terminal). That intent made the drive run without input, and this one changes how its stop, its start and its waits read to a person. Its eighth criterion and its decision 8 are kept for an interrupt: the drive still releases what it can, and this intent adds that a lane whose stop could not be applied keeps its reservation. Whether its press release ("it never leaves an issue held while it waits for someone who isn't coming") and its mechanism's test stand as written depends on open question 1, and two of that question's three answers need them amended.
- built on by: itd-2611141610192096 (the terminal output, whose first live line is this intent's state line) and itd-2611141610201875 (the merge queue and the checkout).
- read by: itd-2611141507299828 (the status line), which shows the stored state and is blocked by this intent.
- related: spc-2611051628562376, the drain intent's open spec, whose step 3 keeps for the product thinker the same kind of choice as open question 2.
- related: iss-2611140902531015 (the take-over from the drive's own terminal), which the first criterion answers. adr-2611050740453233 (a stop is recorded state a person sets), which decision 1 keeps.
- evidence: lab-261114145658-67e1f9b (findings F-3 to F-9, F-11 and F-14 to F-17) and `.abcd/development/research/notes/drive-operations-2026-11.md`.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a person is expected to bring an interrupted or stalled drive back with the start command alone, and to learn its state from one line, because every state the lab found a start stopping on is one the drive can read for itself and the loop already knows its own wait; shown wrong if a drive a person interrupted needs any command but abcd drain --drive to take up the lanes its drain lists, if one lane's wait still stops other lanes opening, if people still read the log to learn the drive's state, or if the design needs a new command, a flag a person must remember or a stored state a person must clear
