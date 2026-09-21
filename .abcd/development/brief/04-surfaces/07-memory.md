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
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `ask` | — | shipped |
| `ingest` | — | shipped |
| `lint` | lint | shipped |


**The store every verb here addresses is the checkout's, resolved from the working directory and never taken to be it.** The front door asks [`gitutil.CheckoutRoot`](../../../../internal/gitutil/repo.go) before it builds a request — the same resolution `capture`, `decide` and `spec` address their stores through: git's toplevel where git will name one, and a refusal in the two remaining states rather than a guess (a repo-shaped tree git will not answer for, and no repository above at all). So a render from a package directory reports the checkout's pages instead of `store not present`, an ingest lands in the checkout's substrate instead of laying a second one beneath the caller, and outside a checkout every verb exits **2** having read nothing and written nothing. It deliberately does not fall through to a marker walk, which would accept any directory carrying the name (iss-2609090947359464). A memory store found **below** the checkout root, on the chain between the caller and it, is named on stderr and left untouched — the deposit an unresolved front door leaves behind, reported to the person standing over it rather than stepped over in silence.

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
`MQ002`, `MQ003`); source-class findings raised page by page, one where a page
rests on a single class of source and one where a page mixes classes without
saying how it weighs them against each other (`MS001`, `MS002`); a missing
licence on an external source (`ML001`); and secret or identity residue in
stored text (`MR001`). `MR001` is the read side of the write-time redactor, run
over every page, the source registry and each stored original: it names the kind
and the line, never the span, and the lint never rewrites the store.

Four of the seven can stop the run. `MR001` is the sharpest: residue in the
store is a fault, never advice. `ML001` and `MS002` join it, because a source
with no licence and a page that silently blends trust levels are both defects in
what the store claims rather than suggestions about how to curate it. `MQ002`
blocks only in its strict form, when one source's quoted coverage passes the
block threshold on unambiguous single-source attribution alone; the same
coverage reached through passages attributable to several sources is capped at a
warning, because that arithmetic cannot prove any one source was over-quoted.
Everything else, `MS001` included, is advisory and leaves the exit code at zero.

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

- [`05-internals/07-memory.md`](../05-internals/07-memory.md): substrate spec
- [`05-internals/09-provenance-substrate.md`](../05-internals/09-provenance-substrate.md): provenance and licence subsystem
- [`../../intents/shipped/itd-36-memory-unification.md`](../../intents/shipped/itd-36-memory-unification.md): the full intent spec with acceptance criteria
- [`research/related-work.md § Karpathy LLM Wiki`](../../research/related-work.md#karpathy-llm-wiki--pattern-source-for-abcdmemory): pattern source
