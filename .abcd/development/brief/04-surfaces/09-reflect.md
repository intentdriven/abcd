# `/abcd:reflect` — Release Retrospective

Close a release with a retrospective somebody will actually read a year later,
without starting from a blank page. The command takes a cut release, reads what
it shipped and how each shipped intent's audit went, and opens a short interview
from that: four asked sections, one clarifying follow-up where an answer is thin,
and a computed fifth. What lands is a five-section README that links out to the
changelog section and to each intent's audit notes rather than copying them, so
the retrospective stays a judgement and never becomes a second copy of the
record.

The grain is the release, deliberately
([adr-2609212115255771](../../decisions/adrs/2609212115255771-phases-and-milestones-are-retired-sequencing-is-dependencies.md)
retired the phase; itd-24 decisions 4 and 5). Per-intent reflection is the
`intent-auditor`'s job, and a retrospective per intent would be a chore nobody
finishes.

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
| `write` | — | shipped |

## Argument

`abcd reflect <release-tag>` takes exactly one positional argument, a **release
tag** the repository holds, in the strict `vMAJOR.MINOR.PATCH` shape
(`v0.11.0`). It is not an intent id: an intent id is refused, naming the intent
audit as the per-intent surface. Bare `/abcd:reflect`
renders help and writes nothing. The person's help lists the verb under
**Release**, beside `launch` (ruling H13).

## The seed

`abcd reflect <release-tag>` renders the seed and writes nothing:

- **The intents the tag shipped**: those that reached `shipped/` between the
  previous release tag and this one, less any whose `shipped_in` names another
  release, plus any in `shipped/` now whose `shipped_in` names this one. A cut
  written the ordinary way stamps no `shipped_in`, so the stamp alone would find
  nothing; it moves a record between releases, the way a hygiene sweep uses it.
- For each, its `impact` and its **audit notes** as counts: the acceptance
  rollup an ingested review writes, and the honoured / diverged / missing gap
  counts. A placeholder, an owed review and an absent section are not audit
  notes; a hand-written audit is, with no counts.
- The shipped intents with **no audit notes**, each with the command that
  audits it: an offer, never a gate.
- The planned intents whose `target_release` names the release and which **did
  not ship**.
- The **changelog section** the cut composed, found by the heading predicate
  the cut and the tagger use.
- The **metrics**: intents shipped, audited and not, the verdict and gap
  distributions, and the dates of this tag and the previous one.
- The four questions the interview asks.

## The interview

`commands/reflect.md` runs it on the host. It offers the audit for each
unaudited intent first and continues either way; it warns about, lists and asks
about the unshipped targeted intents; then the `reflection-composer` agent asks
the four asked sections one question at a time under the GRILL rules. The
metrics section is computed, never asked.

**The thin-answer floor.** An answer is thin when it is empty, restates its
section's heading, or holds fewer than two clauses of at least three words
each. A thin answer is met with that section's one follow-up question before
anything is written, and the reply is filed as the section's follow-up. The
floor is a heuristic, and a cheap one to be wrong about: it costs one question,
never the refusal of the answer that question brings.

## The write

The write sub-verb, handed the interview's answers as a JSON file, is the only
write. It rebuilds the seed, so every refusal the seed makes holds at the write
too, and then refuses, writing nothing:

- while an answer is under the floor with no follow-up, naming each section and
  its question (`thin_answers`);
- while unshipped targeted intents are unconfirmed; a proceed flag is the
  person's confirmation (`unshipped_targets`);
- when the release shipped no intent: "no intent shipped in `<release-tag>` —
  nothing shipped to reflect on" (`nothing_shipped`);
- when the retrospective already exists: it is written once and not edited
  after (`exists`).

The answers file is read strictly: an unknown or repeated key is refused rather
than an answer dropped. Every answer passes the canonical secret scanner before
it is written, and a degraded or unavailable scanner refuses the write. The file
is created exclusively inside the retrospective store, every level of which must
be a real directory, so neither a second run nor a symlinked store can
overwrite or escape.

## The output

`.abcd/development/retrospectives/<release-tag>/README.md`, in the durable
record tier beside the intent store, committed as part of the permanent record:

- frontmatter naming the release, the previous release, the date, the intents,
  which were audited and which not, and the audit receipts that fed the seed;
- a line linking the release's changelog section, and a table linking each
  intent and its audit notes;
- the five sections in order: what went well, what could improve, lessons
  learned, decisions made, and metrics (simple counts, never velocity
  telemetry).

## The nudge

When the release cut writes a cut, its last line says once that a
retrospective for the release is owed and names `/abcd:reflect <release-tag>`
(`retrospective_owed` in its JSON). Nothing repeats it and nothing waits on it.

## The lifeboat

A lifeboat pack carries every retrospective the voyage produced, verbatim, as
`retrospectives/<release-tag>/README.md`, sealed by the record manifest hash with
the other record families. The embark write puts them back into the new
voyage's store, admitting only a strict release-tag directory holding
`README.md`, and the embark lessons view ranks their lessons against the new voyage's brief for the
press-release interview: the three most like it, then the rest as a list
([`03-embark.md`](03-embark.md)).

## Related documentation

- Intent: [`itd-24`](../../intents/shipped/itd-24-reflect-command.md); spec
  `spc-2609211751376504`
- Command page: `commands/reflect.md`; agent: `agents/reflection-composer.md`
- Naming registration: [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)
- The agent catalogue: [`../05-internals/01-agents.md`](../05-internals/01-agents.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd reflect`

Sub-verbs: `abcd reflect write`.

Flags: none.

### `abcd reflect write`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answers` | string |
| `--proceed` | bool |

<!-- surface-appendix:end -->
