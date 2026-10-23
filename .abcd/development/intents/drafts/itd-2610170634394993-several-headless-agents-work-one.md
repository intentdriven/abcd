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
related_intents: [itd-2610170638184486, itd-2610222029228127, itd-2610100704270254, itd-78, itd-115, itd-2609201916151817, itd-82, itd-2609201925079472, itd-148]
supersedes: [itd-2609150819440345]
---

# Headless agents in parallel coordinate through the shared remote, not conversation

## Press Release

> **Several headless agents work one repository at once, and none of them needs a conversation to know what the others are doing.** Each agent takes a piece of work by a claim on the shared remote, says in a line abcd reads which other change it waits on, and gets a share of the machine and of the model budget from abcd rather than from a message. A change that waits on another is not queued until that one lands, and is rebuilt on top of it when it does, so it meets its conflict once. Any session can run the pass that compares what the records say should happen with what is happening and acts on the difference, so no one process has to stay alive, and every wait has a deadline. One central view shows, at a glance, what every session holds, what it is doing and what it waits on, each line saying how long ago it was last seen, while the detail behind each line stays in that session's own checklist on its computer. No build or drain lane abcd launches works under the person's own forge login: each gets a credential of its own, limited to the repository and the acts its work needs and expiring with it, so a session can neither change the project's settings nor reach anything it was not given.
>
> "Last night one session spent its evening telling four others when a pull request had merged, and a new one had to be asked by message to hold its own," said Maya, an autonomous-development practitioner. "My headless agents never read a message. Now each one writes what it waits on where every other agent already looks, abcd holds it until that lands, and when an agent dies its work goes back to the pool instead of waiting on a promise nobody is left to keep."
>
> _Agent-seeded on 2026-10-17 from the evening of 2026-10-16 and a state-of-the-art pass the same night ([research note](../../research/notes/2026-10-17-parallel-headless-agent-coordination.md)); not yet put to the product thinker._

## Why This Matters

abcd's coordination pieces exist but are joined by conversation. Claims (itd-2609091034175565) keep two agents off one record and its files. The load gate keeps preflights from swamping the machine. The merge queue lets one change through at a time. What ties them together on 2026-10-16 was sessions messaging each other: one session asked a new session to hold its records pull request until another landed, and promised to tell four waiting sessions when it merged. A headless agent receives no such message, and a promise lasts only as long as the session that made it.

The same evening showed what a missing deadline costs. The load gate deadlocked for hours between two builds of its own queue rule (iss-2610161634043273), and no wait timed out for eight hours. Two preflights failed on the gate's two-hour limit alone, with every check before it passing (iss-2610161639025702).

Published measurements say the collisions are common, not rare: 19.8% of agent pull requests open at the same time conflicted when one agent made them and 41.7% across agents, and telling an agent what had already landed recovered most of the runs that passed alone and failed together (research note, "Multi-agent coding").

## Mechanism

We expect parallel headless agents to stop colliding and stop idling on each other, because every fact they now pass by message (who holds what, what waits on what, whose turn it is, how much budget is left) becomes a record on the shared remote that each agent already fetches and that any session can act on. Shown wrong if, in a measured run of three or more headless agents, the agent minutes lost to conflicts, rebuilds after a dependency landed, and waits that ended in a timeout are no lower than in a run of the same agents with only reservations and the load gate (dcn-2610180720412979: headless agents cannot coordinate by message, so the shipped coordination is the baseline).

## Scope Conditions

- Holds for agents that can push to the shared remote: people with write access and the cloud sessions they start. An outside contributor reads the records but cannot write one.
- Holds for one repository on one forge whose merge queue is the only route to the default branch.
- Holds for the machine budget (preflights, race runs) only among agents on one machine, which is all the process table shows. The model budget spans every account of one organisation, because the provider's rate limits do.
- Holds where the forge can issue a credential narrower than the person's own login, limited by repository and by act (push to its own branches, open a pull request, read), and expiring; GitHub's App installation tokens and fine-grained tokens are the candidates.
- Holds for the build and drain lanes abcd launches on its own, which start only where the person's own forge login is absent (a cloud session, a container or a separate user account); on a computer where that login is saved, abcd refuses to launch one. The person's own sessions, those working while the person is away included, keep running on their own computer under their login, and a drain the person starts and watches may count as attended (dcn-2610222025104969, dcn-2610222030477927, narrowed by dcn-2610230710469123).
- Holds while at most one minor version of abcd's coordination records is in use at once; a reader meeting a newer record refuses it rather than guessing.

