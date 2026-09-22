---
id: itd-22
slug: harness-portability
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-2]
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

## Open Questions

- Does a host profile live in code (a Go type per host) or in committed configuration the adaptor reads?
- Testing strategy for parity: run the entire acceptance matrix on every adopted host, or a pinned sample per release?
- Where does the parity report live — a dated research note per run, or a receipt-shaped artefact beside the release gates?

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

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
