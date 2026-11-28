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

> "When I am asked whether the drive is working, I run the status command and read one line: stalled since 16:40 on an outage, the runner has not started. I do not read the log for the last line that is not a wait." — Nia, a technical facilitator (they/them)

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

- **Given** a run that was taken over from a terminal, **when** `abcd drain --drive` is typed in that terminal and the take-over claim's conditions hold there (nothing but shells runs beneath the recorded driver besides the drive, and every session marker can be read), **then** the drive takes the run and its lanes go on, with nothing else typed. **Given** the same run where a live pipe or helper runs in that terminal beside the drive, or a session marker cannot be read, **then** the drive leaves that run waiting and names the act that frees it (end the process, close the terminal, or repair or remove the marker file); the take-over is not widened to take it (decision 10).
  - Evidence: lab-261114145658-67e1f9b, F-3 and F-8; the claim's conditions as pull request #1544 put them on the default branch.
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
- **Given** lane hand-backs one after another that share one key (the hand-back's kind, the stage the lane was at and the check of its last logged refusal), **when** three come in a row, the figure read from `pace.stuck_refusals`, **then** the line says `stalled`, since the first hand-back of the run, and names the count, the stage and the kind, instead of `working`. The drive then keeps opening lanes (decision 7). A hand-back at the judgement, by the rule or at a reservation neither counts towards the run nor clears it; a lane that reaches its pull request, or a lane hand-back with another key, clears it (decision 11).
  - Evidence: lab-261114145658-67e1f9b, F-11 (events 1 and 2 of the incident note).
- **Given** an interrupt, **when** the drive ends, **then** every lane it held either carries the stop's mark or keeps its reservation, whatever schema version the run's file states.
  - Evidence: lab-261114145658-67e1f9b, F-16.

## Open Questions

All resolved by the product thinker on 2026-11-15 (decisions 4, 5, 7, 8 and 9). The spec's open note on the take-over claim's conditions was answered on 2026-11-25 (decision 10).

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
   - What it needs: itd-2610222215485265 must be amended. Its press release says the drive never leaves an issue held while it waits for someone who is not coming, and its mechanism is shown wrong if a drive ends with an issue still reserved. Both are to say that a set-aside lane's reservation may stand until its lease lapses. That intent is not edited here; the amendment is filed as iss-2611150635298371, and was made on 2026-11-25 as that intent's decision 12.
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
   - Answer: the drive keeps opening lanes. The count that makes `stalled` was the run's to propose, with its reasons, when the step was planned; it is decision 11.
   - What it costs: agent runs spent on lanes that will fail, and each lane handed back as stuck leaves a mark, a worktree and a branch that a person clears by command, which the first scope condition counts against the design.
8. **A lane whose landing cannot find its record is set aside** (product thinker, 2026-11-15; formerly open question 4).
   - Question: "Is a lane whose landing cannot find its record handed back as stuck, or set aside?"
   - Answer: set aside, not handed back as stuck. The lab's prototype measured this lane set aside with both other lanes landing and nothing left to clear (F-9), so step 3 of the spec is dropped and the set-aside gains the fourth call site.
   - What it costs: the lane holds the drive for the bound, an hour under decision 3, and a drain that a host session steps has no set-aside, so the refusal still goes uncounted there (F-14).
9. **When every lane left is set aside, the drive does not end** (product thinker, 2026-11-15; formerly open question 5, "What is the one act when a drive ends because every lane left is set aside?").
   - Answer: the drive keeps waiting. It tries the set-aside lanes again every hour until its end time (`--until`, or its other end conditions), and the line reads `waiting`, not `stopped`. It does not read `stalled` while those hourly retries are its named next try.
   - Reconciling with the stall criterion: that criterion says a look-again time is not a named end, and the stall rule by time of decision 6 treats only the pace window's next eligible time and a rate limit's named reset as ends. This decision makes the hourly retry of set-aside lanes a named end, so a drive waiting on it reads `waiting` and not `stalled`. The planning step must reconcile the two, and says in the stall rule which look-again times are ends.
   - What it costs, which the product thinker accepted: a drive process stays running, mostly idle, until its end time. A lane whose block still stands is tried again every hour and set aside again.

10. **The first criterion is narrowed to the take-over claim's conditions** (product thinker, 2026-11-25; formerly the spec's open note).
    - Question: the first criterion promised that a drive typed in the terminal a run was taken over from takes the run with nothing else typed, while the claim pull request #1544 put on the default branch takes such a run only when nothing but shells runs beneath the recorded driver besides the drive and every session marker can be read. Is the criterion narrowed to the claim's conditions, or is the claim widened?
    - Answer: the criterion is narrowed. A live pipe or helper in the old terminal, or a session marker that cannot be read, leaves that run waiting, and the refusal names the act that frees it. The take-over is not widened.
    - What it costs: in those cases the run waits for the person, while step 3 of the spec sets its lane aside at the bound so other lanes open. The press release's single command stands for the runs an interrupt left and for a take-over run whose terminal meets the claim's conditions.

