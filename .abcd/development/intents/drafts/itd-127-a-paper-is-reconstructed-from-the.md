---
id: itd-127
slug: a-paper-is-reconstructed-from-the
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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [sources-provenance SOTA note](../../research/notes/2026-10-23-sources-provenance-sota.md).

- **The structural firewall still exists.** Quarto profiles merge a profile's config over `_quarto.yml` and show or hide content with `.content-visible` and `when-profile`, but "`metadata-files` are not resolved in profiles", so the public profile must name its filtered bibliography in the profile config itself (Quarto profiles documentation).
- **Pandoc reads CSL-JSON directly** (`.json`), supports `nocite` and the all-entries wildcard, and `--fail-if-warnings` turns any warning into a non-zero exit (Pandoc manual). Whether a missing citation key warns is not stated and the design does not rely on it.
- **Typst is not a drop-in.** It accepts only BibLaTeX and Hayagriva, not CSL-JSON (Typst bibliography documentation), so a Typst chain needs a `.bib` (or Hayagriva) export of the already-filtered bibliography; the gates run before that export, so its cost is a second export format, not a lost gate.
- **PROV-O names the relations the ledger implies** (`wasDerivedFrom`, `wasQuotedFrom`, `hadPrimarySource`, `wasAttributedTo`); the 2026-07-08 ruling to borrow the vocabulary and not the format stands.
- **No Go-native CSL processor (citeproc) was found.** Go PDF libraries exist (go-pdf/fpdf, "a PDF document generator") but would leave citation formatting to abcd, so a citation-correct render is an external program (Pandoc or Quarto), which answers the open question: the renderer cannot be a bundled dependency.
- **Path 2**: the binary does the deterministic work with no dependency (walk the ledger, group claims by decision, emit one markdown source and the filtered public CSL-JSON, run the post-render check over keys and prose), and the render is an opt-in adapter that finds `quarto`, then `pandoc`, on PATH, passes a local bibliography path with `--fail-if-warnings`, and refuses with the install instruction when neither is present. The markdown plus bibliography is the basic form; the seam is that pair, so a Typst move is a `.bib` exporter. No Go dependency. The independent fit-challenge runs at plan time.

## Open Questions

- Render toolchain and its dependency posture (the repo's no-hard-deps stance vs PDF generation) — unexamined; gates planning.
- (SOTA 2026-10-23) Does the post-render check read rendered prose for confidential titles and aliases (the banlist), or only the citation keys, given that a confidential source can enter a sentence without a citation?
- (SOTA 2026-10-23) Where does the generated public bibliography live between generation and render: the local tier only, or may a repository commit it once it is proven clean?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
