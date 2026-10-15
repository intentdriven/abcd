---
name: decide
description: "Record a decision, list them bare, or mint an ADR with --adr: Writes one decision record or one proposed ADR; refuses a text with nothing slug-able."
argument-hint: "[\"<the decision>\" | --adr \"<title>\"]"
block: people
---

# `/abcd:decide` — record a decision, list them, or file an ADR

One verb, three forms:

- `abcd decide "<text>"` **records a decision**: one new file in
  `.abcd/work/decisions/`, the repository's decision log.
- `abcd decide`, with no operand, **lists every decision recorded**, oldest
  first.
- `abcd decide --adr "<title>"` **mints an architecture decision record**: the
  id, the date, the filename, and the four empty sections the ADR store
  specifies.

Most decisions are the first kind: a dated line saying what was settled and
why, the way the old single log held them. A decision that is hard to reverse,
surprising without context, and the result of a real trade-off earns an ADR as
well.

**Check the binary first.** A binary from abcd 0.14.x or earlier knows only the
ADR form, and spells it without `--adr`: given `decide "<text>"` it mints an ADR
from the text instead of recording a decision, and it refuses `--adr` as an
unknown flag. Run `"${CLAUDE_PLUGIN_ROOT}/abcd" update --check --json`, relay
the installed `version` and, where one is offered, `check.next_step` verbatim,
since it names the command that takes the update for this install's shape. Do
not record the decision by hand in the meantime, and do not run the bare form on
an older binary hoping it lists: it refuses.

## Record a decision

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" decide "<the decision, in one line>" --json
```

The text is one quoted operand, the decision in the words it should be read in.
The verb writes `.abcd/work/decisions/dcn-<id>-<slug>.md`, whose body is
`<date> — <text>`, and nothing else: nothing is appended anywhere, so two
branches that each record a decision merge with nothing to reconcile. The id is
minted through the same seam as every other record, the date comes from it, and
the slug is cut from the text, so a text with no letters or digits in it is
refused. The text passes the secret scanner before anything is written.

Report the `id` and the `path` from the JSON. A record is never edited: a
correction is a new decision that says what it corrects.

It needs the shared tier `.abcd/work/`. A repository without one is refused,
naming `ahoy install`, which lays it.

## List the decisions

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" decide
```

Every decision recorded, oldest first, read from the folder each time: the id,
then the decision. `--json` returns `[{id, date, slug, path, text}]`. A
repository that kept a single `.abcd/work/DECISIONS.md` before the folder still
has it as frozen history: the listing ends with one line naming it, and does not
list its entries. A file in the folder that cannot be read as a record is named
on stderr; relay it. The listing writes nothing.

## Mint an ADR

Check the three admission tests in the store's own README
(`.abcd/development/decisions/adrs/README.md`): hard to reverse, surprising
without context, the result of a real trade-off. If any one is missing, skip the
ADR — record a decision instead, keep file-scoped rationale inline, and file
forward-looking discussion as an RFC and user-facing capability as an intent.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" decide --adr "<title>" --json
```

The title is one quoted operand — a short noun phrase stating the decision in a
line. The verb derives the record's slug from it, so keep it in plain words.

**The binary states nothing.** It owns the parts a human does badly across
branches — allocating an id that cannot collide, and putting the file in the
right place with the right shape. The Context, the Decision, the Alternatives
Considered and the Consequences are yours to write, and the record lands with
`status: proposed` until you set `accepted`. Report the `id` and the `path` from
the JSON, then open the record and write the four sections with the user.

The id is `adr-<yymmddHHMMSS><rrrr>`: a UTC second stamp and four random digits,
drawn through the same seam captures, intents and specs mint through. It reads no
maximum anywhere, which is the point — two branches deciding on the same day
cannot allocate the same number, the collision a hand-numbered ordinal has by
construction. The filename is that stamp followed by the slug, so a directory
listing is in decision order. The hand-numbered records `0001`–`0058` keep their
ids and their filenames, and both vintages resolve everywhere a decision is
cited — `abcd adr-58` and `abcd adr-2609021016286571` both answer.

After the mint:

- Write the four sections, then set `status: accepted` in the same change that
  states the decision in force.
- Add the row to the index table in the store README — the index is a hand
  maintained enumeration, and a decision missing from it is a decision a reader
  does not find.
- Link it both ways: a supersession declares `supersedes` on the successor and
  `superseded_by` on the predecessor, and the record gate refuses a one-way pair.

## From anywhere in the tree

Every form resolves the repository root first, so the reported `path` is
relative to that root and not to the directory you happen to be standing in.
Outside a repository there is nothing to write into or list, and the verb
refuses rather than laying a store where it stands — a record filed outside
every checkout is committed by nothing and read by nothing. If a decisions
folder or an ADR store also exists below the repository root, the verb names it
on stderr and leaves it alone; relay that line, because records sitting there
reach no gate and no release cut.

Exit codes:

- **0** — recorded, listed or minted. Report the `id`, the `date` and the
  `path`; for an ADR, show the user the written skeleton.
- **1** — the decisions folder could not be read. Relay the diagnostic.
- **2** — refused, and **nothing was written**. Relay the diagnostic. The faults
  are a text or title with no slug-able characters in it, `--adr` with no title,
  more than one operand, a repository without the shared tier (for a recording),
  and a working directory with no repository above it, or one whose repository
  git will not answer for.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
