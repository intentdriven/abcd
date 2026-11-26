---
id: itd-2611141610201875
slug: a-drive-keeps-its-own-pull-requests-and
spec_id: spc-2611251808377435
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611141610197753]
severity: minor
related_intents: [itd-2610222215485265, itd-2611062149308723, itd-2611141610192096]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A drive keeps its own pull requests and its own checkout moving

## Press Release

> **A drive keeps its own pull requests and its own checkout moving.** A person starts a drive and leaves. When the merge queue removes one of the drive's pull requests, the drive arms it again, a few times at most at one head, and says so on one line. When the pull request is removed again after those few times, the drive hands the lane back and names the check and the command that re-runs it. At its start and between lanes the drive brings its own checkout up to the default branch whenever that is safe, and when it is not, the drive says what blocks it. Nobody watches the forge for the drive, and nobody moves a file aside so that its checkout can catch up.

> "Four of its pull requests fell out of the queue on a test that fails now and then. I found out from the line that said it had armed each one again." — Iris, a product thinker (they/them)

> "I used to keep a second session open to re-run checks and arm pull requests for the drive. Now I open the forge only when the drive hands a lane back and names the check." — Nia, a technical facilitator (they/them)

## Why This Matters

Over three days of one unattended drive (`.abcd/development/research/notes/drive-operations-2026-11.md`), a second session re-ran four forge checks and re-armed four pull requests by hand (event 3). The drive read each removal correctly, printed what a person should do, and did none of it. Its checkout fell 692 commits behind the default branch, one record missing from it stopped every lane for 23 hours (event 4), and bringing it up to date took setting one of the drive's own uncommitted files aside (event 6).

The lab on drive operations (lab-261114145658-67e1f9b) reproduced both on the pinned build:

- A pull request removed from the merge queue is reported and never re-armed, whatever failed: 0 re-arm, enqueue or re-run calls in four kinds of removal. The report spells the re-arming and gives no command for the re-run (F-6, F-18).
- The drive's own uncommitted records, its draft intents and back-link edits, block its checkout's fast-forward, even when origin holds the same file byte for byte (F-19). A fast-forward never loses a record the drive wrote, because git refuses the whole merge when an incoming commit touches a dirty path (F-12).

The note makes a third point that the lab did not probe. Two fixes reached the default branch while the drive ran, and the drive kept failing the same way, on eleven lanes and then on eight, until a person rebuilt it (events 1 and 2). Nothing tells a person that the drive they started is behind a fix it needs.

### Who it is for

- The product thinker (Iris), who starts a drive and leaves. A check that fails now and then is not their manual step.
- The technical facilitator (Nia), who is asked "is it working?" and should not need to compare the drive's checkout with the default branch to answer.

### What a person does

- Nothing, for a removal that is not the change's own and for a checkout that can catch up.
- For a lane handed back on a real failure: reads the check's name and runs the one command given.

### What a person sees

- One line when a pull request is armed again, saying which and how many times so far.
- A hand-back that names the failed check and the command that re-runs it.
- What blocks the checkout from catching up, when something does: one line among the events when the fast-forward is declined, named again by `abcd implement status` and in the view by login across machines (decision 5).

### What a person never has to know

- That the drive's checkout is behind, or that its own drafts block a fast-forward.
- Which check failed in the merge queue, until the drive hands the lane back.

## Mechanism

We expect flaky checks to stop being manual steps, because the drive already reads each removal and already holds everything it needs to arm the pull request again. Shown wrong if the drive re-arms a pull request at one head more often than the stated number of times, or if people still re-arm the drive's pull requests by hand.

We expect the drive's checkout to stay near the default branch, because the drive writes its own records outside the checkout (decision 2) and git refuses any fast-forward that would touch a dirty path. Shown wrong if a fast-forward loses or rewrites a record the drive wrote or a person's edit, changes what a lane in flight reads, or if the checkout still falls behind until a person moves a file.

## Scope Conditions

