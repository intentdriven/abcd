---
id: itd-113
slug: abcd-s-core-verbs-are-reachable-over-mcp
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
---

# The MCP Front Door Opens — abcd's Core Verbs From Any MCP-Capable Harness

## Press Release

> **abcd's second front door is open: MCP.** The same transport-agnostic core the CLI drives is now reachable as `mcp:abcd:*` tools from any MCP-capable harness — no plugin, no per-host adaptor, no waiting for an integration. Register the server, and intents, capture, guard checks, rules, and history are on the harness's tool palette, returning the same structured results the CLI returns. Under the host-tier policy (adr-39) this is the universal floor: the guarantee every harness starts from, before any native adaptor makes it richer.
>
> "Our harness had no abcd integration, and it didn't matter," said Alice. "One MCP server entry and the whole record was drivable from the agent. When a native adaptor shipped later, nothing we'd built on the floor had to change."

## Why This Matters

adr-23 committed to the MCP server as the second thin front door over the unchanged core, and Phase 3 schedules it ("MCP front door opens") — but no intent, spec, or code exists for it; `internal/surface/mcp` is a reserved, empty seam. adr-39 raises the stakes: the MCP front door is the floor of the whole multi-harness policy — the tier every MCP-capable harness gets with zero per-host work, and the fallback rung every adaptor ladder ends on. Nothing above the floor (per-host adoptions, the eventual open-source default) is credible until the floor exists.

## What's In Scope

- `internal/surface/mcp` as a thin front door: core verbs exposed as `mcp:abcd:*` tools, each returning the same structured result the core hands the CLI — the front door owns transport and input validation at the boundary (adr-23), never logic.
- Server lifecycle from the binary (an explicit serve verb) — stdio transport first; the CLI stays the reliable default front door (brief invariant: every capability reachable with no plugin host present).
- Registration docs: how any MCP-capable harness registers the server (host-neutral; per-host pages arrive with per-host adoptions).
- Parity evidence: an MCP client invoking a verb gets a result equivalent to the CLI invocation — the floor row of itd-22's parity suite.
- **The one stable contract a tool built on abcd reads** (proposed 2026-11-01, to confirm at planning): a verb-shaped addition for a repository's product lives in the host's own plugin format or a script, and reaches abcd only through these tools' structured results and the JSON Schemas generated for them, so extending abcd never means loading code into it.

## What's Out of Scope

- Per-host adaptors, native plugins, and concept mapping — itd-22's lineage and per-host adoption intents (adr-39).
- abcd acting as an MCP *client* (oracle adapters over MCP) — adr-25's separate seam.
- Remote/HTTP transport — stdio first; a further transport is its own decision.
- Any new hook or context-injection behaviour — the floor exposes verbs; dynamic features are what native rungs add.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. Seeded criteria are proposals until the planning interview confirms them._

- **Given** the abcd binary with no plugin host present, **when** the MCP server verb is invoked over stdio by a conforming MCP client, **then** the advertised tool list covers the core verbs designated for the floor AND each tool call returns the same structured result the corresponding CLI invocation returns for the same repository state.
- **Given** an unrecognized or malformed tool input, **when** a mutating verb receives it over MCP, **then** the call fails closed with a structured error and writes nothing (unrecognized input never writes).
- **Given** the MCP front door is live, **when** the surface registry and brief are linted, **then** the new surface has its brief row and surface-registry entry in the same change (spec moves with the surface).
- **Given** a Go MCP dependency is proposed for the implementation, **when** the intent reaches planning, **then** the dependency passes the sota-per-intent path decision with explicit maintainer approval before any `go get` (hard stop).

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md).

- The current MCP revision is 2026-07-28: the `initialize` handshake and protocol sessions are gone, every request carries its version and capabilities in `_meta`, servers MUST answer `server/discover`, results carry `resultType`, roots/sampling/logging are deprecated, and `tools/list` should be deterministic for prompt caching (MCP changelog). Dated revisions through 2025-11-25 still use the stateful handshake (conformance framework), and which revision each host negotiates today was not found, so the floor answers both.
- Structured results map straight onto `outputSchema` and `structuredContent`; input failures are tool errors (`isError`), unknown tools are JSON-RPC errors; tool annotations are untrusted to clients; a no-argument tool uses `{"type":"object","additionalProperties":false}` (MCP tools page).
- The official Go SDK (`modelcontextprotocol/go-sdk`, with Google) supports 2026-07-28 at v1.7.0+ but its `go.mod` requires eight direct modules including `x/tools`, `x/oauth2` and a JWT library; `mark3labs/mcp-go` is at 2025-11-25 and "under active development". Either is a new dependency needing explicit sign-off before any `go get`.
- The no-dependency cost is measured for the older handshake and estimated for the new: the guard's approve sub-verb already serves `initialize`, `ping`, `tools/list` and `tools/call` over stdio in 312 lines of standard library (`internal/surface/cli/guard_approve.go`), speaking 2025-06-18, and the RepoPrompt route is a Streamable HTTP client beside a test server. The 2026-07-28 deltas on top of that are `server/discover`, per-request `_meta` parsing, `resultType` on every result, the required `ttlMs` and `cacheScope` on list results (`CacheableResult`), and `UnsupportedProtocolVersionError` (`-32022`) on a version mismatch; `ping` and `logging/setLevel` are removed in the new revision, so `ping` and `initialize` stay only for pre-2026-07-28 clients.
- Every host prefixes tool names itself (`mcp__abcd__<tool>` in Claude Code for a user- or project-scoped server and `mcp__plugin_abcd_abcd__<tool>` when the server is bundled in the plugin, which is the route abcd ships; `mcp_abcd_<tool>` in Gemini CLI, which also splits policy rules on the first underscore), so abcd's names stay plain `[a-z0-9_]`; the `mcp:abcd:*` spelling in this record is a label, not a wire name.
- Registration is three config shapes: `claude mcp add` with `.mcp.json` at project scope (approval asked), `[mcp_servers.<name>]` in Codex's `config.toml` (`startup_timeout_sec` 10, `tool_timeout_sec` 60), `mcpServers` in Gemini's `settings.json` (`trust` off by default).
- Codex's `writes` approval mode prompts for every tool "not marked read-only", so `readOnlyHint`/`destructiveHint` on each verb decide whether a harness asks first.
- The conformance framework (`npx @modelcontextprotocol/conformance`) is the parity evidence; it is a Node tool, a lab dependency and never a build step.

**Path 2 — a native stdio server shaped to the 2026-07-28 specification, extending the hand-rolled server the binary already carries, with the official Go SDK named as the later replacement for the framing layer only; adopting the SDK instead is a hard stop needing the maintainer's sign-off.** The independent fit-challenge runs at plan time.

## Open Questions

- Which subset of core verbs is the floor's v1 tool list — all read verbs plus which mutating ones?
- MCP SDK versus hand-rolled protocol: the anticipated SDK is a new dependency (hard stop); what is the no-dependency alternative's real cost?
- How does the server resolve the target repository — cwd at launch, a per-call argument, or both?
- (SOTA 2026-10-23) Should the guard's approve server and the front door become one server package before the floor ships, so the binary carries one MCP dispatcher rather than two?
- (SOTA 2026-10-23) Which mutating verbs carry `destructiveHint`, given Codex's `writes` mode prompts on every tool not marked read-only?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
