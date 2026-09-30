---
id: itd-47
slug: oracle-gates-autonomous-mode
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
superseded_by: adr-22
kind_at_supersession: standalone
related_adrs: []
severity: nitpick
---

> **⚠️ Superseded by [ADR-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)** (codex is dropped as a dependency and the Ralph autonomous loop is not ported; the RP→codex→in-session oracle cascade is replaced by host-delegated LLM and the pluggable autonomous seam — see also [ADR-25](../../decisions/adrs/0025-host-delegated-llm-default.md), [ADR-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)). Preserved as historical record per the supersession lifecycle.

# Oracle-Backed Gates Pass Honestly Without A Human In The Loop

## Press Release

> **abcd's `intent-fidelity-reviewer` agent runs its three oracle-backed quality gates (itd-5 self-improvement pre-flight, research-file review, live injection-canary execution) in an autonomous Ralph session without a human in the loop.** Today those gates are structurally un-completable in headless mode: `_build_cli_oracle()` returns `Oracle(MCPBridge())` — the RP MCP leg only, which requires a running RepoPrompt GUI. Ralph running overnight on a server has no GUI. So the three gates either stay `deferred` (honest but blocking spec completion) or get rubber-stamped with fabricated outcomes. Once this intent lands, `_build_cli_oracle()` returns an `Oracle` with both an RP leg and a Codex leg; the Codex leg is reachable headlessly (the `codex` CLI is on PATH), and the gates complete with real outcomes against real oracle round-trips.
>
> "The spc-12 (predecessor store) spec completion review used to stall every time it hit R6," said Ethan, autonomous-loop operator. "I'd come back to a Ralph run that had spent its budget thrashing on a gate it couldn't reach. With the Codex leg wired into the CLI oracle, the same gate runs in the next loop pass and either passes or fails honestly. The run completes — or doesn't — for real reasons."

## Why This Matters

`.work/issues.md` 2026-05-19 line 360 records the blocker: spc-12 (predecessor store; `intent-fidelity-reviewer` agent) ships Role 1 — per-criterion `MET`/`NOT_MET` verdicts on shipped intents — but three of its acceptance gates require a genuine `Oracle.ask()` round-trip that the autonomous run environment cannot satisfy. The three gates are:

- **R6 — itd-5 self-improvement pre-flight.** The candidate prompt is submitted to `lifeboat-oracle` for a clarity rewrite; the rewritten variant must pass the same goldens and be shorter by >10% to be accepted. Decision logged in the CHANGELOG.
- **T1 — research-file oracle review.** The reviewer agent's research artefact (`.abcd/development/research/prompting/agents/intent-fidelity-reviewer.md §7`) is reviewed against oracle judgement.
- **R7 — live injection-canary execution.** The reviewer agent's injection-canary fixture is executed end-to-end through the oracle to demonstrate the injection is ignored.

All three need `_build_cli_oracle()` to reach a real oracle backend. The current implementation returns `Oracle(MCPBridge())`, which is RP-MCP-only and requires an active RepoPrompt GUI — not available in headless mode. spc-13 (predecessor store) wired the Codex CLI as a *backend* (`oracle_codex.py`), but `_build_cli_oracle()` does not consume it. So the gates are reachable only when a human is sitting in front of an RP GUI; the autonomous loop reaches them via the run, can't complete them, and either defers (per the `deferred` permission of spc-12 (predecessor store), extended in `.work/issues.md` 2026-05-19 line 393) or stalls.

The deferral is *honest* — substituting a different reviewer changes the judgement substrate and fails itd-5 honestly — but it leaves spc-12 (predecessor store) in a state where every Ralph completion-review run on the spec produces the same deferral, no progress is made on the gates, and the spec is structurally un-shippable in the loop. The fix is mechanical: extend `_build_cli_oracle()` to wire the Codex leg through a `CodexAgentDispatch` (which spc-11 (predecessor store) / itd-6 work already provides primitives for).

This intent is **a precondition for several downstream specs**: any agent spec that depends on `intent-fidelity-reviewer`'s discipline-checking roles (Roles 2 and 3 — itd-31, itd-34) and any future `lifeboat-oracle` work (itd-5's named reviewer) hits the same gate. Fixing it here unblocks the chain.

## What's In Scope

- **Extend `_build_cli_oracle()` in `scripts/abcd/intent_fidelity_reviewer.py`**
  to construct an `Oracle` that carries both an `MCPBridge` (RP leg) and a
  `CodexAgentDispatch` (Codex leg). The selection logic follows the itd-6
  cascade contract: prefer RP if reachable, fall back to Codex, fall back to
  in-session subagent.
- **Confirm the `oracle_codex.py` integration of spc-13 (predecessor store)** is reachable from this
  call site (the Codex CLI is on PATH per the wiring of spc-13, predecessor store).
- **Re-run the three oracle-backed gates of spc-12 (predecessor store)** (R6, R7, T1) under the
  extended `_build_cli_oracle()` and rewrite the CHANGELOG / research §7 with
  real outcomes — replacing the current `deferred` markers.

