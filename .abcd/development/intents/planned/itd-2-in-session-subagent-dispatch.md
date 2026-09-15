---
id: itd-2
slug: in-session-subagent-dispatch
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: critical
---

# In-Session Subagent Dispatch — the Default Host-Delegated Oracle

## Press Release

> **abcd's default oracle backend needs zero external setup: it delegates to the host's in-session subagent.** Every oracle call — audits, plan-reviews, impl-reviews — routes to an in-session Claude subagent via the host's Task tool. No subscriptions to configure, no MCP servers to install, no CLIs to find. The persona gets Carmack-style reviews from the same `oracle` already running their session, out of the box. Opt-in oracle adapters (RepoPrompt and others) route reviews to a different vendor when a persona configures one — but nothing needs configuring for the default to work.
>
> "I was on a borrowed laptop with no external oracle configured," said Henry, junior developer. "abcd just used the in-session `oracle`. The review wasn't as cross-vendor as an external adapter would give, but it was instant and free — and it worked without me setting anything up."

## Why This Matters

Every external oracle adapter needs external setup — a desktop app, a CLI, a subscription. Requiring any of them before abcd can review anything is friction. So the default oracle path uses the abcd-running Claude session itself: host-delegated, always available, zero setup. External adapters are opt-in, layered above this default for personas who want cross-vendor review.

The challenge: abcd's core process cannot directly call the host's `Task` tool — that's a model-side capability, only Claude (running in the harness loop) can invoke it. Crossing that boundary requires a wire protocol.

## What's In Scope

- **Sentinel-output protocol**: when `AgentDispatch.dispatch_agent(agent_name="in_session", ...)` is invoked, the harness emits a structured payload (JSON) that the orchestrating Claude session reads as a tool result and translates into a `Task` tool invocation. The slash-command markdown (`commands/intent.md`, `disembark.md`, etc.) embeds instructions telling the host: "if the script returned a `task_dispatch` payload, invoke Task with these args."
- **Subagent context isolation**: each in-session dispatch spawns a fresh subagent context (no continuity by design — the no-state-between-calls property is documented as an in-session-specific limitation, distinct from an external adapter's chat-based continuity).
- **Result return path**: the Task subagent's output flows back via the host session into a file or stdout that abcd's loop reads on next iteration.
- **No-continuity caveat**: the same-chat re-review semantics an external oracle adapter may offer do NOT apply to in-session — each iteration is a fresh Task call. The audit-fix loop must re-prime context every iteration. This is acceptable: host-delegated in-session dispatch is the default oracle path, and per-iteration re-priming is its documented tradeoff.
- **Verdict parser interop**: in-session subagent outputs use the same `<verdict>...</verdict>` tag protocol as every oracle adapter.

## What's Out of Scope

- **Autonomously orchestrating multiple in-session subagents** (e.g. for parallel review of different facets) — single subagent per call; multi-agent orchestration is a future intent or part of a broader brief revision.
- **Routing to specific Anthropic models** (Opus vs Sonnet vs Haiku) — uses whatever model the user's Claude Code session is running. abcd doesn't pick.
- **Cost optimisation** (in-session calls bill against the user's Claude subscription; abcd doesn't budget) — covered by itd-17 model effectiveness tracking if relevant.
- **Capability-aware backend selection.** Choosing among the host default and any configured oracle adapters by `(task_class, agent, model_id)` capability is itd-17's concern (capability-aware dispatch), layered ABOVE this intent's default host-delegation. This intent ships only the default path; it never selects backends.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a fresh laptop with no oracle adapter configured and `oracle.backend = "auto"` in `.abcd/config.json`, **when** any abcd command invokes the oracle (e.g. `/abcd:disembark`'s lifeboat-oracle Phase C audit), **then** the audit completes successfully via in-session subagent dispatch and writes a finding JSON to `.abcd/lifeboat/audit/oracle-<ts>.json` without ever attempting an external network call or shell-out to `claude -p`.
- **Given** `oracle.backend = "in-session"` is explicitly pinned, **when** abcd's host script returns its sentinel-output payload, **then** the orchestrating Claude session reads the payload, invokes the host's `Task` tool with the agent name and prompt declared in the payload, and returns the subagent's output back to abcd via the documented result-return path (file or stdout) for the next iteration to consume.
- **Given** an in-session-dispatched subagent emits a Carmack-style review, **when** abcd parses the result, **then** the verdict tag matches the same `<verdict>SHIP|NEEDS_WORK|MAJOR_RETHINK</verdict>` protocol used by every oracle adapter — the parser is backend-agnostic.
- **Given** an audit-fix loop runs over multiple iterations under `oracle.backend = "in-session"`, **when** the second and subsequent iterations dispatch, **then** each iteration is a fresh `Task` call with no inherited `chat_id` continuity AND abcd's loop re-primes the subagent context (artefact + prior verdict + applied fixes) every iteration — the no-state property is documented in `04-surfaces/02-disembark.md` and surfaced in CLI help text.
- **Given** `oracle.backend = "auto"`, **when** an oracle call runs and a persona has configured an opt-in oracle adapter, **then** that adapter handles the call; **and given** no adapter is configured, **then** the call routes to the in-session host-delegated default — host-delegation is the default, never an error for missing external setup.
- **Given** a different host harness (per itd-22's transport-/host-agnostic reach), **when** an in-session dispatch is requested, **then** the dispatch routes through that host's subagent invocation primitive AND the same wire-protocol contract holds — the per-host branch is invisible to the calling agent.

## Open Questions

- What's the canonical wire protocol shape? JSON-on-stdout with a `next_action: "task_dispatch"` field, or some other sentinel? Slash-command markdown needs to know what to look for.
- How does the orchestrating session know to read the wire-protocol payload? The slash-command markdown instructs Claude — but commands like `/abcd:disembark` invoke `scripts/abcd` which spawns a long-running process; how does the post-process payload get back to Claude's view?
- Does this work across hosts (per itd-22)? Each host has its own subagent invocation primitive; the dispatch impl needs a per-host branch.
- What's the right way to test this without spinning up a real subagent? Mock the host's Task tool? Stub the wire-protocol consumer side?
- Does itd-2 need to ship before itd-22 (host-agnostic reach), or can both be designed in parallel since they share the dispatch Protocol surface?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
