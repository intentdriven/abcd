# `/abcd:lab` — Run a Lab

`/abcd:lab` mechanises the lab conventions three hand-run experiments proved
(itd-2609212137128014, spc-2609212141418943). A lab is a throwaway world pinned
at one commit of a repository, run to answer one question. Its evidence lives at
the operator level, its knowledge enters the record only through capture, and
nothing any lab verb does writes into the repository it studies. The procedure
the verb encodes is the discipline record
[itd-2609251624540864](../../intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md).

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
| `matrix` | — | shipped |
| `matrix copy` | — | shipped |
| `matrix review` | — | shipped |
| `matrix judge` | — | shipped |
| `matrix report` | — | shipped |
| `matrix rerun` | — | shipped |

## A model matrix run

A model matrix run in a lab begins with its plan (itd-2610161733316518,
spc-2610161949559050 step 2): the copies each cell makes, the expected cost by source against the spending cap, the load
cap below the machine's cores, the time and the judges each task needs. It is
built from the declaration and the machine's configuration alone: no network
call, no price fetched (a cost is reckoned only from a cell's declared
`cost_per_copy`), and nothing written. Its digest names exactly that plan.
Approving that digest records the run once the person has seen the plan and said
yes to it; the plan is built again and must carry the same digest. The spending
cap comes from the declaration's `spend_cap` or the machine's
`matrix.spend_cap`, never from the repository's configuration. A refusal
exits 2 with nothing written; a halted lab exits 1.

A consented run hands off every pilot copy (spc-2610161949559050 step 8): each
copy's clone of the frozen commit is made under a random alias in the clone
store outside every checkout, the record of which alias is which copy is kept
in the run's state in the lab, and a clone bound for a provider passes the
launch bundle's secret scan first, over its checked-out tree and the frozen
commit's own message, author and committer, a hit refusing the cell. Each hand-off
carries the brief the copy is made from, which names no task id, cell, route,
model or path on this machine. A host cell's copy is the host's to make in its clone and hand back
with a receipt and its transcript; a made code copy carries its hidden tests'
result as the host ran them, and its trial says abcd neither placed nor ran
them: it marks that result, the model served, the harness version and the
times as the host's own word, and says it carries no call receipts. A writing
copy's document is kept only once its trial passes the secret screen, so a
hand-back the screen refuses can be made again. A runner cell's copy waits:
abcd does not launch a runner in a clone yet, and when it does the cell's
runner makes the copy in its clone from the same blind brief
(dcn-2611251746554510); the host never stands in for one, since its answer
would measure the wrong model. A copy handed back is checked for compliance under one overall deadline
before its trial is written, a check copy's findings are recorded and its trial
written before its clone is deleted, and every trial, review request and
judging packet passes the secret screen. The report accounts for every copy
handed off that holds no trial: one the host never handed back, one waiting for
its runner, a refused cell's, and one the run's start never reached are each
shown and kept out of every rate. A review or a judging is drawn once, so it
waits until every copy of its task handed to the host is back. A code task's
copies are then reviewed through blinded requests, each assembled from the
frozen tree and the copy's clone read as data under an opaque label, whose
answers are scored before they are recorded. A writing task's judging packet is
drawn from its made copies with every mention of where a copy ran, its alias,
every task, cell, route and model and every maker withheld, however it is
spelt (split by invisible characters or by separators between its letters, in
look-alike letters, or with its version after it), and a judging session's
verdict file is recorded against it. A check copy's quote of the frozen
material is located in the material as written and shown with every model and
maker withheld, and its finding is keyed on the line as located, so two lines
that differ only in the model they name are never matched as one finding. A
check copy's own words are scrubbed before they are made one line, so a name
broken across lines is withheld from them too.

A recorded run's report is read from its records in the lab and writes nothing
(spc-2610161949559050 step 8). A code task gives each builder's pass rate on
its hidden tests with a 95% interval and its pass^k, what the reviewers caught
of its copies, its cost per passing build stated by source with what that cost
is comparable with, and its model time apart from the time spent waiting on
gates; a copy that never started, or one the compliance check set apart, is
counted apart and in no rate. A pass rate resting on hidden tests' results the
host reported counts them, a limitation names their copies, and every
comparison on them says so, so no ranking on the host's word is shown
unqualified. Each pair of builders is ranked only at the
repeats its power estimate asks for, from the smallest difference in pass rate
the declaration names; otherwise the comparison states the smallest difference
it could have detected and is not ranked. A written doc gives each model's
rating from a regularised Bradley–Terry fit of every judge's ranking, with an
interval bootstrapped over judges and copies and whether it is separable from
the others, how often its copies were good enough, the judges' agreement as
Krippendorff's alpha with an interval (or not estimable, saying why) and each
copy's length; below three judges it gives each judge's ranking and the
good-enough counts only. Guests and the host's own self-blinded rankings are
reported apart. Every report lists the run's limitations and the improvements
it suggests for the next run.

