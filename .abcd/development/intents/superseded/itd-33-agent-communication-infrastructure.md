---
id: itd-33
slug: agent-communication-infrastructure
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-03, from: standalone, to: superseded, reason: "Replaced by reserving work through a claim branch on the shared remote; its one-computer active-work file is excluded by the 2026-10-03 rulings (product thinker, 2026-10-03)" }
blocked_by: [itd-121]
builds_on: [itd-2609201916151817, itd-22]
severity: major
kind_at_supersession: standalone
superseded_by: itd-2609091034175565
---

# Multiple Agents Coordinate On Intent And Spec Work Without Duplicating Effort Or Producing Competing Artefacts

> **Superseded by itd-2609091034175565** on 2026-10-03: Replaced by reserving work through a claim branch on the shared remote; its one-computer active-work file is excluded by the 2026-10-03 rulings (product thinker, 2026-10-03)

## Press Release

> **abcd ships a coordination layer so multiple agents can work on the same intents and specs without duplicating each other's work.** Every agent (Claude Code session, an autonomous run via the pluggable run seam, another harness) declares which `work_item` it's progressing — an `intent_promotion` (`itd-N`), a `spec_task` (`spc-N.M`), or a `command_run` (e.g., `intent.shape`) — by writing to `.abcd/coordination/active-work.json` before starting. Before an agent begins a `work_item`, it calls `take(work_item)`. The result is either *clear* (the agent acquires) or *currently-held-by-X*. Agents in conflict either `yield` (defer to the holder) or `escalate` (surface to the human via bare `/abcd`). The coordination state is plain JSON in the repo — no daemon, no orchestration runtime, no LangGraph/CrewAI/AutoGen dependency. abcd's contribution is a small contract layer (~3 functions, ~8 audit-log event types); the human-facing surface is rendered by the bare `/abcd` dispatcher (itd-20).
>
> "I had Claude Code in one terminal running `/abcd:intent plan itd-33`, and forty minutes later I opened a second terminal and ran the same command — completely forgetting the first was still mid-promotion," said Maya, AI/agent researcher. "Without coordination they'd each have produced a competing `spc-33` spec and I'd have spent another hour deciding which to keep. With this, the second agent reads `active-work.json`, sees Claude Code is mid-`intent_promotion(itd-33)`, and yields with a one-liner explaining who's holding it. No duplicate work, no merge dance."

## Why This Matters

abcd is on a trajectory to host **multiple concurrent agents in the same repo**:

1. **Same human, multiple terminals** — alex opens Claude Code in terminal 1, walks away, opens terminal 2 and reruns the same command. The most common collision pattern in practice.
2. **Autonomous runs** (the pluggable run seam, contextualised by itd-29) — long-running agents that the user kicks off and walks away from. They share a working tree with whatever the user does next.
3. **Multiple harnesses** — Claude Code today, opencode (itd-22); the same project may be opened by a user in one harness while another agent runs in another.
4. **In-session subagent dispatch** (itd-2) — the parent session spawns subagents whose work is *part of* the parent's progress, not a parallel actor (subagents inherit the parent's `agent_id`; they do not hold their own claims).

Today, coordination between these is **emergent at best**:

- **No shared mid-flight state.** Agent A doesn't know that agent B is partway through promoting `itd-N`. Both produce competing specs.
- **No yield/escalate vocabulary.** When two agents do realise they're in conflict, the only resolution is "the user notices and arbitrates" — manual, lossy, slow.

This intent **captures the concern now** so the design conversation can start before the first multi-agent collision lands real work in the bin. The trigger to escalate from `drafts/` to `planned/` is concrete (see "Revisit Triggers"); implementation is deferred until the actual texture of multi-agent collisions is observable.

**Deliberately out of scope: file-level claims.** The earlier draft of this intent proposed a per-file claim/heartbeat protocol to prevent two agents writing the same file at once. That has been cut. Reasons: (a) git merge conflicts surface concurrent text edits loudly, not silently — the failure mode is recoverable; (b) the duplicate-*work* problem (two specs, two parallel plans) is the genuinely-unsolved case where git can't help (different filenames, same intent); (c) the file-claim infrastructure (per-file heartbeats, stale-claim sweeps, `uncoordinated_write_by` flagging, substrate selection from LangGraph/CrewAI/AutoGen) was disproportionate weight for a problem git already solves. If file-level collisions become a recurring real pain, a follow-up intent can revisit.

