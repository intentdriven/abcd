# `/abcd:memory` — Multi-Upstream Curated Knowledge Substrate

User-facing command for the per-project compounding-curated knowledge substrate at `.abcd/memory/`. Design target per itd-36 (idea-1 final shape after 5-round adversarial oracle review); the write core (ingest/ask/bare) traces to the predecessor store's spc-38 (the memory write core) and the lint family to the predecessor store's spc-39 (the memory-coverage lints + the `MQ`/`MS`/`ML` codes). Every bare `spc-38`/`spc-39` on this page is a predecessor-store id cited as provenance, and both numbers collide with live ids in this repo's own store (`.abcd/development/specs/`), where `spc-38` is the record explorer and `spc-39` the relationship chart. The store's hand-numbered ids run well past both. So the store an id belongs to is read from the sentence around it, never from the number; itd-36's `## Implementing specs` section names the same pair, which puts the collision in the record rather than only on this page. itd-36 sits in `intents/planned/` — delivery state is the intent lifecycle's, not this page's (see the [brief README's provenance note](../README.md)).

For the **substrate spec** (page-class enum, source-class taxonomy, curator behaviour, lifecycle class, integration with itd-26 loot), see [`05-internals/07-memory.md`](../05-internals/07-memory.md). This file is the surface contract: what the user types and what happens.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `ask` | — | shipped |
| `ingest` | — | shipped |
| `lint` | lint | shipped |


**The store every verb here addresses is the checkout's, resolved from the working directory and never taken to be it.** The front door asks [`gitutil.CheckoutRoot`](../../../../internal/gitutil/repo.go) before it builds a request — the same resolution `capture`, `decide` and `spec` address their stores through: git's toplevel where git will name one, and a refusal in the two remaining states rather than a guess (a repo-shaped tree git will not answer for, and no repository above at all). So a render from a package directory reports the checkout's pages instead of `store not present`, an ingest lands in the checkout's substrate instead of laying a second one beneath the caller, and outside a checkout every verb exits **2** having read nothing and written nothing. It deliberately does not fall through to a marker walk, which would accept any directory carrying the name (iss-2609090947359464). A memory store found **below** the checkout root, on the chain between the caller and it, is named on stderr and left untouched — the deposit an unresolved front door leaves behind, reported to the person standing over it rather than stepped over in silence.

Bare `/abcd:memory` renders the store's current state and nothing else — never mutates state, and prints no help text (help is `abcd memory --help`). Per the [bare-command-as-render discipline](../02-constraints/04-naming.md). Current sub-verbs (each does something bare cannot):

