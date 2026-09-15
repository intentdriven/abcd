# Provenance / Licence Substrate

A separable plumbing layer used by both `/abcd:memory ingest` (per itd-36) and `/abcd:loot` (a later phase, per itd-26). Owns: licence detection, citation generation, source-hash registry, the restrictive-licence publish gate (per adr-28 the gate's consumer is the lifeboat `/abcd:disembark`, future/inert at launch — see § 4).

## Why it exists as a separate component doc

Without explicit separation, the failure mode is: memory ingest ships with a "temporary" licence layer that itd-26's later verb has to retrofit. The substrate spec is **separable** — designed before either consumer ships, with both consumers calling into the spec'd surface. itd-26 stays in a later phase (verb is the user moment); the substrate is pulled forward alongside itd-36 (because memory ingest needs it).

Two consumers, distinct verbs, one substrate (per the surface-vs-substrate principle established in idea-1 R4):

- **`/abcd:memory ingest`** (itd-36) — knowledge upstreams.
- **`/abcd:loot`** (a later phase, itd-26) — code vendoring.

User moments differ; underlying licence-tracking machinery is the same.

## 1. Licence detection

**SPDX identifier extraction** from declared sources where present. Priority order (first match wins — canonical, implemented by `internal/core/provenance` `detect_licence`):

1. Source has SPDX header in file content (`SPDX-License-Identifier: <id>`) → extract
2. Source has package manifest declaring licence (`package.json` `license` field, `Cargo.toml` `[package] license`, `pyproject.toml` `[project] license`) → extract
3. Source has `LICENSE`/`LICENCE`/`LICENSE.md` file at the source root → parse for SPDX-ID (tag scan + exact-ID first line; no fuzzy full-text matching)
4. HTTP source has `License:` HTTP header (rare) → extract
5. None of the above → mark as `unknown` (explicit; never inferred)

Steps 2–3 run only when the detector is given a `source_root` directory (`detect_licence(text, source_root, http_headers)`). **Substrate capability vs memory-ingest behaviour:** memory ingest of a single local file or URL passes no `source_root`, so it detects only the content SPDX header + HTTP `License:` header; a consumer with a source directory (loot) enables the manifest/LICENSE steps.

A single identifier is matched case-insensitively and stored in canonical SPDX casing via a small local canonical-ID map; an ID outside the map is stored verbatim and flagged `unrecognised`. Compound expressions (`MIT OR Apache-2.0`, `... WITH <exception>`) are stored verbatim, never split; `classify_licence_expression` is the single classifier (`permissive | restrictive | unknown | unrecognised` — any restrictive token anywhere makes the expression restrictive) that the publish-gate policy layers on top of.

**`unknown` is acceptable but must be explicit.** The substrate never silently labels an unknown licence as a permissive default. Lint code `ML001` (spc-39 — the substrate writes `licence: unknown`; spc-39 lints it) blocks on missing licence field; `unknown` value passes the lint but surfaces in `/abcd:memory lint` and `/abcd:loot lint` as a flag for human review.

## 2. Citation generation

Consistent format across code-vendoring and knowledge-ingest:

```yaml
citation:
  type: <code | knowledge>
  origin: "<URL or local path>"
  author: "<author or repo owner>"
  title: "<title>"
  year: <YYYY>
  commit_sha: <sha>      # for code (loot)
  ingested_at: YYYY-MM-DD
  ingested_by: "<abcd command + flags>"
```

Citation is read-only after creation; updates require a new ingest pass (`/abcd:memory ingest --reingest <path>` or `/abcd:loot --reingest <url>`). A reingest of changed content produces a new hash → a new registry entry; a reingest of identical content is a registry hit (counters bump, see § 3) — neither overwrites the prior citation.

## 3. Source-hash registry

`.abcd/memory/.sources_index.json` — DURABLE source metadata, only PARTIALLY regenerable (ADR-13; this supersedes the blanket "regenerable" classification of [`04-universal-patterns.md § 8`](04-universal-patterns.md#8-artefact-lifecycle-taxonomy) for this file): fields about a DISCARDED original (`source_token_count`, `token_count_version`) cannot be rebuilt by crawling the retained pages, so the registry is authoritative for them; only the crawl-derivable subset (the `source_hash` set) is fingerprint-validated against the pages. The key is `sha256(normalised-source-text)` (line endings + trailing whitespace normalised before encoding).

```json
{
  "<sha256-of-normalised-source-text>": {
    "origin": "<URL or path>",
    "licence": "<SPDX-ID|expression|unknown>",
    "source_token_count": <int>,
    "token_count_version": <int>,
    "ingest_count": <int>,
    "first_ingest": "YYYY-MM-DD",
    "last_ingest": "YYYY-MM-DD",
    "consumers": {
      "memory": {
        "class": "<source.class enum value>",
        "citation": { },
        "ingested_at": "YYYY-MM-DD",
        "pages": ["<type>_<domain>_<slug>.md"]
      },
      "loot": { }
    }
  }
}
```

**Why per-content-hash, not per-URL.** Two URLs may serve the same content; same URL may serve different content over time. Hash is the stable identity. Re-ingest of the same content produces same hash → updates `last_ingest` + bumps `ingest_count`; does NOT add a duplicate entry and never drops the durable `source_token_count`/`token_count_version` fields.

**Cross-consumer sharing:** memory ingest and loot share one `sources_index.json`. Shared fields (`origin`, `licence`, `source_token_count`, `token_count_version`, counters) live at the top level of each hash entry; a `consumers` map holds one key per producing verb (`memory`, `loot`), each carrying its OWN `class`/`citation`/`ingested_at`/`pages`. A source that is loot-vendored (code) AND memory-ingested (documentation/notes about that code) yields ONE entry with BOTH keys present — memory never inherits loot's citation, and vice versa. The per-consumer `pages` list lets a re-ingest registry hit name its memory pages without a crawl. Every leaf a memory write introduces into this file (an origin, a licence, a consumer's citation) passes through the store redactor before it is written, and so does every KEY it introduces — a key carrying a secret span is refused rather than rewritten, since renaming a key renames the field the reader looks up. The one exclusion is each consumer's `pages` list: those are page filenames the store RESOLVES (an unnamed page is pruned), their charset is already bounded by the page-name grammar, and rewriting one would unlink a file that keeps its real name. `/abcd:memory lint` scans the file as stored (`MR001`).

## 4. Restrictive-licence publish gate (lifeboat consumer; future/inert at launch)

Per adr-28, the spc-38 restrictive-licence gate is **NOT** the `/abcd:launch` payload's gate. The launch payload manifest (see [`04-launch.md § 2`](../04-surfaces/04-launch.md#2-curated-release-artefact-default-deny)) excludes the entire `.abcd/` namespace — including `.abcd/memory/**` — **wholesale**, so nothing the gate evaluates is ever in the launch publish walk. The gate's real consumer is the **lifeboat** (`/abcd:disembark`), the surface that publishes curated project memory/provenance (adr-35). At launch the gate is **future/inert** against the lifeboat's provenance surface (`02-disembark.md § 5`); `/abcd:launch dry-run` renders its verdicts only as a diagnostic preview, never as enforcement over files launch excludes. The exact verbatim `.abcd/memory/` lifeboat payload (if any) is deferred to the disembark spec that wires the packer.

The gate's substrate integration (consumed by the lifeboat, not launch):

| Gate | Behaviour |
|---|---|
| **`.abcd/memory/sources/` allowlist** | Default-deny. Refuses to surface any file under this path to the gate unless `.abcd/launch-allowlist.json` explicitly names it. Per § 4 of `07-memory.md`. This JSON allowlist re-includes files into the gate's *own evaluation input*, never into the launch publish payload (adr-28 — distinct from the `.abcd/launch.allow` payload-promotion override). |
| **Restrictive-licence detection in the gated payload** | If any file the gate evaluates has a citation with `licence` flagged as restrictive (GPL, AGPL, proprietary, NDA-marked), the gate refuses. The consumer (lifeboat) overrides via `--accept-licence-risk` (logged; never silent). |
| **Unknown-licence detection in the gated payload** | If any gated file has `licence: unknown`, the gate warns (does not block). Surfaces the file path + origin so the user can resolve before publish. |

**The gate consumes the registry; it does not maintain it.** Maintenance is the job of memory ingest + loot (the producers).

## 5. Acceptance criteria (Given-When-Then, per itd-1)

Lint codes referenced below (`ML*`, `MQ*`) ship with spc-39 (the write/lint split, ADR-13) — spc-38's substrate writes `licence: unknown` explicitly; spc-39 lints it.

- **Given** an external source with a declared SPDX licence visible to the detector (an in-file `SPDX-License-Identifier:` header; or a `LICENSE` file / package manifest when the ingest provides a `source_root` — see § 1), **when** the user runs `/abcd:memory ingest <path>`, **then** the resulting memory page(s) carry `source.licence: <spdx-id>` AND a registry entry exists at `.abcd/memory/.sources_index.json[<sha256>]` whose `consumers` map contains the `memory` key.
- **Given** an external source with no declared licence, **when** ingest runs, **then** the page carries `source.licence: unknown` (explicit) AND lint code `ML001` (spc-39) does NOT fire (unknown is acceptable; missing-field would be the violation).
- **Given** the same source content ingested by both `/abcd:memory ingest` (as documentation) and `/abcd:loot` (as code), **when** both have run, **then** the registry has ONE entry with `ingest_count: 2` whose `consumers` map holds BOTH the `memory` and `loot` keys, each with its own `class`/`citation`/`ingested_at`/`pages`.
- **Given** a gated payload (the lifeboat consumer's, per adr-28 — NOT the launch publish payload) includes a file with `citation.licence: GPL-3.0` AND the project is being published as MIT, **when** the gate runs, **then** it refuses with a licence-mismatch error AND lists the offending file + its citation. Override via `--accept-licence-risk` logs the override. (At launch the gate is future/inert; the launch dry-run only renders these verdicts diagnostically.)
- **Given** itd-36 ships with the substrate but itd-26 hasn't shipped, **when** the substrate code is exercised, **then** all citation/licence/registry behaviour works for memory ingest AND the substrate is verified to handle the loot-future case via mock invocations in test fixtures (substrate is separable; memory ingest doesn't depend on loot existing).

## 6. Composition with adjacent surfaces

- **`internal/core/memory` adapter** ([`02-adapters.md`](02-adapters.md)) reads vendor session memory and writes session-memory-class pages. Does NOT consume the provenance substrate (session memory is internal, not external).
- **`/abcd:memory ingest`** (this substrate's primary consumer) consumes for: licence detection on every external source, citation generation on every page, registry update on every ingest.
- **`/abcd:loot`** (later-phase consumer) consumes for: licence detection on vendored repos, citation generation in `.abcd/development/loot/<source>.md`, registry update on every vendor pass.
- **`/abcd:disembark` (lifeboat)** is the gate's real consumer (adr-28): it reads the registry for restrictive-licence enforcement over the curated memory/provenance it publishes; does not write to it. At launch the gate is future/inert — `/abcd:launch` excludes `.abcd/` wholesale and is not the gate's consumer (the launch dry-run only renders the gate's verdicts diagnostically).

## 7. Ship gate

The substrate ships alongside itd-36. The `ML*`/`MQ*` lint family is NOT part of this gate — those codes ship with spc-39; the substrate only writes explicit licence values (including `unknown`) for spc-39 to lint. Acceptance is verified by:

1. Memory ingest of three real external sources (one with SPDX licence declared, one with `unknown`, one with restrictive licence) producing correct frontmatter + registry entries.
2. Gate refusal on a gated payload containing a restrictive-licence-tagged file (mock fixture) — the lifeboat consumer's gate, future/inert at launch (adr-28).
3. Mock loot invocation against a stub URL writes correctly to the same registry, demonstrating substrate separability.

## References

- [`07-memory.md`](07-memory.md) — primary consumer of this substrate
- [`02-adapters.md`](02-adapters.md) — the adapter seam model (provenance substrate is NOT a seam; it's a flat library used by ingest + loot)
- [`04-universal-patterns.md § 8`](04-universal-patterns.md#8-artefact-lifecycle-taxonomy) — source-hash registry is regenerable per the lifecycle taxonomy
- [`itd-26-loot-oss-vendor.md`](../../intents/drafts/itd-26-loot-oss-vendor.md) — later-phase consumer; verb lands in a later phase but substrate ships early
