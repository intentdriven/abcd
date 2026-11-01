---
id: itd-22
slug: harness-portability
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
---

# abcd Reaches Any Harness Through an Adaptor

> **Widened on 2026-09-22** by the product thinker: "any harness" includes **no harness**. abcd run as the binary alone is a first-class way to manage a repository: it calls whichever harnesses are available for the roles that need one, and where a role's runner is unavailable the fallback is a host the operator configured rather than a host session that happens to be there (itd-2609201916056194). The adaptor ladder below gains that rung; the console that would drive it from outside a harness is reframed as an operator console served by the binary (research note, 2026-09-22) and is not this record.


## Press Release

> **abcd runs wherever the user's harness runs.** abcd's core is a transport-agnostic engine, and each harness reaches it through a thin adaptor built on one shared seam. An adaptor climbs a fixed ladder — the host's own plugin format first, any other native seam second, the MCP floor always — and one parity suite proves every host gets the same conventions, intents, and lifeboats. Adding a harness is one adaptor over an unchanged core, never a second copy of abcd.
>
> "We switched harnesses and abcd came with us," said Carol, whose team moved for cost reasons. "Same record, same gates, byte-identical lifeboats. The adaptor was thin because the machinery was already shared."

## Why This Matters

adr-23's portability promise — front doors over an unchanged core — needs shared machinery to be true in practice. Today the hook entrypoints hard-code one host's stdin payload schema, exit-code semantics, and tool naming; without a host profile seam, every port re-solves those, and without one parity suite, "supported on host X" is an unverifiable claim. adr-39 (host-tier policy) sets the tiers — MCP floor, reference host, an eventual open-source default gated on parity — and this intent builds the machinery all of those tiers stand on.

## What's In Scope

- **The host profile seam** — the host-specific payload schema, exit-code semantics, and tool-name assumptions currently hard-coded in the CLI hook entrypoints move behind a host profile; the current host becomes profile #1 with behaviour unchanged.
- **The adaptor ladder as testable behaviour** — each rung declares its capabilities; a rung a host cannot support degrades to the next and reports what it cannot do (loud staging), never a silent no-op.
- **The parity conformance suite, defined once and run per host** — install parity, byte-identical lifeboats (modulo timestamps and oracle adapter ids), in-session subagent dispatch per itd-2's wire-protocol contract, and docs that treat every adopted host as a peer. Its per-check report is the adr-39 default-flip evidence.
- **Docs structure** — a host-neutral install/configuration skeleton that per-host pages slot into without editing existing pages.

## What's Out of Scope

- The MCP front door itself — its own intent; it is the floor the ladder ends on.
- Per-host adoption — one intent per harness at its explicit adoption decision (adr-39), including any concept mapping onto a host's native work-plane.
- The default-host flip — adr-39's gate, exercised with the parity report as evidence; not built here.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. Seeded criteria are proposals until the planning interview confirms them._

- **Given** the hook entrypoints run behind host profile #1, **when** the current host drives them across the existing hook corpus, **then** observable behaviour is identical to the pre-seam baseline — the seam lands with zero behaviour change.
- **Given** a second host profile is declared, **when** its adaptor exercises a rung the host cannot support, **then** the adaptor reports the degradation explicitly AND the parity report records it as a documented host difference, never a silent pass.
- **Given** the parity suite passes on the reference host, **when** the same suite runs against any declared profile, **then** it emits a comparable per-check report suitable as adr-39 default-flip evidence.
- **Given** the host-neutral docs skeleton, **when** a host's page is added at its adoption, **then** no existing page changes AND no committed page names a harness whose adoption is undecided.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md).

