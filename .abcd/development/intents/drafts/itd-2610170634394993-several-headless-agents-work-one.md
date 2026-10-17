---
id: itd-2610170634394993
slug: several-headless-agents-work-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565, itd-2609151838312703, itd-2609221656373558]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610161639025702, iss-2610161634043273, iss-2610162306239284, iss-2610140638074587]
related_intents: [itd-2609150819440345, itd-2610100704270254, itd-78, itd-115, itd-2609201916151817, itd-82, itd-2609201925079472, itd-148]
---

# Headless agents in parallel coordinate through the shared remote, not conversation

## Press Release

> **Several headless agents work one repository at once, and none of them needs a conversation to know what the others are doing.** Each agent takes a piece of work by a claim on the shared remote, says in a line abcd reads which other change it waits on, and gets a share of the machine and of the model budget from abcd rather than from a message. A change that waits on another is not queued until that one lands, and is rebuilt on top of it when it does, so it meets its conflict once. Any session can run the pass that compares what the records say should happen with what is happening and acts on the difference, so no one process has to stay alive, and every wait has a deadline.
>
> "Last night one session spent its evening telling four others when a pull request had merged, and a new one had to be asked by message to hold its own," said Maya, an autonomous-development practitioner. "My headless agents never read a message. Now each one writes what it waits on where every other agent already looks, abcd holds it until that lands, and when an agent dies its work goes back to the pool instead of waiting on a promise nobody is left to keep."
>
> _Agent-seeded on 2026-10-17 from the evening of 2026-10-16 and a state-of-the-art pass the same night ([research note](../../research/notes/2026-10-17-parallel-headless-agent-coordination.md)); not yet put to the product thinker._

## Why This Matters

abcd's coordination pieces exist but are joined by conversation. Claims (itd-2609091034175565) keep two agents off one record and its files. The load gate keeps preflights from swamping the machine. The merge queue lets one change through at a time. What ties them together on 2026-10-16 was sessions messaging each other: one session asked a new session to hold its records pull request until another landed, and promised to tell four waiting sessions when it merged. A headless agent receives no such message, and a promise lasts only as long as the session that made it.

The same evening showed what a missing deadline costs. The load gate deadlocked for hours between two builds of its own queue rule (iss-2610161634043273), and no wait timed out for eight hours. Two preflights failed on the gate's two-hour limit alone, with every check before it passing (iss-2610161639025702).

Published measurements say the collisions are common, not rare: 19.8% of agent pull requests open at the same time conflicted when one agent made them and 41.7% across agents, and telling an agent what had already landed recovered most of the runs that passed alone and failed together (research note, "Multi-agent coding").

## Mechanism

We expect parallel headless agents to stop colliding and stop idling on each other, because every fact they now pass by message (who holds what, what waits on what, whose turn it is, how much budget is left) becomes a record on the shared remote that each agent already fetches and that any session can act on. Shown wrong if, in a measured run of three or more headless agents, the agent minutes lost to conflicts, rebuilds after a dependency landed, and waits that ended in a timeout are no lower than in a run of the same agents coordinating by message.

## Scope Conditions

- Holds for agents that can push to the shared remote: people with write access and the cloud sessions they start. An outside contributor reads the records but cannot write one.
- Holds for one repository on one forge whose merge queue is the only route to the default branch.
- Holds for the machine budget (preflights, race runs) only among agents on one machine, which is all the process table shows. The model budget spans every account of one organisation, because the provider's rate limits do.
- Holds while at most one minor version of abcd's coordination records is in use at once; a reader meeting a newer record refuses it rather than guessing.

## Acceptance Criteria