abcd's stance is **a small contract, not an orchestration substrate**. Plain JSON files, three primitives (`take`/`yield`/`escalate`), three contract functions exposed to itd-20's bare `/abcd` dispatcher. No new framework dependency, no graph runtime, no agent-to-agent chat.

## What's In Scope

- **Agent identity surface.** Every agent declares: `agent_id` (ephemeral; minted at harness invocation as `<harness>-<starttime>-<6char-rand>`), `harness` ∈ `{claude-code, opencode, ralph, external}`, `human` (sourced from git config — the stable cross-session attribute), `started_at`, `last_heartbeat_at` (refreshed every 30s while running), and `working_under` (a single `work_item` ref or `null` between claims). Subagents (in-session `Task` dispatch from itd-2, named subagents like `intent-fidelity-reviewer`) inherit the parent's `agent_id` — they do not register independently.
- **Typed `work_item` claims.** The unit of coordination is a typed work item, not a file region:
  - `intent_promotion(itd-N)` — exclusive. Two agents cannot promote the same intent at once.
  - `spec_task(spc-N.M)` — exclusive at task granularity, **not spec granularity**. Agents A and B can both work `spc-7` if on different tasks (`.2` and `.3`); they collide only if they pick the same task.
  - `command_run(<command-id>)` — exclusive per command + scope. Two simultaneous `intent.shape` runs would race; this generalises the existing `shape.lock` pattern (and forthcoming `consistency.lock`) into a typed family.
  - **Brief and intent-draft *edits* are NOT claimable.** The brief was deliberately split into numbered folders for concurrent editing (per `00-meta.md`); ad-hoc file edits remain git-merge-mediated as today.
- **Three primitives — `take`, `yield`, `escalate`.** No `queue` (no wake-up mechanism; polling solves the slow-work-item case); no `coordinate` (contradicts the no-agent-to-agent-chat boundary). The default success path is `take`; the polite refusal path is `yield`; the human-in-the-loop path is `escalate`.
- **Cooperative-checkpoint escalation.** Escalation is *advisory*, not mid-flight abort. When the human resolves an escalation, agents observe the resolution on their next heartbeat tick (every 30s) and complete-current-work-item-then-yield rather than mid-LLM-call termination. The human who really wants a hard stop can Ctrl-C the agent's terminal — the coordination layer does not signal processes it doesn't own.
- **Escalation menu** (for two agents): `{wait_then_swap, swap_now, sequence, keep_both}`. No `kill agent X` option. Three-or-more-agent conflicts decompose into N–1 pairwise escalations against the holder.
- **Claim lifecycle: three states.** `claim.status` ∈ `{active, paused, released}`. The `paused` state interlocks with itd-29's pause/resume/rewind: paused agents keep heartbeating but their `working_under.status = paused`; resume returns to `active`; rewind releases and re-acquires at the rewind target.
- **Bare `/abcd` shows live agent activity** (rendered by itd-20). itd-33 contributes three contract functions — `render_coordination_status()`, `pending_escalations()`, `resolve_escalation(escalation_id, choice)` — consumed by itd-20's dispatcher. itd-33 introduces NO new `/abcd:coordination` namespace and NO new sub-verbs; the human-facing verbs (bare `/abcd`, top-level `/abcd resolve`) belong to itd-20.
- **State storage partition.** All under `.abcd/coordination/`:
  - `active-work.json` — local-only, gitignored. Per-machine live state. If corrupt or missing, the next agent rebuilds it (empty feed = no claims held).
  - `*.lock` — local-only, gitignored. `flock(2)`-mediated file locks for `command_run` claims. Existing `shape.lock` is the prior art.
  - `audit/<YYYY-MM-DD>.jsonl` — committed. Append-only JSONL, daily UTC rotation. Cross-machine via git: line-level merges, sort by timestamp on read.
  - A `.abcd/coordination/.gitignore` enforces the partition (`active-work.json` and `*.lock` ignored; `audit/` tracked).