- The three harnesses abcd sits beside share one hook idiom: a JSON object on stdin with `session_id`, `transcript_path`, `cwd` and `hook_event_name`; exit 0 with JSON on stdout, exit 2 blocks with stderr as the reason; Claude Code and Gemini CLI treat any other code as a non-blocking warning, while Codex's page leaves other codes undefined (to be tested in the parity suite); `hookSpecificOutput.additionalContext` and `decision`/`reason` (Claude Code, Codex and Gemini CLI hook references). Event names differ (`PreToolUse` against `BeforeTool`), and which events honour exit 2 differs, so a profile is logic, not data.
- Plugin formats have converged on one shape under different manifest names: `.claude-plugin/plugin.json`, a root `plugin.json` with `extensions.com.openai` (Codex), `gemini-extension.json`; each with `skills/`, `hooks/hooks.json` and an MCP server file; Codex plugin hooks bar a plugin from its public directory (vendor plugin references).
- Codex's root `plugin.json` declares the open Agent Plugins 1.0.0 schema (`agent-plugins.org`), a vendor-neutral "interoperability floor": a portable core of `plugin.json`, `skills/` (Agent Skills) and `mcp.json`, with client behaviour, hooks among it, under reverse-domain extension keys; its steering committee draws from Amazon, Cursor, Microsoft, OpenAI and Vercel. It is the obvious seam for the "host's own plugin format first" rung, and whether the adaptor's manifest emitter targets it is for the plan; hook profiles stay logic in code either way.
- The Agent Skills `SKILL.md` format is the one unit all three load unchanged (agentskills.io specification); its own client showcase page rendered empty, so adoption is read from the vendors' pages.
- Hosts now import each other's configuration (`claude import codex --dry-run`), which is the direction the AGENTS.md-only amendment already takes.
- The parity suite has a working model in the MCP conformance framework: requirements per revision, a YAML baseline of accepted gaps scoped to individual checks, `checks.json` and a `tier-check` summary that separates required from informative scenarios.
- LSP's capabilities ("not every language server can support all features") are the precedent for a rung that declares what it cannot do; the Agent Client Protocol is a second stdio floor for abcd *as* an agent, recorded and not recommended.
- The AGENTS.md re-check's cited study (about 20% more cost, no gain in success from always-loaded context files) argues against any per-host prose layer in the docs skeleton.

**Path 2 — native host profiles in code, shaped to the three documented hook contracts, with the MCP conformance framework as the model for the parity report, written as a dated lab artefact.** No new dependency. The independent fit-challenge runs at plan time.

## Open Questions

- Does a host profile live in code (a Go type per host) or in committed configuration the adaptor reads?
- Testing strategy for parity: run the entire acceptance matrix on every adopted host, or a pinned sample per release?
- Where does the parity report live — a dated research note per run, or a receipt-shaped artefact beside the release gates?
- (SOTA 2026-10-23) Each baseline entry in the parity report that accepts a host gap: does it need an issue id, so loud staging survives the baseline mechanism?

### Evidence (2026-08-26, second-harness adaptor lab, local tier)

A local lab drove the existing hook entrypoints end to end from a second
host's native plugin runtime: prompt-router injection with per-domain dedup,
guard block-with-reason, session-start notices, and session-end capture all
worked over an unchanged core. The cost was exactly what this intent
predicts: the adaptor had to re-derive the stdin payload schema, the
exit-code semantics, and the stderr contracts by reading the CLI hook source
— the hand-derivation the host profile seam exists to remove. Two seam
requirements the lab surfaced now sit in the ledger: the prompt-router has no
removal signal for snapshotting clients (iss-2608261550580260), and
`hook session-end` has no machine-readable result channel, only a stderr
string contract (iss-2608261550596333). The lab's verification battery —
injection proven from logs rather than model claims, forged-heading and
separator red-team vectors, guard exit-mapping exhaustiveness, capture
watermark invariants — is a concrete seed for the parity conformance suite.

## Amendments

Amended by itd-2610030814013772 (2026-10-03, its decision 7): abcd writes no tool-specific conventions file for any tool, and setup offers to retire one it finds that only repeats or links to AGENTS.md. This draft's portability work starts from that: each tool is pointed at AGENTS.md through its own configuration, never through a file abcd writes for it.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Linkage note (2026-09-30)

`blocked_by` named itd-2 (in-session subagent dispatch), which is superseded by
itd-2609201916056194. The edge is dropped rather than relinked: the successor,
a delegated agent run through a command-line model runner, does not carry the
in-session dispatch contract this record waited on, and that dispatch is standing
practice the conventions router (AGENTS.md) states. Ruling CF2 of 2026-09-30
counts a blocker kept as a discipline as settled; the run's orchestrator read
the same of a practice the router carries, so the edge has nothing left to
wait on (recorded by the integration lane of autonomous run A).
