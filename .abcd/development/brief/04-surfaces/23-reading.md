# `/abcd:reading` — The Cold-Reading Instrument Surface

A cold reading is an outside look at the design by a reader who has not been told
what the project already believes. Its value depends entirely on that blindness,
and blindness is exactly the thing a reader cannot prove about itself. Every claim
an instrument makes about what it saw would otherwise rest on a disclosure taken
on trust: the repository's tiering is organisational, not an access control, and
nothing in it stops a reading reaching ledger content.

`/abcd:reading` makes the blindness a property of the input instead. A positive
include table names what may travel, at field granularity; the assembled bundle
carries no repository path; and a hashed manifest records what was passed, so a
third party can re-run the assembly and diff the result. What a researcher gets is
a reading whose account of itself can be checked rather than believed.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `assemble` | — | shipped |
| `ingest` | gate | shipped |

Bare `abcd reading` is a third form and a **read-only status render**: the
assembler's version and schema number, the include and exclusion row counts, the
charter path, the position definitions the binary resolves, the staged runs, and
any orphaned ingest waiting to be swept. It writes nothing, and it is where an
operator reads what the instrument currently is before commissioning anything
through it.

## The invocation carries no free text

`assemble` takes a position and a target state, and nothing else. Both are closed
in shape: the position is one of four registered tokens, and the target is `HEAD`
or a hexadecimal commit sha. A branch name or a tag is refused as mutable, because
the manifest's re-runnability rests on a reference that cannot move, and a
positional argument is refused outright.

Closing the invocation at two operands is the point rather than a convenience
([adr-2609021016286571](../../decisions/adrs/2609021016286571-the-invocation-is-a-position-and-a-target-state-and-the-comm.md),
which supersedes
[adr-58](../../decisions/adrs/0058-a-reading-is-commissioned-about-something-so-the-invocation-takes-a-scope.md)).
The reading's object and its question come from its definition, so there is no
channel through which ledger content can travel in the framing of a request. No
repository path is accepted at the invocation either: a path may be named only
inside the committed preset file, where it is reviewed, shape-validated and inside
the dirty gate.

**What a run is handed is not an operand.** The assembler applies the committed
entry for the invoked position, one entry per position, and the manifest records
the entry applied and its hash. Changing what a position reads is a commit to that
file. There is no override at the invocation and nothing to stamp.

**The comparative position derives its candidate set from the record**, because
its object is a prior widening run's pre-admission output
([adr-2609021016272867](../../decisions/adrs/2609021016272867-the-comparative-reading-receives-one-widening-run-s-candidat.md)).
The assembler selects the one committed widening run at the target whose items
carry no disposition and no admission, and hands the reading that run's items
projected to two body fields. Everything else in the readings store stays excluded
there as at every other position, and the manifest asserts it family by family.
None or more than one qualifying run refuses and lists what it looked at, so the
failure is legible rather than silent. A run holding fewer than two candidates is
an interpretation fixed in advance: the position is not exercised, and the assembly
stages a run whose ingest commits that outcome as an empty comparative run naming
the widening run. This is the one positional exception to the prior-run exhaust,
and brief invariant 15 states its limit: one run, two fields, one position.

Assembly reads the working tree, so it refuses unless HEAD resolves to the target
and no included path is uncommitted. A dirty tree cannot be described by a commit
reference, and the manifest would otherwise promise a re-run it could not deliver.
Every refusal exits 2.

## Two rules bind the include table

1. **No include names a directory that contains a record family.** The deny is
   measured from a row's own source downward, so a family's leaf bucket may be
   named individually while a directory above a family may not. A family added
   later, the readings family itself included, is excluded by construction.
2. **A reading's object excludes the material whose state that reading exists to
   change.** The drafts asymmetry and the audit-notes exclusion are its two
   instances: the widening position cannot see the candidate set it is asked to
   widen, and a shipped intent travels as its claim record rather than with the
   audit written against it.

The table is Go data, rendered into the readings family's charter under a test
holding the two to each other. The exclusion floor rides in every manifest, each
entry with the signal by which a reader detects it.

## Two artefacts, and where they land

`assemble` writes the assembled input and the manifest as two separate files: the
input goes to a reader, the manifest stays with the auditor. Without `--out` they
land in the local-tier run directory named after the run. `--out` names a
directory instead, which must be empty or absent, because one run's artefacts are
one run's evidence; each file is written through a temporary name and renamed into
place. With `--dry-run` and no `--out`, nothing is written and the result is
rendered only.

**`--out` costs the run its ingest.** `ingest` resolves a run's manifest from the
local-tier run directory and nowhere else, and `assemble` parks no copy there when
`--out` sent the pair elsewhere. A run assembled to any other directory refuses at
ingest, and bare `abcd reading` does not list it among the staged runs either,
because that listing reads the same one directory. `--out` is for a run whose
artefacts are being inspected or archived; a run meant to come back through
`ingest` lets the default run directory name itself.

