---
id: itd-2611041549471712
slug: from-their-own-account-a-person-types
spec_id: spc-2611041705580524
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201925079472, itd-2611031342011261, itd-2610180912189125]
refines: [itd-2610222215485265]
reverses: [dcn-2610222224054666]
related_issues: [iss-2610121528179959]
severity: major
origin: researcher-authored
production_mode: hand-written
impact: additive
target_release: next
---

# A person stops every unattended run on the machine with one command

## Press Release

> **A person stops every unattended run on their computer from their own account with one command, and nothing but their own command lifts it.** Alice runs abcd unattended on a second account of their computer, and at setup they chose a shared folder on it (`/Users/Shared` on macOS, its counterpart on Linux) for the things the two accounts pass between them. From their own account they type `abcd implement stop`: every run on the machine finishes the tool call in hand, opens no new lane, starts no new agent, writes what each lane holds uncommitted into its record, releases what it reserved, and says so in the next daily digest. An agent's session, a driver or a hook that tries to lift the stop is refused. "I could never reach a run from the terminal it was not in," said Alice, a solo founder. "Now a stop is a file my account writes and every run reads before its next move, and it stays stopped until I say otherwise."

## Why This Matters

The product thinker ruled that watching is no control: a person cannot interrupt a tool call as it runs, so a person watching never makes an unwalled agent acceptable (dcn-2611031722512037). The same fact cuts the other way: a stop can only take hold at a tool-call boundary, so it has to be a recorded state every run reads before each move, not a key the person presses in a terminal they are not at. Today the one such state is the autonomous mode (`abcd mode autonomous`, itd-2610180912189125, adr-2610181003541997), which grants and queues but stops nothing, and its store admits only files this account owns that no one else can write. The person's seat and the run's seat are two accounts on one machine, so no store abcd has today can carry a stop from one to the other; iss-2610121528179959 names the same gap for handovers. A step loop a stopped session left behind kept launching agents on a key nobody watched (iss-2611022247128872).

The product thinker settled the place at the filing (dcn-2611041550139264): the stop lives on the same machine, in a shared folder chosen as an option at setup, on a machine no other person shares, and the store ownership rule admits this one exception. A stop that reaches the machine over a private network or through a cloud adapter is later work. The daily digest (itd-2611041549483696) lives in the same folder, so one choice holds for both.

## Mechanism

