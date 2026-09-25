---
name: decide
description: Mint a decision record (ADR) — allocate its id through the shared record-id seam and lay the store's skeleton, ready for the author to write the decision into.
argument-hint: "<the decision, as a short noun phrase>"
block: people
---

# `/abcd:decide` — file a decision record

A settled decision that is hard to reverse, surprising without context, and the
result of a real trade-off earns an ADR. This mints one: the id, the date, the
filename, and the four sections the store specifies.

**The binary states nothing.** It owns the parts a human does badly across
branches — allocating an id that cannot collide, and putting the file in the
right place with the right shape. The Context, the Decision, the Alternatives
Considered and the Consequences are yours to write, and the record lands with
`status: proposed` until you set `accepted`.

## Before you mint

Check the three admission tests in the store's own README
(`.abcd/development/decisions/adrs/README.md`): hard to reverse, surprising
without context, the result of a real trade-off. If any one is missing, skip the
ADR — file-scoped rationale belongs inline, forward-looking discussion is an RFC,
and user-facing capability is an intent.

## Mint it

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" decide "<title>" --json
```

The title is one quoted operand — a short noun phrase stating the decision in a
line. The verb derives the record's slug from it, so keep it in plain words.

Report the `id` and the `path` from the JSON, then open the record and write the
four sections with the user. Nothing else about the decision is the binary's to
supply.

**The record lands in the checkout's store, from anywhere in the tree.** The verb
resolves the repository root before it mints, so the reported `path` is relative
to that root and not to the directory you happen to be standing in. Outside a
repository there is no decision store to write into, and the verb refuses rather
than laying one where it stands — a decision filed outside every checkout is
committed by nothing and read by nothing. If a decision store also exists below
the repository root, the verb names it on stderr and leaves it alone; relay that
line, because records sitting there reach no gate and no release cut.

## What the id looks like

The id is `adr-<yymmddHHMMSS><rrrr>`: a UTC second stamp and four random digits,
drawn through the same seam captures, intents and specs mint through. It reads no
maximum anywhere, which is the point — two branches deciding on the same day
cannot allocate the same number, the collision a hand-numbered ordinal has by
construction. The filename is that stamp followed by the slug, so a directory
listing is in decision order.

The hand-numbered records `0001`–`0058` keep their ids and their filenames.
Nothing is renumbered, and both vintages resolve everywhere a decision is cited —
`abcd adr-58` and `abcd adr-2609021016286571` both answer.

Exit codes:

- **0** — minted. Report the `id`, the `date` and the `path`, and show the user
  the written skeleton.
- **2** — refused, and **nothing was written**. Relay the diagnostic. A missing
  title and a title with no slug-able characters in it are the two operand
  faults; the third refusal is a working directory with no repository above it,
  or one whose repository git will not answer for, where there is no decision
  store to address at all.

## After the mint

- Write the four sections, then set `status: accepted` in the same change that
  states the decision in force.
- Add the row to the index table in the store README — the index is a hand
  maintained enumeration, and a decision missing from it is a decision a reader
  does not find.
- Link it both ways: a supersession declares `supersedes` on the successor and
  `superseded_by` on the predecessor, and the record gate refuses a one-way pair.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
