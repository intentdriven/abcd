---
id: itd-127
slug: a-paper-is-reconstructed-from-the-provenance-ledger-claims-g
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-76]
severity: minor
impact: additive
---

# A Paper Is Reconstructed From the Provenance Ledger

## Press Release

> **A paper whose claims trace to decisions, and whose decisions trace to sources, is reconstructable rather than rewritten from memory.** The provenance ledger ([itd-76](../shipped/itd-76-source-provenance-ledger.md), which this intent `refines`) already records which source influenced which decision with what claim. Paper reconstruction walks that trail: claims grouped by decision, citations resolved from the bibliography, rendered to PDF and HTML from one markdown source. The public render is proven clean twice, independently: structurally — it renders from a *generated* bibliography that omits confidential entries, so an unpermitted key fails the build — and by a deterministic post-render check of the output's citations against both gates (`permission_status` on the source, `cited_publicly` on the ledger line).
>
> "When the paper behind a decision is finally published, I flip one flag and the citation appears in the next render," said Alice, "with the whole influence trail already written."

## Why This Matters

The ledger's value compounds only if something reads it back. Reconstruction turns the eager, cheap capture of influence into the expensive artefact — a publishable paper — without a second bookkeeping pass, and the double-proof render is where the [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md) two-gate rule pays off at publication time: the build cannot cite what the human has not deliberately released.

## Acceptance Criteria

> _Seeded by the itd-76 planning session, unconfirmed — the planning interview must walk these with the maintainer._

- Given ledger lines across decisions, when reconstruction runs, then claims are grouped by decision and every resolvable citation comes from the generated public bibliography.
- Given a citation whose source fails either gate, when the render runs, then the build fails structurally, and a passing render also passes the independent post-render citation check.

## Open Questions

- Render toolchain and its dependency posture (the repo's no-hard-deps stance vs PDF generation) — unexamined; gates planning.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
