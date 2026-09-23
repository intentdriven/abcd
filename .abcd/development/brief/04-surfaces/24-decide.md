# `/abcd:decide` — File a Decision Record

`/abcd:decide` mints an architecture decision record: it allocates the id,
derives the slug and the date, and writes the store's skeleton into
`.abcd/development/decisions/adrs/`. It is the ADR store's **minting** verb, the
counterpart of the read-only `abcd adr-N` dispatch the
[`08-abcd.md`](08-abcd.md) chapter describes. It is not the store's only writer:
[`/abcd:embark`](03-embark.md) unpacks a lifeboat's decision records into the same
store, which is a restore rather than a mint.

## Behaviour

Given the quoted title as its one operand, the JSON form emits
`{ "id": "adr-<stamp>", "slug": "<kebab-case>", "title": "<title>",
"date": "YYYY-MM-DD", "path": ".abcd/development/decisions/adrs/<stamp>-<slug>.md" }`
and writes that one file, laying the store's directories down first where the
checkout does not already hold them. Nothing else lands on disk. The plain render
names the same four values and the status the record lands with. Exit 0 when the record lands, exit 2 for
an operand fault — no title, or a title with nothing slug-able in it — with
nothing written.

**The path is relative to the repository root, from anywhere in the tree.** The
front door resolves the checkout root before it builds the request, so a mint
from a package directory lands in the checkout's own store and no second store
appears beneath the caller. The resolution is
[`gitutil.CheckoutRoot`](../../../../internal/gitutil/repo.go), the same one
`capture` addresses its ledger through — git's toplevel where git will name one,
and a refusal in the two remaining states rather than a guess: a repo-shaped tree
git will not answer for, and no repository above at all. Outside a checkout the
verb therefore exits **2** and writes nothing, because there is no decision store
to address and laying one where the caller stood is what a lost record looks like
one directory further out. It deliberately does not fall through to a marker
walk, which would accept any directory carrying the name (iss-2609090947359464).

A decision store found **below** the checkout root, on the chain between the
caller and it, is named on stderr and left untouched — the deposit an
unresolved front door leaves behind, reported to the person standing over it
rather than stepped over in silence. The note states what is there; moving a
record is a judgement no verb makes.

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

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd decide`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
