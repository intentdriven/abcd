---
id: itd-6
slug: rp-mcp-only-integration
spec_id: spc-2609211950427074
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609170822093401, itd-2609201925079472, itd-2609201916056194]
severity: minor
impact: additive
---

> **⚠️ Framing superseded by [ADR-25](../../decisions/adrs/0025-host-delegated-llm-default.md)** (host-delegated LLM is the default; RepoPrompt is one optional oracle adapter among many, not abcd's single integration — see also [ADR-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). The intent itself stays live and is scheduled in Phase 0's `## Scope` as the oracle adapter seam: read "abcd's only RP integration is MCP" below as the contract of the *RP adapter*, not of abcd — the RP-specific mechanics (MCP bridge, cascade position, `chat_id` semantics) are adapted to the adapter seam at spec time.

# RepoPrompt over MCP is one opt-in reviewer route, beside the host's own agents

> **Re-filed on 2026-09-21** by the product thinker: not abcd's one integration but one opt-in reviewer adapter. abcd is host-delegated by default; with `oracle.review = rp` configured, a lane's reviews go to RepoPrompt over MCP, and when it is unreachable the host's own agent runs the review and the receipt says so. The three-step cascade, the setup discovery and the non-Mac flow this record first described are dropped; the press release and scope below are read through this paragraph and the Decisions section. itd-7 waits on this record.


## Press Release

> **abcd has exactly one integration with RepoPrompt: the MCP API.** abcd never picks an `oracle`, never reads RP's preset selection, never spawns its own subprocess for code review. It calls RP via MCP and RP uses whatever `oracle` the persona has configured for whatever task — Claude via the persona's subscription, Codex via the persona's subscription, Gemini, any preset RP knows. The persona configures `oracle` backends inside RP once; abcd uses them forever. Zero abcd-side `oracle` logic, zero "which preset?" prompts, zero hard-coded routing.
>
> **Status: built as the re-filed route, on 2026-10-12.** The re-filed scope, [spc-2609211950427074](../../specs/closed/spc-2609211950427074-rp-mcp-only-integration.md), built it from nothing: `oracle.review = rp` opts a lane's reviews in, and `internal/adapter/repoprompt` speaks MCP, with the standard library alone, to the server the person names. The `RPUnavailable` error, the `MCPBridge` and the `oracle.py` audit-fix loop this record names belong to an earlier Python lineage: its spec `spc-5-rp-mcp-integration-declare` (predecessor store; not the `spc-5` in this repository's spec store) and its ADR-02 and ADR-03 (not this repository's adr-2 and adr-3). None of them is in this binary, and `go.mod` carries no MCP dependency. See the Implementation status section.
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

- **Same-chat re-review semantics** (codified abcd rule, narrowed by ADR-02 § 3): when abcd re-runs an oracle/review/audit after applying fixes (plan-review → fix → re-review; impl-review → fix → re-review; lifeboat-oracle → fix → re-audit), the re-call MUST stay in the **same RP chat** — never `--new-chat`, never fresh `rp builder`. RP chats accumulate context (original artefact + first review + fix summary); same-chat re-runs let the model do incremental "are these fixes correct?" checks instead of starting from scratch. The harness `mcp_call` for an RP audit MUST return `chat_id` in `McpResult`; the audit-fix loop in abcd's `oracle.py` MUST thread that ID back as the `chat_id` arg on the next call. **Narrowed (ADR-02 Criterion 3b):** "same chat" means within one `abcd-cli` command invocation's stdio session. Cross-invocation chat continuation requires fresh GUI approval and is out of scope for autonomous operation. Same rule applies whether the backend is RP, Codex, or in-session subagent (in-session uses `Task` with continuation prompts). **Verdict direction across iterations**: the verdict can change in EITHER direction across audit-fix iterations — a fix can resolve issues (NEEDS_WORK→SHIP) AND a fix can introduce regressions (SHIP→NEEDS_WORK). Both are valid signal; abcd's `re_audit` MUST NOT reject downgrades (mirroring the anti-pattern list of spc-2, predecessor store). **Lifecycle narrowing (added after spc-5 (predecessor store), per ADR-02 § 3):** "same chat" now narrows further — it means **same-MCP-session / same-`MCPBridge`-instance only**. In spawn mode the session is per `abcd-cli` invocation; in host-reuse mode (ADR-03) the session lives for the lifetime of the injected host harness. A `chat_id` is only meaningful within the `MCPBridge` instance that produced it — cross-bridge `chat_id` reuse is undefined behaviour, not a supported continuation path.

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

These questions were settled against the earlier Python lineage's design — its spc-5 spec (predecessor store) and its ADR-02 and ADR-03, not this repository's spc-5, adr-2 and adr-3 — and the Phase 0 harness-interface research note ([`01-harness-interface.md`](../../research/notes/01-harness-interface.md)). The answers stand as design input for the re-filed adapter; the `MCPBridge`, `McpResult`, `RPUnavailable` and `oracle.py` they name are that lineage's, and none of them is in this binary.

- **Does RP MCP support the long-running, async-result pattern abcd needs (e.g., a 5-minute Carmack review)? Or is it strictly synchronous within an MCP call lifetime?**
  Resolved by ADR-02 § 4: the `MCPBridge` contract is synchronous within an MCP call lifetime — `mcp_call` blocks for the call's duration. There is no async-result handle. The long-running case is handled by a generous per-tool `call_timeout_s` budget (`oracle_send` / `context_builder` get 600 s) inside one held-warm stdio session, not by an async poll.
- **If RP MCP returns a chat ID for long-running work, how does abcd poll/listen for completion?**
  Resolved by ADR-02 §§ 3–4: there is no polling. The call is synchronous; `mcp_call` returns when the tool call returns. The `chat_id` on `McpResult` is for *same-session re-review threading*, not completion polling. The async-vs-sync decision referenced for "Task 5's harness.py" is settled — the harness method stays synchronous (ADR-01 § 3 lock), and the concrete sync↔async bridge is internal to the `MCPBridge` of spc-5 (predecessor store).
- **Chat identity and continuation — what does a `chat_id` mean, and can a chat be resumed across `abcd-cli` invocations?**
  Resolved by ADR-02 § 3 and the spc-5 (predecessor store) `.6` exception mapping: a `chat_id` is meaningful only within the `MCPBridge` instance / MCP session that produced it. Cross-invocation (and cross-bridge) chat continuation is **not supported** — RP's GUI approval gate forecloses it, and any RP-infrastructure failure surfaces as the typed `RPUnavailable` (`OSError` subclass) declared by spc-5 (predecessor store). "Same chat" therefore means same-MCP-session only; the spc-5 (predecessor store) `.6` failure-path mapping routes every unreachable-RP path through `RPUnavailable` so callers cascade cleanly rather than relying on a stale `chat_id`.

## Resolved Questions

- **Failure semantics: if an MCP call to RP times out, does abcd retry, fall through to in-session, or both?**
  Resolved by ADR-02 (spc-4-phase-0-p1-patch-viability-framing-mcp.4, predecessor store):
  On any RP-infrastructure failure (`RPUnavailable` — subprocess spawn fail, `startup_timeout_s`
  expiry, RP approval denial via `McpError: Connection closed`, `call_timeout_s` expiry, or
  mid-call transport failure), `oracle.py` routes to `dispatch_agent(agent_name="codex", ...)`.
  No silent retry within the RP transport; fall-through IS the retry (to the next cascade level).
  Timeout defaults: `startup_timeout_s = 10.0 s` (combined spawn + initialize); `call_timeout_s`
  default 30 s with per-tool overrides. See ADR-02 §§ 4–6. The re-filed Decisions (3) replace the
  Codex step: an unreachable RepoPrompt falls back to the host's own agent, with the receipt
  saying so, and this repository has no `oracle.py`.

## Implementation status

_Prerequisites, recorded 2026-09-29 (iss-2609240227236354)._ The readiness gate
reads this record READY, but its spec (spc-2609211950427074) stands on four
things the tree does not hold yet, and `builds_on` names the four intents
behind them:

- **The validator stage the adapter implements.** `abcd build` ships its loop's
  first pieces, and the validators are a later piece of the same spec
  (itd-2609201916151817, spec spc-2609202134338445 open; see
  `internal/core/implement/loop/loop.go`).
- **The `oracle.review` route.** The layered configuration reader
  (`internal/core/layered`) exists; the review-route intents it serves,
  itd-2609170822093401 and itd-2609201925079472, are still planned with specs
  open.
- **The command-line runner** the adapters chapter entry sits beside
  (itd-2609201916056194, spec spc-2609221533057881 open).
- **An MCP client.** The spec's Approach names one as a new dependency, subject
  to the new-dependency sign-off; `go.mod` carries none.

The readiness gate does not read `builds_on`, so these edges inform a reader
and a run that picks this record; they do not change the gate's verdict.

_Built on 2026-10-12_ by
[spc-2609211950427074](../../specs/closed/spc-2609211950427074-rp-mcp-only-integration.md),
from nothing: the validate stage and the command-line runner had landed, the
`oracle.review` key is read through `internal/core/layered` by the delivery's own
reader, and the MCP client is the standard library's, so `go.mod` still carries no
MCP dependency. Before it, a grep for `MCPBridge`, `RPUnavailable` and `RepoPrompt`
over `internal/` and `cmd/` found only the scanner's RepoPrompt session-key pattern
(`internal/adapter/scanner/patterns.go`), a guard corpus line, and the doc comment of the
configuration layer (`internal/core/layered`) naming `oracle.review` as a consumer it serves.
The bridge, the typed error
and the host-reuse path an earlier Python lineage's `spc-5-rp-mcp-integration-declare` (predecessor store) describes
belong to that lineage, not to this binary, so no part of the route was a foundation to build on.
The four acceptance criteria above are the re-filed set, and
[spc-2609211950427074](../../specs/closed/spc-2609211950427074-rp-mcp-only-integration.md)
carries all of them.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-d270e2e0acfb -->
Fidelity review — receipt rcp-d270e2e0acfb (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:ac9edc6f4a31d514575d1b4b0d066d26901309910e8b635ba651cdbbca57f36f
Input attestations: diff:d1255b8c59858e7a795005631f89cb28c1b659b6..141a1b9be0e3887c6d6533eda0d9c328f2497c52, the change PR #976 (feat/repoprompt-reviewer-cloud) brought onto main on 2026-10-15: 35 commits, 67b6e22e0 to 331d18bc0, including the spec close 94128c891; located from git history, since the request names no range@-; repository:2e03c61e454bf09ecda8cee0ae2f377157b522f7 (origin/main when reviewed); the route's code is unchanged since 141a1b9be apart from two assertions in validate_review_test.go@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 1

Audit owed (receipt rcp-d270e2e0acfb): ac-1 INCONCLUSIVE. Remedy: re-run the audit. Carried by iss-2610241426406882.

Per-criterion verdicts:
- ac-1 — INCONCLUSIVE: abcd's side is built: oracle.review is read from either file, implement step hands a ruthless or security review given to the host to DriveReview, the adapter sends it as one MCP Streamable HTTP session, and the return is checked by the same parseVerdict and recorded through the same receipt as a host's. Whether the requests reach RepoPrompt cannot be settled from the inputs. Every test answers from mcptest, a server the test owns, and the brief says no test reaches a real RepoPrompt. The adapter assumes RepoPrompt serves Streamable HTTP at a loopback URL and offers oracle_send taking message, and nothing in the delivery checks either. Ruling H1b (dcn-2609290000000007) says itd-6 is finished only after one run against the real RepoPrompt app, and no such run is recorded. The route is also narrower than the Given: oracle.review = rp alone sends nothing, because the route needs oracle.rp.url in the machine's file (the repository's file is refused) and otherwise falls back as absent.
  evidence: internal/surface/cli/build.go:1045 — "review, lerr := loop.LoadReviewRoute(roots, agentPrompt)"
  evidence: internal/surface/cli/build.go:1059 — "res, err = loop.DriveReview(ctx, root, id, res, loop.DefaultStages(), loop.Options{}, review)"
  evidence: internal/adapter/repoprompt/repoprompt.go:244 — "args := map[string]any{"message": request}"
  evidence: internal/core/implement/loop/validate.go:946 — "route := runner.RouteRecord{Asked: rr.Name, Ran: rr.Name}"
  evidence: internal/core/implement/loop/validate.go:569 — "word, err := parseVerdict(string(raw), reviewerVerdicts[v.Role])"
  evidence: internal/core/implement/loop/validate_review_test.go:80 — "func TestTheReviewsGoToRepoPromptOverMCPAndTheLoopRecordsTheirVerdicts(t *testing.T) {"
  evidence: internal/adapter/repoprompt/mcptest/mcptest.go:3 — "no test reaches a real RepoPrompt or the network"
  evidence: .abcd/development/brief/05-internals/02-adapters.md:247-249 — "RepoPrompt's MCP server must therefore answer over HTTP at that address; abcd starts no bridge to a server that speaks only stdio."
  evidence: .abcd/work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md:6 — "itd-6 is finished only after running once against the real RepoPrompt app, a check owed to a person"
  evidence: internal/core/oracle/review.go:108 — "if f.Layer != layered.Machine {"
  evidence: internal/core/implement/loop/validate.go:1331 — "return "", &reviewFailure{runner.ReasonAbsent, rr.Absent}"
- ac-2 — MET: With the route configured and nothing answering (connection refused, a bound run out, no server named, a refused or unparsable answer, a return with no verdict, a step interrupted), DriveReview hands every review with no usable return to fallBack. fallBack records a FallbackReceipt (role, asked rp, reason, abcd's detail, ran host) in the run's state and record, retried while the lock is busy, and the step's next move names it. The host is handed the same brief and return path, and its return is then recorded as any validator's. Loop and front-door tests hold each reason, and status and record count the fallbacks.
  evidence: internal/core/implement/loop/validate.go:923 — "return fallBack(repoRoot, runID, res, aw, rr, err, o)"
  evidence: internal/core/implement/loop/validate.go:1003 — "fb := runner.FallbackReceipt{At: o.now(), Role: aw.Role, Asked: rr.Name, Reason: runner.ReasonFailed, Detail: fallbackDetail(cause), Ran: runner.Host}"
  evidence: internal/core/implement/loop/drive.go:218-220 — "the %s was routed to %s, which was %s (%s); %s runs it"
  evidence: internal/core/implement/loop/validate_review_test.go:198 — "func TestAReviewTheRouteDoesNotReturnFallsBackToTheHostWithAReceipt(t *testing.T) {"
  evidence: internal/core/implement/loop/validate_review_test.go:296 — "func TestAnInterruptedStepFallsBackAsInterrupted(t *testing.T) {"
  evidence: internal/surface/cli/build_review_surface_test.go:115 — "func TestAReviewTheOracleDoesNotReturnFallsBackAndIsCountedAtTheFrontDoor(t *testing.T) {"
- ac-3 — MET: With oracle.review unset or host, LoadReviewRoute returns nil and DriveReview passes the step's result through untouched, and oracle.rp is not read at all off the rp route. A test holds the step's result and the state byte-identical to the plain host-driven step, with no request rendered. The adapter's files import no process, file, environment or system-call API. No other abcd code references RepoPrompt or writes oracle.review or oracle.rp; the scanner's pre-existing sessionKey pattern only redacts.
  evidence: internal/core/implement/loop/validate.go:867 — "if rv.Route != oracle.ReviewRP {"
  evidence: internal/core/implement/loop/validate.go:902 — "if rr == nil || res.Awaiting == nil || !res.handed || res.Route != nil || res.Fallback != nil {"
  evidence: internal/core/oracle/review.go:86 — "if rv.Route != ReviewRP {"
  evidence: internal/core/implement/loop/validate_review_test.go:320 — "func TestAnUnconfiguredRouteLeavesTheReviewAndTheStateByteIdentical(t *testing.T) {"
  evidence: internal/core/oracle/review_test.go:37 — "func TestTheReviewRouteIsTheHostUnlessConfigured(t *testing.T) {"
  evidence: internal/adapter/repoprompt/nothing_test.go:17 — "func TestTheAdapterStartsNothingAndReadsNothing(t *testing.T) {"
  evidence: internal/adapter/repoprompt/repoprompt.go:9 — "The adapter connects and nothing else. It never starts RepoPrompt or any"
- ac-4 — MET: The adapters chapter puts both reviewer routes under one heading, the command-line runner and RepoPrompt over MCP as adjacent bullets, each naming its opt-in (roles.< role>.runner; oracle.review = rp with oracle.rp.url in the machine's file). The build command page puts the RepoPrompt paragraph right after the command-line runner's, naming the same opt-in, and the implement page does the same.
  evidence: .abcd/development/brief/05-internals/02-adapters.md:189 — "### Reviewer routes — the command-line runner, and RepoPrompt over MCP"
  evidence: .abcd/development/brief/05-internals/02-adapters.md:197 — "- **The command-line runner** (`internal/core/runner`, itd-2609201916056194)"
  evidence: .abcd/development/brief/05-internals/02-adapters.md:226-228 — "**RepoPrompt over MCP** (`internal/adapter/repoprompt`, itd-6) sends the review to the RepoPrompt the person runs, which answers with whichever model the person configured in it. The opt-in is `oracle.review = rp` in the"
  evidence: commands/build.md:401 — "A role can run through a command-line runner instead of an agent you start."
  evidence: commands/build.md:443-444 — "A build lane's reviews can go to RepoPrompt over MCP instead of a reviewer you start. This is an opt-in: `oracle.review` set to `rp` in the repository's"
  evidence: commands/implement.md:287 — "With `oracle.review` set to `rp` (the repository's `.abcd/config.json` or"

Gap audit:
- honoured:
  - Host-delegated by default, and oracle.review = rp in the repository's or the machine's file opts a lane's reviews in, the repository's file winning (decision 1).
    evidence: internal/core/oracle/review.go:76 — "v, err := layered.Get(s, ReviewKey, ReviewHost, func(name string) error {"
    evidence: internal/core/oracle/review_test.go:52 — "func TestOracleReviewIsReadFromTheRepositoryOrTheMachine(t *testing.T) {"
  - abcd picks no oracle and no model and reads no RepoPrompt preset: the request carries only the tool's message (and new_chat when the tool declares it), and RepoPrompt answers with the model the person configured.
    evidence: internal/adapter/repoprompt/repoprompt.go:6 — "RepoPrompt answers with whichever model the person configured in it"
    evidence: internal/adapter/repoprompt/repoprompt.go:244 — "args := map[string]any{"message": request}"
  - abcd spawns nothing for a review and never starts, installs or configures RepoPrompt: the adapter only connects to the address it is handed, and a test holds its imports to no process, file or environment API.
    evidence: internal/adapter/repoprompt/nothing_test.go:17 — "func TestTheAdapterStartsNothingAndReadsNothing(t *testing.T) {"
    evidence: internal/adapter/repoprompt/repoprompt.go:9 — "The adapter connects and nothing else. It never starts RepoPrompt or any"
  - An unreachable RepoPrompt falls back to the host's own agent, and a receipt says it fell back and why, counted per route and per role (decision 3).
    evidence: internal/core/implement/loop/validate.go:1003 — "Asked: rr.Name, Reason: runner.ReasonFailed, Detail: fallbackDetail(cause), Ran: runner.Host}"
    evidence: internal/core/implement/loop/drive.go:218 — "st.Fallbacks = append(st.Fallbacks, fb)"
    evidence: internal/surface/cli/build_review_surface_test.go:115 — "func TestAReviewTheOracleDoesNotReturnFallsBackAndIsCountedAtTheFrontDoor(t *testing.T) {"
  - Reviews only, not audits: only the ruthless and security reviewers take the route, and the intent-auditor stays on the host (decision 2).
    evidence: internal/core/implement/loop/validate.go:906 — "words, ok := reviewerVerdicts[aw.Role]"
    evidence: internal/core/implement/loop/validate_review_test.go:1027 — "func TestAnAuditorsVerdictIsNeverTheRoutes(t *testing.T) {"
  - MCP is spoken with the standard library alone, so go.mod carries no MCP dependency; the delivered range leaves go.mod and go.sum unchanged (Status paragraph).
    evidence: internal/adapter/repoprompt/repoprompt.go:2-3 — "a client of the Model Context Protocol's Streamable HTTP transport, spoken with the standard library alone"
    evidence: go.mod:5-13 — "require ("
  - The brief's adapters chapter and the command page show the adapter as one entry beside the command-line runner, with the opt-in named (spec scope 5).
    evidence: .abcd/development/brief/05-internals/02-adapters.md:189 — "### Reviewer routes — the command-line runner, and RepoPrompt over MCP"
    evidence: commands/build.md:443-444 — "A build lane's reviews can go to RepoPrompt over MCP instead of a reviewer you start. This is an opt-in"
- diverged:
  - The Given 'oracle.review = rp in the repository's or the machine's configuration' sends the reviews to RepoPrompt (re-filed paragraph, ac-1, CHANGELOG.md:109). In the delivery the route also needs oracle.rp.url, which only ~/.abcd.noindex/config.json may declare (a repository declaring oracle.rp is refused). With oracle.review = rp alone, every review falls back as absent. The spec records this as an agent's choice for review, and open issue iss-2610151422150430 notes the changelog line omits the URL.
    evidence: CHANGELOG.md:109 — "With `oracle.review = rp` in the repository's or the machine's abcd configuration, a `build` lane sends its ruthless and security reviews to RepoPrompt over MCP"
    evidence: internal/core/oracle/review.go:108 — "if f.Layer != layered.Machine {"
    evidence: internal/core/implement/loop/validate.go:1331 — "return "", &reviewFailure{runner.ReasonAbsent, rr.Absent}"
    evidence: .abcd/development/specs/closed/spc-2609211950427074-rp-mcp-only-integration.md:70 — "2026-10-12 (agent's choice, for review): **The server's address is the"
  - 'It calls RP via MCP' (first scoped as the mcp__RepoPrompt__* tools) is delivered as MCP's Streamable HTTP transport only, to a loopback URL the person names. A RepoPrompt MCP server that speaks only stdio is out of reach, since abcd starts no bridge. The tool shape (oracle_send taking message, optional new_chat) is assumed, not checked against RepoPrompt.
    evidence: .abcd/development/brief/05-internals/02-adapters.md:247-249 — "RepoPrompt's MCP server must therefore answer over HTTP at that address; abcd starts no bridge to a server that speaks only stdio."
    evidence: internal/adapter/repoprompt/repoprompt.go:55 — "const DefaultTool = "oracle_send""
    evidence: internal/adapter/repoprompt/repoprompt.go:329 — "the %s tool takes no message argument, so it cannot take a review request"
  - The typed RPUnavailable of the press release's quote is delivered as *repoprompt.Error with a Kind (unreachable, timeout, refused, unparsable, failed) that becomes the fallback's reason. No RPUnavailable type exists, as the Status paragraph says.
    evidence: internal/adapter/repoprompt/repoprompt.go:125 — "KindUnreachable Kind = "unreachable""
    evidence: internal/adapter/repoprompt/repoprompt.go:134 — "type Error struct {"
  - The in-scope same-chat re-review semantics (a re-review MUST stay in the same RP chat, with chat_id threaded back) are delivered as a new chat for every review (new_chat: true when the tool declares it), and no chat_id is read. The re-filing's list of drops does not name this rule; the spec records the drop as an agent's choice for review.
    evidence: internal/adapter/repoprompt/repoprompt.go:246 — "args["new_chat"] = true"
    evidence: .abcd/development/specs/closed/spc-2609211950427074-rp-mcp-only-integration.md:99-100 — "the intent's same-chat re-review rule belongs to the predecessor lineage the re-filing set aside"
  - The original scope's three-step cascade (RP MCP, Codex CLI, in-session) and ahoy's one-time setup discovery were dropped by the 2026-09-21 re-filing. The delivery is one route that falls straight back to the host. ahoy's presence check ('open RP and try again' once per session), which the re-filing does not name, is not built: reachability is found per review and recorded as the fallback's reason.
    evidence: internal/core/implement/loop/validate.go:1003 — "Ran: runner.Host}"
    evidence: internal/adapter/repoprompt/repoprompt.go:11 — "for a server: the one address it reaches is the URL its caller hands it"
- missing:
  - The live run against the real RepoPrompt app. Ruling H1b says itd-6 is finished only after one such run, a check owed to a person. The delivery moved the intent to shipped/ with every test on a test-owned MCP server and no record of the run, and no ledger issue tracks it as owed; the command-line runner's equivalent is tracked as iss-2609301519558538.
    evidence: .abcd/work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md:6 — "itd-6 is finished only after running once against the real RepoPrompt app, a check owed to a person"
    evidence: .abcd/development/brief/05-internals/02-adapters.md:276-277 — "No test reaches a real RepoPrompt: every one runs against an MCP server the test owns on the loopback address."
    evidence: internal/adapter/repoprompt/mcptest/mcptest.go:3 — "no test reaches a real RepoPrompt or the network"
<!-- abcd-review-end receipt=rcp-d270e2e0acfb -->

### Linkage note (2026-09-30)

`builds_on` named itd-2 (in-session subagent dispatch), which is superseded by
itd-2609201916056194. The edge is dropped: its live successor is already in this
record's `builds_on`, so the relink would list it twice.

## Grounds

- pursued: the run's reviews are its scarcest step, one at a time on a weekly budget that ran out mid-pilot, and a second route is the relief; we expect the person's own configured reviewer to carry some of them; shown wrong if nobody opts in within a release