- The forge data the code parses today cannot tell a failure that is the change's from one that is not. For a removal it reads the reason, the merge group's workflow runs (name, branch, commit, conclusion) and that commit's check runs (name, conclusion); none names a file, a path or a test. Two removals that differ only in whether the failed check tests the change's files read identically, byte for byte (F-10). The drive therefore does not try to tell them apart: it re-arms a removal always, up to a few times at one head (decision 1), and a real failure spends those few merge-queue cycles before it is handed back. <!-- cond: cond-2611251808376685 -->
- A checkout whose HEAD is the default branch and an ancestor of origin's, with no merge, rebase or stash in progress and nothing staged (F-12). <!-- cond: cond-2611251808376820 -->
- The records the drive writes are outside the checkout (decision 2), so a dirty path in the checkout is never one of the drive's own records, and the drive needs no list to tell its records from a person's edit (F-12, case D). <!-- cond: cond-2611251808376388 -->
- No lane in progress at the moment of a fast-forward. A lane parked on its pull request already re-reads the default branch at each move. <!-- cond: cond-2611251808374463 -->
- A source build never updates itself (adr-2611062149324491). A fast-forwarded checkout is newer records and source, and the running binary is unchanged. <!-- cond: cond-2611251808377425 -->
- The lab measured the drain's move on fixtures, not a drive on a live forge, and wired neither re-arming rule into the loop. <!-- cond: cond-2611251808376133 -->

## What's Out of Scope

- The drive rebuilding itself from source. The lab did not probe it, and decision 4 rules that the drive does nothing new for a fix it lacks.
- Re-running a forge check the forge refuses to re-run (the incident's fifth pull request, whose workflow runs were created and never started).
- A surface for what the drive's reservations cover, and a way to clear many handed-back lanes at once (the note's event 10).
- The state line itself, the set-aside of a blocked lane and the claim at the start. Those are itd-2611141610197753, which this intent builds on: its one-line reports follow that intent's state line, and where a blocked fast-forward is shown is the fifth open question.
- The terminal's drawing of these lines (itd-2611141610192096).
- Any new command, any flag a person must remember, and any stored state a person must clear.

## Acceptance Criteria

- **Given** a pull request the merge queue removed, **when** it has been re-armed fewer times at that head than the stated number, **then** the drive re-arms it and says so on one line; otherwise it hands the lane back naming the check and the command that re-runs it (decision 1). The number is three re-arms at one head (decision 7).
  - Evidence: lab-261114145658-67e1f9b, F-6, F-10 and F-18.
- **Given** a drive's checkout behind origin's default branch, with no dirty path that an incoming commit changes and no lane in progress, **when** the drive starts or a lane ends, **then** the checkout is fast-forwarded, including over a commit that changes a hook or a setting the drive holds, after which the drive reads its hold again (decisions 3 and 6); **given** a dirty path that an incoming commit changes, **then** it is not, and the drive writes one event line that says what blocks it, which `abcd implement status` names and the view by login across machines carries, while the state line keeps its one `on` field (decision 5). The records the drive writes are outside the checkout (decision 2).
  - Evidence: lab-261114145658-67e1f9b, F-12 and F-19.

## Open Questions

_All resolved: questions 1 to 4 by the product thinker on 2026-11-15 (decisions 1 to 4), and question 5 by the technical facilitator on 2026-11-25 (decision 5), which is kept below as it was put._

