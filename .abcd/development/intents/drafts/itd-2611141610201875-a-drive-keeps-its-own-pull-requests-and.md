---
id: itd-2611141610201875
slug: a-drive-keeps-its-own-pull-requests-and
spec_id: null
kind: null
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

> **A drive keeps its own pull requests and its own checkout moving.** A person starts a drive and leaves. When the merge queue removes one of the drive's pull requests for a failure that is not the change's own, the drive arms it again, a stated number of times at most, and says so on one line. When the failure is the change's, the drive hands the lane back and names the check and the command that re-runs it. Between lanes the drive brings its own checkout up to the default branch whenever that is safe, and when it is not, the state line says what blocks it. Nobody watches the forge for the drive, and nobody moves a file aside so that its checkout can catch up.

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
- On the state line: what blocks the checkout from catching up, when something does.

### What a person never has to know

- That the drive's checkout is behind, or that its own drafts block a fast-forward.
- Which check failed in the merge queue, when the failure is not the change's.

## Mechanism

We expect flaky checks to stop being manual steps, because the drive already reads each removal and already holds everything it needs to arm the pull request again. Shown wrong if the rule the product thinker chooses re-arms a real failure of the change more often than the stated number of times, or if people still re-arm the drive's pull requests by hand.

We expect the drive's checkout to stay near the default branch, because the drive has the list of every record it wrote and git refuses any fast-forward that would touch one. Shown wrong if a fast-forward loses or rewrites a record the drive wrote, changes what a lane in flight reads, or if the checkout still falls behind until a person moves a file.

## Scope Conditions

- The forge data the code parses today cannot tell a failure that is the change's from one that is not. For a removal it reads the reason, the merge group's workflow runs (name, branch, commit, conclusion) and that commit's check runs (name, conclusion); none names a file, a path or a test. Two removals that differ only in whether the failed check tests the change's files read identically, byte for byte (F-10). The rule for the first criterion therefore waits on the first open question.
- A checkout whose HEAD is the default branch and an ancestor of origin's, with no merge, rebase or stash in progress and nothing staged (F-12).
- The drive can tell its own records from a person's edit only by the list its drain keeps of what it wrote. Git cannot tell them apart (F-12, case D).
- No lane in progress at the moment of a fast-forward. A lane parked on its pull request already re-reads the default branch at each move.
- A source build never updates itself (adr-2611062149324491). A fast-forwarded checkout is newer records and source, and the running binary is unchanged.
- The lab measured the drain's move on fixtures, not a drive on a live forge, and wired neither re-arming rule into the loop.

## What's Out of Scope

- The drive rebuilding itself from source. The lab did not probe it, and the fourth open question asks what the drive should do instead.
- Re-running a forge check the forge refuses to re-run (the incident's fifth pull request, whose workflow runs were created and never started).
- A surface for what the drive's reservations cover, and a way to clear many handed-back lanes at once (the note's event 10).
- The state line itself, the set-aside of a blocked lane and the claim at the start. Those are itd-2611141610197753, which this intent builds on: its one-line reports and its "what blocks it" use that intent's state line.
- The terminal's drawing of these lines (itd-2611141610192096).
- Any new command, any flag a person must remember, and any stored state a person must clear.

## Acceptance Criteria

- **Given** a pull request the merge queue removed, **when** the rule the product thinker chooses says the failure is not the change's, **then** the drive re-arms it, at most a stated number of times at one head, and says so on one line; otherwise it hands the lane back naming the check and the command that re-runs it. This criterion cannot be written more exactly until the first open question is answered.
  - Evidence: lab-261114145658-67e1f9b, F-6, F-10 and F-18.
- **Given** a drive's checkout behind origin's default branch, whose every dirty path is a record the drive wrote and none of which an incoming commit changes, and no lane in progress, **when** a lane ends, **then** the checkout is fast-forwarded; **given** any other dirty path, **then** it is not, and the state line says what blocks it. This criterion depends on the second and third open questions.
  - Evidence: lab-261114145658-67e1f9b, F-12 and F-19.

