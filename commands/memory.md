---
name: memory
description: "Render the memory store's status: Writes nothing; refuses outside a git checkout."
argument-hint: "[<empty>] | ingest <path-or-https-url> [--keep-original] | ask <question> | lint"
block: people
---

# `/abcd:memory` — curated knowledge substrate

The per-project compounding-curated knowledge substrate at `.abcd/memory/`.
Bare invocation **performs zero writes**.

**The substrate addressed is the checkout's, from anywhere in the tree.** Every
verb here resolves the repository root before it reads or writes, so a bare
render run from a package directory reports the checkout's pages rather than
"store not present", and an ingest lands in the checkout's store rather than
laying a second one under the directory you happen to be standing in. Outside a
repository there is no substrate to address: the verb exits **2**, reads
nothing and writes nothing, because pages filed outside every checkout are
committed by nothing and read by nothing. If a memory store also exists below
the repository root, the verb names it on stderr and leaves it alone; relay that
line, because pages sitting there are read by no `ask`, no `lint`, and no reader
of the checkout's store.

## Status (bare)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory --json
```

Summarise the JSON: `pages` and `by_class` (page count per source class),
`last_ingest`, any `contradictions`, and per-source `headroom` lines. The bare
render never rebuilds or mutates the coverage index.

## Ingest a source

Distil an external source (transcript / article / URL) into typed, cited
memory pages. A remote source must be **https**: a plaintext `http://` source
is refused by name, and a redirect that leaves https is refused per hop, because
the store copies a fetched source's text — and the licence header lifted out of
it — verbatim into durable provenance. PDF is a later-phase seam: the binary
rejects a PDF source with a clear error, because no text-extraction dependency
is wired.

A credential carried by the URL never reaches the store: basic-auth userinfo
(`https://user:pass@host/doc`) and credential-shaped query keys (`token`,
`api_key`, `apikey`, `access_token`, `password`, `secret`, case-insensitive —
names that carry a secret in essentially every usage, so an addressing
parameter such as `?key=` is never truncated) are stripped before the fetched address becomes the stored
origin and title, and masked in every fetch-failure message — including the
transport's own error, which is unwrapped to its cause so it cannot re-print
the address behind the mask. The rest of the address is reproduced unchanged.

**You** are the distiller: read the source, produce the
`DistilledPage` JSON array, and pass it to the binary via `--pages-json`
(a file, or `-` for stdin). The binary computes the provenance, licence, and
content hash, validates every page, and writes atomically.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory ingest <path-or-https-url> --pages-json distilled.json --json
```

Add `--keep-original` to retain the source at
`.abcd/memory/sources/<sha256>.<ext>` (the lifeboat licence gate — not launch —
governs its export). Report `status`, `licence`, and the written `pages`. An
already-known source re-ingests from the registry with no `--pages-json`.

## Ask memory

Deterministic retrieval over the store, then a cited answer:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory ask "<question>" --json
```

The default answer is the deterministic citation-renderer over the top-ranked
pages; every citation references `source_class`, `citation`, and `source_hash`.
`--top-n` sets the retrieval depth; `0` uses the pinned default. Optionally
file the answer back as a new page with `--file-back --page-json <file|->` (one
`DistilledPage` object you produce from the retrieved matches).
Report the `answer` and, if present, the `file_back` result.

## Lint

Full-store curator health-check — per-page quotation budgets, cumulative source
coverage, source-class and licence advisories, and secret or identity residue
in stored text (`MR001`, a blocker: the store's write-time redactor run over
every page, the sources registry and each text kept-original, and every page
name — each page file's and each registry back-link's — judged as the write
side judges a filename, for a hard-fail secret alone; the finding names the kind
and the line, never the span, and lint never rewrites the store):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory lint --json
```

It rebuilds the regenerable `.coverage_index.json` and writes a report under
`.abcd/.work.local/logs/memory/lint-<ts>/`. Summarise `summary.blockers` /
`summary.warnings` / `summary.infos` and each finding's `code` and `message`.
Blockers exit nonzero; warn-only exits 0.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
