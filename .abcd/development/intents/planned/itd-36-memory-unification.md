---
id: itd-36
slug: memory-unification
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-28]
severity: major
---

# Knowledge That Compounds, Not Knowledge That Re-Derives

> **Packaging framing per [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md) (supersedes adr-18).** The restrictive-licence gate's consumer is the **lifeboat** (`/abcd:disembark`), future/inert at launch: the curated release excludes `.abcd/**` wholesale, so `/abcd:launch` never evaluates what the gate guards. The canonical framing lives in the brief (`05-internals/09-provenance-substrate.md § 4`, `07-memory.md § 4`, `04-surfaces/04-launch.md § 2`).

## Press Release

> **abcd ships `/abcd:memory` — a per-`voyage` curated knowledge substrate that compounds across sessions instead of being re-derived from scratch every conversation.** Researchers and engineers running a `voyage` under abcd can drop external sources (PDFs, transcripts, articles, talks) into the `voyage`, and abcd reads them, distils them into typed entity/topic pages with full citation, and discards the original by default. The same `.abcd/memory/` namespace also receives synthesised principles from session memory (existing behaviour), cross-references from oracle reviews and `.abcd/work/` notes, and synthesis output from `/abcd:dredge` (itd-25, a separate intent). New sub-verb `/abcd:memory ask <question>` queries the substrate and returns synthesis with citations; `/abcd:memory lint` enforces quotation budgets, licence declarations, and cross-source provenance hygiene. The pattern is the [`Karpathy LLM Wiki`](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) (April 2026) — *"compile once, query repeatedly"* replacing RAG-shaped re-derivation.
>
> "Last quarter I'd open a session, paste the same ten paragraphs of background context, and lose half the conversation rebuilding shared ground," said Carol, technical lead. "Now I point the agent at `.abcd/memory/` and we pick up where we left off — and I can see exactly where each claim came from. The default-no-original rule means I can ingest a 200-page paper without worrying about copyright; the lint catches when a derivative page quotes too much. Nothing I've ever paid for accidentally ends up in a public repo."

## Why This Matters

Today's `.abcd/memory/` is the receiving end of a single upstream — vendor session memory (Claude / opencode), distilled by the memory writer into domain-grouped principle pages. That's the per-project compounding-curated agent-experience layer. Useful, but narrow.

There's a real gap: **per-project durable knowledge has multiple legitimate upstreams, not just session memory.** Research-shaped projects ingest external papers; mixed-licence projects ingest internal notes alongside paywalled academic sources; long-running projects accumulate patterns from oracle reviews and the issue ledger. Today the user has no abcd surface for any of this — they paste context into chat, lose it at session end, and re-derive it next session. RAG patterns help but re-derive synthesis on every query. The Karpathy LLM Wiki pattern (April 2026, gist 5k+ stars) names a different shape: synthesis happens at ingest, not query — cross-references pre-built, contradictions flagged during maintenance.

**The reframe lands cleanly because abcd doesn't need a new namespace.** `.abcd/memory/` already exists as the per-project compounding-curated artefact. Widening its upstream funnel from "vendor session memory only" to "session memory + external sources + reviews + notes + dredge synthesis + spec-extracted theory" is structurally smaller than creating a new top-level namespace — and avoids the visibility-rule violations that would come with a separate `.abcd/knowledge/` directory (per the locked decision in [`05-internals/03-configuration.md § 1`](../../brief/05-internals/03-configuration.md): "no exceptions to the visibility rule").

**Default-no-originals is what makes ingest copyright-safe.** External sources may be paywalled, NDA'd, or under restrictive licences. Storing the original creates a copyright-laundering vector. Default-no-originals + bounded per-page quotation (≤5%, no contiguous span >150 words) + cumulative source-coverage lint (≤25% deduplicated coverage across all pages citing the same source, span-level dedup) closes the laundering vector for the common case. `--keep-original` is the explicit opt-in for sources the user owns or has clear redistribution rights for; the restrictive-licence gate (consumed by the lifeboat `/abcd:disembark`, per adr-28) then refuses to surface anything under `.abcd/memory/sources/` unless an explicit allowlist entry exists.

**Loot substrate reuse compresses cost.** itd-26 `/abcd:loot` (per itd-26) and `/abcd:memory ingest` need the same machinery: licence detection, citation generation, source-hash registry, the lifeboat's restrictive-licence gates. This intent ships the substrate as a separable component spec at [`05-internals/09-provenance-substrate.md`](../../brief/05-internals/09-provenance-substrate.md). itd-26's verb sits on top — no re-implementation, no fragile bespoke licence layer.