5. **"Where does a person see that the checkout's fast-forward is blocked?"** (F-12; the lab's brief put it on the state line)
   - What is known: the state line of itd-2611141610197753 has one `on` field, for what the drive is working or waiting on, and one act. A drive whose fast-forward is declined is usually working on a lane, so the block is not what the line is on. The lab's brief says "the state line says what blocks it" and its state-line design has no field for it.
   - Resolved (decision 5): one event line when the fast-forward is declined, named in `abcd implement status` and carried in the view by login; not on the state line.

## Decisions

1. **The drive always re-arms a pull request the merge queue removed, up to a few times at one head** (product thinker, 2026-11-15; formerly DQ-6).
   - Answer: always, with no new read of the forge. A real failure of the change spends those few merge-queue cycles before it is handed back. The exact number was the run's to propose when the step was planned; it is decision 7.
   - What it costs: the merge-queue cycles a real failure spends, which is the price of reading no more of the forge than the code parses today (F-10).
2. **The records a drive writes are kept outside the checkout until a person takes them** (product thinker, 2026-11-15; formerly DQ-8).
   - Answer: the hand-back notes and the other records the drive writes go in each account's own abcd folder on that machine, and not in the checkout, until a person takes them in.
   - What it costs, which the product thinker accepted: another account's sessions and the forge do not see those records until they are moved in.
3. **A running drive may fast-forward over a commit that changes a hook or a setting it holds** (product thinker, 2026-11-15; formerly DQ-9).
   - Answer: yes, and it re-reads its hold at that boundary, as it already re-executes for a new release.
   - What it costs: the hold's guarantee, that nothing the drive's git runs changed since the start, becomes that nothing changed since the last boundary.
4. **When the default branch carries a fix the drive lacks, the drive does nothing new and relies on releases** (product thinker, 2026-11-15; formerly the fourth question).
   - Answer: a release build already takes each new release at a lane boundary (itd-2611062149308723), and a source build never updates itself (adr-2611062149324491). Nothing is added.
   - What it costs: a fix reaches a drive only when a release is cut, and a drive on a source build is not covered.
5. **A declined fast-forward is one event line, not part of the state line** (technical facilitator, 2026-11-25, on delegation; formerly the fifth question).
   - Answer: when the fast-forward is declined, the drive writes one event line that says what blocks it. `abcd implement status` names the block, and the view by login across machines carries it, so a person need not be at the drive's terminal to learn it. The state line keeps its one `on` field, for what the drive is working or waiting on.
   - Why: a drive whose fast-forward is declined is usually working on a lane, so the block is not what the state line is on, and a second field there would cost the line its one-glance reading.
   - What it costs: the block is not on the line a person reads first; it is one command away, and it scrolls up among the events at the terminal.
6. **The drive fast-forwards at its start as well as between lanes** (technical facilitator, 2026-11-25, on delegation).
   - Answer: the second criterion is widened to the drive's start. A start is a boundary like the end of a lane, so it fast-forwards under the same conditions and reads its hold after, in place of refusing a checkout that is behind a tip whose hooks have changed and naming a hand-typed `git pull --ff-only`.
   - Why: decision 3 already lets a running drive fast-forward over a hooks change at a boundary, and with several machines landing work a checkout behind origin at the start is the normal state each morning.
   - What it costs: a start may now move the checkout before its first move, which a person who left a deliberate older checkout would not expect; the same safety conditions as at a lane's end apply, so a dirty path an incoming commit changes still stops it.
7. **Three re-arms at one head** (technical facilitator, 2026-11-25, at planning; decision 1 left the number to the run).
   - Answer: the loop arms a pull request the merge queue removed again at most three times at one head, a constant beside the watch's own bound of three toggles per head (`maxToggles`). The count and the forge time of the removal last re-armed for are new fields of the lane's landing, which takes run-state schema 32 (spc-2611251808377435, step 1).
   - Why: a real failure then costs at most three merge-queue cycles before a person hears of it, the same bound the watch already keeps.
   - What it costs: a pull request whose flaky check fails four times at one head is handed back though one more cycle might have landed it.

## Typed Links

- builds_on: itd-2611141610197753 (a drive carries on from one command, and says in one line when it cannot). The one-line reports of both criteria follow that intent's state line, and its set-aside is what keeps a lane that cannot land from holding the others while this intent is unbuilt.
- related: itd-2610222215485265 (the drive itself). itd-2611062149308723 (a drive takes each release itself) and adr-2611062149324491, which bound decision 4. itd-2611141610192096 (the terminal output).
- context: dcn-2611062201286033 (a merge-queue entry dropped with no failing test is re-armed until it lands, and a real failure gets fixed).
- evidence: lab-261114145658-67e1f9b (findings F-6, F-10, F-12, F-18 and F-19) and `.abcd/development/research/notes/drive-operations-2026-11.md`, events 1 to 4 and 6.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
