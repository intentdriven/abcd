# `/abcd:decide` — File a Decision Record

`/abcd:decide` mints an architecture decision record: it allocates the id,
derives the slug and the date, and writes the store's skeleton into
`.abcd/development/decisions/adrs/`. It is the ADR store's **write** verb, the
counterpart of the read-only `abcd adr-N` dispatch the
[`08-abcd.md`](08-abcd.md) chapter describes.

## Behaviour

```bash
abcd decide "<title>" --json
```

emits `{ "id": "adr-<stamp>", "slug": "<kebab-case>", "title": "<title>",
"date": "YYYY-MM-DD", "path": ".abcd/development/decisions/adrs/<stamp>-<slug>.md" }`
and writes exactly that one file. The plain render names the same four values
and the status the record lands with. Exit 0 when the record lands, exit 2 for
an operand fault — no title, or a title with nothing slug-able in it — with
nothing written.

**The path is relative to the working directory, not to the repository root.**
The verb passes the process's current directory straight to the core, which
joins the store's relative directory onto it and creates the tree; it never
resolves a repository root, and it never requires one. Run from a subdirectory
it mints a second store there, and run outside a repository altogether it
exits 0 and creates `.abcd/development/decisions/adrs/` wherever it stands,
with a JSON `path` that reads exactly like the repository store's. This is a
divergence from `capture`, which resolves the root and writes to
`<root>/.abcd/work/issues/open/` from anywhere in the tree, and it is a defect
rather than a design choice: neither this chapter, `abcd decide --help`, nor
`commands/decide.md` names the condition. Until it is closed, invoke the verb
from the repository root.

The title is one quoted operand. It reaches the committed filename by way of the
derived slug, so it passes the canonical scanner before anything is derived from
it, exactly as the intent store's quoted-text create does.

## The id is minted, never counted

The id is `adr-<yymmddHHMMSS><rrrr>` — the twelve-digit UTC second stamp and the
four-digit uniform suffix every other minting family draws through
`recordid.Minter` ([adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md),
mechanics per spc-33). The mint reads **no maximum**: not the store's, not the
citations'. That is the property the family was moved for. A hand-numbered
ordinal is allocated by reading the directory, so two branches deciding on the
same day allocate the same number — which is what happened when `0055` and
`0056` were each minted twice, and it is the collision the ruling of 2026-09-01
(`.abcd/work/DECISIONS.md`, that date) closed by construction.

The filename is the stamp followed by the slug, so a directory listing is in
decision order and the two vintages do not interleave: every ordinal is shorter
and smaller than every stamp, so the hand-numbered records sort first in both
the lexical listing and the numeric index order.

**`0001`–`0058` keep their ids and their filenames.** Nothing is renumbered, and
every reader of an ADR id admits both vintages: dispatch on `adr-45` and on a
stamped id both resolve, and the record gates pass over freshly minted stamped
skeletons. What they do not share is one derivation. `recordid.CanonADRID` (for
a cited id) and `recordid.ADRFileID` (for a filename) are the canonical pair,
and only the citation resolver, the `abcd <record-id>` dispatch and `decide`
itself call them. Three readers carry their own: the `record_schema` gate and
the context-citation-currency gate share a locally defined handle regex and
ADR-filename regex inside `internal/core/lint` (taking `recordid.FilenameNumRe`
for itd, spc and iss but not for adr); the site's decisions index derives
handles in `internal/core/site`, a package that does not import
`core/recordid` at all; and the lifeboat re-implements the pair as
`gvCanonADRID` / `gvADRIDFromFilename`, whose own comment records that it
agrees with the other two by inspection. The lifeboat's native ADR source adds
a fourth local filter for numbered filenames. Four parallel derivations agreeing
by inspection is the shape a divergence hides in, and consolidating them is
open work rather than a claim this chapter can make.

## What the verb does not do

It writes an **empty** record. The four sections the store README specifies —
Context, Decision, Alternatives Considered, Consequences — arrive as the
questions they answer, and the record lands with `status: proposed`, because a
freshly minted file is a draft whose decision is not yet locked and only its
author can say otherwise. The author sets `accepted` in the change that states
the decision in force.

It does not maintain the store README's index table, and it does not link the
record to anything: a supersession is a declared pair the `record_schema` gate
checks in both directions, and declaring one is an act of judgement.

## Where this sits

- The store, its admission tests, its lifecycle and its index:
  [`decisions/adrs/README.md`](../../decisions/adrs/README.md).
- The record-id scheme the mint belongs to: invariant 11 in
  [`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md).
- The plugin surface: `commands/decide.md`.