## Acceptance Criteria

- Given an agent whose branch waits on an open pull request, and says so in a line abcd reads, when it asks abcd to land the branch, then abcd does not queue it while that pull request is open, rebuilds the branch on top of it once it merges, and queues the result.
- Given a declared dependency that is closed without merging, cannot be resolved, or closes a cycle, when abcd reads it, then abcd refuses to land the branch and names the dependency and the reason, rather than waiting.
- Given an agent holding a claim that stops renewing it, when the claim expires and a delay has passed, then another agent can take the work, and a push or tag abcd makes for the first agent under its old claim is refused because it carries a renewal that is no longer a fast-forward, and a queue entry is refused by a re-read of the claim made just before it (a window of one call, stated, not claimed closed).
- Given three headless agents on one machine with a budget of two preflights, when all three reach their preflight at once, then two run and the third waits, and no wait lasts beyond its stated deadline without ending in a named refusal.
- Given the session running the coordination pass is stopped mid-pass, when another session runs the pass, then it reaches the same decisions from the records on the remote alone, with nothing lost.
- Given two agents running different versions of abcd's coordination records, when they read each other's records, then neither waits on the other in a cycle, and the one meeting a newer format than it knows refuses with a message naming the version.
- Given several sessions on different computers, each keeping its own checklist locally, when the person or any session opens the central view, then it shows one line per session (what it holds, its one-line state, what it waits on, when it was last seen) built from the short lines the sessions published, with no program having to stay running.
- Given a session that changes what it holds or waits on, when it next writes its checklist, then it publishes its short line only after the checklist is written, and publishes nothing when nothing in the line changed.
- Given a session that stops without notice, when the central view is opened, then its line reads as live, then as not seen for a stated time, then as work another session may take over after a delay, then it is gone; it never reads as working once its renewals have stopped.
- Given an autonomous session started by abcd, when it pushes, opens a pull request or reads the forge, then it does so under a credential issued for it alone, limited to the repository and the acts its work needs and expiring with it, never under the person's own login; and when it tries an act outside that set (changing a repository setting or variable, deleting a branch under a protected prefix, reaching a repository outside its token's reach), the forge refuses it (narrowed per dcn-2610180720415625: a token that can push can delete an unprotected branch).
- Given the person's own login holds admin rights on the repository, when an autonomous session runs, then nothing the session can reach carries those rights, so a setting only the maintainer may change (the reservation cap among them) stays out of its reach.
- Given a computer where the person's forge login is in a place abcd checks (the forge CLI's saved login, a token variable, a stored-credential file naming the host), when abcd is asked to launch a build or drain lane there, then abcd refuses to launch it, names each place it found a login without reading or printing the login, and says what it cannot see (a keychain-only login, an SSH key); and given a cloud session, a container or a separate user account with none of them, the lane starts with its own App token, which travels through abcd's own credential channel and so never trips the check. The person's own sessions are not checked (dcn-2610230710469123).
- Given the repository sets a sub-agent budget of eight and two sessions are running on one computer, when both start sub-agents, then together they never hold more than the budget, and a session that finds the budget spent waits for a slot under a stated deadline rather than starting one. Whether the pool also spans computers is open below.
- Given an agent about to re-submit a branch after a change it overlaps with has landed, when abcd rebuilds it, then the agent is given a summary of what landed since its base.

## SOTA

Surveyed 2026-10-17 against primary sources ([research note](../../research/notes/2026-10-17-parallel-headless-agent-coordination.md)):

