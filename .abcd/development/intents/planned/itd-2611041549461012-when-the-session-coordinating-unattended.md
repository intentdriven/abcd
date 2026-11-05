---
id: itd-2611041549461012
slug: when-the-session-coordinating-unattended
spec_id: spc-2611041706004822
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472, itd-2611031342011261, itd-2610180912189125]
refines: [itd-2610222215485265]
related_issues: [iss-2611022247128872, iss-2611032348373007, iss-2610241107435264, iss-2610241414414823, iss-2611040337228910, iss-2610110915210571]
severity: major
origin: researcher-authored
production_mode: hand-written
impact: additive
target_release: next
---

# A session that takes over a run steps on from what abcd wrote down

## Press Release

> **When the session coordinating unattended runs dies, the next session picks the runs up from what abcd wrote down, and nothing is started twice.** Alice runs abcd unattended on a second account of their computer. When the session driving the runs is gone (a crash, a lost connection, a compaction that forgot), the next session types `abcd implement resume`: it lists every run this machine drives, each lane's stage, each agent still alive or lost, the receipts that arrived while nobody collected them, the pull requests waiting on the queue and how far each lane's base has moved, and the work each lane left uncommitted, and it steps on from there. An agent whose receipt is waiting is collected before any fresh one starts, and work left uncommitted is saved aside and named before anyone builds over it. "I used to find a second implementer paid for the work the first had finished, and a step loop nobody owned still launching agents," said Alice, a solo founder. "Now a restart reads the record instead of guessing."

## Why This Matters

The restart half of an unattended run has its evidence in the loop's own issues. A step loop a stopped session left in the background kept stepping its run and launching agents on a key nobody watched, and the successor could not learn from the run who drove it, from which process, or which key its launches spent (iss-2611022247128872); the run state's `Driver` is a string naming the kind of driver, not a live record. A finished agent was stranded when the step that would collect it was refused, and every later step asked for a fresh implementer though the receipt sat at the awaited path, which an unattended run would have paid for twice (iss-2611031337115764). Stopping a stage left no record of what it wrote, and a resume would have started a fresh implementer over ten changed files (iss-2611032348373007). A session's work in flight does not survive the host's compaction unless it wrote it down (iss-2610222026223032). A crashed copy stranded reserved records and left pull requests drifting (iss-2610241107435264, iss-2610241414414823), and a resume says nothing of how far a lane's base has moved (iss-2611040337228910). On 2026-11-03 the coordinator itself was a host session plus a polling script, so there is no coordinator to restart, only a session to replace; iss-2611031146255825 names the missing drive.

What already exists is the pieces: the run state and the drain state, each lane's checklist (`abcd checklist`), the restart of a gone implementer with its uncommitted work saved aside (`internal/core/implement/loop/restart.go`), the drain's first-signal cancel and second-signal kill, the run log's `stop` and `intervention` events (`abcd implement log`), and the launch ledger (itd-2611031342011261), whose entries carry each agent's process id so a reader judges liveness itself. The drain spec (spc-2610230813348813) already promises that a signal leaves a lane resumable and a second drive resumes it. This intent makes those pieces one path; its sibling, the stop switch (itd-2611041549471712), is the person's way to end the work on purpose.

## Mechanism

We expect a successor to step on without starting anything twice because every fact it needs is a record the run wrote as it went, the live driver on the run state, each launch and its process in the ledger, each lane's work on its checklist and in the receipt path the await names, and the resume reads those records in one listing before it moves, collecting what arrived before it launches. The claim is shown wrong if a resume starts a fresh agent for an await whose receipt is already at its path, or builds a stage over uncommitted work without naming it first.

## Scope Conditions

- Holds for runs abcd drives on one machine: build runs, drain lanes and the terminal drive (itd-2610222215485265), whose plugin page stays an agent's command and whose terminal form stays the person's way to drive it (dcn-2611041550139264). A hosted runner's run is resumed by its host. <!-- cond: cond-2611041706003922 -->
- A process killed outright is outside the promise; the resume treats its agents as lost and its work as saved aside. <!-- cond: cond-2611041706002354 -->
- Recorded state means abcd's own files: the run and drain states, the checklist, the launch ledger and the log; never a session's transcript or memory. <!-- cond: cond-2611041706009161 -->
- Holds on macOS and Linux, where `~/.abcd.noindex/` is abcd's home; there is no Windows target. <!-- cond: cond-2611041706005578 -->