A check task is judged in two phases (spc-2610161949559050 step 9). The run
first joins its copies' findings by their exact keys; a copy the compliance
check set apart, or one never started, is in no comparison. Where the
declaration names a matcher cell, the findings no key joins go to it alone, each
under an opaque id with nothing of the copy, cell or model that raised it, and
its answer is read as untrusted input, refused whole when any part of it fails;
with no matcher those findings stay rows of their own and its agreement is not
estimable. The person then judges each distinct finding alone, shuffled, under
a code, with no model named and no verdict proposed, and their verdicts lock
once. Only then is the second phase shown: each finding a row and each model a
column, the verdict the proposer cell proposes where one is declared, each
model's precision, unique valid findings, recall and overlap, and the matcher's
agreement on a sample of its pairs the person labels, reported from ten
labelled pairs up. The matcher and the proposer are cells the person declares,
on their own routes and never on a builder's route or model, and each is
optional: nothing is chosen for the person. The model is compared by its name
with what carriers add to one model set aside: the provider prefix, a known
carrier's dotted region and vendor (`us.anthropic.`) and, behind it, the
carrier's revision tail (`-v1:0`), a routing variant such as `:free` or
`:latest`, a context suffix such as `[1m]`, a date or `@` stamp, a trailing
`-latest`, case, spaces, and a dot or a dash between version digits.
A size or quantisation tag (`:7b`), a model's own dotted name (`aya.expanse`)
and the version number itself still tell two models apart, a host builder is
compared by the model its copies were served, and the plan's judges note asks the person to check the
routes shown, since a name read this way can still miss one model sold under
two names.
One on a runner route waits for a launcher abcd does not have yet, and the host
never answers for it: the plan's judges line says so, and the judging stops at
that cell's stage saying so. Nothing of a check is filed or
written where the release gates read. A code task may name its hidden tests, a
folder outside the checkout, and the command that runs them; the printed plan
shows both, the folder quoted and each word of the command quoted where it
holds anything but plain printable text, with the line the person consents to:
that abcd will run each copy's code with that command, and that until it runs
hidden tests itself a code copy's result is the host's report, as above.

A model run is re-runnable, not reproducible. A re-run starts from a recorded
run, named by its id in the lab or by a run's folder carried from elsewhere,
and is planned from that run's own declaration under the person's consent to
its own plan; a run whose frozen commit is not the lab's pin is refused, and
so is a carried folder holding anything that is neither a file nor a folder,
which is never opened. The
re-run keeps a copy of the first run's records beside its own, each passed
through the secret screen, and its report shows, field by field, where it
drifted: the run's record and each copy's trial, with both values, leaving out
the ids and clocks that differ by construction. A re-run hands off its own
pilot copies as a run does.

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
store, and descends from the pin by the parents its commits store; it has no
remote; and the hooks path a
session in it would run, read with the operator's global configuration in
force and judged as git expands it (`~`, `~user`, `%(prefix)/`), resolves
inside the lab; an empty value, or one git cannot expand, is refused. Dual binary: the work binary is a regular file,
never a link to an operator-level installation; its embedded vintage, read from
its build metadata without running it, is the pin and unmodified; it is the
binary the first passing preflight pinned by its sha256, since the work binary
is never rebuilt; and a test binary, when there is one, is a separate file.
The dual-binary group holds a lab of abcd's own repository: only that snapshot
can build an abcd work binary. It fails closed, holding the lab when either
signal says abcd — the root commit is abcd's, or the pin commit carries
`cmd/abcd/main.go`, read at the pin so a later commit that removes it cannot
lift the gate; a pin git cannot read holds it — so a shallow or rewritten copy
of abcd never skips it. In a
lab of any other repository each binary check passes marked not applicable,
saying why, in the artefact and the JSON alike; the mint's next steps and the
INTENTION's lifecycle map name no `bin/abcd`, and a probe record names the work
binary not applicable rather than absent.

