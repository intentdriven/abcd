# `/abcd:scribe` — The Ledger Scribe's Context and Ingest

The scribe ([`agents/scribe.md`](../../../../agents/scribe.md)) transcribes a
reading run's records and the researcher's dispositions into the ledger's
declared shapes, and authors nothing. Its access rule is the reading
assembler's exact inverse: a reading receives a positively included slice of the
shipped repository and no ledger; the scribe receives ledger content and no
shipped tree. `/abcd:scribe` is the verb that holds that rule by construction,
on the idiom [`/abcd:reading`](23-reading.md) holds the read block by: an
assembler with an allow list, a manifest that makes the exclusion checkable, and
an ingest that validates the returned output before anything is written
([adr-2609021016275803](../../decisions/adrs/2609021016275803-no-session-holds-both-a-reading-and-the-ledger-and-a-per-run.md);
itd-2609020625402599).

It is a verb of its own and never a sub-verb of `/abcd:reading`, because the two
contexts must never share a front door.

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
| `assemble` | — | shipped |
| `ingest` | — | shipped |

## Building the context

Assembly takes one ingested reading run and the researcher's dispositions text.
The run must carry its commit marker: the scribe transcribes dispositions
against records the ledger already holds, so the run's reading records come from
the store, never from a raw reading output handed over a second time.

The context is positive inclusion at directory grain. Its allow list is derived
from the issue ledger's own directory list — the reading records, dispositions,
admissions, surprises and reframes, and the three status directories — so a
record family the ledger declares later is inside the scribe's world, and
outside every reading's, by the same declaration. The collector walks those
directories and nothing else and refuses a symlink inside them; an allow-list
assertion then refuses any item whose path lies outside the list, whatever route
it arrived by, so an item that reached the context by a future route is a
refusal rather than a disclosure. The shipped tree, the brief, the intents, the
specs, the decisions, the local tier and the session-transcript store are
excluded because no walk starts in them. The supplied text is carried verbatim,
with the caller's home directory and the repository root taken out of it.

The context and its manifest are parked in the local tier, or in an
operator-named directory that must be empty and may not be one a reading's
include table reaches: a context parked there would hand the next reading the
ledger. The manifest names every ledger path passed with its length and hash,
the hash of the scribe definition, the hash of the context and of the supplied
text, the allow list, and the exclusions with the signal behind each. Nothing in
the durable tier is touched, so a session assembled and never ingested leaves
no trace beside the run.

Both artefacts carry the scribe's per-run context stamp, which names the scribe
kind, the run and a digest of the ledger the context holds. A session handed the
context carries the stamp in its retained transcript, and the history store's
separation check ([`11-history.md`](11-history.md)) reports any transcript
carrying the reading stamp and the scribe stamp of one run.

## Ingesting what the scribe returned

The scribe returns one JSON document naming its run and the context it was
handed, with four outputs: the records to file (dispositions, admissions and
surprises), fidelity flags, outstanding items, and refusals. Nothing is written
until all of the following hold:

- **The context is proven.** The context on disk hashes to its parked manifest,
  and the payload cites that hash.
- **The supplied text is the researcher's.** The context and the manifest are
  parked in the local tier, where a scribe session granted tools could rewrite
  both and recompute every hash that binds them, so their agreement is no
  witness to what the researcher wrote. The ingest therefore takes the
  researcher's dispositions file again, the one assemble was handed and the
  session never was, scrubs it as assemble did, and requires the manifest's
  supplied hash and the context's supplied copy both to equal it. Every check
  below reads that text.
- **Nothing is authored.** The payload is decoded against closed shapes at every
  level, so a key the scribe may not author is refused by name with the entry it
  sat on. A disposition or an admission for an item the supplied text never names
  is one the researcher did not supply. A disposition's state must stand as a
  whole word, in any case, on a line of the supplied text that names its item,
  and an admission, which writes an acceptance, needs that line to admit or
  accept the proposal: the state is the ruling, and a state another item's line
  carries is not the researcher's answer to this one. A ground, an exit
  condition or a surprise that does not stand verbatim in the supplied text,
  once whitespace is folded, is one the researcher did not write. The scribe
  reformats; the check is that every word it carries was already there.