An output directory the include table can reach is refused when it is named,
because writing a run where the table reaches it commits the next run's
contamination. And both artefacts are refused as input wherever an admitted path
holds one, recognised by the type tag they carry, so a run committed before that
refusal existed cannot ride in either.

Run identifiers are minted per adr-45, from a mint that reads no maximum, so two
checkouts assembling in the same window cannot converge on one id.

## What a reading would cost

Every assembly, `--dry-run` included, reports the size of the item text it
assembled — bytes and an estimated token count, in total and per kind — so what a
reading would be handed is known before one is commissioned (itd-198). The estimate
is byte-derived rather than a tokenizer's count, and the render says so beside the
number rather than letting a reader take it for a measurement.

## `ingest` checks what the reading was licensed to produce

`ingest` validates the JSON a reading returned and writes its reading records. It
is the output-contract idiom the repository already carries — an agent emits JSON,
a deterministic verb validates it, the verb writes the record — and it adds a check
no structural schema performs: what the reading was **licensed** to produce, not
only what it saw.

Three properties distinguish it. **Item identifiers are minted by the verb**, so
the payload carries none and one it supplies is refused as an unknown field. **The
supply regime is the definition's**, read from the position's definition file and
compared against the output's own claim, with no operand and no configuration key
able to reach it. And **named provenance is enforced at every regime**: an item
whose pattern field is empty or absent is refused, which the definitions instruct
and nothing else checks.

The regime gate has one layer, and it is structural. A regime that declares
reserved names has them matched against the item's own **keys** — its declared
fields, and the keys of any nested object the contract does not define — never
against the words inside a value. Keys are compared folded, so a reserved name
respelled in code points that render the same refuses as itself. Three of the four
regimes declare a row; the generative regime declares none, so no name is reserved
at the widening position. That is designed rather than overlooked: the generative
body schema is two fields, so any other key is refused as unknown regardless, and
the generative licence is the widest of the four, with its constraint falling at
admission rather than at ingest. The shipped `reading ingest --help` states it
outright, and the consequence to hold is that at one of the four positions this
gate performs no check of its own.

**The gate refuses only a real decision field.** A reading reports: it quotes the
record's own disposition line, says what a clause settles, what a paper recommends,
what a suite scores, and which section says a fix is merged while another says
pending. That is most of what a reading legitimately does. The gate once carried a
second layer, a registry of named signatures over an item's prose, and it could not
tell a reading that proposes from one reporting somebody else proposing: measured
over thirty-four realistic outputs it caught fourteen, every one of them for
quoting the document it read. It is gone rather than softened, because a gate that
reads prose is still reading prose. What remains carries no bound of that kind: a
field is present or it is not.

## Refusal, and what survives one

Refusal has two granularities. An item-level violation refuses that item and lands
the rest, naming the ordinal, the rule and the field. A list-level violation refuses
the run, and a refused run leaves a durable record once the run's identity is
proven: a refusal file carries the run metadata and the named reason and no items,
so a rerun is a new run with a new id rather than an amendment.

Identity is proven when the run id resolves to a manifest parked in the run
directory whose content hash matches the payload's, and until it does, nothing is
written anywhere. A malformed envelope, a run id nothing is parked under, a hash
disagreement, and a manifest naming another run or another position each refuse
before that point and leave no refusal record at all. That is deliberate: a refusal
record is a record about a run, and a payload that has not yet shown which run it
belongs to has nothing to be recorded against.

Writes are staged. Nothing durable is written or deleted until the whole payload
validates; the reading records land as one batch; and the run metadata is written
**last**, as the commit marker, so a run without one never happened. An interrupted
ingest leaves a stage in the local tier. Every later invocation names the orphan,
and the next one whose payload validates sweeps it, in one of two ways decided by
that run's own commit marker:

| the orphaned run | what the sweep does |
|---|---|
| reached no commit marker | rolls the run back: its reading records come out of the committed ledger and its run directory goes, because the run never happened. The ids removed are reported on every exit, including a failing one |
| reached its commit marker | is left entirely alone and the stage is cleared, because the stage is a leftover from a crash after the marker and the run is complete |

A refused run reports the orphans it left in place instead of sweeping them: the
sweep is a delete in the committed tier, and a refused run never reaches one.

## What this surface does not claim

It never runs a reading. It produces the input a reading would be given;
dispatching that input to a reader is host work.

The bundle is pathless by construction, which is the half of the isolation the
binary enforces. The other half — that the dispatching host grants the reader no
repository access — is a host obligation stated in the plugin surface, and it is
disclosed as an obligation rather than claimed as an enforcement.

Timing and target selection stay the operator's choice. The manifest and the run
record make that visible after the fact rather than preventing it. Prose-borne
warmth inside an admitted chapter has no structural signal: the chapter-level
include bound and the glossary discipline carry it, and it is disclosed as residue.

## References

- Plugin command: [`commands/reading.md`](../../../../commands/reading.md)
- The family's charter and the rendered include table:
  [`.abcd/development/readings/README.md`](../../readings/README.md)
- The construal's admissibility:
  [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md)
- Invariants 14 and 15: [`03-invariants.md`](../02-constraints/03-invariants.md)