- **Kubernetes controllers and leases, Chubby sequencers:** mature. Adopted as the pattern: coordination is a pass that compares wanted and current state held in shared records, a claim is a compare-and-swap with an expiry and a generation number that a later act re-checks, and a dead holder's claim is retaken only after a delay. The shared remote already gives abcd the compare-and-swap (a push that is not a fast-forward is refused).
- **Job queues with timeouts (Faktory, Temporal):** mature. Adopted: work comes back by timeout, never by noticing a death, and every wait has a wait-to-start limit, a per-attempt limit and a heartbeat.
- **Merge queues and dependency lines (GitHub merge queue, Mergify `Depends-On:`, git-spice, Aviator):** mature. GitHub's merge queue stays the only route to the default branch. The dependency line and the footprint routing are the parts abcd has to add, because the queue it uses reads neither.
- **Multi-agent coding products (Claude Code agent teams, Copilot's cloud agent):** young. Agent teams coordinate through a shared task list but are bound to one session on one machine, so they cannot carry coordination across accounts or into the cloud. Not adopted as the layer.
- **A coordinator cluster (etcd, a Temporal server):** mature but out of scale for one repository. Not adopted.

Path taken: 2. The forge and the shared remote are the native floor (the queue, compare-and-swap pushes, branches cloud sessions can push), and the pass, the dependency line, the generation number and the budgets are bespoke to abcd's records, behind a seam.


**Carried in from itd-2610240850336695, its Decision 19 (product thinker,
2026-10-24).** When sessions push under a credential of their own, the
repository gains a ruleset refusing non-fast-forward pushes on every branch,
with a bypass for the person's admin role: a session and abcd then cannot
force-push anywhere, and the person still can, through `gh` or the forge's web
interface. Until then the person's grant ships with the force-push stop on
feature branches resting on this machine's pre-push hook and the guard.

**The bundle gains a third member (product thinker, 2026-10-24).** The
person's grant, itd-2610240850336695, joins the autonomous-agents bundle with
this intent and the checklist, so its force-push stop lands with this intent's
credential split and the all-branch ruleset above.

## Open Questions

Answered on 2026-10-18 by the planning prep (facilitator, each a decision
record) and on 2026-10-22 by the product thinker:

- Who runs the pass: any session on demand, before it acts; no timer lease (dcn-2610180720366282).
- Where "waits on" lives: a `Depends-On:` line in the pull request body, mirrored as a hint on the status line (dcn-2610180720362692).
- Merge or restack: merge, restacking only after a hand squash or rebase merge (dcn-2610180720369860).
- The machine budget: the load gate's slots, weighted per job (dcn-2610180720376153). The model budget: shared across sessions (dcn-2610222025116937, the product thinker's reversal of dcn-2610180720379757).
- The credential: a GitHub App token minted for the person, eight-hour expiry (dcn-2610180720379765, dcn-2610180723332655); the App registered once in the browser (dcn-2610180720389628); shipped with the coordination (dcn-2610222025116835).
- The release cut: the announcements intent's release reservation, no new lease (dcn-2610180720382472).
- "I am merging, hold": a hold is a reservation; the announcement informs (dcn-2610180720389352).
- Push rate: renewals hourly, the line only on change, back-off on refusal (dcn-2610180720390166).

Still open, for the product thinker's sign-off:

- **Which unattended runs the login check covers: settled 2026-10-23 by the product thinker** (dcn-2610230710469123, decision 12). Only the build and drain lanes abcd launches on its own; the person's own sessions keep the login they run under.
- **How a session's token is issued: settled 2026-10-22, on delegation** (dcn-2610222048069284). A user access token from the device flow, so no client secret ships anywhere; the refresh token stays on the person's computer, outside every session; each launch gets an eight-hour token, and a cloud session's pushes stop when it lapses, queued for the person's return.

- **How far the shared model budget reaches.** Sessions on one computer can share a pool kept on that computer, as the load gate's slots already are; sessions on several computers or accounts would need the pool kept on the shared remote, which costs a push each time a sub-agent starts or ends. The scope condition above says the model budget spans every account of one organisation; the product thinker's answer of 2026-10-22 said "shared across sessions" without naming the reach.

## Decisions

