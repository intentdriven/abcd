---
id: adr-22
slug: bundled-deps-as-pluggable-adapters
status: accepted
date: 2026-07-06
supersedes: [adr-14, adr-15, adr-17, itd-47]
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-21, adr-23, adr-25, adr-26, adr-27, adr-29]
---

# ADR-22: Bundled dependencies become pluggable adapters over a native default

## Context

abcd bundled five external tools and treated them as load-bearing: specstory
(transcript capture), RepoPrompt and codex (the review/oracle backends),
flow-next (spec and task orchestration), and Ralph (the autonomous run loop).
Each hard dependency forced abcd to carry machinery whose only job was to make
someone else's tool safe to live inside abcd: the overlay manifest and
dispatcher that survived a `/flow-next:ralph-init` re-vendor, the argv-sentinel
session mirror, the `ahoy doctor` bypass probes, the abstraction boundary that
warned an operator they had reached past the wrapped surface. That subsystem
existed **only because** the tools were mandatory and re-vendored themselves on
top of abcd's own state.

The rebuild removes the premise. No external tool is a required dependency.

## Decision

Every previously-bundled tool becomes an **opt-in adapter over a native
default**. abcd works with none of them installed; each is a capability an
operator can plug in, never a floor abcd stands on:

- **specstory** → native local redacted transcript store
  ([ADR-29](0029-native-transcript-corpus.md)) is the default; specstory is an
  optional capture source.
- **RepoPrompt / codex** → host-delegated LLM
  ([ADR-25](0025-host-delegated-llm-default.md)) is the default oracle;
  native/CLI/API/MCP backends are opt-in adapters.
- **flow-next** → native minimal spec/task store
  ([ADR-26](0026-native-spec-layer-ccpm-backend.md)) is the default; the companion
  harness `ccpm` is the primary deeper backend. flow-next is **not** built.
- **Ralph** → the autonomous run is a pluggable seam
  ([ADR-27](0027-autonomous-run-pluggable-seam.md)), not a Ralph port.

Because no tool re-vendors itself onto abcd's state, the entire
overlay/dispatcher/session-mirror/boundary subsystem **ceases to exist**. This
ADR therefore **supersedes** the three decisions that built and tuned it — and
itd-47, the intent that rested on the same bundled cascade:

- ADR-14 — the
  degraded-fallback guard over the required overlay manifest.
- ADR-15 — the warn-not-block
  abstraction boundary, the argv sentinel, and the artifact-only doctor probes.
- ADR-17 — the
  `rp chat-send` override machinery layered onto the bundled RP path.

## Alternatives Considered

- **Keep the tools as hard deps, keep the overlay.** Preserves the shipped
  integration work. Rejected: the overlay is pure tax — its every component
  exists to contain a mandatory foreign tool, and it re-accrues on every
  upstream release. Mandatory bundling is the thing the rebuild is removing.
- **Drop the tools entirely, no adapters.** Simplest. Rejected: RP/codex/ccpm
  are genuinely valuable when present; discarding the ability to plug them in
  throws away capability the adapter seam keeps cheap.
- **Chosen: native default + opt-in adapter per tool.** abcd stands on its own
  code; a present tool is an upgrade, never a prerequisite.

## Consequences

- The overlay manifest, `scripts/ralph/flowctl` dispatcher, `ralph_mirror.py`
  session shim, the `--abcd-driven` sentinel, and the doctor bypass probes are
  all removed. `/abcd:ralph-up`, `--fresh`, and the re-vendor mandate go with
  them — there is nothing to re-apply because nothing is re-vendored.
- The abstraction boundary as a *reach-past warner* is retired: with native
  defaults there is no wrapped foreign surface to reach past. abcd's guarantees
  live in its own core, not in a warning that an operator left it.
- Each adapter carries its own thin capability contract (interface + native
  fallback), so a missing or misbehaving tool degrades to the native path
  rather than breaking abcd.
- The doctor's job narrows from policing foreign bypass artifacts to checking
  abcd's own health and which adapters are wired.
- **The first declared exception (proposed 2026-10-16, signed off by the
  product thinker on 2026-10-22, dcn-2610222041417934).** The fence around the agents abcd's runner launches
  ([adr-2610161621415708](2610161621415708-abcd-s-command-guard-steers-agents-an-os.md),
  spc-2610161751056096) is Anthropic's sandbox-runtime (`srt`), an external
  program abcd does not ship. On the runner route alone it is a requirement,
  not an adapter that degrades: a fence has no native default to degrade to,
  since abcd writes no fence of its own and an unfenced launch is what the
  fence exists to remove. A machine without it refuses a runner launch, and
  the person's per-machine off switch (`runner.sandbox.off` in
  `~/.abcd.noindex/config.json`) takes the place of degradation. Every other
  route still needs no external tool. Where `srt` is missing, abcd offers to
  install it and installs only after the person's explicit yes
  (dcn-2610222041424027).
