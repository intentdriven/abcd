---
term: record
bounded_context: core
definition: One identified, filed document that a command mints and a lint gate reads — an itd-N, spc-N, adr-N, iss-N or rdg-id. "The development record" is the whole durable corpus those records make up, and "a record family" is one lifecycle-bucketed set of them; each of the three is qualified where the other two could be read.
aliases: ["record file"]
forbidden_synonyms: ["doc", "artefact", "entry", "ticket"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/brief
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# record

A **record** is one filed document with a stable id, minted by a command and readable by a
gate: an intent (`itd-N`), a spec (`spc-N`), an ADR (`adr-N`), an issue (`iss-N`), or a cold
reading (`rdg-<yymmddHHMMSS><rrrr>`). Lifecycle is directory-as-truth — a record moves between
buckets rather than carrying a status field.

## Senses

Three senses, one word. The glossary keeps the bare form for the single file and qualifies the
other two (iss-2609012245352480).

| Sense | The one spelling | Where it lives |
|---|---|---|
| One filed document with an id | **a record** | the family folders under [`development/`](../../../README.md) and [`work/issues/`](../../../../work/issues/README.md) |
| The whole durable corpus — brief, intents, specs, ADRs, principles, roadmap, plans, research, readings | **the development record** | [`.abcd/development/`](../../../README.md) |
| One lifecycle-bucketed set of records of a single kind | **a record family** | the four families `embark` writes, per [`04-surfaces/03-embark.md`](../../04-surfaces/03-embark.md) |

**A fourth use is a claim, not a place.** The cold-reading detection position asks where the
shipped tree is in tension with **the claim record** — the record read as a set of assertions
about the software, as against the tree that implements them. Written in full, it never
collides with the corpus sense.

**Where the record disagrees.** [`development/README.md`](../../../README.md) states that
"Issues are not a record folder: the ledger lives in the working tier at `../work/issues/`"
([adr-32](../../../decisions/adrs/0032-issue-ledger-is-working-tier-data.md)), while
[`04-surfaces/03-embark.md`](../../04-surfaces/03-embark.md) counts issues among "the four
record families (ADRs, issues, intents, specs)". Both hold: the glossary fixes **record family**
as a property of the records — an id, a mint command, lifecycle buckets — and **tier** as a
property of where they are filed. Issues are a record family in the working tier. What
`development/README.md` denies is a *durable-tier folder*, not the family.

## When to use

Use "a record" for one file with an id. Use "the development record" for the corpus a reader is
asked to work from. Use "a record family" when the subject is the set and its buckets — what
`embark` writes, what a lint gate walks, what a lifeboat packs.

## When NOT to use

Do not call the brief a record: the [brief](brief.md) is revised in place and holds no id or
lifecycle bucket. Do not call a working-tier note or a local scratch file a record — an
artefact with no id and no minting command is neither, and the tiering under `.abcd/` decides
where it belongs. Do not use "record" for the released binaries: the development record travels
with every repository checkout and release source archive, never inside a built artefact.

## Examples

- "A change that fixes a captured issue resolves it in the same change" — the issue is one record.
- "The development record is the durable 'what / why' the build works from."
- "`embark` writes only the four record families, verbatim, into their canonical locations."

## Related terms

- [brief](brief.md) — the living root document, not a record in the filed-and-identified sense
- [intent](intent.md), [spec](spec.md) — two of the families
- [ledger](ledger.md) — the working-tier issue ledger is a record family with a store's name
- [surface](surface.md) — the front doors that mint records
