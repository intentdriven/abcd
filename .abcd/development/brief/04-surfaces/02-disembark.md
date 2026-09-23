# `/abcd:disembark` — Pack a Lifeboat

Leave a project in a form somebody can pick up: point the packer at any
repository and it writes, elsewhere, the highest-fidelity account of that
project's theory it can ground from what is on disk. Every claim in the pack
cites the source it came from, and everything it could **not** ground is
reported rather than invented, as a named blank with the question a human must
answer.

Three properties make it safe to run on a repository you care about. It never
writes to the source, and a test fingerprints the source tree before and after
to prove it. It refuses any destination abcd did not produce, so it cannot
overwrite someone's directory. And it scans the planned bytes for secrets
before writing any of them, refusing the whole pack rather than redacting.

> **Model of record: [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md).** The packer is read-only and out-of-tree, the voyage log lives at the operator level (`~/.abcd/voyage/<source-root-sha>/`, never committed), and the review returns the registered `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` verdicts. The coverage experiment (itd-88) leads: the pack carries only what abcd could ground, and `coverage.{json,md}` carry what is missing, what was searched, and the question a human must answer.

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the packer and the round-trip ship in [Phase 6](../../roadmap/phases/phase-6-lifeboat.md). The coverage experiment is pulled out of Phase 6 and sequenced ahead of it, per adr-35.

> **Recovery humility.** The lifeboat is the highest-fidelity proxy of a project's theory we can leave behind. It is not the theory. The theory of any non-trivial project lives in the people who built it, the conversations where decisions were made, and the alternatives they rejected before this one — what Naur (1985) called the lived activity of building. The lifeboat is the floor we can carry across a session, machine, or team boundary. See [`01-product/03-mental-model.md § The Naurian gap`](../01-product/03-mental-model.md#the-naurian-gap--modification-axis).

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
| `coverage` | — | shipped |
| `graveyard` | — | shipped |
| `review` | review | shipped |
| `pack` | — | shipped |
| `plan` | — | shipped |
| `press-release` | — | shipped |
| `principles` | — | shipped |
| `probe` | — | shipped |


Bare `/abcd:disembark` prints the sub-verb list and flags, and mutates nothing.
The verbs divide into three jobs.

**Look before you pack.** The probe reports, read-only, which brief sections the
source can ground, which come back blank, and what was searched; it is the
coverage experiment's readout, and it writes nothing anywhere. The plan is the dry
run: the full file set a pack would write, without writing it. Both take the
source repo as an optional argument defaulting to the current directory, and
both can widen the scan to files git ignores, which the report then says.

**Pack.** Packing takes the source repo and the destination and writes the
lifeboat. Both paths are positional and required, and there is no shorthand for
either: the lifeboat lands out-of-tree at a destination the operator chose, and
the source is never written to. It can widen its scan to ignored files as well,
exactly as the probe and the plan do.

**Synthesise over an already-packed lifeboat.** The press release, the
principles and the review each run in one of two modes: deterministic from the
packed evidence, or validating a host-produced JSON payload passed on a flag.
The graveyard has the validating mode alone and asks for its payload by name when
none is given: what an abandoned approach taught is not something the packed
files can be read for deterministically, so there is no second mode to fall back
on. The validating mode is cite-or-be-dropped, and the review carries the
registered verdict. The coverage aggregate is cross-repo: hand it probe reports
and it returns the section-by-repo table.

**Not built yet:** `to-spec-kit`, which would export shipped intents to GitHub
Spec Kit format alongside the lifeboat (itd-23).

## 1. Architecture (a single deterministic pass)

Packing is one deterministic Go run. It dispatches no agents and runs
no model passes: the synthesis artefacts are written later, by the synthesis
sub-verbs, over an already-packed lifeboat.

```
INVENTORY (read-only)
  walk the source and its record families → the planned lifeboat file set,
  each brief section grounded from the source per internal/core/lifeboat/mapping.go,
  plus a per-section coverage report (grounded | partial | blank)
                           │
                           ▼
DESTINATION SAFETY GATE
  refuse unless <dest> is absent, empty, or carries a parseable _provenance.json
  → never overwrite a directory abcd did not produce (adr-35)
                           │
                           ▼
SECRET SCAN (before any write)
  scan the planned bytes; a hard-fail secret refuses the whole pack — never redact
                           │
                           ▼
WRITE
  write to a staging directory, then swap it into <dest>; _provenance.json is
  written last — the commit marker and the gate key for a later re-pack
                           │
                           ▼
VOYAGE LINE (operator-local)
  append one line to ~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl;
  a failed append never fails the pack — the written _provenance.json is authoritative
```

