---
id: itd-6
slug: rp-mcp-only-integration
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2]
severity: minor
---

> **⚠️ Framing superseded by [ADR-25](../../decisions/adrs/0025-host-delegated-llm-default.md)** (host-delegated LLM is the default; RepoPrompt is one optional oracle adapter among many, not abcd's single integration — see also [ADR-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). The intent itself stays live and is scheduled in Phase 0's `## Scope` as the oracle adapter seam: read "abcd's only RP integration is MCP" below as the contract of the *RP adapter*, not of abcd — the RP-specific mechanics (MCP bridge, cascade position, `chat_id` semantics) are adapted to the adapter seam at spec time.

# RP-Only Integration: abcd Talks to RepoPrompt via MCP, Period

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

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a macOS user with RepoPrompt installed and `oracle.backend = "rp"` (or `"auto"` with RP available), **when** any abcd command invokes the oracle (lifeboat-oracle, press-release-composer, intent-fidelity-reviewer, plan-review, impl-review, prompt SOTA audit), **then** the call goes through `mcp__RepoPrompt__*` tools exclusively — no `claude -p` subprocess spawn, no direct OpenAI / Anthropic / Google API call, and no preset-selection prompt to the user.
- **Given** abcd issues an RP MCP audit call, **when** the call returns, **then** `McpResult.chat_id` is populated AND the audit-fix loop in `oracle.py` threads that `chat_id` back as the `chat_id` argument on the next call — never `--new-chat`, never a fresh `rp builder`. This holds within a single `abcd-cli` invocation across plan-review→fix→re-review, impl-review→fix→re-review, and lifeboat-oracle→fix→re-audit cycles. (Per ADR-02 § 3: "same chat" is scoped to within one `abcd-cli` command invocation; cross-invocation chat continuation requires fresh GUI approval and is not supported for autonomous operation.)
- **Given** an audit-fix iteration produces a verdict change in either direction (NEEDS_WORK→SHIP after fixes; SHIP→NEEDS_WORK after a regression), **when** abcd's `re_audit` runs, **then** both directions are accepted as valid signal — no rejection of downgrades, no special-casing of upgrades.
- **Given** RP MCP is unreachable (RP not running, MCP server config missing, network failure), **when** abcd attempts an oracle call with `oracle.backend = "auto"`, **then** the resolution chain falls through to Codex CLI (if `codex` is on PATH) and then to in-session subagent (per itd-2) — three-step cascade, surfaced in the run log so the user can see which backend served the call.
- **Given** the user runs `/abcd:ahoy` for the first time on a macOS machine with RP installed, **when** ahoy's setup discovery runs, **then** the RP MCP config path is detected (in `~/Library/Application Support/RepoPrompt/MCP/` or the project's `.mcp.json`), recorded in `.abcd/config.json` → `oracle.rp.mcp_config_path`, reachability is tested, and `oracle.backend` is locked to `"rp"` on success or to the next available backend on failure (with a one-time hint about how to enable RP later).
- **Given** the user has configured Claude, Codex, and Gemini as separate model presets inside RP, **when** abcd issues different oracle call types (review, audit, question), **then** abcd makes no preset-selection decision — RP routes to whatever the user has configured for that call type. abcd's logs record only the MCP call shape, never the resolved model.
- **Given** a non-Mac user with Codex CLI but no RP, **when** abcd's resolution chain runs, **then** Codex CLI is selected and the user is never prompted about RP setup; RP-related friction is invisible to non-Mac users.

## Open Questions

- What does the RP MCP API actually expose for "review this prompt" vs "ask this question" vs "audit this content"? Need to verify which `mcp__RepoPrompt__*` tools cover the abcd oracle use cases (lifeboat-oracle, press-release-composer, intent-fidelity-reviewer, prompt SOTA audit, plan-review, impl-review). _(Open: this is an RP-API-shape sub-question; spc-5 delivered the `MCPBridge` transport but not the per-oracle-call tool mapping.)_
- How does this interact with itd-22 (OpenCode portability)? OpenCode probably has its own equivalent integration pattern (its own MCP setup, or a different surface entirely). The harness's `mcp_call(server, tool, args)` shim should treat "RP" as one server name; OpenCode's equivalent picks up via its own server config.
- **Standardised review-chat naming — naming affordance sub-question (still Open).** When abcd opens an RP chat for an oracle/review call, the chat should carry a deterministic, identifiable label — e.g. `spc-6: Phase 1 reconciliation` (or `<itd-N>: <slug>` for intent-stage work) — not RP's auto-generated `untitled-chat-<hex>`. Field evidence (2026-05-16, spc-6 plan-review): the RepoPrompt MCP `oracle_send` tool exposes **no chat-name parameter** — it always auto-names — and the MCP toolset has no rename op. Naming only worked via `flowctl rp chat-send --chat-name`, a path broken against rp-cli 2.x. itd-6 must still settle: does abcd's RP integration name chats at creation (needs an MCP affordance RP may not currently provide — verify against the RP MCP API), name the *tab* instead (the `builder` path does title tabs from its summary), or rename post-creation? A consistent `<id>: <label>` convention makes review chats findable — RP windows accumulate dozens of review tabs. Cross-ref: the `feedback-rp-window-per-repo` agent-memory note has the full rp-cli 2.x / flow-next skew diagnosis. _(The chat-**identity / continuation** half of this question — what a `chat_id` means and whether it survives across invocations — is now resolved; see "Resolved (post-spc-5)" below. Only the cosmetic naming-affordance half stays Open.)_

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
