# `/abcd:reflect` — Phase Retrospective

> **Not built yet.** There is no `reflect` verb on the binary, no
> `commands/reflect.md`, no `reflection-composer` agent under `agents/`, and no
> `.abcd/retrospectives/` tree in the working tree. The backing intent sits in
> [`intents/planned/`](../../intents/planned/itd-24-reflect-command.md)
> (itd-24); delivery state is the intent lifecycle's, not this page's (see the
> [brief README's provenance note](../README.md)). The prose below records the
> design contract in present tense as the brief's intents do.

Close a phase of work with a retrospective somebody will actually read a year
later, without starting from a blank page. The command takes a completed
phase, reads the audit receipt that phase already produced, and turns its
per-item verdicts into a short interview: five seeded questions, one clarifying
follow-up where an answer is thin. What lands is a five-section README that
links out to the phase, the audit and the specs rather than copying them, so
the retrospective stays a judgement and never becomes a second copy of the
record.

The grain is the phase, deliberately. Per-intent reflection is the
`intent-auditor`'s job, and a retrospective per intent would be a chore nobody
finishes.

## Argument

The command takes exactly one positional argument: a **phase id**, which is a
filename stem in [`roadmap/phases/`](../../roadmap/phases/) (`phase-1-ahoy`,
say). It is not an intent id and not a spec id, and `/abcd:reflect <itd-N>` is
refused. Bare `/abcd:reflect` renders help and writes nothing.

## What it does

1. Selects the **latest** phase-audit receipt whose phase matches the argument,
   read from the local-ephemeral logs tier.
2. Runs the composer agent as a seeded single-pass interview, its questions
   drawn from the receipt's per-item acceptance verdicts. A thin answer triggers
   one clarifying question; a deliberately empty section renders an explicit
   "none recorded" line rather than being omitted.
3. Shells a deterministic writer with the collected answers as JSON. The
   command markdown performs **zero writes**: every write goes through that
   writer, which renders the README, records the consumed receipt path, and
   links to the phase doc, the audit report and the member specs.

The writer is the single dispatch target and the only writer, and it is fully
testable without the agent: JSON answers in, README out.

## The five-section template

The retrospective always carries these five sections, in this order: what went
well, what could improve, lessons learned, decisions made, and metrics
(qualitative plus simple counts, never velocity telemetry).

## Refusals

The writer refuses, each refusal naming the phase-audit prerequisite, when: the
argument is not a phase id; the answers are hollow (a bare `{}` on stdin, all
"none recorded") and `--allow-empty` was not passed; no phase-audit receipt
exists for the named phase; the latest receipt is empty-audited, so nothing
shipped to reflect on; or a retrospective already exists and `--overwrite` was
not passed.

It also enforces write-site containment: the resolved target must be inside the
retrospectives tree, and receipt-supplied spec ids are shape-validated before
they are rendered into link text.

## Where the receipt lives

The receipt shape this surface consumes is the predecessor store's phase-audit
report, and the predecessor wrote it under `.abcd/logbook/`. **That location is
retired here.** A 2026-07-12 maintainer adjudication (iss-36 and iss-56,
resolved as iss-73) placed runtime artefacts in the gitignored
`.abcd/.work.local/logs/` tier instead, and a detector holds it:
`TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source
under `internal/` so much as names `logbook`. A delivered `reflect` therefore
reads its receipt from `.abcd/.work.local/logs/`; the retired path survives in
this record as the predecessor's, never as a path to implement against.

## Output path is unsettled

Output is fixed at `.abcd/retrospectives/<phase-id>/README.md`, committed as
part of the phase's permanent record. That path is a peer of `.abcd/work/` and
`.abcd/development/`, and it is **not one of the three tiers** `AGENTS.md`
fixes. Delivering itd-24 therefore has to place the tree in an existing tier or
record a decision admitting a fourth; until then the output path is a design
target's proposal rather than a settled location.

## Scope of the first version

- A single seeded interview pass. Multi-turn depth is a recorded future
  extension.
- No auto-triggering after a phase closes: reflection is deliberately
  on-demand.
- A missing audit is refused rather than repaired inline.
- Links are limited to the phase doc, the audit report and the member specs.
  The receipt carries no intent ids, so intent links are a recorded future
  extension.

## Lifeboat forward requirement

The lifeboat must pack every phase retrospective a voyage produced, so the full
reflection arc travels between voyages. Because `/abcd:reflect` is not built and
produces no retrospectives yet, this is a **documented forward requirement on
the disembark pack**, recorded here and in the itd-24 acceptance so a later
reader treats it as a requirement rather than a shipped capability.

## Related documentation

- Intent: `itd-24` (`../../intents/planned/itd-24-reflect-command.md`)
- Naming registration: [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)
- The agent catalogue a composer would join: [`../05-internals/01-agents.md`](../05-internals/01-agents.md)