- **Bare `/abcd:memory`** — render: page count by class (e.g., "23 session_memory + 8 external_pdf + 4 oracle_review + 2 spec_modification_grammar"), last-ingest timestamp, and the recent contradictions. No suggested next actions, and no mutation. The JSON render carries one element the text render drops: a `drift` list, which says `index stale; run an ingest` or `contradictions register stale; run an ingest` when `index.md` or `contradictions.md` no longer hash-match what the store's pages would render. Quotation-budget headroom per source renders READ-ONLY from the spc-39 `.coverage_index.json`: when the index is present AND fingerprint-fresh (a read-only crawl recomputes the current fingerprint and matches the stored one) it shows per-source warn/block headroom; fingerprint drift shows a "stale — run /abcd:memory lint" hint; an absent index an info line; a malformed index or crawl failure a non-fatal "headroom unavailable" line. The bare render never rebuilds or mutates the index.
- **`/abcd:memory ingest <path-or-https-url>`** — register an external source (transcript / article / https URL — a plaintext `http://` source is refused by name and a redirect leaving https is refused per hop, since the store copies fetched text and the licence lifted from it verbatim into durable provenance; a credential the URL carries — basic-auth userinfo, or a credential-shaped query key such as `token` or `api_key` — is stripped before the fetched address becomes the stored origin and title, and masked in every fetch-failure message; PDF is a later-phase seam — the binary rejects PDF sources with a clear error until a text-extraction dependency is wired) as typed entity/topic pages with citation frontmatter, appending to the ingest log. The host agent is the distiller: it reads the source, produces the `DistilledPage` JSON array, and passes it via the load-bearing `--pages-json <file|->` flag (required for a new source; an already-known source re-ingests from the registry without it); the binary computes provenance, licence, and content hash, validates every page, and writes atomically. **Default: do NOT store original.** Flag-shaped modifier: `--keep-original` (opt-in storage at `.abcd/memory/sources/<sha256>.<ext>`; the later-phase lifeboat licence gate — `/abcd:disembark`, NOT launch, per adr-28 — is designed to refuse publish without an explicit allowlist entry; launch excludes `.abcd/**` wholesale per [`04-launch.md § 2`](04-launch.md#2-curated-release-artefact-default-deny)).
- **`/abcd:memory ask <question>`** — query memory by domain + class; synthesise an answer with citations (every citation references `source.class` + `citation` + `source_hash`); optionally file the result back as a new memory page (flag-driven: `--file-back` with `--page-json <file|->`, the host-produced answer page; `--top-n <int>` sets retrieval depth, 0 uses the pinned default).
- **`/abcd:memory lint` (spc-39)** — full-store curator health-check: per-page quotation budgets (`MQ001`), cumulative source coverage (`MQ002`), coverage-unavailable diagnostic (`MQ003`, info), source-class single-class advisory (`MS001`), cross-class without weighting note (`MS002`), missing licence on `external_*` (`ML001`), secret or identity residue in stored text (`MR001`, blocker — the read side of the write-time store redactor, run over every page, `.sources_index.json` and each text kept-original; the finding names the kind and the line, never the span, and lint never rewrites the store). ALWAYS crawls the full repo store, rebuilds the regenerable `.coverage_index.json`, emits findings to `.abcd/.work.local/logs/memory/lint-<ts>/report.{json,md}`. Exit: blockers → nonzero; warn-only → 0 (curator advisory — see [`06-lint.md §2`](../05-internals/06-lint.md#2-severity-model)). Mutates no memory-store state (coverage index + logbook report are its only writes). Per ADR-13's write/lint split, the spc-38 write core ships ingest/ask/bare; spc-39 ships this lint family ONLY — contradictions are rendered by spc-38's reconciliation (surfaced by the bare render), orphan/stale-claim audits are deferred.

## 1. Default flow — distil, cite, discard

```
/abcd:memory ingest <path>
    │
    ▼
PROBE
  - Compute sha256 of source content
  - Look up in .abcd/memory/.sources_index.json (the provenance substrate per
    itd-36/spc-38, the provenance capability; distinct from the
    session-transcript store that keys transcripts on the root-commit SHA)
  - If found: bump ingest_count, update last_ingest, return cached citation
  - If new: continue
    │
    ▼
LICENCE DETECT (per 05-internals/09-provenance-substrate.md § 1)
  - Parse source for SPDX-ID (in-file SPDX header + HTTP `License:` header — memory
    ingest passes no source root, so the LICENSE-file and package-manifest steps
    are inert on this surface)
  - On ambiguous / missing: record `licence: unknown` explicitly, no prompt
    (spc-39's `ML001` is what lints it — see § 2)
  - Later phase (not in the shipped ingest, which classifies restrictive licences
    but never rejects): reject if licence is restrictive AND project is public
    (--accept-licence-risk override)
    │
    ▼
DISTIL (host-delegated: the host agent is the distiller, supplying the
        DistilledPage array via --pages-json; the binary validates every page.
        The principle-distiller curator role is a Phase 6 design target per
        05-internals/01-agents.md)
  - Read source content
  - Produce N entity/topic pages: <type>_<domain>_<slug>.md
  - Each page carries source: { class, citation, licence, source_hash, ingested_at, weighting_note? }
  - Cross-reference to existing memory pages (topic-hash dedup)
  - Apply per-page quotation budget as curation discipline (the MQ001 lint
    that enforces it computes at LINT time — spc-39's `/abcd:memory lint` —
    never at ingest)
    │
    ▼
WRITE
  - .abcd/memory/<type>_<domain>_<slug>.md (new pages or updates)
  - .abcd/memory/README.md (store skeleton, scaffolded on first write)
  - .abcd/memory/index.md (regenerated catalog)
  - .abcd/memory/log.md (append: ## [YYYY-MM-DD HH:MM] external_pdf | <slug> — <summary>)
  - .abcd/memory/contradictions.md (if curator surfaces conflict with existing pages)
  - .abcd/memory/.sources_index.json (registry update)
    │
    ▼
DISCARD ORIGINAL (default behaviour)
  - Source path + hash recorded for re-ingest only
  - Original NOT stored at .abcd/memory/sources/
  - The log carries no discard notice — log.md entries are the fixed page-write
    line only (`## [YYYY-MM-DD HH:MM] <class> | <slug> — <summary>`)
```

`--keep-original` opts the user into storing the original at `.abcd/memory/sources/<sha256>.<ext>`. Later phase: the spc-38 restrictive-licence gate refuses to publish anything under `.abcd/memory/sources/` unless `.abcd/launch-allowlist.json` explicitly names the file — the shipped ingest classifies restrictive licences but never gates. Per adr-28 this gate is the **lifeboat's** (`/abcd:disembark`), NOT launch's — launch excludes `.abcd/**` wholesale and never publishes `.abcd/memory/sources/`; the gate is future/inert at launch.

## 2. Acceptance Criteria (Given-When-Then, per itd-1)

See [the full acceptance criteria](../../intents/planned/itd-36-memory-unification.md#acceptance-criteria) in itd-36's intent spec. Surface-level summary:

- **Bare**: bare `/abcd:memory` renders current state; never mutates.
- **Ingest default-no-original**: original NOT stored unless `--keep-original`; citation + source_hash recorded; quotation budget applied per page (enforced at lint time by spc-39's `MQ001`, never at ingest).
- **Ingest with `--keep-original`**: original stored at `.abcd/memory/sources/<sha256>.<ext>`; the later-phase lifeboat licence gate (`/abcd:disembark`, not launch — adr-28) is designed to refuse publish without allowlist.
- **Ask**: synthesises answer with per-citation provenance (class + citation + source_hash); optionally files result back.
- **Lint (spc-39, not spc-38 behaviour)**: emits `MQ001` / `MQ002` / `MQ003` / `MS001` / `MS002` / `ML001` / `MR001` codes; cumulative coverage uses span-level dedup. spc-38 writes `licence: unknown` explicitly; spc-39's `ML001` is what lints it.
- **Schema extension on existing**: existing flat-named pages preserved; `index.md` generated over them; `source.class: session_memory` backfilled as default.
- **Cross-consumer registry**: the provenance substrate's `.abcd/memory/.sources_index.json` (per itd-36/spc-38, the provenance capability; not to be confused with the ahoy history store) is shared with itd-26 loot (a later phase, not yet built); same hash → same registry entry.

## 3. Runtime-log layout

```
.abcd/.work.local/logs/memory/
├── ingest-<utc-ts>/                # later phase — comes with the principle-distiller curator (Phase 6)
│   ├── ingest-report.{json,md}     # source path, sha256, distilled page count, citation, licence
│   └── distil-trace.json           # principle-distiller per-page output trace (debug)
├── ask-<utc-ts>/                   # later phase — comes with the principle-distiller curator (Phase 6)
│   └── ask-report.{json,md}        # question, retrieved page slugs, synthesised answer with citations
└── lint-<utc-ts>/
    └── report.{json,md}            # lint findings: MQ001/MQ002/MQ003/MS001/MS002/ML001/MR001 with locations
```

Only `lint-<utc-ts>/report.{json,md}` is written by the shipped store (spc-39's lint); ingest and ask write no reports — the `ingest-` and `ask-` trees come in a later phase with the principle-distiller curator. Runtime artefacts (reports only) live in the gitignored `.abcd/.work.local/logs/` tier.

## 4. Composition with adjacent surfaces

- **`/abcd:disembark`** exports curated project memory/provenance into the lifeboat (designed behaviour; itd-36 doesn't change disembark's source-mapping). What the lifeboat is specified to carry is the curated provenance surface named in [`02-disembark.md §5`](02-disembark.md) — root `_provenance.json` (the lifeboat marker and manifest) and `coverage.{json,md}` (per-section grounded/partial/blank status) — **not** a verbatim `.abcd/memory/` payload; declaring an exact `.abcd/memory/`-verbatim payload is deferred to the disembark spec that wires the lifeboat packer (adr-28). The recovery-humility framing on disembark/embark applies: the lifeboat is the floor of recoverable theory, not theory itself.
- **`/abcd:embark`** unpacks a lifeboat's record families into the receiving repo; carrying `.abcd/memory/` forward is designed behaviour. Source-class enum carries forward; receiver runs `/abcd:memory lint` (spc-39) post-unpack to verify quotation budgets and licences haven't drifted.
- **`/abcd:launch`** does **not** consume the provenance substrate's licence gate (adr-28): the public launch payload excludes `.abcd/**` — including `.abcd/memory/**` — wholesale as policy, so launch never publishes the files the gate checks. The restrictive-licence gate's real consumer is the **lifeboat** (`/abcd:disembark`, above), the surface that publishes curated project memory/provenance; the later-phase gate is designed to refuse publish on restrictive-licence files and warn on `licence: unknown`. At launch the gate is future/inert — launch excludes `.abcd/**` wholesale, so `/abcd:launch dry-run` surfaces no licence-gate verdict (its gate roster is six: the secret/PII scan, installability smoke and citation-baseline gates run for real, marker-block and documentation-auditor report `not_implemented` as Phase-5 deferred, and semantic-receipts reports a third status, `host-run`, naming the commit it found no receipts for; see [`04-launch.md § 2`](04-launch.md#2-curated-release-artefact-default-deny) and adr-28).
- **`/abcd:dredge`** (a later phase, itd-25) writes synthesis output to `.abcd/memory/<type>_<domain>_<slug>.md` with `source.class: dredge_synthesis`. Distinct verb (storage vs operation per dredge-pushback in idea-1 R4); shared destination namespace.
- **Native** specs inherit from itd-37 modification grammar — at spec completion, `principle-distiller` (the agent ships, with its prompt and its `disembark principles` verb, per [`05-internals/01-agents.md`](../05-internals/01-agents.md); the curator role described here is the part that does not) extracts the spec's `## Modification Grammar` section into `spec_modification_grammar_<spec_id>.md` (append-only) and updates curator-merged `modification_grammar_<domain>.md` (compounding-curated). User does not invoke `/abcd:memory` for this — the extraction is designed to be automatic on spec completion.

## 5. Cost shape

| Item | Cost |
|---|---|
| Schema extension | 4 new sibling files (`README.md`, `index.md`, `log.md`, `contradictions.md`) + typed `source:` frontmatter; no migration of existing flat-named files |
| Curator role on `principle-distiller` | Role extension on a shipped agent (per [`05-internals/01-agents.md`](../05-internals/01-agents.md): the prompt and its `disembark principles` verb ship; this role does not). itd-31 precedent: the design-target agent roster stays at 16 |
| Provenance/licence substrate | Separable spec at [`05-internals/09-provenance-substrate.md`](../05-internals/09-provenance-substrate.md); shared with the later-phase itd-26 loot |
| Lifeboat licence-gate extension (adr-28) | `.abcd/memory/sources/` allowlist + restrictive-licence detection over the lifeboat's gated payload (`/abcd:disembark`); NOT a launch payload gate — launch excludes `.abcd/**` wholesale, so the gate is future/inert at launch |
| Lint codes | New family: `MQ001` (per-page quotation), `MQ002` (cumulative coverage), `MQ003` (coverage-unavailable diagnostic, info), `MS001` (source-class single-class), `MS002` (mixed-class without weighting note), `ML001` (licence missing), `MR001` (secret or identity residue in stored text — report only, never rewrite) |

**Tight coupling** (logged as a principal risk): itd-36 is structurally non-decomposable — sub-verbs need the schema; schema needs the curator role; curator role needs the lint codes. Partial ship not meaningful.

## References

- [`05-internals/07-memory.md`](../05-internals/07-memory.md) — substrate spec (page-class enum, source classes, curator behaviour)
- [`05-internals/09-provenance-substrate.md`](../05-internals/09-provenance-substrate.md) — provenance/licence subsystem (shared with the later-phase loot verb)
- [`../../intents/planned/itd-36-memory-unification.md`](../../intents/planned/itd-36-memory-unification.md) — full intent spec with acceptance criteria + adversarial worked-example ship gate
- [`research/related-work.md § Karpathy LLM Wiki`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory) — pattern source