11. **The hand-back stall count is three in a row with one key, read from `pace.stuck_refusals`** (technical facilitator, 2026-11-25, on delegation; decision 7 left the count to the run to propose).
    - Answer: three lane hand-backs in a row that share one key, the hand-back's kind, the lane's stage and the check of its last logged refusal, make the line read `stalled`. The reason's text is not in the key, since it carries run ids, counts and minutes. Hand-backs at the judgement, by the rule or at a reservation neither count nor clear the run, because they are the drain working; a lane that reaches its pull request clears it, and so does a lane hand-back with another key.
    - Why three, and why that key: in the incident note's first event eleven of twenty stuck lanes failed one way over about two days, and at three the line turns after about fifteen agent runs; at two, neighbouring issues failing one check is common, and at five two more lanes fail first. Three identical refusals already make a lane stuck (`pace.stuck_refusals`), and three identical hand-backs making the drive stalled is the same idea one level up, so no new setting is added.
    - What it costs: a person who raises `pace.stuck_refusals` for flaky lanes also makes the line slower to turn. A false `stalled` costs one look, since the drive carries on.

## Typed Links

- builds_on: itd-2610222215485265 (`abcd drain --drive` works through the ledger from a plain terminal). That intent made the drive run without input, and this one changes how its stop, its start and its waits read to a person. Its eighth criterion and its decision 8 are kept for an interrupt: the drive still releases what it can, and this intent adds that a lane whose stop could not be applied keeps its reservation. Its press release ("it never leaves an issue held while it waits for someone who isn't coming") and its mechanism's test need amending, since decision 4 leaves a set-aside lane's reservation to lapse; the amendment is filed as iss-2611150635298371.
- built on by: itd-2611141610192096 (the terminal output, whose first live line is this intent's state line) and itd-2611141610201875 (the merge queue and the checkout).
- read by: itd-2611141507299828 (the status line), which shows the stored state and is blocked by this intent.
- related: spc-2611051628562376, the drain intent's open spec, whose step 3 keeps for the product thinker the same kind of choice that decision 5 answers for the drive.
- related: iss-2611140902531015 (the take-over from the drive's own terminal), which the first criterion answers. adr-2611050740453233 (a stop is recorded state a person sets), whose consequence that nothing resumes by itself when a stop lifts decision 1 departs from for a drive's start.
- evidence: lab-261114145658-67e1f9b (findings F-3 to F-9, F-11 and F-14 to F-17) and `.abcd/development/research/notes/drive-operations-2026-11.md`.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-da2f8a9cec53 -->
Fidelity review — receipt rcp-da2f8a9cec53 (verifier intent-auditor claude-fable-5-2).

Provenance: intent-auditor@claude-fable-5-2 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:6437f40245447b83f983e70e096a081b8c136aeb40244eefedbf445d170fefec
Input attestations: diff:origin/main...f185082fa73bdf9c74a0f620563948c570976731 (spec step 5)@sha256:adbe601b4676a7e4e7b287eb928b6a3bb103767182d4868dee1a548364d4d35f; commit:f185082fa73bdf9c74a0f620563948c570976731 (tree read for steps 1 to 4, on origin/main 2de8cb51548a4bcc058a54eeeb847a48b8d68f13)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The claim takes a run its own terminal took over and a drive's start from that terminal lands three lanes with no other command; with a helper beside the drive or an unreadable marker the run is left and the refusal names the act (end the process, close the terminal, remove or repair the file), the claim unwidened.
  evidence: internal/core/implement/loop/control_terminal_test.go:101 — "func TestADriveTakesARunItsOwnTerminalTookOver(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:234 — "func TestADrivesStartFromItsOwnTerminalLandsThreeLanes(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:494 — "func TestARunLeftForAProcessBesideTheDriveNamesTheAct(t *testing.T) {"
  evidence: internal/core/implement/loop/control_terminal_test.go:569 — "func TestADriveWaitsWhereASessionMarkerCannotBeRead(t *testing.T) {"
  evidence: internal/core/implement/loop/control.go:449 — "end process %d (%s), and this drive's next look takes the run where nothing else runs beneath it; or close the terminal the run was taken over from"
  evidence: internal/core/implement/loop/control.go:434 — "remove or repair each file in %s that cannot be read as a session marker, %s first"
- ac-2 — MET_WITH_CONCERNS: takeEarlier takes interrupted runs at the start, settled or unsettled, at the earlier schema version, parked on a pull request, and with the drain replaced, each asserted to name no typed take-over. Concern: the start does not take EVERY interrupted run in the checkout as the second clause says: a run that builds an intent and a run naming no driver with no sign its session ended are left, and the lines for them name other commands (`abcd implement resume --take`, `abcd implement step --run`). The earlier version tested is SchemaVersion-1, not the lab's 25.
  evidence: internal/core/implement/loop/terminal_claim_test.go:251 — "func TestAnUnsettledSignalMarkOnAnEarlierVersionIsSettledAndTaken(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:293 — "func TestARunParkedUnderASignalIsTakenAndFinishedOnceMerged(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:371 — "func TestARunWhoseDrainWasReplacedIsTakenWithTheRest(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal.go:2805 — "func (t *terminal) takeEarlier(ctx context.Context) (string, error) {"
  evidence: internal/core/implement/loop/terminal.go:2876 — "builds an intent, which a drive does not take up: `abcd implement resume --take %s` takes it over"
  evidence: internal/core/implement/loop/terminal.go:3027 — "names no driver and carries no mark, so whether the session that let it go has ended cannot be told, and it is left as it stands; `abcd implement step --run %s` claims it and steps it on"
  evidence: internal/core/implement/loop/terminal_claim_test.go:732 — "func TestAnIntentBuildIsNotTakenAtTheStart(t *testing.T) {"
- ac-3 — MET: A run whose live driver is another terminal or a host session is not taken, the start names the holding process and lands the other lanes; a person's stop that stands is left and named, and once lifted the next start takes the run up and reserves it again.
  evidence: internal/core/implement/loop/terminal_claim_test.go:396 — "func TestAHaltedRunWhoseLiveDriverIsInAnotherTerminalIsLeftAndNamed(t *testing.T) {"
  evidence: internal/core/implement/loop/control_terminal_test.go:140 — "func TestADriveUnderASessionDoesNotTakeTheRun(t *testing.T) {"
  evidence: internal/core/implement/loop/control_terminal_test.go:219 — "func TestADriveContendsWhereItsTerminalIsNotTheDriver(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:351 — "func TestAPersonsStopThatStandsIsLeftAndNamed(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:327 — "func TestALiftedPersonsStopIsTakenUp(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: blocked() sets a lane aside at pace.stuck_minutes with its reason, the other lanes land, the lane is tried again once they are done, and with every lane left set aside the drive waits and retries hourly to its bound. Concern: the reservation is NOT left to lapse for every set-aside lane as the criterion says; a lane set aside because a live driver holds its run keeps its passive renewals (dcn-2611270753395749, recorded by the building run, not a product-thinker ruling amending decision 4).
  evidence: internal/core/implement/loop/terminal.go:3147 — "func (t *terminal) blocked(runID, site, why string) bool {"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:118 — "func TestARunWithALiveDriverInAnotherTerminalIsSetAside(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:197 — "func TestABlockThatClearsJustAfterTheSetAsideLands(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:418 — "func TestADrainWithACapOfOneWaitsOnItsSetAsideLane(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal.go:151 — "const terminalSetAsideRetry = time.Hour"
  evidence: internal/core/implement/loop/terminal.go:3351 — "func (t *terminal) awaitSetAside(ctx context.Context) string {"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:147 — "a lane a live driver holds is left to lapse: set aside %v, note %v, LeftToLapse %v, renewed by implement status %v"
  evidence: .abcd/work/decisions/dcn-2611270753395749-a-lane-a-drive-sets-aside-because-a-live.md:6 — "keeps its reservation's passive renewals"
- ac-5 — MET: The landing's lane-less missing-record refusal is marked recordAbsent and routed to blocked(), so the lane is set aside at the bound under --until and with none, stays in progress (not handed back), and the other lanes land; a review-set or configuration refusal that names no lane is logged on no lane and sets nothing aside.
  evidence: internal/core/implement/loop/terminal.go:2354 — "if r.recordAbsent && t.blocked(runID, blockRecord, r.Reason) {"
  evidence: internal/core/implement/loop/reservation.go:1466 — "rf.recordAbsent = true"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:325 — "func TestTheLandingsMissingRecordIsSetAside(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_setaside_test.go:456 — "func TestAMachineRefusalThatNamesNoLaneSetsNothingAside(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: DriveLine composes `<state> <elapsed> · on <what> · you: <act>` once from the stored DriveState, and the terminal's state: line, `abcd implement status` and the board's facilitator view and --json are asserted byte for byte equal. Concern: the bare board, the default product view, shows only the state word and its until (`building: drive waiting`), with no since, no on-what and no act, so a person reading the board without `--view facilitator` does not see the line the criterion promises (dcn-2611280725169564, recorded by the building run).
  evidence: internal/core/implement/loop/drivestate.go:190 — "func DriveLine(s DriveState, now time.Time) string {"
  evidence: internal/surface/cli/drain_drive_state_test.go:152 — "func TestTheTerminalTheStatusAndTheBoardShowOneDriveLine(t *testing.T) {"
  evidence: internal/surface/cli/drain_drive_state_test.go:190 — "func TestTheStatusCommandAndTheBareBoardPrintTheDrivesLine(t *testing.T) {"
  evidence: internal/surface/cli/drain_drive_state_test.go:215 — "the product view does not show the drive's word alone"
  evidence: internal/core/board/board.go:378 — "func driveLine(d *Drive, f Frame, sym symbols) string {"
  evidence: .abcd/work/decisions/dcn-2611280725169564-the-board-s-product-view-shows-a-drive-s.md:6 — "The board's product view shows a drive's state as its word and its until alone"
- ac-7 — MET_WITH_CONCERNS: ReadDriveState judges the recorded process by id and start time against the process table and a gone drive reads `stopped ... not heard from since <written>`, never working, on both status surfaces. Concern: where the process table cannot be read or the record names no start, the reader falls back to the drive's own next-write time, so a dead drive still reads as stored (working included) until that time passes; a far-future next write is taken at its word (dcn-2611280815384891).
  evidence: internal/core/implement/loop/drivestate_read.go:69 — "if s.State != DriveStopped && driveGone(s.Process, s.Next, now) {"
  evidence: internal/core/implement/loop/drivestate_read_test.go:82 — "func TestAGoneDriveReadsStoppedNeverWorking(t *testing.T) {"
  evidence: internal/surface/cli/board_drive_test.go:81 — "func TestTheBoardReadsAGoneDriveAsStopped(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_read.go:127 — "return next.IsZero() || now.After(next)"
  evidence: internal/core/implement/loop/drivestate_read_test.go:134 — "func TestADriveThatCannotBeJudgedByItsProcessReadsByItsNextWrite(t *testing.T) {"
  evidence: .abcd/work/decisions/dcn-2611280815384891-a-drive-s-state-reader-takes-the-stored.md:6 — "a record with no process start and a far-future next write reads working until that time"
- ac-8 — MET: stallWord reads waiting before a named end and stalled once the stuck bound has passed since the wait's start, whatever the reasons; since survives alternating reasons; an outage and a limit with no reset carry no until and read stalled past the bound; the hourly set-aside retry is a named end and reads waiting however long.
  evidence: internal/core/implement/loop/drivestate.go:157 — "func stallWord(start, until time.Time, bound time.Duration, now time.Time) string {"
  evidence: internal/core/implement/loop/drivestate_test.go:67 — "func TestADriveSinceSurvivesAlternatingReasons(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_test.go:46 — "func TestADriveWaitIsNeverStalledBeforeItsNamedEnd(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_test.go:424 — "func TestAnOutageReadsStalledFromItsFirstFailure(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_test.go:479 — "func TestARateLimitSetsAnUntilOnlyWhereItNamesAReset(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_test.go:570 — "func TestEveryLaneSetAsideWaitsUntilTheRetry(t *testing.T) {"
- ac-9 — MET: handBackStreak counts lane hand-backs sharing kind, stage and check; at pace.stuck_refusals (three bundled) the line reads stalled since the first, on `3 lanes handed back in a row, stuck at validate (lint)`; judgement, rule and reservation hand-backs neither count nor clear, a landed lane or another key clears; the streak only overlays the displayed state, so the drive keeps opening lanes.
  evidence: internal/core/implement/loop/drivestate_test.go:240 — "func TestTheHandBackStreakReadsStalled(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_test.go:591 — "func TestThreeLaneHandBacksInARowReadStalled(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate.go:357 — "func (s handBackStreak) over(st DriveState, bound int) DriveState {"
  evidence: internal/core/implement/loop/drivestate.go:334 — "func (s handBackStreak) stalled(bound int) bool {"
- ac-10 — MET: An interrupt applies the stop (mark, halted) to a run the drive drives, at the earlier version and at this one; a run it could not claim gets no mark and keeps its reservation, every member's included, with the lapse time said, and the next start reads it as reserved.
  evidence: internal/core/implement/loop/terminal_stop_test.go:606 — "func TestAnInterruptAppliesTheStopToAnEarlierVersionsRunItsTerminalDrove(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_stop_test.go:633 — "func TestAnInterruptKeepsTheHoldOfARunItCouldNotClaim(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_stop_test.go:672 — "func TestAnInterruptKeepsEveryMembersHold(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_stop_test.go:698 — "func TestAnInterruptAtThisVersionIsUnchanged(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal.go:4011 — "func (t *terminal) keepUnstopped(runID string) bool {"

Gap audit:
- honoured:
  - After an interrupt, with or without a rebuild, the same command carries on with the lanes its drain lists and nothing else is typed for them.
    evidence: internal/core/implement/loop/terminal_claim_test.go:234 — "func TestADrivesStartFromItsOwnTerminalLandsThreeLanes(t *testing.T) {"
    evidence: internal/core/implement/loop/terminal_claim_test.go:251 — "func TestAnUnsettledSignalMarkOnAnEarlierVersionIsSettledAndTaken(t *testing.T) {"
  - A lane the drive cannot take is set aside with its reason, and the drive opens the next one.
    evidence: internal/core/implement/loop/terminal.go:3147 — "func (t *terminal) blocked(runID, site, why string) bool {"
    evidence: internal/core/implement/loop/terminal_setaside_test.go:325 — "func TestTheLandingsMissingRecordIsSetAside(t *testing.T) {"
  - One line says working, waiting, stalled or stopped, since when, on what, and the one act or nothing to do; the terminal and `abcd implement status` show the same line.
    evidence: internal/core/implement/loop/drivestate.go:190 — "func DriveLine(s DriveState, now time.Time) string {"
    evidence: internal/surface/cli/drain_drive_state_test.go:152 — "func TestTheTerminalTheStatusAndTheBoardShowOneDriveLine(t *testing.T) {"
  - When every lane left is set aside the drive does not end; it retries hourly and reads waiting.
    evidence: internal/core/implement/loop/terminal.go:3351 — "func (t *terminal) awaitSetAside(ctx context.Context) string {"
    evidence: internal/core/implement/loop/drivestate_test.go:570 — "func TestEveryLaneSetAsideWaitsUntilTheRetry(t *testing.T) {"
  - Where output is not a terminal, a `state:` line is printed each time the state changes.
    evidence: internal/surface/cli/drain_drive.go:912 — "say("state: " + loop.DriveLine(s, s.Written))"
    evidence: internal/surface/cli/drain_drive_state_test.go:55 — "func TestADriveStateLineIsPrintedOnAChangeAndNotOtherwise(t *testing.T) {"
  - An interrupt leaves every lane the drive held marked or reserved, at any schema version the build reads.
    evidence: internal/core/implement/loop/terminal_stop_test.go:633 — "func TestAnInterruptKeepsTheHoldOfARunItCouldNotClaim(t *testing.T) {"
- diverged:
  - The board shows the same line: delivered in the facilitator view and --json only; the default product view shows the state word and its until alone.
    evidence: internal/core/board/board.go:378 — "func driveLine(d *Drive, f Frame, sym symbols) string {"
    evidence: commands/drain.md:181 — "the product thinker's view shows only the state and when its wait ends"
  - A set-aside lane's reservation is left to lapse: delivered only for a hold no one works; a lane a live driver holds keeps its renewals.
    evidence: internal/core/implement/loop/terminal.go:3203 — "func (t *terminal) setLaneAside(s SetAsideLane, again bool) {"
    evidence: .abcd/work/decisions/dcn-2611270753395749-a-lane-a-drive-sets-aside-because-a-live.md:6 — "Only a hold no one works is left to lapse"
  - The same command carries on with every run the interrupt left in the checkout: delivered for issue runs; an intent's build and a driverless run with no sign its session ended are left, and their lines name another command.
    evidence: internal/core/implement/loop/terminal.go:2876 — "`abcd implement resume --take %s` takes it over"
    evidence: internal/core/implement/loop/terminal.go:3027 — "`abcd implement step --run %s` claims it and steps it on"
  - A drive that ended without a last line never reads working: delivered where the process table can be read; otherwise the reader trusts the drive's next-write time.
    evidence: internal/core/implement/loop/drivestate_read.go:127 — "return next.IsZero() || now.After(next)"
  - The one new stored thing is the drive's state: the delivery also stores a left-to-lapse note per reservation.
    evidence: internal/core/claim/renewdue.go:203 — "func LeaveToLapse(root, id, commit string) error {"
- missing:
  - Evidence for a closed terminal, a reboot and a run file left by a real older build: no delivered test covers them, and the live proof is still open.
    evidence: .abcd/development/intents/shipped/itd-2611141610197753-a-drive-carries-on-from-one-command-and.md:77 — "A closed terminal, a kill, a reboot and a file left by a real older build were not probed"
  - A lane left for an unreadable process table being set aside at the bound: no test in the set-aside suite exercises it.
    evidence: internal/core/implement/loop/terminal_setaside_test.go:118 — "func TestARunWithALiveDriverInAnotherTerminalIsSetAside(t *testing.T) {"

Scope-condition dispositions:
- cond-2611141617190715 — narrowed: No command or flag was added by step 5 and none by the step commits, and nothing stored must be cleared by a person; but the drive's state is no longer the one new stored thing: a left-to-lapse note per reservation is also stored in the checkout, ended by the next reservation pushed for the id, and lines for runs outside the take rules name existing commands.
  narrowing: Holds as no new command, no flag to remember and no stored state a person must clear; does not hold as 'the one new stored thing', since the delivery also stores left-to-lapse notes beside the drive's state.
  evidence: internal/core/claim/renewdue.go:203 — "func LeaveToLapse(root, id, commit string) error {"
  evidence: internal/core/claim/renewdue.go:191 — "const lapseRelDir = stateRelDir + "/lapse""
  evidence: internal/core/implement/loop/drivestate.go:43 — "const DriveStateRel = driveStateDir + "/drive-state.json""
  evidence: internal/core/implement/loop/terminal_setaside_test.go:324 — "nothing is left for a person to clear"
- cond-2611141617198456 — survived: The state is one file in the drive's own checkout's local tier, read by status surfaces run in that checkout, and one drive holds a checkout at a time.
  evidence: internal/core/implement/loop/drivestate.go:47 — "const driveStateDir = ".abcd/.work.local/logs""
  evidence: internal/core/implement/loop/drivestate_read.go:48 — "func ReadDriveState(repoRoot string, now time.Time) (DriveState, error) {"
  evidence: commands/drain.md:97 — "One drive holds a checkout at a time"
  evidence: internal/surface/cli/board_drive_test.go:190 — "func TestTheBoardReadsNoDriveOutsideAManagedCheckout(t *testing.T) {"
- cond-2611141617198101 — survived: Runs at the version before this build's are read, claimed and stopped by the delivered tests, and a file stating a newer version is refused by the read before any decode or claim.
  evidence: internal/core/implement/loop/state.go:2120 — "if err == nil && peek.SchemaVersion > SchemaVersion {"
  evidence: internal/core/implement/loop/state_text_test.go:23 — "const schemaVersionPrevious = SchemaVersion - 1"
  evidence: internal/core/implement/loop/terminal_claim_test.go:251 — "func TestAnUnsettledSignalMarkOnAnEarlierVersionIsSettledAndTaken(t *testing.T) {"
- cond-2611141617194898 — narrowed: With an unreadable process table the start leaves the run, names why and lands the other lanes; no delivered test sets such a lane aside at the bound, and the state reader, a new consumer of the table, falls back to the next-write time instead.
  narrowing: Shown for the start leaving the run and naming the process table; the set-aside at the bound under an unreadable table is exercised by no test, and the status reader under an unreadable table judges by the drive's next-write time.
  evidence: internal/core/implement/loop/terminal_claim_test.go:413 — "func TestAnUnreadableProcessTableLeavesTheRun(t *testing.T) {"
  evidence: internal/core/implement/loop/drivestate_read.go:119 — "that cannot be read, falls back to next"
- cond-2611141617192337 — narrowed: Both starts the lab did not build are built and tested (a lifted stop's run taken up, a replaced drain's run taken with the rest), and a stop that stands is left. But the start does not take every interrupted run in the checkout: intent builds and driverless runs with no sign their session ended are left and named. A closed terminal, a reboot and a file from a real older build remain unprobed.
  narrowing: A start takes every interrupted ISSUE run whose driver is gone, is the drive's own terminal, or shows a sign its session ended; an intent's build and a driverless run with no settled mark or lapsed reservation are left with another command named.
  evidence: internal/core/implement/loop/terminal_claim_test.go:327 — "func TestALiftedPersonsStopIsTakenUp(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal_claim_test.go:371 — "func TestARunWhoseDrainWasReplacedIsTakenWithTheRest(t *testing.T) {"
  evidence: internal/core/implement/loop/terminal.go:2876 — "builds an intent, which a drive does not take up"
  evidence: internal/core/implement/loop/terminal_claim_test.go:588 — "func TestADriverlessRunWithAnUnsettledMarkIsLeft(t *testing.T) {"
- cond-2611141617198387 — survived: Every delivered proof still runs on a fixture clock with a stand-in process table and stand-in stages; no figure from a live drive was added, and the live-proof intent is still planned.
  evidence: internal/core/implement/loop/terminal_setaside_test.go:333 — "o.Until = w.clock.t.Add(3 * time.Hour)"
  evidence: internal/core/implement/loop/terminal_claim_test.go:398 — "claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)"
  evidence: .abcd/development/intents/planned/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:1 — "itd-2610222215485265 is still under planned/"
<!-- abcd-review-end receipt=rcp-da2f8a9cec53 -->

## Grounds

- pursued: a person is expected to bring an interrupted drive back with the start command alone, and to learn its state from one line, because every state the lab found a start stopping on is one the drive can read for itself and the loop already knows its own wait; shown wrong if a drive a person interrupted needs any command but abcd drain --drive to take up every run the interrupt left in the checkout, if one lane's wait still stops other lanes opening, if people still read the log to learn the drive's state, or if the design needs a new command, a flag a person must remember or a stored state a person must clear
