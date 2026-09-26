# `/abcd:lab` — Run a Lab

`/abcd:lab` mechanises the lab conventions three hand-run experiments proved
(itd-2609212137128014, spc-2609212141418943). A lab is a throwaway world pinned
at one commit of a repository, run to answer one question. Its evidence lives at
the operator level, its knowledge enters the record only through capture, and
nothing any lab verb does writes into the repository it studies. The procedure
the verb encodes is the discipline record
[itd-2609251624540864](../../intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention-to-discard-and-every.md).

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `mint` | — | shipped |
| `preflight` | gate | shipped |
| `record` | — | shipped |
| `sweep` | gate | shipped |
| `harvest` | — | shipped |

## The store

The lab store is machine-scoped and keyed on the repository's root commit, as
the transcript store is
([`../05-internals/03-configuration.md` § The lab store](../05-internals/03-configuration.md#the-lab-store)):
one registry line per lab in the repository's lane, and one lab home per lab
beside it. Every level is created one real directory at a time and never
through a symlink, the store is private to the account (directories `0700`,
files `0600`), and every write inside a lab home goes through a containment root
opened on that home, so a link planted inside it cannot carry a write out.
Labs that predate the keyed layout sit at the top of the store with their own
index; the verb neither reads nor writes them.

Bare invocation lists the repository's labs, read-only: each lab's pin, its
probe count, and the gates holding it halted. An absent store is an empty list
and is not created.

## The lifecycle

Minting a lab for a question creates its home, appends its registry line, and
lays down a **snapshot**: a standalone clone of the repository detached at the
pin, which is HEAD unless another commit is named. The clone runs under the
isolated git environment, so no hook the operator's configuration names fires
while it is made, and its remote is removed, so nothing done in the lab world
reaches the checkout it came from. The lab's documents are scaffolded: the
INTENTION (the question, the hypothesis, the measures, the STOP conditions and
the amendments chain, with the lifecycle mapped onto the home), the findings
log, the corrections log and the amendments log. A mint that fails part-way
removes the home it made, and nothing else.

The **preflight** writes its artefact and runs two groups of checks.
Harness isolation: the lab's own HOME holds no link out of the lab; the
snapshot is a standalone clone rather than a linked worktree, borrows no object
store, and descends from the pin; it has no remote; and the hooks path a
session in it would run, read with the operator's global configuration in
force and judged as git expands it (`~`, `~user`, `%(prefix)/`), resolves
inside the lab; an empty value, or one git cannot expand, is refused. Dual binary: the work binary is a regular file,
never a link to an operator-level installation; its embedded vintage, read from
its build metadata without running it, is the pin and unmodified; it is the
binary the first passing preflight pinned by its sha256, since the work binary
is never rebuilt; and a test binary, when there is one, is a separate file.

**Recording** a probe scaffolds its record — the input, the command line, the
exit status and the two output streams, empty, and a note naming the artefact
the probe observed (the work binary's vintage and sha256, when there is one).
The verb runs nothing: whoever runs the lab runs the probe's command and
redirects its output into the scaffold, so no command reaches the shell through
the verb. A probe is recorded once and never overwritten.

The **retraction sweep** reads every correction the lab recorded and searches
the lab's own documents for its literal — the pattern, not the instance —
listing every file and line where it still stands. The snapshot, the lab's
HOME and binaries, transcripts and probe records are not swept: they are the
world and the instruments, not claims. A correction whose literal is shorter
than three characters is refused as noise.

The **harvest** assembles the lab's harvest in the lifeboat's section shape —
intention, method, what worked, what is open, candidates, coverage — from the
INTENTION, the findings log and the probe records. Every finding cites its
probe records by path; every product finding is listed as a capture candidate
with the capture line that files it, found during the lab, and flagged where no
refutation was attempted; procedure findings are listed as amendment candidates.
Coverage grades each section grounded, partial or blank, the lifeboat's
vocabulary. The harvest files nothing, and claims no cost the lab did not
measure.

## Halt and record

A gate refusal halts the lab and is recorded, never adapted around. When the
preflight or the sweep fails, the verb writes its artefact, appends a gate
finding to the findings log naming every failed check or correction, and holds
the lab halted on that gate. A halted lab records no probe. The halt lifts only
when the same gate passes again; the finding stays in the log, and the harvest
of a halted lab leads with it. A refusal that repeats a standing halt reuses its
finding rather than logging it twice. The sweep's finding names corrections by
number and line, so it is never itself an instance of the text it retracts.

No claim outlives its input: the harvest refuses a finding that cites no probe
record or evidence file, or cites one that is missing or incomplete, and writes
nothing. A harvest written by hand is never overwritten.

## Exit codes

`0` done; `1` a gate refused (the preflight, the sweep, the harvest's citation
check) or a halted lab refused a probe, with the artefact rendered and the
finding recorded; `2` the request was refused — no checkout, an unknown or
malformed lab id, a bad probe name, question or pin — with nothing written. The
JSON output carries each result whole, and a refusal is the `{"abcd":"error",…}`
envelope on stdout. Every path is written in tilde form.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd lab`

Sub-verbs: `abcd lab harvest`, `abcd lab mint`, `abcd lab preflight`, `abcd lab record`, `abcd lab sweep`.

Flags: none.

### `abcd lab harvest`

Sub-verbs: none.

Flags: none.

### `abcd lab mint`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--pin` | string |

### `abcd lab preflight`

Sub-verbs: none.

Flags: none.

### `abcd lab record`

Sub-verbs: none.

Flags: none.

### `abcd lab sweep`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
