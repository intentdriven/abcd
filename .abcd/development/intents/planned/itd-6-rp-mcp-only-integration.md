---
id: itd-6
slug: rp-mcp-only-integration
spec_id: spc-2609211950427074
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2]
severity: minor
impact: additive
---

> **⚠️ Framing superseded by [ADR-25](../../decisions/adrs/0025-host-delegated-llm-default.md)** (host-delegated LLM is the default; RepoPrompt is one optional oracle adapter among many, not abcd's single integration — see also [ADR-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). The intent itself stays live and is scheduled in Phase 0's `## Scope` as the oracle adapter seam: read "abcd's only RP integration is MCP" below as the contract of the *RP adapter*, not of abcd — the RP-specific mechanics (MCP bridge, cascade position, `chat_id` semantics) are adapted to the adapter seam at spec time.

# RepoPrompt over MCP is one opt-in reviewer route, beside the host's own agents

> **Re-filed on 2026-09-21** by the product thinker: not abcd's one integration but one opt-in reviewer adapter. abcd is host-delegated by default; with `oracle.review = rp` configured, a lane's reviews go to RepoPrompt over MCP, and when it is unreachable the host's own agent runs the review and the receipt says so. The three-step cascade, the setup discovery and the non-Mac flow this record first described are dropped; the press release and scope below are read through this paragraph and the Decisions section. itd-7 waits on this record.


## Press Release

> **abcd has exactly one integration with RepoPrompt: the MCP API.** abcd never picks an `oracle`, never reads RP's preset selection, never spawns its own subprocess for code review. It calls RP via MCP and RP uses whatever `oracle` the persona has configured for whatever task — Claude via the persona's subscription, Codex via the persona's subscription, Gemini, any preset RP knows. The persona configures `oracle` backends inside RP once; abcd uses them forever. Zero abcd-side `oracle` logic, zero "which preset?" prompts, zero hard-coded routing.
>
> **Status (post-spc-5): the RP MCP bridge/foundation is implemented.** spc-5 declares the typed `RPUnavailable` error (in `internal/core/...`) and delivers a concrete `MCPBridge` — the ADR-02 spawn implementation shipped, plus the ADR-03 host-reuse hook for host-connected Claude Code. What this intent describes as the *three-step cascade* (RP MCP → Codex CLI → in-session subagent), the one-time ahoy RP setup discovery, and the non-Mac flow are **follow-up work, deferred** beyond spc-5 — see the Implementation status section. This is "the bridge is built and typed", not "the RP MCP path is fully wired end-to-end".
>
> "I had wired up Claude, Codex, and Gemini in RP with task-specific presets," said Bob, staff engineer. "I'd worried abcd would keep asking me which to use. The RP MCP bridge just calls RP; when RP is not reachable it raises a typed `RPUnavailable` so the tooling can react cleanly instead of guessing. RP picks the `oracle`. I don't think about it."

## Why This Matters

abcd's oracle backend chain (brief § 6.7) treats RP as one of three transports (RP MCP / Codex CLI / in-session subagent). That framing implied abcd had model-selection logic to manage. **It doesn't.** RP is not a transport — it's an orchestrator that already abstracts over Claude, Codex, Gemini, and any model the user has configured.

Phase 0's plan-review surfaced a real friction point: getting the right RP model selected for a review felt like work. The fix is **not** "abcd auto-detects the active model" — it's "abcd's only RP integration is MCP, and RP handles everything else." The user configures models inside RP once (per their tastes, per their subscriptions), and abcd never touches that configuration.

This intent re-frames the brief's RP integration: drop "select RP backend with preset awareness" complexity; lock "MCP call to RP, full stop". The brief § 6.7 already implies this — itd-6 makes it explicit.

## What's In Scope

- **Single integration point**: abcd calls RP via `mcp__RepoPrompt__*` tools. Period. No subprocess spawning of `claude -p`, no direct LLM API calls, no model detection.
- **RP MCP server presence check**: ahoy verifies RP MCP is reachable; if not, surface "open RP and try again" once (per session)
- **Drop "active model detection"**: previous draft of this intent assumed abcd needed to read RP's active preset. It doesn't. abcd issues MCP calls; RP routes to whatever the user has configured for that call type.
- **Three-transport oracle chain stays**: RP MCP (preferred) → Codex CLI subprocess (alternative for non-Mac / no-RP users — RP is macOS-only) → in-session subagent (final fallback). The itd-6 architectural correction is specifically about the RP-side: abcd never reads RP's preset, never picks an RP model, never spawns `claude -p` to bypass RP. Codex is a separate transport the user explicitly chooses when they don't have RP.
- **Failure mode**: if RP MCP is unreachable, abcd falls through to Codex if configured, then in-session subagent. Three-step cascade.
- **One-time RP setup discovery**: ahoy detects the RP MCP server config (in `~/Library/Application Support/RepoPrompt/MCP/` or `.mcp.json`), notes it in `.abcd/config.json` → `oracle.rp.mcp_config_path`, and tests reachability. If reachable: lock `oracle.backend = "rp"`. If not: lock `oracle.backend = "codex"` (if Codex CLI present) or `"in-session"` (final fallback) and surface a one-time hint about how to enable RP later.

- **Same-chat re-review semantics** (codified abcd rule, narrowed by ADR-02 § 3): when abcd re-runs an oracle/review/audit after applying fixes (plan-review → fix → re-review; impl-review → fix → re-review; lifeboat-oracle → fix → re-audit), the re-call MUST stay in the **same RP chat** — never `--new-chat`, never fresh `rp builder`. RP chats accumulate context (original artefact + first review + fix summary); same-chat re-runs let the model do incremental "are these fixes correct?" checks instead of starting from scratch. The harness `mcp_call` for an RP audit MUST return `chat_id` in `McpResult`; the audit-fix loop in abcd's `oracle.py` MUST thread that ID back as the `chat_id` arg on the next call. **Narrowed (ADR-02 Criterion 3b):** "same chat" means within one `abcd-cli` command invocation's stdio session. Cross-invocation chat continuation requires fresh GUI approval and is out of scope for autonomous operation. Same rule applies whether the backend is RP, Codex, or in-session subagent (in-session uses `Task` with continuation prompts). **Verdict direction across iterations**: the verdict can change in EITHER direction across audit-fix iterations — a fix can resolve issues (NEEDS_WORK→SHIP) AND a fix can introduce regressions (SHIP→NEEDS_WORK). Both are valid signal; abcd's `re_audit` MUST NOT reject downgrades (mirroring spc-2 spec's anti-pattern list). **Lifecycle narrowing (added post-spc-5, per ADR-02 § 3):** "same chat" now narrows further — it means **same-MCP-session / same-`MCPBridge`-instance only**. In spawn mode the session is per `abcd-cli` invocation; in host-reuse mode (ADR-03) the session lives for the lifetime of the injected host harness. A `chat_id` is only meaningful within the `MCPBridge` instance that produced it — cross-bridge `chat_id` reuse is undefined behaviour, not a supported continuation path.