## What's Out Of Scope

- **Building `lifeboat-oracle` agent.** itd-5's named reviewer agent does
  not exist. The R6 gate (clarity rewrite) is reviewer-specific —
  substituting a different reviewer changes the judgement substrate. This
  intent's R6 work has two honest paths: (a) ship `lifeboat-oracle` as part
  of this intent's scope, OR (b) extend the `deferred` permission to "until
  `lifeboat-oracle` exists" with a re-run hook for when it does. Decide at
  plan; lean (b) (separation of concerns).
- **itd-6 cascade epic in full.** itd-6 AC#4 (full three-step cascade with
  in-session fallback) is the broader work. This intent uses the RP+Codex
  legs already available; it does not ship the in-session subagent leg
  (depends on itd-2).
- **`_build_cli_oracle()` callers other than `intent_fidelity_reviewer.py`.**
  If other call sites construct the CLI oracle, they likely have the same
  problem, but the fix surface here is the specific call site of spc-12 (predecessor store). Wider
  audit deferred to itd-6 cascade epic.

## Acceptance Criteria

- *Given* the extended `_build_cli_oracle()`, *when* it is called in a Ralph environment with no RP GUI running but the `codex` CLI on PATH, *then* it returns an `Oracle` whose first reachable backend is the Codex leg; oracle calls succeed against real Codex round-trips.
- *Given* `intent_fidelity_reviewer` running the R6 gate (itd-5 self-improvement pre-flight) in a Ralph environment, *when* the gate executes, *then* a real oracle round-trip happens, a real outcome is logged in `agents/CHANGELOG.md`, and the entry is no longer `deferred` — unless `lifeboat-oracle` is the named reviewer and is missing, in which case the `deferred` permission is extended explicitly with that named cause (per `.work/issues.md` 2026-05-19 line 393).
- *Given* the R7 gate (live injection-canary execution) running in a Ralph environment, *when* the gate executes, *then* the canary fixture is submitted through the oracle, the output is inspected for injection success, and the gate emits a definite pass/fail — not `deferred`.
- *Given* the T1 gate (research-file oracle review) running in a Ralph environment, *when* the gate executes, *then* the research artefact is reviewed against oracle judgement and a real verdict lands in `research/prompting/agents/intent-fidelity-reviewer.md §7`.
- *Given* an RP GUI is also running, *when* the extended `_build_cli_oracle()` is called, *then* it prefers the RP leg (matching itd-6 cascade ordering); the Codex leg is the fallback, not the default.

## Implementing specs

itd-47 is implemented across multiple specs. The single-valued frontmatter
`spec_id` records the **primary** delivering spec (spc-27); the remaining spec is
recorded here because `spec_id` holds one value and would understate scope.
This section is the canonical multi-spec implementation index:

- **spc-27** (primary) — oracle CLI Codex leg for autonomous mode (the `_build_cli_oracle()` extension + Codex-leg gates).
- **spc-32** — Phase-3 closeout sweep (the remaining oracle-gate hardening delivered under the closeout).

## Open Questions

- **`lifeboat-oracle` scope.** Ship it as part of this intent (broader scope,
  unblocks R6 fully) or defer to a sibling intent and document the
  extended `deferred` permission for the R6 entry only? Lean: defer
  `lifeboat-oracle` to a sibling intent — it's a full agent prompt
  spec with its own discipline gates (itd-5 applies to *it*), not a
  one-line wiring change.
- **In-session subagent leg.** itd-6 AC#4 names a three-step cascade
  (RP → Codex → in-session). This intent ships the first two. The
  in-session leg depends on itd-2 (which has no spec yet). Document the
  partial as an explicit deferral, plumb the third leg later.
- **Test surface for the Codex leg.** spc-13 (predecessor store) tested `oracle_codex.py` in
  isolation; this intent needs at least one integration test that exercises
  the extended `_build_cli_oracle()` end-to-end against a real Codex CLI
  invocation in a Ralph-like environment. Where does that test live —
  under `tests/abcd/`, a new `tests/ralph/`, or as a script under
  `.work/`? Decide at plan.

## Related

- **spc-12** (predecessor store; `intent-fidelity-reviewer` agent) — the spec whose oracle gates
  this intent unblocks.
- **spc-13** (predecessor store; headless Codex CLI oracle wiring) — provides `oracle_codex.py`,
  the Codex leg this intent's `_build_cli_oracle()` consumes.
- **itd-6** (RP-MCP-only integration / oracle cascade) — the broader
  cascade work; this intent ships the first two legs.
- **itd-5** (prompt-quality additions discipline) — the discipline whose
  pre-flight gate (R6) is one of the three this intent restores.
- **itd-2** (in-session subagent dispatch) — the prerequisite for the
  cascade's third leg, deliberately left for a later spec.
- **`.work/issues.md` 2026-05-19 line 360** — the canonical blocker entry.
- **`.work/issues.md` 2026-05-19 line 393** — the extended `deferred`
  permission entry; this intent removes most of the deferral cases.