1. **The draft is split (2026-10-17).** The product thinker confirmed the decomposition as proposed: this intent holds the capability alone. The trust rule (every wait has a deadline, a claim's generation number is re-checked before a push, queue or tag, and a reader refuses a newer coordination record) becomes an ADR at planning, and the stance that coordination lives on the shared remote, never only in conversation, becomes a principle. The table is in the decomposition-calibration note under this date.
2. **Ownership with take-over lives here, and the check before an unattended run does not (2026-10-17).** The product thinker chose to keep the check of what could block a long run, which asks every session what only the person can unblock, in the per-session checklist draft, done by asking sessions by message. Work owned with an expiry, and taken over when its holder stops renewing, stays in this intent; it had one defensible home, so it was recorded rather than asked. The checklist draft is where a session keeps its own state; this intent is what sessions share.
3. **The central view is one short published line per session, with the details kept locally (2026-10-17).** The product thinker asked for a central place to see at a glance what is taken and worked on, with each session's details kept within that session so they survive compaction, and asked for a state-of-the-art pass on the link between the two ([research note](../../research/notes/2026-10-17-parallel-headless-agent-coordination.md), "The central view and each session's detail"). Offered a short line per session with the details local, or the full checklist uploaded as well so another computer could resume a dead session, they chose the short line: each session's checklist on its own computer is the source of its detail, the session publishes one short line about itself when that line changes, the claim renewal it already makes shows it is alive, and the view is every line put together, built by whoever reads it. A dead session's unpublished steps are lost; full uploads can be added on top later.
4. **This intent supersedes the register draft (2026-10-17).** The register draft (itd-2609150819440345) stated the problem this intent answers: who holds what was settled only in conversation. Its want, a central place to see who holds what, is decision 3's view, and its claim branch shipped as reservations. Its helper program on each machine, reachable over a private network to show work not yet pushed, is dropped: it needs a process left running, which decision 3 avoids. It can be raised again as a new idea.
5. **Autonomous sessions never use the person's own forge login (2026-10-17).** The product thinker ruled that autonomous sessions work under a weaker credential of their own, given on a need-to-know basis, and that this belongs to this intent and to the autonomous-agents bundle. It was found that day that every session on the machine used the maintainer's own `gh` login, which holds admin rights on the repository, so any session could change a setting meant for the maintainer alone (the reservation cap, captured the same day).
6. **The central view says what each session is actually doing (2026-10-17).** Restated from the product thinker's ruling relayed by the checklist draft's session: the person must never wrongly believe agents are working or waiting. A line's state is therefore derived from what the machine and the remote can see (the renewal, the process, the last commit), and what an agent writes about itself is a hint beside it, because agents' own status reports lag.

7. **Unattended runs happen only where the person's forge login is absent (2026-10-22, C1).** The product thinker chose a real guarantee, set up once, over saying plainly that the login is still within reach (dcn-2610222025104969). abcd checks for a saved login before an unattended run starts and refuses where it finds one (dcn-2610222030477927, facilitator).
8. **Only the short line is shared now; full checklist uploads come later (2026-10-22, C2).** Settles the two rulings of 2026-10-17 in favour of later and keeps decision 3 as written (dcn-2610222025114805). The later uploads are itd-2610222029228127.
9. **Agents' own App tokens ship in the same release as the coordination (2026-10-22, C3).** The person registers abcd's GitHub App once in the browser before that release is usable (dcn-2610222025116835).
10. **The model budget is shared across sessions (2026-10-22, C-confirm).** A typed reversal of dcn-2610180720379757 and of the product thinker's 2026-10-10 answer "each abcd run" in itd-2610100704270254: the budget is one pool every session draws on, and one busy session can slow the others, which the product thinker accepted (dcn-2610222025116937). The press release keeps "a share of the model budget".
11. **The trust rule and the stance are written (2026-10-22).** The trust rule is adr-2610222026262980, proposed until the plan's sign-off, then a brief invariant; the stance is the principle `coordination-lives-on-the-shared-remote`.
12. **Only abcd's own helpers run away from the person's login (2026-10-23, narrowing C1).** The product thinker ruled that the build and drain lanes abcd launches on its own must run where the person's GitHub login is absent, while the person's own sessions working while they are away keep running on their own computer under their login, and a drain the person starts and watches may count as attended (dcn-2610230710469123). Decision 7's guarantee therefore covers abcd's launches, not the person's own sessions, and the login check runs before a launch only.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