- **Audit-log schema** (durable contract, eight event types):
  - `agent_registered` — agent process starts, first reads `active-work.json`. Fields: `agent_id, harness, human, started_at`.
  - `take` — agent acquires a clear work_item. Fields: `agent_id, work_item.{type,id}, at`.
  - `release` — agent finishes work_item normally. Fields: `agent_id, work_item.{type,id}, at, outcome ∈ {completed, abandoned}`.
  - `lapsed` — stale-claim sweeper auto-releases (last_heartbeat_at older than 3× heartbeat interval). Fields: `agent_id, work_item.{type,id}, at, last_heartbeat_at, reason: heartbeat_lapsed`.
  - `yield` — agent voluntarily yields a held claim (cooperative checkpoint). Fields: `agent_id, work_item.{type,id}, at, to_agent_id` (if known).
  - `escalate_requested` — agent invokes escalate against a held work_item. Fields: `requesting_agent_id, holding_agent_id, work_item.{type,id}, escalation_id (UUID), at`.
  - `escalate_resolved` — human resolves an escalation. Fields: `escalation_id, choice ∈ {wait_then_swap, swap_now, sequence, keep_both}, resolved_by_human, at`.
  - `agent_unregistered` — agent process exits cleanly. Fields: `agent_id, at`.
  - Every line carries `schema_version: 1` and an ISO-8601 millisecond UTC `ts`. Heartbeats are NOT logged (high-frequency noise; liveness lives in `active-work.json`). Failed `take` attempts are NOT logged (only state-changing events appear).
- **Always-on.** No opt-out flag. No "single-agent mode" toggle. The cost of always-on is a few file reads/writes per work_item — invisible if no second agent ever appears, and the same-human-two-terminals case alone justifies the overhead.

## What's Out of Scope

- **File-level claims.** Cut entirely (see "Why This Matters"). Two agents writing the same file is git-merge territory; abcd does not introduce a per-file claim protocol.
- **Real-time agent-to-agent chat / `coordinate` primitive.** Agents do not negotiate split/merge with each other. When two agents need to compromise, the answer is `escalate` to the human.
- **`queue` primitive.** No wake-up mechanism exists; polling solves the slow-work-item case naturally. Add later if real collisions show queue earns its place.
- **Mid-flight agent abort.** Escalation is cooperative-checkpoint — agents observe revocation on next heartbeat and finish-then-yield. Hard stop = Ctrl-C, not a coordination primitive.
- **Authoritative shared memory across agents.** Each agent retains its own context window; coordination shares *which work_item is held*, not *thinking*. Sharing thinking is itd-25 (dredge) territory.
- **Substrate selection (LangGraph / CrewAI / AutoGen).** None of these are needed for the narrowed scope. The contract is plain JSON files plus `flock(2)` for `command_run` claims.
- **Subsuming Claude Code's own subagent tool.** Claude Code's `Agent` / `Task` tool stays the in-harness primitive; subagents inherit the parent's `agent_id` rather than registering as independent agents.
- **Distributed-systems-grade consensus.** Two agents on the same machine sharing a working tree is the hot path. Cross-machine via git is eventually consistent — sufficient.
- **Permission / authorisation matrix.** "Which agents can edit which files" is out of scope; coordination is about *avoiding duplicate work*, not *enforcing policy*. Permission is itd-18 territory.
- **Real-time UI dashboard.** Bare `/abcd` is text-shaped (terminal-friendly). Graphical dashboards are a future question.
- **Coordination across non-abcd projects.** A second project that doesn't ship abcd is invisible to this layer.

## Acceptance Criteria

