# SOTA survey — several coding agents on one machine and one repository: claims, leases, worktrees, locks

Dated 2026-09-30. The sweep iss-2608230943533581 asked for before itd-33
(draft) proceeds, run under the
[research protocol](2026-08-22-sota-research-protocol.md) as far as one
author can run it (see the review record). One host-run research pass
(primary sources read on 2026-09-30) plus a survey of what this repository
already ships, challenged for fit per
[`prefer-sota`](../../principles/prefer-sota.md). The transport question
(how sessions pass messages) is covered by
[`2026-09-15-agent-messaging-on-a-local-network.md`](2026-09-15-agent-messaging-on-a-local-network.md)
and is not repeated here.

The capture names four questions: what substrates the field uses for
same-machine multi-agent work on one repository; whether a claim-and-yield
contract of itd-33's shape is state of the art or a local invention; how
other systems make a session's working tree unambiguous; and what the
evidence says about coordination overhead against parallel throughput.

## 1. The substrates in use

| System | Isolation of the tree | Claim on work | Liveness | Enforcement |
| --- | --- | --- | --- | --- |
| The first harness's agent teams (experimental) | none by default; "Two teammates editing the same file leads to overwrites. Break the work so each teammate owns a different set of files" | a shared task list; "Task claiming uses file locking to prevent race conditions"; tasks carry dependencies | a lead that is fixed for the session; idle notifications | none on files; hooks (`TaskCreated`, `TaskCompleted`, `TeammateIdle`) can refuse a transition |
| Cursor parallel agents | "Worktrees let Agent work in isolated Git checkouts"; a machine-wide cap of 25 worktrees, swept every six hours | none; the person applies or deletes each result | n/a | the worktree boundary |
| MCP Agent Mail (MIT, independent) | none | "advisory file reservations (leases)" on paths or globs, exclusive or shared, with a TTL, renewable and releasable | TTL expiry | advisory: conflicts are reported "alongside grants"; an optional pre-commit hook blocks a commit that touches another agent's exclusive reservation |
| git itself | `git worktree add` "refuses to create a new worktree when <commit-ish> is a branch name and is already checked out by another worktree" | n/a | `git worktree lock` protects a worktree's metadata from pruning, not its contents | the refusal, unless `--force` |
| abcd today | the machine-scoped worktree store (convention; its verbs are itd-2609091014076309, drafted) | `abcd implement claim`: one file per record by exclusive create, a lease (default 2h, 1m to 24h), a lapsed lease claimable again and logged, contention at exit 3, the back-off logged (shipped by itd-2609221656373558) | the lease | refusal at claim; a second session is also refused a second live claim |

Evidence tiers: vendor documentation for the harness and Cursor; project
documentation for MCP Agent Mail and git.

## 2. Is claim-and-yield of itd-33's shape state of the art?

**The claim half is established; the yield-and-escalate half is local.**

- **Leases are forty years old.** A lease is a lock with a term, so a
  crashed holder cannot hold forever (Gray and Cheriton, SOSP 1989). Every
  system above that claims anything uses one: the harness's task lock, Agent
  Mail's TTL reservations, abcd's claim files. itd-33's heartbeat-and-lapse
  (three missed 30-second beats) is a lease renewed by heartbeat.
- **A lease without a fencing token is unsafe when the holder can pause.**
  "If the GC pause lasts longer than the lease expiry period, and the client
  doesn't realise that it has expired, it may go ahead and make some unsafe
  change"; the remedy is "a number that increases ... every time a client
  acquires the lock", checked by whatever accepts the write (Kleppmann,
  2016, practitioner tier). An agent session is the extreme case: it can sit
  in a long model call, or be suspended with the laptop lid, far longer than
  any lease. Neither itd-33 nor `implement claim` carries a fencing token
  today; git's own refusal to check one branch out twice is the only fence
  in the stack.
- **Typed work items, not files.** itd-33 cut file-level claims; the field
  splits. The harness and Cursor claim tasks or isolate trees and leave files
  to the person; Agent Mail claims files. abcd's shipped claim is on a record
  id (plus optional declared paths), which is itd-33's `intent_promotion` /
  `spec_task` idea at record granularity.
- **Yield and escalate** (`wait_then_swap`, `swap_now`, `sequence`,
  `keep_both`) have no counterpart in any system surveyed. The closest is
  Agent Mail returning conflicts beside grants and leaving the response to
  the agents. This half is a local invention, and the evidence does not yet
  say it is needed: the shipped claim refuses at exit 3 and the session backs
  off, which is yield without a vocabulary.

## 3. Making the tree unambiguous