## Acceptance Criteria

- **Given** a run, **when** a session drives it, **then** the state records the live driver (session, process id, start time) and each in-flight launch with its route, and a second driver is refused while the first is alive (iss-2611022247128872).
- **Given** a driver that is gone, **when** the next session runs the resume, **then** it lists every run on the machine with each lane's stage, each agent alive or lost by its process, each receipt that arrived uncollected, each pull request waiting and how far each lane's base has moved (iss-2611040337228910), and the uncommitted work each lane holds; it collects an arrived receipt before it ever starts a fresh agent (iss-2611031337115764).
- **Given** a lane with uncommitted work and no live agent, **when** the resume steps it, **then** the work is saved aside and named in the record before a fresh agent starts, never built on silently (iss-2611032348373007).
- **Given** a drain stopped while awaiting a judgement, **when** it is resumed, **then** the judgement can be given or the entry is cleared (iss-2610110915210571).
- **Given** a coordinator session that lost its context, **when** it resumes, **then** everything it needs is in the listing above and in the checklist, so nothing depends on the transcript (iss-2610222026223032).
- **Given** a run the stop switch halted (itd-2611041549471712), **when** the stop is lifted and a session resumes, **then** the listing shows what the stop left in each lane and the reservations it released, and the resume reserves again before it steps.

## Open Questions

_None open. Every question the scratch draft of 2026-11-04 put for this half had one answer on the record: a stop takes hold at a tool-call boundary, never within one (dcn-2611031722512037); a coordinating session is walled where abcd starts it, and one the person starts off a profile is warned (dcn-2611031722512037, dcn-2611041550129278); the run's account of its launches is the ledger (itd-2611031342011261, decisions 7 to 12); a gone agent restarts fresh from its last commit with its uncommitted work saved aside (iss-2610080620372731)._

## Decisions

- 2026-11-04, the product thinker (dcn-2611041550139264): the restart path and the stop switch are two records, this one and itd-2611041549471712, as the scratch draft's decomposition proposed; the routing is confirmed.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop releases the run's claims on its work, so a resume after a stop reserves again before it steps.
- 2026-11-04, the product thinker (dcn-2611041550139264): the drain's plugin page stays an agent's command and the terminal form stays; the runs this path resumes include the terminal drive.

## Decomposition

Hand-run four-piece table, 2026-11-04, from the scratch draft that held the stop switch and the restart path together; the product thinker confirmed the split (dcn-2611041550139264).

| Part | Kind | Home |
|---|---|---|
| the resume listing, the live-driver record, collect-before-launch, saved-aside work, the drain's judging entry | capability | this intent |
| one command stops every run on the machine at its next move, lifted only by the person | capability | itd-2611041549471712, the stop switch |
| a stop is a recorded state read at each move, never a signal to a terminal; an agent's session may never lift it | trust rule | an ADR owed with the stop switch, extending adr-2610240850193281 (the grant permits, this withdraws) and dcn-2611031722512037; a brief invariant when accepted; not filed here |
| a run's account of itself lives in abcd's files, never in a session's memory | stance | a candidate principle, for the product thinker to confirm; not filed |
| the driver fields, the resume's output | plumbing | the brief's loop chapter, through this intent's spec |

Verdict: SPLIT, confirmed. Typed links: `refines` itd-2610222215485265 (its spec's resume story becomes this path for every run); `builds_on` itd-2609201925079472 (the loop's pacing and checkpoints), itd-2611031342011261 (the ledger that names each agent's process) and itd-2610180912189125 (the autonomous mode's store, the shape a recorded state mirrors); `related_issues` as the frontmatter lists, whose remedies are the criteria above. Advisory reversal flags: none.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
