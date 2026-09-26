---
name: scribe
description: "Assemble a ledger scribe's context and ingest what it transcribed: Writes nothing bare; refuses an unknown sub-verb."
argument-hint: "assemble --run <rdg-N> --dispositions <path> [--out <dir>] [--dry-run] | ingest --scribe-json <path> --dispositions <path> [--context <path>]"
block: agents
---

# `/abcd:scribe` — the ledger scribe's context and ingest

The scribe (the `abcd:scribe` agent) transcribes a reading run's records and the
researcher's dispositions into the ledger's declared shapes, and authors
nothing. Its access rule is the reading assembler's exact inverse: a reading is
handed a slice of the shipped repository and no ledger; the scribe is handed the
ledger and no shipped tree. This verb holds that rule by construction. It builds
the scribe's context from the issue ledger's own directories and nothing else,
writes a manifest naming every path it passed, and refuses a returned payload
that authored anything.

Two things this surface does not do. It never runs the scribe: it produces the
context a scribe session is handed, and dispatching that session is host work.
And it never judges what it transcribes: a state, a ground or a resolution the
researcher's text does not carry is refused, never supplied.

## The host obligation

**Hand the scribe session the context file and nothing else.** No repository
access, no reading bundle, no transcript, no other file. The session is not the
reading session and never becomes one: a session that held a reading bundle may
not be handed a scribe context, and the reverse. The verb cannot enforce what a
host gives a session; `abcd history separation` reports afterwards whether any
retained transcript carries both a reading stamp and a scribe stamp of one run.

## Assemble

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" scribe assemble --run <rdg-N> --dispositions ./dispositions.md --json
```

`--run` names an ingested reading run: its commit marker must exist, because the
scribe transcribes dispositions against records the ledger already holds, read
from the store rather than from a raw reading output handed over again.
`--dispositions` names the researcher's dispositions text in whatever form they
wrote it; it is read whole and carried verbatim, with the home directory and the
repository root taken out of it.

The context carries every record under the issue ledger's own directories — the
reading records, dispositions, admissions, surprises and reframes, and the open,
resolved and won't-fix issues — derived from the ledger's directory list, so a
record family the ledger declares later is included the day it is declared.
Nothing outside those directories is walked, a symlink inside them or at any
directory above them is refused, and an item outside them is refused whatever
route it arrived by.

Report from the JSON: `run`, `item_count`, `context_stamp`, `context_sha256`
(the scribe's output cites it), `out_dir` and `artefacts`. The context and the
manifest are parked in `.abcd/.work.local/scratch/scribe-runs/<rdg-N>/`, or
under `--out`, which must be empty or absent and may not be a directory a
reading's include table reaches. The durable record is untouched: a session
assembled and never ingested leaves no trace beside the run. `--dry-run` writes
nothing unless `--out` names somewhere to write. A second assembly into an
occupied directory is refused: one directory holds one session's evidence.

Every refusal exits 2: a run id that is not one, a run that was never ingested,
a missing `--dispositions`, an out directory a reading can reach, and a
symlinked ledger directory.

## What the scribe returns

One JSON document, which the scribe definition states for the session:

```json
{
  "_type": "abcd.scribe.output/1",
  "run": "rdg-N",
  "context_sha256": "<the context_sha256 assemble reported>",
  "dispositions": [{"item": "rdi-N", "state": "accepted", "grounds": "…",
                    "exit_condition": "", "supersedes": "", "recurs": []}],
  "admissions": [{"item": "rdi-N", "grounds": "…"}],
  "surprises": [{"occasioned_by": "rdi-N", "text": "…"}],
  "fidelity_flags": [{"first": "…", "second": "…"}],
  "outstanding": ["rdi-N"],
  "refusals": [{"subject": "…", "reason": "…"}]
}
```

The scribe cannot compute `context_sha256`; the host puts the value
`assemble` reported into the payload, or hands it to the session with the
context.

## Ingest

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" scribe ingest --scribe-json ./scribe-output.json --dispositions ./dispositions.md --json
```

`--dispositions` names the researcher's dispositions text again, the same file
`assemble` was handed, and is required. `--context` names the context when
`assemble` wrote it under `--out`; the manifest is read from beside it. Nothing is written until all of the following
hold, and any failure exits 2 naming the field and the item:

- **The context is proven**: it hashes to its parked manifest, and the payload
  cites that hash. A payload from another session is refused.
- **The supplied text is the researcher's**: the parked context and manifest sit
  where a scribe session with tools could rewrite them, so the manifest's
  supplied hash and the context's supplied copy must both equal the file
  `--dispositions` names, and every check below reads that file. Never hand
  the scribe session that file's path.
- **Nothing is authored**: a key outside the shapes above (a `resolution`, a
  `pattern`, a `position`, anything) is refused by name, and so is a key the
  payload repeats at any depth, which is never read last-wins; a disposition or an
  admission for an item the supplied dispositions never name is refused; a
  `state` that does not stand as a whole word in its item's part of a line
  (the whole line when the line names no other item, else the text from the
  item's id to the next item id) is refused, and so is an admission whose
  item's part neither admits nor accepts it; a `grounds`, an `exit_condition` or a surprise `text` that does not stand
  verbatim in the supplied text once whitespace is folded is refused. The scribe
  reformats; it never adds a word.
- **Every answer is the run's**: a disposition, admission or outstanding item
  that is not one of the run's items is refused, and one item takes one answer.
- **Nothing is passed over in silence**: every item of the run with no standing
  disposition is answered, listed as outstanding, or named in a refusal.
- **The run has no promoted scribe manifest yet**: the durable tier is
  write-once, so once a session has landed records over a run, a later answer
  to it is written with the capture verbs. An ingest that lands no record
  promotes nothing and leaves the run open to a later session.

Then the dispositions, the admissions and the surprises are written, in that
order, through the capture verbs' own functions, each with its own redaction and
refusals. At the widening position that includes the ordering gate: no
disposition and no admission lands until a committed comparative run names the
widening run. The first refusal stops the ingest, the render names what landed
before it, and the manifest stays parked, so a rerun drops what landed rather
than minting it twice.

Report from the JSON: the `dispositions`, `admissions` and `surprises` written,
each with its id; `outstanding`; every `fidelity_flags` entry, **unresolved** —
never pick one side of a flag, it is the researcher's to resolve; every
`refusals` entry; and `manifest`, the promoted manifest beside the run, absent
when the ingest landed no record. Flags
and refusals are never written into a record; in the JSON a hidden or
terminal-control rune in either arrives percent-encoded (`%E2%80%AE`), so quote
the value as it stands.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