- **Given** an agent calls `take(work_item)` against a project with no existing `.abcd/coordination/active-work.json`, **when** the call executes, **then** the file is created on first call, the claim succeeds, and an `agent_registered` + `take` pair lands in the audit log — no error, no warning, no required setup.
- **Given** agent A holds `intent_promotion(itd-33)` and is heartbeating, **when** agent B calls `take(intent_promotion(itd-33))`, **then** B receives `currently-held-by-A` with A's `agent_id`, `human`, `working_under`, and `last_heartbeat_at` age — without any race condition resulting in two simultaneous holders.
- **Given** agent A holds `spec_task(spc-7.2)`, **when** agent B calls `take(spec_task(spc-7.3))`, **then** the claim succeeds (task-level granularity, not spec-level) — both agents progress different tasks of the same spec concurrently.
- **Given** agent A holds a claim and crashes (no heartbeat for >3× heartbeat interval, default 90s), **when** any agent next reads `active-work.json` or attempts a conflicting `take`, **then** the stale claim is auto-released, the next agent acquires cleanly, and a `lapsed` audit-log entry is written with `reason: heartbeat_lapsed`.
- **Given** agent A holds a work_item and agent B requests escalation via `escalate`, **when** the human runs bare `/abcd`, **then** the coordination section lists the escalation with `escalation_id`, requester, holder, work_item, and age — and the human resolves via `/abcd resolve <escalation_id> <choice>` (verb owned by itd-20, calling itd-33's `resolve_escalation`). The agents observe the resolution on their next heartbeat tick and act according to `choice ∈ {wait_then_swap, swap_now, sequence, keep_both}`.
- **Given** the human runs bare `/abcd` while two agents are active and no escalations are pending, **when** the dispatcher returns, **then** the coordination section is renderable in ≤5 lines listing each agent's `agent_id`, `harness`, `human`, `working_under`, and `last_heartbeat_at` age — sufficient to answer "what's running in this repo right now" without consulting any other source.
- **Given** an autonomous run completes a work_item and exits, **when** the interactive user later runs bare `/abcd`, **then** the recent-activity summary surfaces the `release` event with `outcome: completed` and a pointer to the resulting spec, regardless of whether the interactive user was watching when the run finished. (Durability follows from the audit log being append-only and committed.)
- **Given** an autonomous loop is `pause`d via itd-29's pause primitive, **when** another agent reads coordination state, **then** the loop's `claim.status` shows `paused` (not `released`); on `resume` it returns to `active`; on `rewind` the claim is released and a fresh `take` happens at the rewind target — visible as paired `release` + `take` events in the audit log.

## Revisit Triggers

This intent moves from `drafts/` to `planned/` when ANY of the following happens:

1. **First duplicate-work incident** — two agents independently begin work on the same intent or spec, producing competing artefacts.
2. **First user reports stranded work** — an agent edited files but the user couldn't reconstruct who, when, or why.
3. **itd-2 ships and is used in anger** — once in-session subagent dispatch is real and frequent, the parent/child overlap window becomes observable. (Subagents inherit parent identity per this intent's contract; the trigger is whether *parent-level* multi-agent collisions become real.)
4. **opencode harness ships (itd-22)** — multi-harness operation makes coordination's value concrete and surfaces the cross-harness contract requirements.
5. **Two consecutive successful multi-agent runs** — proves the substrate works enough that the absence of coordination becomes the next bottleneck.

The first user to hit (1)–(2) is asked to record the texture in the `.abcd/work/issues/` ledger so the design proceeds against real evidence, not guesses.

**Note on the previous file-clobber trigger.** An earlier draft included "first merge conflict caused by two agents editing the same file" as a trigger. That trigger has been removed: the narrowed scope (work_item claims, not file claims) makes file-level merge conflicts an out-of-scope failure mode for itd-33. They remain a real pain — but a separate intent's pain.

## Open Questions

- **Heartbeat interval.** Recommend 30s heartbeat, 90s stale threshold (3× interval), configurable per-project. The previous draft's longer list of open questions has dissolved under the narrowed scope: substrate choice (no substrate); state-under-git partition (resolved — see "What's In Scope"); subagent identity inheritance (resolved — subagents inherit parent ID); itd-29 interaction (resolved — three-state claim); itd-2 interaction (resolved — parent inheritance); corruption recovery (`active-work.json` is local-only and trivially regenerable); naming (coordination, with `take`/`yield`/`escalate` as the primitives); cross-machine merge (JSONL line-level merge); external-agent flagging (cut with file claims).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Linkage note (2026-09-30)

`builds_on` named itd-2 (in-session subagent dispatch), which is superseded by
itd-2609201916056194. The edge is dropped rather than relinked: the successor,
a delegated agent run through a command-line model runner, does not carry the
in-session dispatch contract this record built on, and that dispatch is standing
practice the conventions router (AGENTS.md) states. Ruling CF2 of 2026-09-30
counts a blocker kept as a discipline as settled; the run's orchestrator read
the same of a practice the router carries, so the edge has nothing left to
build on (recorded by the integration lane of autonomous run A).

## References

- Adjacent intents: itd-2 (in-session subagent dispatch — subagents inherit parent identity per this intent's contract), itd-22 (opencode portability — first cross-harness consumer), itd-29 (autonomous-run resilience — three-state claim interlocks with pause/resume/rewind), itd-15 (self-dogfooded SOTA audit — must remain coordination-aware), itd-20 (top-level `/abcd` dispatcher — owns the human-facing render and resolve verbs that consume itd-33's three contract functions), itd-18 (permission templates — adjacent but distinct concern).
- Brief contracts: glossary entries for `work_item.type`, claim primitives, escalation choices, `release.outcome`, `claim.status` registered in `02-constraints/04-naming.md` (Reserved vocabulary table); audit-log path canonicalised in `05-internals/04-universal-patterns.md`.
- Methodological precedents: file locks (`flock`/`fcntl`); the actor model (Erlang/OTP); the autonomous run-seam pattern (single-agent today, multi-agent scaling tomorrow). The earlier draft's reference to LangGraph / CrewAI / AutoGen as substrate candidates has been dropped — the narrowed scope needs none of them.
