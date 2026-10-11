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

## Open Questions

- Which subset of core verbs is the floor's v1 tool list — all read verbs plus which mutating ones?
- MCP SDK versus hand-rolled protocol: the anticipated SDK is a new dependency (hard stop); what is the no-dependency alternative's real cost?
- How does the server resolve the target repository — cwd at launch, a per-call argument, or both?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