## 2. Recency rule

Later structural artefacts supersede earlier ones: spec numbers, ADR
`Superseded-By` headers, file modification times, git log. **Never resolve
recency semantically.** When structure and content disagree, the shipped pack
records the structural answer only.

*Not built yet: routing such disagreements to a transcript-reading distiller
that emits a finding for the unrecorded-decisions report. Viability is gated on
itd-11 (draft); transcript signal density is measured in
`../../research/notes/transcript-sampling.md`.*

## 5. Output shape

The plan lists the file set a pack writes, and a pack that completes writes that
same tree. **The two agree on paths, not on outcome.** The secret scan is a
pack-only seam, injected so the lifeboat core stays free of the scanner adapter,
and mandatory: a pack called without one refuses. So the plan never runs it, and
its output carries no signal that the pack will refuse. A tree that plans
cleanly and holds a hard-fail secret in planned content packs to nothing and
exits non-zero.

Each path derives from the brief-section-to-lifeboat-path mapping in
`internal/core/lifeboat/mapping.go`: a section grounds into its file where the
source supports it, and `coverage.{json,md}` record every section that stays
blank, what was searched, and the question a human must answer. The graveyard is
a section of its own.

```
<dest>/                                 # operator-chosen, outside the source repo (adr-35)
├── _provenance.json                    # the lifeboat marker and re-pack gate key, written last
├── coverage.json                       # per-section status (grounded|partial|blank), evidence, what was searched
├── coverage.md                         # rendered
├── brief/                              # the brief, section by section, grounded from the source
│   ├── 01-product/ … 06-delivery/
│   └── glossary/
├── graveyard/                          # what the project tried and abandoned
│   ├── abandoned.json
│   └── archaeology.json
├── rescue/                             # the spine: the intent corpus where one exists, else the commit history
│   ├── spine.md                        # commit-history spine, written where no record store exists
│   ├── intents/{drafts,planned,shipped,superseded,disciplines}/   # intent corpus, verbatim
│   └── specs/{open,closed}/            # spec store, verbatim
├── docs/
│   └── adrs/                           # ADRs copied verbatim
└── activity/
    └── issues/{open,resolved,wontfix}/ # curated issue ledger snapshot
```

`_provenance.json` is what makes the pack checkable by a third party. It carries
the schema version and generator, the source name and root SHA, the tiers
present, a `manifest_sha256` over every other file, a `record_manifest_sha256`
over the record-derived families alone, the omissions, and a `pass_b_exemption`
present only when no transcript tier grounded the package, so an unmarked
lifeboat marshals as it always has and embark can say which it is.

The synthesis sub-verbs add the rest afterwards: the press release writes
`press-release.{json,md}`, the principles write `principles.{json,md}`, the review
writes the verdict artefact, and the graveyard validates and writes the lesson
JSON. None of these exist at pack time.

The lifeboat is written out-of-tree, so the source repo has nothing to
gitignore.

## 6. Per-phase acceptance

Each phase passes when **both gates** succeed.

1. **Review gate**: the `lifeboat-reviewer` review on phase outputs returns a
   registered verdict with specific findings rather than vague approval, and the
   gate passes on `SHIP`. The review reaches a model through the oracle seam,
   host-delegated by default (per
   [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)); an opt-in
   oracle adapter runs it when wired, and never blocks.
2. **Round-trip gate**: stage outputs feed cleanly into the next stage's
   expected inputs. A packed lifeboat verifies against its manifest and is
   consumed by the synthesis sub-verbs without parse errors.

Acceptance is checked across a validation corpus with documented per-repo
exemptions where a feature genuinely does not apply. *The corpus manifest
(`.abcd/corpus.json`) is not yet in the tree.*

## 7. Acceptance

> **Open question (adr-35):** the shipped voyage line records enough for a bare
> invocation to say when the source last disembarked and where, so the question
> is whether bare invocation resolves the source root SHA from the working
> directory (as the probe and the plan already default to it) and reads that log.
> The first criterion below is gated on that decision.