## What's Out of Scope

- Auto-launching RP if not running (out of scope; user opens RP themselves)
- Configuring presets/models inside RP (RP's UI handles this; abcd reads, never writes)
- Routing different oracle calls to different RP "modes" (e.g., "this audit needs Opus") — abcd issues an MCP call, RP picks; if the user wants a specific model for a specific call type, they configure RP accordingly
- Cost tracking per model (covered by itd-17 model effectiveness tracking — independent concern)
- Spawning Claude Code subprocesses with `claude -p` — explicitly NOT how abcd works; abcd is the consumer, not the spawner
- Direct OpenAI / Google / Anthropic API integrations — out of scope; route everything through RP
- **Capability-aware cascade routing** (added 2026-05-08 per idea-4 jagged-frontier review). The cascade defined by this intent (RP MCP → Codex CLI → in-session subagent) is and stays availability-driven (per `04-universal-patterns.md § 7` "fixed cascade"). Capability-aware routing — when Frontier Awareness ships — is a *pre-cascade selector* layer above the cascade, NOT a modification. The selector picks which backend the cascade *starts from* based on `(task_class, agent, model_id) → preferred_backend_ranking`; the fixed cascade per this intent and itd-2 begins from that backend without contract change.

## Scope Conditions

None stated.

## Mechanism

We expect a reviewer route a person has already configured and paid for to be used over one abcd would have to own, because the review is the run's scarcest step and the person's own tool is where their model choices already live; shown wrong if nobody opts in within a release of it shipping.

## Acceptance Criteria

- **Given** `oracle.review = rp` in the repository's or the machine's abcd configuration, **when** a `build` lane reaches its reviews, **then** the ruthless and security review requests are sent to RepoPrompt over MCP and each returned verdict is recorded by the loop as any validator's is.
- **Given** the adapter is configured and RepoPrompt is unreachable, **when** a review is due, **then** the host's own agent runs it and the receipt states that the review fell back and why; no review is silently skipped.
- **Given** the adapter is not configured, **when** a review runs, **then** nothing differs from today, and abcd never spawns, installs or configures RepoPrompt.
- **Given** the adapter ships, **when** the brief's adapters chapter and the command page are read, **then** the adapter is one entry beside the command-line runner, with the opt-in named.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **One opt-in reviewer route**, not the integration; the host stays the default.
2. **Reviews only**, not audits.
3. **Unreachable falls back to the host**, with the receipt saying so.
4. **itd-7 waits** on this record and is not in the run.

## Open Questions

_None open._

## Resolved (post-spc-5)

These questions were settled by the spc-5 spec and the Phase 0 harness-interface research note ([`01-harness-interface.md`](../../research/notes/01-harness-interface.md)), together with the spc-5 `.6` host-reuse / failure-mapping work.

- **Does RP MCP support the long-running, async-result pattern abcd needs (e.g., a 5-minute Carmack review)? Or is it strictly synchronous within an MCP call lifetime?**
  Resolved by ADR-02 § 4: the `MCPBridge` contract is synchronous within an MCP call lifetime — `mcp_call` blocks for the call's duration. There is no async-result handle. The long-running case is handled by a generous per-tool `call_timeout_s` budget (`oracle_send` / `context_builder` get 600 s) inside one held-warm stdio session, not by an async poll. spc-5's concrete `MCPBridge` implements exactly this.
- **If RP MCP returns a chat ID for long-running work, how does abcd poll/listen for completion?**
  Resolved by ADR-02 §§ 3–4: there is no polling. The call is synchronous; `mcp_call` returns when the tool call returns. The `chat_id` on `McpResult` is for *same-session re-review threading*, not completion polling. The async-vs-sync decision referenced for "Task 5's harness.py" is settled — the harness method stays synchronous (ADR-01 § 3 lock), and the concrete sync↔async bridge is internal to spc-5's `MCPBridge`.
- **Chat identity and continuation — what does a `chat_id` mean, and can a chat be resumed across `abcd-cli` invocations?**
  Resolved by ADR-02 § 3 and the spc-5 `.6` exception mapping: a `chat_id` is meaningful only within the `MCPBridge` instance / MCP session that produced it. Cross-invocation (and cross-bridge) chat continuation is **not supported** — RP's GUI approval gate forecloses it, and any RP-infrastructure failure surfaces as the typed `RPUnavailable` (`OSError` subclass) declared by spc-5. "Same chat" therefore means same-MCP-session only; the spc-5 `.6` failure-path mapping routes every unreachable-RP path through `RPUnavailable` so callers cascade cleanly rather than relying on a stale `chat_id`.

## Resolved Questions

- **Failure semantics: if an MCP call to RP times out, does abcd retry, fall through to in-session, or both?**
  Resolved by ADR-02 (spc-4-phase-0-p1-patch-viability-framing-mcp.4):
  On any RP-infrastructure failure (`RPUnavailable` — subprocess spawn fail, `startup_timeout_s`
  expiry, RP approval denial via `McpError: Connection closed`, `call_timeout_s` expiry, or
  mid-call transport failure), `oracle.py` routes to `dispatch_agent(agent_name="codex", ...)`.
  No silent retry within the RP transport; fall-through IS the retry (to the next cascade level).
  Timeout defaults: `startup_timeout_s = 10.0 s` (combined spawn + initialize); `call_timeout_s`
  default 30 s with per-tool overrides. See ADR-02 §§ 4–6.

## Implementation status

_Added post-spc-5. The spc-5 spec (`spc-5-rp-mcp-integration-declare`) implemented the RP
MCP bridge/foundation — the typed `RPUnavailable` error (`internal/core/...`,
spc-5 `.1`), the concrete `MCPBridge` ADR-02 spawn implementation (spc-5 `.2`/`.5`), and the
ADR-03 host-reuse code path (spc-5 `.6`/`.7`). Operational availability under host-connected
Claude Code is still subject to RP's GUI approval gate — an approval denial surfaces as the
typed `RPUnavailable` rather than a hang. spc-5 does **not** complete this intent end-to-end:
the following acceptance criteria are explicitly deferred to follow-up work._

- **AC#3 — Verdict-direction (both-directions accepted across audit-fix iterations): DEFERRED.**
  Requires the `oracle.py` audit-fix loop. spc-5 ships only the `MCPBridge` transport, not the
  `re_audit` caller. Follow-up target: the `oracle.py` cascade spec (downstream of spc-5).
- **AC#4 — Three-step cascade (RP MCP → Codex CLI → in-session subagent): DEFERRED.**
  spc-5 delivers the RP transport and the typed `RPUnavailable` signal that the cascade catches,
  but the cascade itself (Codex CLI fallthrough, in-session subagent fallback, run-log
  surfacing of the serving backend) is not in spc-5. Follow-up target: the `oracle.py` /
  itd-2 cascade spec.
- **AC#5 — Ahoy setup discovery (RP MCP config-path detection + `oracle.backend` lock): DEFERRED.**
  spc-5 does not touch `/abcd:ahoy`. The MCP config-resolution order is specified in ADR-02 § 1,
  but the ahoy-side discovery, `.abcd/config.json` write, and one-time hint are follow-up work.
  Follow-up target: the ahoy / setup-discovery spec.
- **AC#6 — Non-Mac flow (Codex-only users never prompted about RP): DEFERRED.**
  Depends on AC#4's cascade and AC#5's setup discovery being in place. Follow-up target:
  the same cascade + setup-discovery follow-up epics.

AC#1 (MCP-only call path, no `claude -p` spawn) and AC#2 (`chat_id` threading within one
invocation) are *foundationally* satisfied by spc-5's `MCPBridge` + ADR-02 § 3, but their
end-to-end gate runs only when the `oracle.py` callers above land — they are not claimed
shipped here.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## Grounds

- pursued: the run's reviews are its scarcest step, one at a time on a weekly budget that ran out mid-pilot, and a second route is the relief; we expect the person's own configured reviewer to carry some of them; shown wrong if nobody opts in within a release