- Given an agent whose branch waits on an open pull request, and says so in a line abcd reads, when it asks abcd to land the branch, then abcd does not queue it while that pull request is open, rebuilds the branch on top of it once it merges, and queues the result.
- Given a declared dependency that is closed without merging, cannot be resolved, or closes a cycle, when abcd reads it, then abcd refuses to land the branch and names the dependency and the reason, rather than waiting.
- Given an agent holding a claim that stops renewing it, when the claim expires and a delay has passed, then another agent can take the work, and a push or queue from the first agent under its old claim is refused because the claim's generation number has moved on.
- Given three headless agents on one machine with a budget of two preflights, when all three reach their preflight at once, then two run and the third waits, and no wait lasts beyond its stated deadline without ending in a named refusal.
- Given the session running the coordination pass is stopped mid-pass, when another session runs the pass, then it reaches the same decisions from the records on the remote alone, with nothing lost.
- Given two agents running different versions of abcd's coordination records, when they read each other's records, then neither waits on the other in a cycle, and the one meeting a newer format than it knows refuses with a message naming the version.
- Given an agent about to re-submit a branch after a change it overlaps with has landed, when abcd rebuilds it, then the agent is given a summary of what landed since its base.

## SOTA

Surveyed 2026-10-17 against primary sources ([research note](../../research/notes/2026-10-17-parallel-headless-agent-coordination.md)):

- **Kubernetes controllers and leases, Chubby sequencers:** mature. Adopted as the pattern: coordination is a pass that compares wanted and current state held in shared records, a claim is a compare-and-swap with an expiry and a generation number that a later act re-checks, and a dead holder's claim is retaken only after a delay. The shared remote already gives abcd the compare-and-swap (a push that is not a fast-forward is refused).
- **Job queues with timeouts (Faktory, Temporal):** mature. Adopted: work comes back by timeout, never by noticing a death, and every wait has a wait-to-start limit, a per-attempt limit and a heartbeat.
- **Merge queues and dependency lines (GitHub merge queue, Mergify `Depends-On:`, git-spice, Aviator):** mature. GitHub's merge queue stays the only route to the default branch. The dependency line and the footprint routing are the parts abcd has to add, because the queue it uses reads neither.
- **Multi-agent coding products (Claude Code agent teams, Copilot's cloud agent):** young. Agent teams coordinate through a shared task list but are bound to one session on one machine, so they cannot carry coordination across accounts or into the cloud. Not adopted as the layer.
- **A coordinator cluster (etcd, a Temporal server):** mature but out of scale for one repository. Not adopted.

Path taken: 2. The forge and the shared remote are the native floor (the queue, compare-and-swap pushes, branches cloud sessions can push), and the pass, the dependency line, the generation number and the budgets are bespoke to abcd's records, behind a seam.

## Open Questions

- Who runs the coordination pass: any session on demand (each `land`, `build` or `drain` runs it before acting), a session that holds a lease to run it on a timer, or both? The state-of-the-art pass advises against a coordinator process that must stay alive.
- Where does "waits on" live: a `Depends-On:` line in the pull request body (Mergify's form, visible on the forge), a field on the claim record on the remote (visible before any pull request exists), or both?
- Does the dependent branch get rebuilt by merging the landed dependency in or by restacking it, given the repository allows merge, squash and rebase merges and a restack rewrites shas a `resolved_by.commit` stamp may cite?
- Where does the machine budget live: the load gate's slots (`load.preflights`) extended to every heavy job as weighted slots, or a separate pool? Does the sub-agent budget (itd-2610100704270254) join it, and is it sized from the provider's rate-limit headers or a fixed count?
- Does the release cut become a lease the merge path checks, replacing the rule that the cutting session has the final say by message?
- How does this relate to the register draft (itd-2609150819440345) and the announcements intent (itd-2609151838312703): does it consume them, or does one of them grow into this?

## Decisions

1. **The draft is split (2026-10-17).** The product thinker confirmed the decomposition as proposed: this intent holds the capability alone. The trust rule (every wait has a deadline, a claim's generation number is re-checked before a push, queue or tag, and a reader refuses a newer coordination record) becomes an ADR at planning, and the stance that coordination lives on the shared remote, never only in conversation, becomes a principle. The table is in the decomposition-calibration note under this date.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