We expect a stop to take hold on every run because each run already reads a recorded state before each move (the autonomous mode's store is the shape), and a file in the shared folder written by the person's account is one more such read; the runs stop at their next boundary, which is the only place a stop can act. The claim is shown wrong if a run makes a move after the stop file exists, or if any process but the person's own command removes it.

## Scope Conditions

- Holds for runs abcd drives on one machine: build runs, drain lanes and the terminal drive (itd-2610222215485265); a hosted runner's run is stopped by its host. <!-- cond: cond-2611041705583688 -->
- Holds where the person chose the shared folder at setup; a machine without one has no stop that crosses accounts, and `abcd ahoy` says so. The folder is one option of setup, never a requirement. <!-- cond: cond-2611041705589732 -->
- Holds on a machine no other person shares: the folder is readable by every account on it, and the ruling that admits it rests on the machine being the person's alone. <!-- cond: cond-2611041705586060 -->
- A stop takes hold at the next move of each run; an agent mid-call finishes that call. A process killed outright is outside the promise, and the restart path (itd-2611041549461012) treats its agents as lost. <!-- cond: cond-2611041705583744 -->
- The stop is the person's act from their own account; an agent's session is refused the command. Lifting it is the same person's act. <!-- cond: cond-2611041705584558 -->
- Holds on macOS and Linux; there is no Windows target. <!-- cond: cond-2611041705581125 -->

## Acceptance Criteria

- **Given** the person runs the stop from their own account, **when** any run on the machine takes its next move, **then** it opens no lane, hands out no stage, starts no agent, and writes a `stop` event naming who set it and when; an agent already running finishes its call and its receipt is collected, never acted on further; and only then, with no agent live in the lane, is each lane's uncommitted work saved aside as a restart does and its reservations released so a peer may take the work (dcn-2611041550139264).
- **Given** the stop holds, **when** an agent's session, a driver or a hook tries to lift it, **then** it is refused; only the person's own command lifts it, and the lift is a recorded event.
- **Given** a terminal run, **when** Ctrl-C or a hang-up reaches it, **then** the signal is a stop: the running agent's call finishes and its receipt is collected, then the lane's uncommitted work is saved aside and its claims released, and the drive prints the run and the lane; a second press kills the process group at once (dcn-2611041608057544). This reverses dcn-2610222224054666, under which the open lane kept its reservation for the next drive to resume.
- **Given** the stop holds, **when** the last run in the account that runs the work ends under it, **then** the stop composes that day's digest page before it ends, in that account (itd-2611041549483696), naming the stop, every run it halted and what each left; a page already written for the day is rewritten whole.
- **Given** setup, **when** the person is offered the shared folder, **then** the walkthrough says what it is for, that it is read by every account on the machine, and that it is meant for a machine no other person shares; declining leaves every other part of abcd working, and the stop then reaches only runs in the person's own account.
- **Given** the stop file's home, **when** abcd reads it, **then** it accepts this one file outside its owner-only rule, written by the person's account, and refuses a stop file any other account wrote, naming it.

## Open Questions

_None open. The three questions the scratch draft of 2026-11-04 put are the first three decisions below._

## Decisions

- 2026-11-04, the product thinker (dcn-2611041550139264): the stop lives on the same machine, in a shared folder chosen as an option at setup (`/Users/Shared` on macOS, its counterpart on Linux), on a machine no other person shares; the store ownership rule admits this one exception; a stop over a private network or through a cloud adapter is later work. The scratch draft's second option, made deliberate.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop lets an agent in a call finish that call, then stops.
- 2026-11-04, the product thinker (dcn-2611041550139264): a stop releases the run's claims on its work.
- 2026-11-04, the product thinker (dcn-2611041550139264): the stop switch and the restart path are two records; the routing is confirmed.
- 2026-11-04, the product thinker (dcn-2611041608057544): Ctrl-C in a terminal run is a stop: the call finishes and claims are released; a second press kills at once.

## Decomposition

Hand-run four-piece table, 2026-11-04, from the scratch draft that held the stop switch and the restart path together; the product thinker confirmed the split (dcn-2611041550139264).

| Part | Kind | Home |
|---|---|---|
| one command stops every run on the machine at its next move, lifted only by the person; the shared folder offered at setup | capability | this intent |
| the resume listing, the live-driver record, collect-before-launch, saved-aside work | capability | itd-2611041549461012, the restart path |
| a stop is a recorded state read at each move, never a signal to a terminal; an agent's session may never lift it; one shared folder on an unshared machine is the one file outside the owner-only rule | trust rule | an ADR owed, extending adr-2610240850193281 (the grant permits, this withdraws) and dcn-2611031722512037, and narrowing the store ownership rule by this one exception (dcn-2611041550139264); a brief invariant when accepted; not filed here |
| a run's account of itself lives in abcd's files, never in a session's memory | stance | a candidate principle, for the product thinker to confirm; not filed |
| the stop file's shape, the setup walkthrough's wording, the digest's line | plumbing | the brief's loop chapter, through this intent's spec |

Verdict: SPLIT, confirmed. Typed links: `refines` itd-2610222215485265 (a stop ends its drive the same way); `builds_on` itd-2609201925079472, itd-2611031342011261 and itd-2610180912189125; `related_issues` iss-2611022247128872 and iss-2610121528179959. `reverses` dcn-2610222224054666 (dcn-2611041608057544): a signal no longer leaves the lane's reservation held; the call finishes and the claims are released. Flagged, never auto-classified: this intent narrows the autonomous mode's reading that only `abcd mode back` ends it; a stop ends the work without ending the mode.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