- **Given** any abcd-aware terminal, **when** the user runs bare
  `/abcd:disembark`, **then** the dispatcher prints the available sub-verbs and
  flags only, and mutates nothing. *(Gated on the open question above: once
  resolution from the working directory is decided, bare invocation also reads
  the voyage log and shows when the source last disembarked and where, plus
  suggested next actions.)*
- **Given** any source repo, **when** any sub-verb runs against it, **then** the
  source tree is unchanged afterwards. The evidence is a fingerprint, not a
  content hash: the test accumulates every entry's relative path, mode and size
  under the source before and after the run and asserts the two match, skipping
  `.git`, whose internal bookkeeping is not the source of truth. Two mutations
  therefore sit outside the assertion's sight: a rewrite that preserves a file's
  size, and any write under `.git`. No path under the source repo is ever a
  destination. *(The fingerprint is asserted for the probe, the plan and the pack,
  which are the three sub-verbs that open the source at all. The review also takes
  a source repo, but only to check that it is a real directory and to take its
  name for the attestation: the content is never read, so there is nothing for a
  fingerprint to catch.)*
- **Given** a corpus repo with an intent corpus, ADRs, and a memory backend
  present, **when** a full pack runs to completion, **then** the destination
  contains all sections in [§ 5](#5-output-shape) and the review returns a
  registered verdict of `SHIP` with specific findings.
- **Given** a corpus repo where one source is sparse, **when** a full pack runs,
  **then** the run **succeeds**: the affected section is omitted from the brief
  rather than fabricated, and coverage records it as blank with what was
  searched and the question a human must answer. A blank is a first-class
  result, not a failure or an exemption footnote.
- **Given** the user runs the probe, **when** it completes, **then** every
  adapter's probe runs in parallel, the coverage report is rendered to stdout
  with each section marked grounded, partial or blank plus what was searched,
  nothing is written into the source, and the run takes a small fraction of the
  time a full pack would.
- **Given** the probe run across the validation corpus, **when** the reports are
  aggregated, **then** the aggregate reports the section-coverage delta between
  a rich-record repo and a git-only repo: the experiment's readout, and the
  evidence the packer's section list is built to (itd-88, adr-35).
- **Given** the user runs the plan, **when** it completes, **then** the source

  inventory runs end to end, the would-be writes are listed as file paths, and
  nothing is written to the source or the destination.
- **Given** a destination that is neither absent, nor an empty directory, nor
  one carrying a parseable `_provenance.json`, **when** a pack targets it,
  **then** the run **refuses** and writes nothing. abcd never overwrites a
  directory it did not produce, and there is no backup copy.
- **Given** any pack run completes, **when** the lifeboat is written, **then** a
  line is appended to the voyage log carrying the run's schema version, event,
  timestamp, `manifest_sha256`, source name, source root SHA, destination, file
  count and bytes written (per
  [`03-embark.md § 7`](03-embark.md#7-voyage-layout--embarkdisembark-provenance-and-history)).
  The line carries no oracle backend and no verdict, because nothing produces
  them yet and an empty field would be a lie. Its `manifest_sha256` matches the
  lifeboat's own `_provenance.json`, whose hash pins every other packed file.
- **Given** a destination carrying a parseable `_provenance.json` from a
  previous run, **when** a new snapshot lands there, **then** the previous
  snapshot is replaced and its manifest remains in the voyage log. There is
  never a versioned pair of snapshot directories: history is preserved in the
  manifest log, not in stale copies.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd disembark`

Sub-verbs: `abcd disembark coverage`, `abcd disembark graveyard`, `abcd disembark pack`, `abcd disembark plan`, `abcd disembark press-release`, `abcd disembark principles`, `abcd disembark probe`, `abcd disembark review`.

Flags: none.

### `abcd disembark coverage`

Sub-verbs: none.

Flags: none.

### `abcd disembark graveyard`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--lessons-json` | string |

### `abcd disembark pack`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--include-ignored` | bool |

### `abcd disembark plan`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--include-ignored` | bool |

### `abcd disembark press-release`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--press-release-json` | string |

### `abcd disembark principles`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--principles-json` | string |

### `abcd disembark probe`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--include-ignored` | bool |

### `abcd disembark review`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--review-json` | string |

<!-- surface-appendix:end -->
