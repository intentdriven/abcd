# `.abcd/memory/` — Multi-Upstream Curated Knowledge Substrate

`.abcd/memory/` is abcd's compounding-curated knowledge artefact (lifecycle class per [`04-universal-patterns.md § 8`](04-universal-patterns.md#8-artefact-lifecycle-taxonomy)). It accumulates synthesised knowledge across project lifetime from multiple upstream pipelines, structured per the [Karpathy LLM Wiki pattern][karpathy-llm-wiki] (April 2026): raw sources → curated wiki → schema. abcd realises the pattern at the destination namespace; the wiki is the *organisation* of memory, not a separate artefact.

Per itd-36 (storage model) and itd-39 (scope-aware retrieval).

## 0. Memory scopes and routing

Memory exists at the two `.abcd/` scopes (see [`03-configuration.md`](03-configuration.md#the-two-abcd-scopes)):

| Scope | Location | What lands here |
|---|---|---|
| **repo** | in-tree `.abcd/memory/` | **The primary home.** Project-shaped knowledge — pitfalls, decisions, principles tied to the project being built. Most memory is repo-scoped. |
| **user** | `~/.abcd/memory/` | **Personal preferences** and cross-project principles that have no single repo home (e.g. a preferred phrasing convention, or a lesson that applies to every project). |

**Routing rule.** Which scope a curated page lands in is decided by `principle-distiller` at curation time, by the *kind* of knowledge — not by where the source happened to sit:

- Default → **repo**. Anything project-shaped.
- Promote to **user** when the knowledge is a personal preference, or a principle that applies across projects with no single repo home.

**The vendor boundary.** abcd never writes to `~/.claude/.../memory/`. The Claude Code memory directory is a *read-only harvest source* for `dev-sync memory` (see [`02-adapters.md`](02-adapters.md)); curated output is written to the scope-appropriate `.abcd/memory/`, never back upstream.

**Retrieval is not flat inheritance.** An agent does **not** load the union of both scopes — that is precisely the context-overflow failure mode. Retrieval is keyword-recall + budget-bracketed injection; see § 9.

## 1. Page model

Memory pages use the existing flat naming `<type>_<domain>_<slug>.md`. The flat form already encodes Karpathy's three axes: `<type>` is page class; `<domain>` is topic; `<slug>` is per-page identifier. **No migration of existing flat-named files.** abcd ships schema *extension*, not schema replacement.

```
.abcd/memory/
├── README.md                 # schema documentation (this file's content + page-class enum + lint contract)
├── index.md                  # generated catalog over all pages, one line per page (class + domain + summary)
├── log.md                    # append-only ingest record: ## [YYYY-MM-DD HH:MM] <upstream_class> | <slug> — <summary>
├── contradictions.md         # surfaced contradictions register; each entry cross-links the conflicting pages
├── <type>_<domain>_<slug>.md # individual pages (existing convention)
├── <type>_<domain>_<slug>.md
├── ...
└── sources/                  # opt-in only via --keep-original; default-deny at the lifeboat licence gate (per § 4; adr-28 — NOT a launch payload gate)
    └── <sha256>.<ext>
```

`index.md` and `log.md` are generated/maintained by `principle-distiller` (the curator). `contradictions.md` is curator-surfaced. Existing memory pages from pre-itd-36 projects are NOT renamed; the schema extension generates `index.md` over them in place, and `source:` frontmatter is backfilled with `source.class: session_memory` as the default on first `/abcd:ahoy` after itd-36 ships.

## 2. Page-class enum

Every memory page declares its source class via typed `source:` frontmatter. The enum is **closed, PR-to-extend** (governance via brief edit to [`02-constraints/04-naming.md § Reserved vocabulary`](../02-constraints/04-naming.md)):

| `source.class` | Upstream | Mutability |
|---|---|---|
| `session_memory` | Vendor session memory (Claude / opencode / future), distilled by the `internal/core/memory` adapter via `dev-sync memory` | Re-distilled on each sync; lossy upstream |
| `external_pdf` | User-ingested PDF via `/abcd:memory ingest <path>` | Immutable post-ingest (original NOT stored unless `--keep-original`) |
| `external_transcript` | User-ingested transcript via `/abcd:memory ingest <path>` | Immutable post-ingest |
| `external_article` | User-ingested article (URL or local) via `/abcd:memory ingest <path-or-url>` | Immutable post-ingest |
| `oracle_review` | Synced from `.abcd/work/reviews/` (RepoPrompt / codex review artefacts) | Immutable post-sync |
| `work_notes` | Curated from working notes in `.abcd/work/` (shared, committed) | User-mutable upstream |
| `issue_ledger` | Synthesised entries from the `.abcd/work/issues/` ledger (per itd-4 capture) | Immutable post-create |
| `dredge_synthesis` | Cross-corpus synthesis output from `/abcd:dredge synth` (per itd-25 — a later phase) | Per-run; durable knowledge |
| `spec_modification_grammar` | Per-spec theory extraction from a spec's `## Modification Grammar` section (per itd-37) at spec completion | Append-only per spec |
| `modification_grammar` | Compounding-curated cross-spec synthesis on the same domain (per itd-37) | Curator-merged across specs |

## 3. Typed `source:` frontmatter

Every memory page carries a typed `source:` block. `citation` is the **object** shape from [`09-provenance-substrate.md § 2`](09-provenance-substrate.md#2-citation-generation) — a mapping, never a string — so one canonical citation shape serves both the provenance registry and memory pages.

A **single-source** page carries a scalar `source.class`:

```yaml
---
source:
  class: <one of the enum values above>
  citation: { type: knowledge, origin: "<url|path>", author: "...", title: "...", year: 2026, ingested_at: 2026-06-09, ingested_by: "..." }   # required for external_*
  licence: "<spdx-id|declared-by-user|unknown>"        # required for external_*
  source_hash: <sha256>                                # required for external_* and dredge_synthesis
  ingested_at: YYYY-MM-DD
recall: [<keyword>, <keyword>, ...]                    # itd-39 — keywords that surface this page (see § 9)
---
```

A **multi-source** page carries `source.classes` — the authoritative summary set, derived from each `sources[].class` at write time (lint reads `source.classes`) — plus a `sources:` list nested under `source:` where **each entry carries its own `class`** (required, so per-citation provenance survives) alongside its `citation`/`licence`/`source_hash`/`ingested_at`:

```yaml
---
source:
  classes: [external_pdf, session_memory]
  weighting_note: "<text>"                             # required when classes mix (see § 5)
  sources:
    -
      class: external_pdf
      citation: { type: knowledge, origin: "<url|path>", author: "...", title: "...", year: 2026, ingested_at: 2026-06-09, ingested_by: "..." }
      licence: CC-BY-4.0
      source_hash: <sha256>
      ingested_at: YYYY-MM-DD
    -
      class: session_memory
      citation: { type: knowledge, origin: "<session>", author: "...", title: "...", year: 2026, ingested_at: 2026-06-09, ingested_by: "..." }
      licence: unknown
      source_hash: <sha256>
      ingested_at: YYYY-MM-DD
recall: [<keyword>, ...]
---
```

`source.weighting_note` nests under `source:`, is required exactly when a page mixes ≥ 2 source classes, and never appears on a single-source page.

`recall` is the retrieval key (per itd-39): the prompt-router hook keyword-matches incoming prompts against it. `principle-distiller` populates `recall` at curation time from the page's domain and salient terms; absent or empty `recall` means the page is reachable only via `/abcd:memory ask`, never auto-injected.

**Curator rules** (enforced by `principle-distiller` + lint):

- Pages with single-source-class synthesis are flagged as **low cross-validation** (advisory, lint code `MS001`; not blocking). Surfaces in `/abcd:memory lint` (spc-39).
- Pages mixing source classes (e.g., `session_memory` + `external_pdf`) require a `weighting_note` field acknowledging the asymmetric trust gradient between class types. Lint blocks (lint code `MS002`) until the note is added.
- Pages with `class: external_*` without a `licence` field block at lint (lint code `ML001`). `unknown` is acceptable — must be explicit.
- Stored text carrying a secret or identity span the write-time store redactor would refuse or rewrite blocks at lint (lint code `MR001`) — pages, `.sources_index.json` and text kept-originals alike. Page names — each page file's and each registry back-link's — are judged by the write side's own filename verdict (`filenameHardFailKinds`: the name, its type/domain/slug components and every underscore suffix, at the hard-fail bar alone), because `_` is a word character and a token embedded in a name has no boundary the free-text scan can anchor on. A back-link that is a well-formed page name is kept out of the registry's free-text scan, as the write side keeps it out of the leaf walk, so a prose-shaped name the hostname heuristic happens to match is not reported as network residue. Lint names the kind and the line, never the span, and never rewrites the store; the operator repairs.

## 4. Default-no-originals + `--keep-original`

`/abcd:memory ingest <path>` reads the source, distils into typed entity/topic pages with citation frontmatter, and **discards the original by default**. Source path + hash recorded for re-ingest only.

`--keep-original` flag opts the user into storing the original at `.abcd/memory/sources/<sha256>.<ext>`. The spc-38 restrictive-licence gate refuses to publish anything under `.abcd/memory/sources/` unless an explicit allowlist entry is written. Per adr-28 this gate is the **lifeboat's** (`/abcd:disembark`), NOT the launch payload's: `04-launch.md § 2` excludes the entire `.abcd/` namespace from the public launch payload wholesale, so launch never publishes `.abcd/memory/sources/` and is not the gate's consumer. At launch the gate is future/inert; its real consumer is the lifeboat, the surface that publishes curated project memory/provenance (adr-35).

**Why default-no-originals.** External sources may be paywalled, NDA'd, or under restrictive licences. Storing the original creates a copyright-laundering vector unless paired with explicit per-source provenance + licence-tagging. Default-no-originals + bounded quotation (§ 5) closes the laundering vector for the common case; `--keep-original` is the explicit-opt-in for the case where the user owns the source or has clear redistribution rights.

## 5. Quotation budgets (two layers)

**Per-page bounded quotation:**
- ≤5% of source per derivative page.
- No contiguous quoted span >150 words.
- Lint warns above threshold (lint code `MQ001`).

**Cumulative source-coverage lint** (closes per-page-budget × N-pages laundering vector):

```
total_source_coverage_pct(source_hash) =
  (deduplicated quoted-span tokens across all pages citing source_hash)
  / (total tokens in source)
```

- Span-level dedup — multiple pages legitimately quoting the same passage count once.
- Thresholds: warn at 15%, fail at 25% (lint code `MQ002`).
- Configurable per project in `.abcd/memory/config.json` → `quotation_budget`.
- Runs at lint pass (not at ingest). On fail: refuses next quotation block from that source until coverage drops via summarisation rewrite of existing pages.

**Coverage state** lives at `.abcd/memory/.coverage_index.json` — a regenerable index (per § 8 lifecycle taxonomy) rebuilt by full crawl on demand. Drift between the index and source-of-truth pages IS the lint signal.

**Documented limitation (not an implementation item):** the quotation lint catches *quotation*, not *close paraphrase*. A page that paraphrases an entire source at 100% coverage in its own words is a derivative work and isn't fair-use protected. Users ingesting copyrighted material under restrictive licences should not rely solely on the lint.

## 6. The curator — `principle-distiller` extended

`principle-distiller` (Pass C agent, per [`01-agents.md`](01-agents.md)) curates `.abcd/memory/`. Per itd-36, its scope extends to:

| Input | Source | Behaviour |
|---|---|---|
| Existing memory pages | `.abcd/memory/<type>_<domain>_<slug>.md` | Read for cross-source dedup (topic-hash) |
| Vendor session memory | `internal/core/memory` adapter (Claude / opencode) | Distill into `session_memory` pages (existing behaviour, unchanged) |
| External sources (new at itd-36) | `/abcd:memory ingest` invocations | Distill into typed `external_*` pages with citation frontmatter |
| `spec_modification_grammar` (new at itd-37) | A spec's `## Modification Grammar` section at spec completion | Append per-spec page; run domain-curator pass to update `modification_grammar_<domain>.md` |
| `dredge_synthesis` (itd-25, a later phase) | `/abcd:dredge synth` runs | Write synthesised entries to memory with `source.class: dredge_synthesis` |

**The agent count stays at 15** (per itd-31 precedent: agent count grows by user-facing responsibility, not by audit subtype). `principle-distiller` gains role responsibilities, not agent peers. Soft-cap risk acknowledged: if the third role ever burst coherence, fork the curator role into a sibling agent.

## 7. `/abcd:memory` command

User-facing surface (per itd-36):

| Verb | Behaviour |
|---|---|
| Bare `/abcd:memory` | Status + help + render of current memory state (page count by class, last-ingest timestamp, recent contradictions). Per the [bare-command-as-render discipline](../02-constraints/04-naming.md). Quotation-budget headroom renders per external source READ-ONLY from the spc-39 `.coverage_index.json`: a fingerprint-fresh index shows per-source warn/block headroom, fingerprint drift shows a "stale — run /abcd:memory lint" hint, an absent index an info line, and a malformed index/crawl failure a non-fatal "headroom unavailable" line. The bare render never rebuilds or mutates the index (the `lint` sub-verb is the sole writer). |
| `/abcd:memory ingest <path-or-https-url>` | Read external source, distil into typed pages with citation, append to log. A remote source must be https (admission and every redirect hop), the posture of `abcd update`; a credential the URL carries — basic-auth userinfo or a credential-shaped query key — is stripped before the fetched address becomes the stored origin and title, and masked in every fetch-failure message (the transport's own `*url.Error` is unwrapped to its cause, so it cannot re-print the address behind the mask). Default: do NOT store original. Flag: `--keep-original` (opt-in storage; the lifeboat licence-gate allowlist is required before a kept original could ever publish — via `/abcd:disembark`, not launch; adr-28). |
| `/abcd:memory ask <question>` | Query memory by domain + class; synthesise answer with citations; optionally file result back. |
| `/abcd:memory lint` **(spc-39)** | Full-store curator health-check: ALWAYS crawls the whole repo store (the cumulative `MQ002` needs the full corpus), runs the `MQ`/`MS`/`ML`/`MR` family (`MQ001`/`MQ002`/`MQ003` quotation budgets + coverage, `MS001`/`MS002` source-class advisories, `ML001` missing licences, `MR001` secret or identity residue in stored text), rebuilds the regenerable `.coverage_index.json`, and writes its run report to `.abcd/.work.local/logs/memory/lint-<ts>/report.{json,md}` in the gitignored local-ephemeral tier, per the iss-36 and iss-56 adjudication resolved as iss-73. Mutates NO memory-store state — the coverage index + that report are its only writes. Exit: blockers → nonzero; warn-only → exit 0 (curator advisory; the recorded divergence from the standalone grammar, see [`06-lint.md §2`](06-lint.md#2-severity-model)). **Not part of spc-39:** contradiction surfacing is RENDERED by spc-38's reconciliation into `contradictions.md` (surfaced by the bare render, not lint-coded), and orphan/stale-claim audits are DEFERRED to a later intent — `lint` runs neither. |

## 8. Cross-cutting integration

- **itd-26 `/abcd:loot`** uses the same provenance/licence subsystem as memory ingest (see [`09-provenance-substrate.md`](09-provenance-substrate.md)). the later-phase `/abcd:loot` verb sits on top of this substrate; the substrate ships alongside itd-36.
- **itd-25 `/abcd:dredge`** stays a distinct verb (per the dredge-pushback in idea-1 R4: storage vs operation; user-moments differ). Dredge synthesis output writes to `.abcd/memory/<type>_<domain>_<slug>.md` with `source.class: dredge_synthesis`; per-run report stays in the gitignored local-ephemeral tier at `.abcd/.work.local/logs/dredge/<ts>/`, the tier that replaced the retired runtime location (iss-73).
- **itd-37 modification grammar** extracts per-spec theory into `spec_modification_grammar_<spec_id>.md` (append-only per itd-36 lifecycle taxonomy) and curator-merged per-domain `modification_grammar_<domain>.md` (compounding-curated). See [`02-disembark.md`](../04-surfaces/02-disembark.md) and [`03-embark.md`](../04-surfaces/03-embark.md) for recovery-humility framing.

## 9. Retrieval — scope-aware, budget-bracketed (itd-39)

itd-36 (above) is the **storage** model. itd-39 is the **retrieval** model: how the right memory reaches an agent at the right moment without overflowing context. It builds on infrastructure abcd already has rather than introducing a parallel system.

**One engine, two payloads.** itd-3's prompt router is abcd's in-plugin adoption of [CARL][carl]'s recall engine — a `UserPromptSubmit` hook with keyword recall, signature dedup, and force-refresh-every-N. It is a sub-command of the abcd binary reached through the hook manifest, not a script file in `hooks/`. CARL's same engine drives two payloads: procedural rules *and* a memory layer. itd-3 adopted the rules payload; itd-39 adopts the memory payload. The hook is extended to scan each prompt against **both** `rules.json` domains **and** memory-page `recall:` frontmatter, and to inject matching memory pages alongside matching rules. Same scan, same dedup, same refresh discipline.

**Context brackets — the bounded-retrieval control.** § 5's quotation budgets bound *ingest*; nothing yet bounds *retrieval*. itd-39 adopts CARL's context brackets — injection adapts to remaining context window:

| Bracket | Window free | Memory injection |
|---|---|---|
| `FRESH` | ≥ 70% | matched pages injected in full |
| `MODERATE` | 40–70% | only highest-relevance matched pages, in full |
| `DEPLETED` | < 40% | matching `index.md` lines only — titles, not bodies |

This is what makes "memory never overflows" an enforced property rather than a hope: a long session degrades gracefully to titles-only instead of crowding out the work.

**Two-scope query, narrower wins.** The hook queries the `index.md` of both memory scopes (repo / user — § 0). It injects keyword-matched, bracket-filtered pages — never the union of both scopes. On a recall-keyword collision across scopes, the **narrower scope wins**: repo overrides user.

**Per-prompt, not sticky.** Consistent with itd-3, recall is independent per prompt — no cross-prompt "pin this page" mode in scope. A session-sticky mode is a later-phase candidate (see itd-39 Open Questions).

**Diagnostic (a design target; not built).** `abcd memory recall [keyword]` would show which pages a prompt or keyword surfaces and in which bracket — explainability, per the bare-command-as-render discipline. The shipped surface carries `ingest`, `ask` and `lint` and nothing else.

See [`../../intents/drafts/itd-39-scope-aware-memory-retrieval.md`](../../intents/drafts/itd-39-scope-aware-memory-retrieval.md) for the full intent, acceptance criteria, and the architectural-audit reconciliation with itd-3's global-rules rejection.

## References

- [Karpathy LLM Wiki gist (April 2026)][karpathy-llm-wiki] — pattern source for the wiki organisation
- [`research/related-work.md`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory) — full prior-art comparison
- [`04-universal-patterns.md § 8`](04-universal-patterns.md#8-artefact-lifecycle-taxonomy) — artefact-lifecycle taxonomy this file's lifecycle class derives from
- [`02-adapters.md`](02-adapters.md) — `internal/core/memory` semantic adapter (vendor-agnostic dispatcher)
- [CARL][carl] — recall-engine prior art; itd-3 adopts its rule payload, itd-39 its memory payload + context brackets
- [`itd-39`](../../intents/drafts/itd-39-scope-aware-memory-retrieval.md) — scope-aware memory retrieval intent

[karpathy-llm-wiki]: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
[carl]: https://github.com/ChristopherKahler/carl