## What's In Scope

- **`/abcd:memory` command** with sub-verbs `ingest` / `ask` / `lint`. Bare invocation renders state per the universal bare-command-as-render discipline. Surface contract: [`04-surfaces/07-memory.md`](../../brief/04-surfaces/07-memory.md). Substrate spec: [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md).
- **Schema extension on `.abcd/memory/`** (NOT migration): three new conventional sibling files (`index.md` generated catalog, `log.md` append-only ingest history, `contradictions.md` curator-surfaced register) plus typed `source:` frontmatter on every memory page. Existing flat-named pages preserved; schema extends in place.
- **Typed `source:` frontmatter** with closed-enum source-class taxonomy: `session_memory` / `external_pdf` / `external_transcript` / `external_article` / `oracle_review` / `work_notes` / `issue_ledger` / `dredge_synthesis` / `spec_modification_grammar` / `modification_grammar`. Every external source carries `citation` + `licence` + `source_hash` + `ingested_at`.
- **Default-no-originals + `--keep-original`**: ingest reads → distils → discards by default; flag opts into storage at `.abcd/memory/sources/<sha256>.<ext>`; the lifeboat licence gate refuses to surface `sources/` files without an explicit allowlist entry.
- **Quotation budgets, two layers**: per-page (`MQ001`: ≤5% per page, no >150-word contiguous span) + cumulative source-coverage (`MQ002`: ≤25% deduplicated coverage across all pages citing one `source_hash`, span-level dedup, configurable threshold).
- **Source-class lints**: `MS001` (advisory: single-source-class synthesis is low cross-validation), `MS002` (blocks: cross-class synthesis without `weighting_note` field), `ML001` (blocks: `external_*` page missing `licence` field; `unknown` is acceptable but must be explicit).
- **Curator role on `principle-distiller`** (existing agent; itd-31 precedent — agent count stays 15). Inputs extended: external-source ingest pipeline added alongside existing session-memory + reviews + work-dir + code-rescuer inputs. New output mandate: typed `source:` frontmatter on every emitted page. New lint pass: `MQ`/`MS`/`ML` codes.
- **Provenance/licence substrate** at [`05-internals/09-provenance-substrate.md`](../../brief/05-internals/09-provenance-substrate.md) — separable spec; SPDX licence detection; citation generation; source-hash registry at `.abcd/memory/.sources_index.json` (regenerable per the lifecycle taxonomy); the lifeboat's restrictive-licence gates. Shared with itd-26 loot.
- **Karpathy citation** in [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md) (component description) + [`research/related-work.md`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory) (prior-art table). NOT in `01-product/03-mental-model.md` (per idea-1 R5 review: mental model is for axes, not components; six-week-old citations don't belong in the project's most-stable file).
- **Brief edits**: artefact-lifecycle taxonomy added to [`05-internals/04-universal-patterns.md § 8`](../../brief/05-internals/04-universal-patterns.md#8-artefact-lifecycle-taxonomy) (regenerable / append-only / compounding-curated). Reserved-vocabulary entry in [`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md) for `source.class` + lifecycle classes.

## What's Out of Scope

- **A separate `.abcd/knowledge/` namespace.** Initially proposed (idea-1 pre-review draft); killed by RP review (visibility-rule lock; namespace-creep). Memory unification at the *destination* (existing `.abcd/memory/` namespace) is the right shape; new top-level was wrong.
- **Schema migration of existing flat-named pages.** Current `<type>_<domain>_<slug>.md` already encodes Karpathy's three axes (type = page class; domain = topic; slug = entity ID). Renaming to nested directories would be cosmetic and would touch every existing memory page; not justified.
- **`/abcd:dredge` absorption.** `/abcd:dredge` (itd-25) stays a distinct verb. Its synthesis output writes to `.abcd/memory/<type>_<domain>_<slug>.md` with `source.class: dredge_synthesis` (shared destination namespace), but the verb is not folded into `/abcd:memory`. User moments differ (dredge = "look across what we already have"; ingest = "I have a new source"); folding into one surface mismatches the user's mental grouping. Independent revisit triggers preserved.
- **Cross-project frontier sharing.** Per-project in this intent (privacy + curation overhead). Cross-project sharing surfaces as a fresh idea when triggers fire (≥3 projects with non-trivial accumulated knowledge AND user-driven request).
- **Close-paraphrase detection.** The quotation lint catches *quotation*, not *close paraphrase*. A page that paraphrases an entire source at 100% coverage in its own words is a derivative work and isn't fair-use protected. This intent acknowledges this as a documented limitation in `05-internals/07-memory.md § 5`; semantic-similarity infra to detect paraphrase is a separate intent.
- **Auto-classify the upstream.** This intent requires the user to invoke `/abcd:memory ingest <path>` explicitly. Auto-detection (e.g., scanning `~/Downloads/*.pdf` for ingest candidates) is deferred — a separate intent if friction proves real.
- **MCP server for runtime memory editing.** Other knowledge frameworks ship MCP servers for `add_page`, `merge_pages`, etc. This intent ships JSON-on-disk + CLI surface only; the curator agent (`principle-distiller`) edits via standard file-edit tools. MCP integration is deferred if friction is real.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per [itd-1 acceptance gates](../disciplines/itd-1-acceptance-gates.md). These gates are checked by `intent-fidelity-reviewer` Role 1 when this intent moves to `shipped/`._

- **Given** an external source with declared SPDX licence in a `LICENSE` file, **when** the user runs `/abcd:memory ingest <path>`, **then** the resulting memory page(s) carry `source.licence: <spdx-id>` AND the original is NOT stored at `.abcd/memory/sources/` AND a registry entry exists at `.abcd/memory/.sources_index.json[<sha256>]` whose `consumers` map contains the `memory` key.
- **Given** an external source ingested without `--keep-original`, **when** the user later runs `/abcd:memory ask <question>` referencing that source, **then** the answer cites the source by `(class, citation, source_hash)` but never reproduces the original; if the original is needed for re-distillation, the user is prompted to re-ingest.
- **Given** an external source ingested with `--keep-original`, **when** `/abcd:disembark` runs the restrictive-licence gate, **then** the gate refuses to surface `.abcd/memory/sources/<sha256>.<ext>` unless `.abcd/launch-allowlist.json` explicitly names it (the allowlist re-includes files into the gate's own evaluation input, never into the launch payload — adr-28).
- **Given** a memory page with `source.class: external_pdf` and ≥150-word contiguous quotation, **when** `/abcd:memory lint` runs, **then** the lint emits `MQ001` and points at the offending span.
- **Given** N memory pages collectively citing one `source_hash` with cumulative deduplicated coverage ≥25%, **when** `/abcd:memory lint` runs, **then** the lint emits `MQ002` and refuses any further quotation block from that source until coverage drops below the threshold (via summarisation rewrite of existing pages).
- **Given** a memory page with `source.class: session_memory` only (no cross-class inputs), **when** `/abcd:memory lint` runs, **then** the lint emits `MS001` (low cross-validation; advisory only — does not block).
- **Given** a memory page mixing `session_memory` + `external_pdf` sources without a `weighting_note` field, **when** `/abcd:memory lint` runs, **then** the lint emits `MS002` and blocks until the weighting note is added.
- **Given** an external source ingested without a `licence:` field, **when** `/abcd:memory lint` runs, **then** the lint emits `ML001` and blocks until the licence is declared (`unknown` is acceptable but must be explicit).
- **Given** an existing pre-itd-36 project with flat-named memory files, **when** `/abcd:ahoy` runs after itd-36 ships, **then** the existing files are NOT renamed; `index.md` is generated over them; `source.class: session_memory` is backfilled as the default.
- **Given** dredge synthesis output (itd-25), **when** dredge runs, **then** the dredge agent writes its synthesised entries to `.abcd/memory/<type>_<domain>_<slug>.md` with `source.class: dredge_synthesis` AND the per-run report stays at `.abcd/logbook/dredge/<ts>/`.
- **Given** the same source content is ingested by both `/abcd:memory ingest` (as documentation) and `/abcd:loot` (as code, per itd-26), **when** both have run, **then** the registry has one entry with `ingest_count: 2` whose `consumers` map holds both the `memory` and `loot` keys (memory and loot share one substrate registry).
- **Given** the lifeboat's gated payload includes a file with `citation.licence: GPL-3.0` AND the project is being published as MIT, **when** `/abcd:disembark` runs the restrictive-licence gate, **then** the gate refuses with a licence-mismatch error AND lists the offending file. Override via `--accept-licence-risk` logs the override in the disembark report.
- **Given** this intent's cost-discipline boundary, **when** a promotion proposes adding `known_failure_modes` (or any other dynamic-state field) to memory frontmatter, **then** the proposal is rejected as scope creep — those fields belong to other intents (Frontier Awareness, per a separate intent, for `known_failure_modes`).

## Implementing specs

itd-36 is implemented across multiple specs. The single-valued frontmatter
`spec_id` records the **primary** delivering spec (spc-38); the remaining spec is
recorded here because `spec_id` holds one value and would understate scope.
This section is the canonical multi-spec implementation index:

- **spc-38** (primary) — `/abcd:memory` write core (the memory substrate, ingest, registry).
- **spc-39** — `/abcd:memory lint` quality gate (quotation-budget / licence / provenance lint).

## Ship gate — adversarial worked examples

Per the idea-1 R5 review: the push replaces "≥3 projects in anger" (unachievable pre-ship) with **three adversarial worked examples** demonstrated before itd-36 is closed. Confirmatory examples are not enough; pick projects where ingest *might fail*:

1. **Research-shaped project** — heavy external PDFs, papers, talks. Test: does the synthesis stay coherent across 20+ external sources? Does the cumulative coverage lint trigger correctly? Are entity/topic pages distinguishable from a stack of PDF summaries?
2. **Tooling project** (e.g., abcd itself dogfooding) — almost no external sources; mostly session memory + reviews + notes. Test: does the schema feel cluttered with `source.class: session_memory` defaults? Does `index.md` provide value when most pages are principle-shaped?
3. **Mixed-licence project** — open-source docs + proprietary internal notes + paywalled academic sources. Test: do the licence gates and quotation budgets behave correctly? Does the lifeboat licence gate correctly refuse to surface proprietary content under `--keep-original`?

If all three produce coherent `.abcd/memory/` outputs, this intent ships. If one produces sprawl or fails the licence gates, scope is reduced and the failed example becomes the primary debug target.

## Open Questions

- **Persona for the press release** — Carol is product-lead per `personas.json`. The current draft assigns Carol "technical lead" instead — verify that the persona registry's role assignment doesn't conflict, OR pick a persona whose role-hint is "researcher" / "engineer" / "lead investigator" instead. (Closing fix expected at promotion time.)
- **Lint code numbering** (`MQ001`/`MQ002`/`MS001`/`MS002`/`ML001`) — illustrative; verify against [`05-internals/06-lint.md`](../../brief/05-internals/06-lint.md) reservation table at promotion time. Adjacent reserved codes (`SD001` per the bare-command-as-render discipline; `VR001` per the vocabulary-registration requirement; `MG001`-`MG004` per itd-37) should not collide.
- **Recursive ingest** — does `/abcd:memory ingest` accept a directory (recursive ingest of all files) or only one source per call? Operational decision, not architectural. Lean: one source per call (avoids accidental "ingest the whole filesystem" mistake); directory-walk via `--recursive` flag is a candidate if friction is real.
- **Cumulative coverage state location** — `.abcd/memory/.coverage_index.json` (per-source cumulative coverage) is its own JSON registry rebuilt by full crawl on demand. Drift between the index and source-of-truth pages IS the lint signal. Verify location matches existing dotfile conventions in `.abcd/memory/`.

## Audit Notes

_Empty. Populated by `intent-fidelity-reviewer` Role 1 (single-document fidelity per the itd-1 discipline) when this intent moves to `shipped/`._

## References

- Full assessment in a local working note (unmigrated), with 5-round review trail (chat `adversarial-review-llm-w-4970CE`); pre-review framings preserved in chat record.
- [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md) — substrate spec (page-class enum, source classes, curator behaviour, lifecycle class).
- [`05-internals/09-provenance-substrate.md`](../../brief/05-internals/09-provenance-substrate.md) — provenance/licence subsystem shared with itd-26 loot.
- [`04-surfaces/07-memory.md`](../../brief/04-surfaces/07-memory.md) — surface contract for `/abcd:memory`.
- [`research/related-work.md § Karpathy LLM Wiki`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory) — pattern source.
- [`itd-1-acceptance-gates.md`](../disciplines/itd-1-acceptance-gates.md) — companion discipline; this intent's acceptance criteria conform to its Given-When-Then shape.
- [`itd-37-modification-grammar.md`](../disciplines/itd-37-modification-grammar.md) — companion discipline; ships alongside itd-36 (page-classes `spec_modification_grammar` + `modification_grammar` are part of itd-36's source-class enum).
- [`itd-25-dredge-cross-corpus-synthesist.md`](../drafts/itd-25-dredge-cross-corpus-synthesist.md) — sibling intent; writes to `.abcd/memory/` with `source.class: dredge_synthesis`; verb stays distinct (storage vs operation per dredge-pushback in idea-1 R4).
- [`itd-26-loot-oss-vendor.md`](../drafts/itd-26-loot-oss-vendor.md) — sibling intent; consumes the same provenance/licence substrate this intent ships.

[karpathy-llm-wiki]: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f "Karpathy LLM Wiki gist (April 2026)"