**Recording** a probe scaffolds its record — the input, the command line, the
exit status and the two output streams, empty, and a note naming the artefact
the probe observed (the work binary's vintage and sha256, when there is one).
The verb runs nothing: whoever runs the lab runs the probe's command and
redirects its output into the scaffold, so no command reaches the shell through
the verb. A probe is recorded once and never overwritten.

The **retraction sweep** reads every correction the lab recorded and searches
the lab's own documents for its literal — the pattern, not the instance —
listing every file and line where it still stands. The snapshot, the lab's
HOME and binaries, transcripts and each probe's five capture files are not
swept: they are the world and the instruments, not claims. Nor are the
records a procedure writes under `state/matrix/` (a matrix run's declaration,
run and trial records): they are what a run was given and measured, never
rewritten, so a retraction swept into one could never be applied. Only a
regular file a procedure could have written there, within the size it writes,
is passed over; anything else there is swept like any document. A probe's
`record.md` is prose the harvest cites, and is swept. The sweep is fail-closed:
a document it cannot read — larger than it reads, binary (a NUL byte in its
head), or not a regular file — is listed by path, and while any correction is
recorded that fails the sweep, since a retracted claim may stand in it unseen.
A correction whose literal is shorter than three characters is refused as
noise.

The **harvest** assembles the lab's harvest in the lifeboat's section shape —
intention, method, what worked, what is open, candidates, coverage — from the
INTENTION, the findings log and the probe records. Every finding cites its
probe records by path; every product finding is listed as a capture candidate
with the capture line that files it, found during the lab, and flagged where no
refutation was attempted; procedure findings are listed as amendment candidates.
Coverage grades each section grounded, partial or blank, the lifeboat's
vocabulary. The harvest files nothing, and claims no cost the lab did not
measure.

A lab that collected the decision adapter's shadow pairs
(itd-2609221009495079) gets one more section. When a judgement type is
configured for shadow (`oracle.judgements.<type>` set to
`shadow:<provider>/<model>`, [`03-configuration.md`](../05-internals/03-configuration.md)),
the host decides and the adapter answers beside it. Each pair is appended to
`pairs.jsonl` in the newest lab home minted for the repository: the host's
answer, the adapter's answer, the adapter's probability, the state's SHA-256 and
size, and the model that answered, or why it did not. That is the only file the
adapter writes, and it is never written into the repository. Two steps go
through the router today, so theirs are the types whose pairs are kept: the
drain's judgement before a lane opens (`eligibility-residual`), and embark's
lesson ranking (`embark-lesson-ranking`), one pair per lesson once the host's
scores are given ([`03-embark.md`](03-embark.md)). Every other type stays on
the host whatever its route says
([`03-configuration.md`](../05-internals/03-configuration.md)).
Here the harvest
computes something rather than only assembling prose: from `pairs.jsonl` it
counts, for each judgement type, the pairs, the pairs compared (the adapter
answered), the pairs that agreed and the pairs skipped (the state was larger
than the model's window, or the call failed). It reports the agreement rate,
and the compared and agreed counts in four probability bands (below 0.50,
0.50–0.70, 0.70–0.90, 0.90 and above). Lines that do not parse are counted
apart and never guessed at. The section ends with the line a research note
filed from it carries, naming the lab home and the count. That note is the
evidence a ruling turns a type on with. A lab with no pairs has no such
section.

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

`0` done; `1` a gate refused or a halted lab refused a probe: the preflight and
the sweep refuse with the artefact rendered and the finding recorded, while the
harvest's citation check writes nothing and sets no halt, and a halted lab
refusing a probe writes nothing and names the standing finding; `2` the request
was refused — no checkout, an unknown or malformed lab id, a bad probe name,
question or pin — with nothing written. The
JSON output carries each result whole, and a refusal is the `{"abcd":"error",…}`
envelope on stdout. Every path is written in tilde form.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd lab`

Sub-verbs: `abcd lab harvest`, `abcd lab matrix`, `abcd lab mint`, `abcd lab preflight`, `abcd lab record`, `abcd lab sweep`.

Flags: none.

### `abcd lab harvest`

Sub-verbs: none.

Flags: none.

### `abcd lab matrix`

Sub-verbs: `abcd lab matrix copy`, `abcd lab matrix judge`, `abcd lab matrix report`, `abcd lab matrix rerun`, `abcd lab matrix review`.

| Flag | Type |
|---|---|
| `--approve` | string |

### `abcd lab matrix copy`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answer` | string |
| `--form` | string |
| `--receipt` | string |
| `--transcript` | string |

### `abcd lab matrix judge`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--bad` | string |
| `--good` | string |
| `--host` | string |
| `--judges` | string |
| `--labels` | string |
| `--matcher` | string |
| `--proposals` | string |
| `--table` | bool |
| `--verdicts` | string |

### `abcd lab matrix report`

Sub-verbs: none.

Flags: none.

### `abcd lab matrix rerun`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | string |

### `abcd lab matrix review`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answers` | string |

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