## Open Questions

Each question below is the product thinker's to decide, and none is answered here. The first three are quoted as the lab met them, with the answers it can defend and what each costs. The lab did not probe the fourth: the question is the incident note's, and its answers and costs are set out by the run that filed this draft, to widen the choice and not to recommend one.

1. **DQ-6. "When may the drive re-arm a pull request the queue removed?"** (F-10, F-18)
   - Always, up to N times at one head: no new read of the forge. Cost: a real failure of the change spends N merge-queue cycles before it is handed back.
   - Only when a new read says the failed tests are not the change's (the failed job's annotations or log): the rule P3 states. Cost: a new forge surface to parse and keep working, and a mapping from test names to the change's files.
   - Only when the same workflow also failed for another pull request in the same listing: no new read. Cost: a flaky check that fails alone is still handed back, and a real failure is still re-armed when a neighbour fails.
   - Never; spell the re-run command and put it on the state line. Cost: every flaky check stays a manual step, with the command now given.

   Context, not an answer: a ruling already on record says that a merge-queue entry dropped with no failing test is re-armed until it lands, and a real failure gets fixed (dcn-2611062201286033). The lab found that the forge data the code parses today cannot tell the two apart (F-10), so that ruling cannot be applied by the drive without either a new read of the forge or an accepted number of wasted re-arms.

2. **DQ-8. "Should the drive land the records it writes?"** (F-12, F-19)
   - Yes, through a records-only pull request per hand-back or per batch: the tree stays clean and the checkout can always fast-forward. Cost: pull requests the drive opens for its own bookkeeping, through the same queue.
   - No; fast-forward only when the manifest rule holds, and say when it does not. Cost: the checkout still falls behind on the first clash, with a line that says so.
   - Write them outside the checkout until a person takes them. Cost: drafts a person cannot see where they expect them.

3. **DQ-9. "May a running drive fast-forward over a commit that changes a hook or a setting it holds?"** (F-12, case F)
   - No: decline and say so. Cost: the checkout stays behind until a restart.
   - Yes, and re-read its hold at that boundary, as it already re-executes for a new release. Cost: the hold's guarantee (nothing the drive's git runs changed since the start) becomes "since the last boundary".

4. **"What should a drive do when the default branch carries a fix it lacks?"** (the incident note's events 1 and 2; not probed by the lab)
   - Nothing more than itd-2611141610197753 gives: the state line says `stalled` when lanes are handed back in a row for one reason. Cost: the line cannot say that a fix exists, so a person has to work out that a rebuild is the remedy.
   - Say how far its own build is behind the default branch, on the state line or at the start. Cost: a source build is behind after every merge, so the line says so nearly always, and being behind is not the same as lacking a fix it needs.
   - Rely on releases: a release build already takes each new release at a lane boundary (itd-2611062149308723), and a source build never updates itself. Cost: a fix reaches a drive only when a release is cut, and a drive on a source build, which the incident's was, is not covered.
   - A source build rebuilds itself from its fast-forwarded checkout between lanes. Cost: a binary with no release, no checksum and no signed provenance replaces the running one, which adr-2611062149324491 rules out as written; that record would have to change, and the lab tested none of it.

## Typed Links

- builds_on: itd-2611141610197753 (a drive carries on from one command, and says in one line when it cannot). The one-line reports and the "what blocks it" of both criteria are that intent's state line, and its set-aside is what keeps a lane that cannot land from holding the others while this intent is unbuilt.
- related: itd-2610222215485265 (the drive itself). itd-2611062149308723 (a drive takes each release itself) and adr-2611062149324491, which bound the fourth open question. itd-2611141610192096 (the terminal output).
- context: dcn-2611062201286033 (a merge-queue entry dropped with no failing test is re-armed until it lands, and a real failure gets fixed).
- evidence: lab-261114145658-67e1f9b (findings F-6, F-10, F-12, F-18 and F-19) and `.abcd/development/research/notes/drive-operations-2026-11.md`, events 1 to 4 and 6.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
