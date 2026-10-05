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
| `interview` | — | shipped |

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

## In a plain Terminal

The interview sub-verb runs the same interview with no host session
(spc-2610030911534855). The `reflection-composer` runs on the runner the person
routed it to in their own machine's configuration, once per turn: each turn's
brief carries the seed, the asking rules and the answers so far, and the role
returns the next question as a receipt of the shared question type, or `done`
with the answers object. abcd checks each question before it is drawn
(sanitised first, then the structural check and the asking limits), draws it
when stdin, stdout and stderr are all terminals, and records each answer, marked
as answered in the Terminal, in the answers record in the local tier. A question
that fails the check is the runner's invalid answer: its fallback receipt goes
into the record and, with no configured host to fall back to, the interview
stops, exit 1, keeping the answers given. `done` is filed through the write
below, with every floor and refusal unchanged; a thin answer the write refuses
goes back to the role, at most twice, to ask the section's follow-up. A
fault of the write that is not one of its refusals exits 1, since the answers
record stands by then.

The role is granted Read, for its turn's brief, and Write, for its receipt,
and changes no file of the repository: abcd reads the working tree's state
(git's listing of what differs from HEAD, each path's content hashed; every path
git ignores, the local tier among them, by its mode, size and modification
time; git's own hooks, info files, configuration, HEAD and refs, and each
submodule's hooks and configuration, and every worktree's entry under
`worktrees/` (its `HEAD`, `commondir`, `gitdir`, `config.worktree` and
`locked`, an entry made or removed noticed), read in the repository's common
git directory and where a link there leads, any hooks directory
`core.hooksPath` names outside the tree, read where its links lead, and the
push receipts in the local tier of every worktree git lists, each hashed; left out are only the run's own turn
directory, where the role writes its receipt, a checkout's local transcript
store, a file named `.DS_Store` anywhere and `.claude/scheduled_tasks.lock` at
the root, which execute nothing, every other `.claude/` path watched) before
and after each dispatch, and any path that changes during a dispatch stops the
interview, whatever changed it (abcd cannot tell the role's writes from another
program's, so two interviews run in one checkout stop each other), exit 1,
naming each and keeping the answers given. What the links one reading follows
lead to is read within 1,024 entries and 64 MiB in all, and a reading past that
stops the interview, exit 1, naming the link.

A drawn question takes a choice, not typed prose, so the role offers drafts of
a section's answer, and the retrospective carries the drafts the person chose.

It refuses before anything runs, exit 2, writing nothing:

- when no route of the person's reaches a runner: the role's route is the host,
  no fallback host is set, and the refusal names the role's key, the machine's
  file, and the setup interview, which needs no route;
- off a terminal with no answers file. With one, each question is written as
  plain text and answered by ordinal (`Q1`, `Q2`, ...); the file running out
  refuses, exit 2, naming the question, and records nothing. An entry's place
  is its own `answered_in`, else the answered-in flag's, `Terminal` by default.

Unshipped targeted intents refuse as the write does, before any runner starts,
until the proceed flag confirms them. Ctrl-C at a question, or while the runner
writes one, exits 130 and keeps the answers given.

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
it is written, and a degraded or unavailable scanner refuses the write; after
the redaction every control and bidi byte in an answer is masked with `?`, line
by line, so the record reads as its bytes say. The file is created exclusively
inside the retrospective store, every level of which must be a real directory,
so neither a second run nor a symlinked store can overwrite or escape, and under
the intent store's lock, the lock a lifeboat embark writes under, so an embark
carrying a retrospective for the same release is serialised with it.

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
- The plain-Terminal interviews: spec `spc-2610030911534855`
- Naming registration: [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)
- The agent catalogue: [`../05-internals/01-agents.md`](../05-internals/01-agents.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd reflect`

Sub-verbs: `abcd reflect interview`, `abcd reflect write`.

Flags: none.

### `abcd reflect interview`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answered-in` | string |
| `--answers` | string |
| `--proceed` | bool |

### `abcd reflect write`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answers` | string |
| `--proceed` | bool |

<!-- surface-appendix:end -->
