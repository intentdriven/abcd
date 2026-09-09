# `/abcd:memory` — Curated Knowledge Substrate

Keep what a project learns from the things it reads, in a form that is still
usable a year later: distilled pages rather than stored documents, each carrying
the citation, licence and content hash it came from, so an answer drawn out of
the store can always be traced back to a source and re-checked. The store lives
in the repo at `.abcd/memory/`, so it travels with the project and is reviewable
in a diff.

The cost is the discipline: sources are distilled and discarded by default
rather than hoarded, the quotation budget is a curation rule rather than a
storage limit, and licence and provenance are recorded at ingest instead of
reconstructed later.

The **substrate spec** (page-class enum, source-class taxonomy, curator
behaviour, lifecycle class) is
[`05-internals/07-memory.md`](../05-internals/07-memory.md). This file is the
surface contract: what the user types and what happens.

> **Provenance, and a warning about ids.** The surface traces to itd-36, which
> sits in `intents/planned/`. The write core and the lint family were specified
> in a predecessor store whose `spc-38` and `spc-39` collide with live ids in
> this repo's own store. So on this page the store an id belongs to is read from
> the sentence around it, never from the number.

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


**Bare `/abcd:memory`** renders the store's state and nothing else: how many
pages there are by class, when the last ingest happened, the recent
contradictions, and per-source quotation-budget headroom. It never mutates and
never rebuilds an index. The JSON render carries one element the text render
drops, a `drift` list saying that the catalogue or the contradictions register
no longer hash-matches what the store's pages would render, so a reader knows
the numbers are stale rather than wrong. Headroom is read-only in the same
spirit: a fresh index shows per-source warn and block headroom, a drifted one
says to run the lint, and an absent or unreadable one says the headroom is
unavailable rather than guessing at it.

**`/abcd:memory ingest <path-or-https-url>`** registers an external source as
typed pages with citation frontmatter, and appends to the ingest log. The host
agent is the distiller: it reads the source, produces the distilled pages, and
passes them through `--pages-json`, which is load-bearing and required for a
source the store has not seen (an already-known source re-ingests from the
registry without it). The binary computes provenance, licence and content hash,
validates every page, and writes atomically.

Three refusals matter, because the store copies fetched text and its licence
verbatim into a durable artefact. A plaintext `http://` source is refused by
name, and a redirect that leaves https is refused per hop. A credential the URL
carries (basic-auth userinfo, or a query key such as `token` or `api_key`) is
stripped before the fetched address becomes the stored origin, and masked in
every fetch-failure message. And PDF is a later-phase seam: the binary rejects
PDF sources with a clear error until a text-extraction dependency is wired.

**The original is not stored by default.** `--keep-original` opts into keeping
it under `.abcd/memory/sources/`. The licence gate that would police publishing
such a file belongs to the lifeboat, not to launch (adr-28): launch excludes
`.abcd/**` wholesale, so it never publishes the files the gate checks. That gate
is a later phase, and the shipped ingest classifies restrictive licences without
ever refusing on them.

**`/abcd:memory ask <question>`** synthesises an answer with per-citation
provenance, every citation naming its source class, citation and content hash.
`--file-back` with `--page-json` files the host-produced answer back as a new
memory page; `--top-n` sets retrieval depth.

**`/abcd:memory lint`** is the curator health-check over the whole store. It
always crawls the full store, rebuilds the regenerable coverage index, and
writes its findings to a run log under `.abcd/.work.local/logs/memory/`. It
mutates no memory-store state, and its exit code is the decision: blockers exit
non-zero, warnings alone exit 0, because most of what it reports is curator
advice rather than a fault.

Seven codes ship, in four families: per-page and cumulative quotation budgets
with a diagnostic for the case where coverage cannot be computed (`MQ001`,
`MQ002`, `MQ003`); source-class advisories for a single-class store and for a
cross-class store with no weighting note (`MS001`, `MS002`); a missing licence
on an external source (`ML001`); and secret or identity residue in stored text
(`MR001`, the one blocker). `MR001` is the read side of the write-time
redactor, run over every page, the source registry and each stored original: it
names the kind and the line, never the span, and the lint never rewrites the
store.

The quotation budget is applied as curation discipline at distil time and
computed at lint time. Nothing enforces it at ingest.

## What ships, and what does not

The write core (bare, `ingest`, `ask`) and the lint family are on the binary.
Contradictions are rendered by the write core's own reconciliation and surfaced
by the bare render; orphan and stale-claim audits are deferred.

Not built: a curator role on the `principle-distiller` agent (the agent and its
`disembark principles` verb ship; this role does not), ingest and ask run
reports, user-scope memory outside the repo, and the automatic extraction of a
spec's modification grammar into the store at spec completion.

## Composition with adjacent surfaces

- **`/abcd:disembark`** is designed to export curated project memory and
  provenance into the lifeboat. What the lifeboat is specified to carry is the
  curated provenance surface named in
  [`02-disembark.md`](02-disembark.md), not a verbatim copy of the store;
  declaring an exact payload is deferred to the disembark spec that wires the
  packer (adr-28). The recovery-humility framing applies: the lifeboat is the
  floor of recoverable theory, not the theory.
- **`/abcd:embark`** carrying the store forward is designed behaviour. The
  source-class enum carries forward, and the receiver runs `memory lint` after
  unpacking to check that quotation budgets and licences have not drifted.
- **`/abcd:launch`** does not consume the licence gate at all: the public
  payload excludes `.abcd/**` wholesale as policy, so `launch --dry-run`
  surfaces no licence verdict (see
  [`04-launch.md § 2`](04-launch.md#2-curated-release-artefact-default-deny)).

**Tight coupling**, logged as a principal risk: itd-36 is structurally
non-decomposable, because the sub-verbs need the schema, the schema needs the
curator role, and the curator role needs the lint codes. A partial ship is not
meaningful.

## References

- [`05-internals/07-memory.md`](../05-internals/07-memory.md) — substrate spec
- [`05-internals/09-provenance-substrate.md`](../05-internals/09-provenance-substrate.md) — provenance and licence subsystem
- [`../../intents/planned/itd-36-memory-unification.md`](../../intents/planned/itd-36-memory-unification.md) — full intent spec with acceptance criteria
- [`research/related-work.md § Karpathy LLM Wiki`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory) — pattern source
