# `/abcd:embark` — Unpack a Lifeboat

Start a repository from someone else's record rather than from nothing. Point
embark at a packed lifeboat and it writes that project's decisions, issues,
intents and specs into their canonical places in your repo, verbatim, and opens
with the coverage blanks a human still owes the record. What you get is a
working store on the first day; what you are told, before any of it, is exactly
what the pack could not ground.

It is safe to aim at a repository that already has content. A file the lifeboat
would write and the target already holds with different bytes is a conflict, and
**any** conflict refuses the whole write: nothing is written, and one bulk
report names every conflicting path so you can resolve them and re-run. A file
whose bytes already match is an idempotent skip, so a re-run is a clean no-op.

> **What ships is `from` and `probe`, and `from` takes no flags** (only the
> global `--json`). The richer surface this chapter designs — a `scan` discovery
> sub-verb, and the `from` modifiers `--force`, `--archive` and
> `--refresh-audit` — is **not built yet**; [§ The design-target
> surface](#the-design-target-surface) holds it, and nothing outside that
> section describes it as present.

> **Recovery humility.** The lifeboat is the highest-fidelity floor the originating session could leave behind; it is not the activity that produced it. **When something here does not make sense, hunt the originating session before trusting the lifeboat blindly**: ask the prior author, surface the conversation where the decision happened, look at the rejected alternatives. See [`01-product/03-mental-model.md § The Naurian gap`](../01-product/03-mental-model.md#the-naurian-gap--modification-axis).

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the lifeboat round-trip ships in [Phase 6](../../roadmap/phases/phase-6-lifeboat.md). [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) is the model of record, and it settled four things against this chapter's earlier design: the voyage log is operator-level rather than in-tree, there is no in-tree lifeboat home and no shorthand for one, writes go through `os.Root` containment plus independent path validation rather than ordinary file writes, and a refusal path never writes (the core returns the conflicts and the surface renders them). One more, and it is the load-bearing one: **lifeboat text is never injected verbatim into `CLAUDE.md`.** The current marker block is re-injected instead. That is the difference between a data leak and a persistent instruction implant.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `from` | — | shipped |
| `probe` | — | shipped |


Bare `/abcd:embark` prints dispatcher help and mutates nothing.

- **`embark from <lifeboat-dir> [target-dir]`** unpacks into the target, which
  defaults to the working directory. The lifeboat path is required and is always
  an explicit path to a destination a disembark wrote: there is no in-tree
  lifeboat home to expand a shorthand to. The round-trip and self-test case is
  ordinary, not special-cased: `disembark pack <repo> <dest>` followed by
  `embark from <dest>`.
- **`embark probe <lifeboat-dir> [target-dir]`** answers the same question
  read-only: what would land where, does the lifeboat verify against its
  manifest, is its schema version one this build understands. It writes nothing
  and runs no product audit.

## 1. Source lookup

Lifeboats are *output*, and they land out-of-tree at a destination the operator
chose ([`02-constraints/01-platform.md § Lifeboat path`](../02-constraints/01-platform.md#lifeboat-path)).
There is no in-tree lifeboat home and no repo-local registry of inbound
lifeboats, so embark always reads from an external source: the destination some
disembark wrote to. Path resolution is therefore one step: validate the given
path and use it.

## 2. Conflict-based refusal

There is no emptiness gate. A target that merely carries unrelated files is not
a conflict. Embark refuses only when a planned target path already holds
differing bytes, is a non-regular target, is a duplicate target, or sits under a
non-directory parent. On any such conflict the core **writes nothing**: it
returns the conflict set it found, and the surface renders it. A refusal that
writes a file would be a transport-agnostic-core violation.

## 3. Scaffold steps

Embark is a deterministic Go run: it reads the lifeboat, plans, refuses on any
conflict, then writes the record families plus the marker block. No interactive
scaffolder and no model sit in the write path.

0. **Read the lifeboat.** The four record families — ADRs, issues, intents,
   specs — plus the report-only files that inform the run. The lifeboat is
   untrusted input: embark verifies its `manifest_sha256` against the on-disk
   tree, over every hashed file, and refuses a symlink or an oversize file
   anywhere inside. A tampered hashed record or an added stray file is refused.
   The post-pack synthesis layer sits outside the manifest seal deliberately,
   because it is written after the hash: those files carry their own per-entry
   integrity (cite-or-be-dropped, the registered-verdict gate) rather than the
   hash.
1. **Plan.** Map each record file to its target path and classify it as a
   create, as unchanged (byte-identical), or as a conflict. On any conflict,
   embark writes nothing and refuses.
2. **Write the record families verbatim** to their canonical locations, through
   two-layer containment (an `os.Root` boundary plus independent lexical path
   validation), skipping the unchanged. Bucketed families keep their source
   bucket: issues by state, intents by lifecycle stage, specs by open or closed.
   Terminology, docs and the memory store are **not** embark families; they do
   not travel.
3. **Re-inject the current abcd marker block** into the target `CLAUDE.md`
   between its markers, idempotently. Never `AGENTS.md`, and never a verbatim
   copy of lifeboat prose. The block is the modular-rules-loader block (itd-3);
   principles surface through the loader's domain rules on demand.
4. **Report**, blanks first: any pass the lifeboat declares exempt, then the
   coverage blanks a human must answer, then what was embarked into where, the
   written and unchanged counts with the per-family breakdown, and the marker
   action. The report-only files that informed the run are tallied by `probe`
   alone. They ride on the `from` result and are reachable through `--json`, but
   the rendered report omits them, because a human reading the outcome of a
   write wants what landed, and the probe is where a human asks what a lifeboat
   holds.

## 4. Conflict UX

The core returns the conflict set and the surface renders it as a **single bulk
report**: one line per conflicting target path with its conflict kind, never a
per-file barrage and never a file written by the core. The shipped `from` writes
nothing on any conflict and exits non-zero.

```
2 conflict(s) (nothing was written):
  • .abcd/development/specs/open/spc-12.md  (exists-differs)
  • .abcd/work/issues/open/iss-7.md  (exists-differs)

nothing was written — resolve the conflicts and re-run
```

The conflict list is a value the core hands back. If the operator wants it on
disk, the surface writes it; the core does not.

## 5. The coverage handoff

Both `probe` and `from` open on what a human still owes the record, before any
write summary. Two things print there, in this order.

**A declared pass exemption.** The lifeboat's provenance carries an optional
exemption for pass B, the pass that mines chat transcripts for the rationale
nobody wrote down. No source tier this build packs reads a transcript store, so
disembark writes the declaration into every pack it produces, and embark carries
it through as the first line of both reports.

The declaration outlives the blanks below it, and that is the point of it being
a declaration rather than a silent gap: a brief section pass B would have
grounded reads as a pass that never ran, not as work a human has been left. The
field is an omitted pointer rather than a boolean, so a package that predates it
marshals exactly as it always did, and the branch that stops the declaration is
exercised before any transcript adapter exists. An exemption that could not stop
would become a false claim in a durable artefact the day pass B ships.

**The coverage blanks.** Past the exemption, an absent coverage report prints
nothing, a degraded one prints a one-line note, and a present one prints each
unanswered brief section with the question that grounds it, human-owned sections
marked as yours to write. A present coverage with no blanks prints nothing:
there is nothing to answer.

## 6. Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare
  `/abcd:embark`, **then** the dispatcher prints help listing the shipped
  sub-verbs and the global `--json` flag, and mutates nothing.
- **Given** a lifeboat and a conflict-free target, **when** `embark from` runs
  (the target defaulting to the working directory), **then** the four record
  families land at their canonical locations under the target, the current abcd
  marker block is re-injected into the target `CLAUDE.md`, and everything else
  in the lifeboat informs the report but is never written.
- **Given** a repo disembarked to a destination, **when** `embark from` runs on
  it in an empty target, **then** the round-trip completes with no shorthand and
  no special case: the destination is an ordinary explicit path.
- **Given** a target holding a file that conflicts with a planned write,
  **when** `embark from` runs, **then** the command refuses, the core returns
  the conflict list **without writing any file**, and the surface renders it as
  one bulk report. A target that merely holds unrelated files is not a conflict.
- **Given** the user runs `embark probe`, **when** it completes, **then** the
  lifeboat is inspected against the target (file tree, schema validation,
  would-be writes), no target mutation occurs, and the user sees a report ready
  to inform the decision to run `embark from`.

## The design-target surface

None of the following is built. Each is described here as the intended design,
and the acceptance criteria that name it are gated on it shipping.

- **`embark scan`** would list lifeboat destinations — directories carrying a
  parseable `_provenance.json`, the same marker the destination safety gate keys
  on — ranked by modification time and presented as candidates, with no
  unpacking, and `--deep` for a wider walk. It is what a user reaches for before
  `embark from` when they are not sure where lifeboats live.

  > **Open question (adr-35):** where `scan` searches. Walking the parent
  > directory made sense when a lifeboat lived inside its producing repo, so
  > siblings of the working directory *were* the candidate set. Destinations are
  > now operator-chosen and need not sit beside the repo being embarked into.
  > Either the sibling walk is kept as a cheap heuristic, or scan is given
  > explicit roots (an argument, a configured search path, or the voyage
  > records). adr-35 does not settle this, and it must be decided before `scan`
  > is specified; the depth semantics of `--deep` fall out of whatever that
  > decides. The same note is carried in
  > [`02-constraints/01-platform.md § Embark sources`](../02-constraints/01-platform.md#embark-sources).

- **`from --force`** would turn the bulk conflict report into a single
  resolution prompt (keep target, replace target, merge where possible, or
  abort) and apply the chosen resolution uniformly: one decision, shown its full
  scope before it is asked.
- **`from --archive`** would copy the input lifeboat verbatim into the voyage
  store before unpacking, for the case where the source repo will disappear. Off
  by default, because the source path and hash are enough while the source
  repo persists.
- **`from --refresh-audit`** would re-run the oracle product audit against the
  current lifeboat content and report the drift against the disembark-time
  audit.
- **Voyage provenance on embark**: an `embark/provenance.json` under the voyage
  store recording the source path, the source manifest hash, the timestamp and
  the files written (see [§ 7](#7-voyage-layout--embarkdisembark-provenance-and-history)).
- **`embark from-spec-kit`**, ingesting a GitHub Spec Kit project directory as
  starter draft intents (itd-23).
- **Workspace registration**: a lifeboat carrying an external tool's workspace
  file would prompt to register it where that tool is installed, and warn
  gracefully and continue where it is not.

## 7. Voyage layout — embark/disembark provenance and history

Lifeboat *operations* write provenance and history to
**`~/.abcd/voyage/<source-root-sha>/`**: the operator level, keyed on the
root-commit SHA exactly as the history store is, and therefore never committed
(adr-35, superseding adr-4's in-tree location). The lifeboat itself is written
out-of-tree to the operator-chosen destination
([`02-disembark.md § 5`](02-disembark.md#5-output-shape)) and holds only the
latest snapshot; it does not accumulate.

```
~/.abcd/voyage/<source-root-sha>/            ← operator level, keyed like the history store; never committed
├── embark/                                  ← not built yet
│   ├── provenance.json                      ← source path, manifest hash, timestamp, files written
│   └── from/<timestamp>/                    ← --archive: verbatim copy of input lifeboat (opt-in)
└── disembark/
    └── history.jsonl                        ← append-only manifest log of every disembark
```

Keying on the source repo's root-commit SHA is what lets voyage survive a
rename, a remote move, or the source repo being deleted entirely, and it is why
voyage may hold absolute source paths without ever putting them in a committed
file.

`disembark/history.jsonl` is append-only, one JSON object per run:

```json
{
  "schema_version": 2,
  "event": "disembark",
  "at": "2026-05-04T14:30:00Z",
  "manifest_sha256": "abc123...",
  "source_name": "abcd-cli",
  "source_root_sha": "def456...",
  "dest": "../abcd-lifeboat",
  "files": 214,
  "bytes": 1048576
}
```

> **The entry is deliberately minimal** (adr-35): it carries the manifest
> identity, the source name and root SHA, the destination, and the file and byte
> counts. Not a verdict, a label, or a recipient list. An empty field is a lie in
> a schema; add any of these when something actually populates it.

Manifests are small (a file list and hashes, not contents), so the log answers
"what did this repo's lifeboat look like at point T?" without keeping stale
snapshots around. Acceptance for the disembark-side write lives in
[`02-disembark.md § 7`](02-disembark.md#7-acceptance).