The field's answer is **one checkout per agent, created by the tool, not
remembered by the agent** (Cursor; the harness's worktree isolation for
subagents; git's branch refusal). What none of the sources address is the
failure iss-2608230847432285 records: a shell whose working directory
silently returns to the shared checkout. Isolation by worktree only helps if
every command is anchored to it. The practices that close that gap are
local: absolute paths in every command, a first command that proves `HEAD`
is the expected sha, and a lint of the tree before a commit (this run's
common lane rules hold all three). No primary source was found that
measures how often agents lose their directory.

## 4. Coordination overhead against throughput

- "Multi-agent systems use about 15× more tokens than chats", and "most
  coding tasks involve fewer truly parallelizable tasks than research, and
  LLM agents are not yet great at coordinating and delegating to other
  agents in real time" (Anthropic engineering, vendor tier).
- Across 260 configurations on six benchmarks, the authors report "a
  robust capability-saturation effect", and "relative performance change
  compared to single-agent baseline ranges from +80.8% on decomposable
  financial reasoning to -70.0% on sequential planning"; "architectures
  without centralized verification tend to propagate errors more than those
  with centralized coordination" (Kim et al., arXiv 2512.08296, research
  tier; quoted from the abstract, the body was not read).
- A taxonomy of 1,600 annotated traces across seven frameworks finds 14
  failure modes in three families, one of them "inter-agent misalignment"
  (Cemri et al., arXiv 2503.13657, research tier).
- The harness's own guidance: "Start with 3-5 teammates", "Three focused
  teammates often outperform five scattered ones", and coordination overhead
  "increases" with team size (vendor tier).

**Run testimony, not measurement.** This repository's autonomous runs cap
live agents (five, ruled 2026-09-29) and have logged three coordination
incidents of the lock-and-kill kind: a pattern kill that took down two peer
lanes' gates (2026-09-23); a hand-rolled lock loop that removed a lock
another lane held, so two preflights ran at once (2026-09-29); and a peer
worktree created inside the checkout that made a concurrent preflight
flake (iss-2608261331317889). Each was fixed by a narrower rule: kill only
your own process group; a lock released only if its owner file still names
the releasing process (a fence in all but name); worktrees outside the tree.

## Adversary filter

- **A small contract, not an orchestration substrate** (itd-33's own
  stance) survives: no system surveyed shows a framework beating plain
  files plus exclusive create on one machine.
- **Host-agnostic.** The harness's agent teams are experimental, one team per
  session, and vendor-specific; abcd cannot build on them, only interoperate.
- **No new dependency.** Agent Mail brings a server, SQLite and a Python
  runtime; its reservation *shape* is worth borrowing, not the software.
- **Machine-scoped, not in the tree.** itd-33 places live state under
  `.abcd/coordination/` in the working tree; the shipped claim lives in the
  machine-scoped run store, keyed on the root commit, which the worktree
  principle and the incident above favour.
- **Measure before building yield.** The shipped `implement report` already
  derives a comparison of division modes from the run log; that log is
  where the need for escalation would show.

## Verdict (for itd-33's planning; not a ruling)

1. **Keep the claim, on the shipped seam.** itd-33 should build on
   `abcd implement claim` (exclusive create, lease, lapse, log in the
   machine-scoped run store) rather than a second claim file under
   `.abcd/coordination/`. The shape is state of the art.
2. **Add a fencing token before extending the lease's reach.** A
   monotonically increasing claim generation, written into the claim file
   and re-checked by every write the claim protects (a commit, a record
   move, a lock release), closes the pause-past-expiry hole the preflight
   lock already had to close by hand.
3. **Defer yield and escalate** until the run log shows contention that
   exit-3-and-back-off did not resolve; the four-way escalation menu has no
   precedent and no measured need.
4. **Make the tree unambiguous by construction**: ship the worktree store's
   verbs (itd-2609091014076309), so a lane's checkout is created and
   recorded by a verb, and have every lane-facing verb refuse when its
   working directory is not the checkout its claim names.
5. **Cap parallelism by evidence**: the field's three-to-five and this run's
   five agree; revisit only with the run log's own comparison.

What would show this wrong: a run in which two sessions hold one record
without a lapse in between (the claim is broken), or a run log in which
back-off repeatedly costs more than an escalation would have (yield is
needed after all).

## Review record

2026-09-30: authored in one pass by an implementer in autonomous run A (lane
drainResearch), with the fit-challenge run in-pass by the author. The
protocol's second research agent and its two independent reviewers were not
run (the run forbids a lane from spawning agents); that review is owed
before itd-33's planning cites this note.

## Sources (all accessed 2026-09-30)

- [Claude Code — Orchestrate teams of Claude Code sessions](https://code.claude.com/docs/en/agent-teams)
- [Cursor — Worktrees](https://cursor.com/docs/configuration/worktrees)
- [MCP Agent Mail — repository and README](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [git-worktree documentation](https://git-scm.com/docs/git-worktree)
- [Gray and Cheriton, Leases: an efficient fault-tolerant mechanism for distributed file cache consistency (SOSP 1989), ACM DL record](https://dl.acm.org/doi/10.1145/74850.74870) — the record page refused an automated fetch (HTTP 403); cited from the bibliographic record
- [Kleppmann, How to do distributed locking (2016)](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html)
- [Anthropic — How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Kim et al., Towards a Science of Scaling Agent Systems (arXiv 2512.08296)](https://arxiv.org/abs/2512.08296)
- [Cemri et al., Why Do Multi-Agent LLM Systems Fail? (arXiv 2503.13657)](https://arxiv.org/abs/2503.13657)