- **Every answer is the run's, once.** An answered or outstanding item must be one
  of the run's items, and one item takes one answer.
- **Nothing is passed over in silence.** Every item of the run with no standing
  disposition is answered, listed as outstanding, or named in a refusal. An item
  already answered in the ledger is not owed again, which is what lets a rerun
  after a partial ingest drop what landed.
- **The run has no promoted scribe manifest.** The durable tier is write-once, so
  the refusal comes before any write rather than after the records land.

One scribe session per run lands records: this is a departure from the spec,
which assumes a rerun re-proves the same context and says nothing of a second
session. The promoted manifest is write-once beside the run, so once an ingest
has landed a record and promoted it, a later answer to that run is written with
the capture verbs directly, not through the scribe. An ingest that lands no
record — every item outstanding, or refused — promotes nothing, so it cannot
lock the run: its manifest stays parked, and a later session over the run is
assembled once the parked directory is cleared, or into an operator-named
directory.

The records are then written in payload order — dispositions, admissions,
surprises — through the capture verbs' own functions, each under the ledger lock
it takes for itself and each with the redaction and refusals it already applies.
The verb adds no validation path of its own beyond the authoring refusal. Two
inherited refusals meet a scribe payload whole: the ordering gate refuses any
disposition or admission at the widening position until a committed comparative
run names the widening run, and the admission writer's one-ground rule refuses
an admission whose ground differs from the standing acceptance's. The first
refusal from any write stops the ingest and names what landed before it.

Fidelity flags and refusals are carried into the result unresolved and never
into a record. Once every write has landed, and when at least one record did,
the manifest is promoted beside the run through the reading store's one
durable-tier writer, write-once. That
directory is denied to every assembly by the exclusion floor, so the next
reading cannot see it.

Every refusal of either sub-verb exits 2; a refusal after something landed
renders what landed first.

## Disclosed limits

- The context is assembled from the ledger as it stands on disk. A scribe that
  needs ledger content the working tree does not hold is outside this scope.
- The verb cannot enforce what a host hands a session. The plugin surface states
  the obligation, and the separation check can only see what a host retained:
  where a host assembles context before anything is retained, the check reports
  the property unobserved and the scribe definition's protocol remains the gate.
- The researcher's dispositions file is the ingest's one witness, and it is a
  file the operator names. A scribe session that learns its path and rewrites
  it before the ingest is outside what the verb can see; the host obligation
  covers it, since the session is handed the context and nothing else.
- Promotion follows what this ingest landed, not what the session landed. A
  rerun after a partial ingest that drops everything that landed, leaving only
  outstanding items, promotes nothing, so the records the first attempt wrote
  have no promoted manifest beside the run; the parked one still names the
  context they came from.
- The state check reads words, not sense. A line that names a state only to
  negate it ("rdi-N: not accepted") still carries it, and a line naming two
  states carries both, so the verb refuses a state the item's line does not
  carry and cannot tell which of two it carries the researcher meant. The
  definition holds the scribe to the ruling the line gives.

## References

- The definition: [`agents/scribe.md`](../../../../agents/scribe.md), and its
  protocol in [`05-internals/01-agents.md`](../05-internals/01-agents.md).
- The decision: [adr-2609021016275803](../../decisions/adrs/2609021016275803-no-session-holds-both-a-reading-and-the-ledger-and-a-per-run.md).
- The plugin surface: `commands/scribe.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd scribe`

Sub-verbs: `abcd scribe assemble`, `abcd scribe ingest`.

Flags: none.

### `abcd scribe assemble`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--dispositions` | string |
| `--dry-run` | bool |
| `--out` | string |
| `--run` | string |

### `abcd scribe ingest`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--context` | string |
| `--dispositions` | string |
| `--scribe-json` | string |

<!-- surface-appendix:end -->
