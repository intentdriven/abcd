# `/abcd:disembark` — Pack a Lifeboat

> **Model of record: [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md).** The packer is read-only and out-of-tree (`disembark pack <source-repo> <dest>`; the source is never written to, and a test fingerprints its tree before and after), the voyage log lives at the operator level (`~/.abcd/voyage/<source-root-sha>/`, never committed), a destination safety gate refuses any directory abcd did not produce, and the review returns the registered `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` verdicts. The coverage experiment (itd-88) leads: the brief carries only what abcd could ground, each claim citing its source; `coverage.{json,md}` carry what is missing, what was searched, and the question a human must answer.

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the packer and the round-trip ship in [Phase 6](../../roadmap/phases/phase-6-lifeboat.md). The **coverage experiment (itd-88) is pulled out of Phase 6** and sequenced ahead of it, per adr-35.

> **Recovery humility.** Disembark packs the highest-fidelity proxy of the project's theory we can leave behind. It is not the theory. The theory of any non-trivial project lives in the people who built it, the conversations where decisions were made, and the alternatives they rejected before this one — what Naur (1985) called the lived activity of building. The lifeboat is the floor we can carry forward across a session, machine, or team boundary; it is not the activity itself. See [`01-product/03-mental-model.md § The Naurian gap`](../01-product/03-mental-model.md#the-naurian-gap--modification-axis) for the framing.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

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


Bare `/abcd:disembark` prints the sub-verb list and flags only — never mutates state. Current sub-verbs (as the shipped binary exposes them):

- **`/abcd:disembark pack <source-repo> <dest>`** — pack a lifeboat from `<source-repo>` to `<dest>`. Both paths are positional and required, and there is **no `home` shorthand**: the lifeboat lands out-of-tree at an operator-chosen destination and the source repo is never written to (adr-35). Its deterministic flow is described in § 1 below.
- **`/abcd:disembark probe [source-repo]`** — read-only inspection of the source (the repo argument is optional and defaults to the current directory): which brief sections it can ground, which come back blank, and what was searched. Renders the coverage report to stdout and writes nothing into `<source-repo>`; `coverage.{json,md}` are written only by `pack` (adr-35). It is the coverage experiment's readout (itd-88).
- **`/abcd:disembark plan [source-repo]`** — the dry-run (the repo argument is optional and defaults to the current directory): the full lifeboat file set a pack would write, without writing anything.
- **`/abcd:disembark coverage <report.json>…`** — aggregate probe reports into the cross-repo section×repo coverage table.
- **`--include-ignored`** on `pack`, `probe`, and `plan` — also read files git ignores. It widens the scan, and the report says so.
- **Synthesis sub-verbs over a packed lifeboat** — `graveyard` (validate host-produced lesson JSON and write the survivors, cite-or-be-dropped), `review` (assess a packed lifeboat against its source repo — a registered verdict + cited findings), `press-release` (compose the lifeboat's press release), and `principles` (distil principles from the ADRs). Each is deterministic-or-validate-host-JSON.
- **Later phase: `to-spec-kit <path>`** — export shipped intents to GitHub Spec Kit format alongside the lifeboat (per itd-23); not yet shipped.

## 1. Architecture (a single deterministic pass)

`disembark pack <source-repo> <dest>` is one deterministic Go run. It reads the source (never writing to it) and lands every byte under `<dest>`:

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

The pack dispatches no agents and runs no LLM passes. The synthesis artefacts — `press-release.{json,md}`, `principles.{json,md}`, the review artefact, and the validated graveyard lessons — are written by the separate synthesis sub-verbs run over an already-packed lifeboat, each deterministic-or-validate-host-JSON.

## 2. Recency rule

Hard rule: later structural artefacts supersede earlier ones — spec numbers, ADR `Superseded-By` headers, file mtimes, git log. **Never resolve recency semantically.**

When structure and content disagree (e.g., a later spec mentions an earlier decision approvingly without restating it), the shipped pack records the structural answer only. *(Design target, not yet shipped: route such disagreements to a transcript-reading distiller agent that emits a finding for the unrecorded-decisions report — viability gated on itd-11 (draft); transcript signal density per `../../research/notes/transcript-sampling.md`.)*

## 5. Output shape

`disembark plan [source-repo]` lists the file set a pack writes, and a pack that completes writes that same tree to `<dest>`. The two agree on paths, not on outcome: the secret scan is a Pack-only seam (`SecretScan`, injected so the lifeboat core stays free of the scanner adapter, and mandatory — a pack called without one refuses), so `plan` never runs it and its output carries no signal that the pack will refuse. A tree that plans cleanly and holds a hard-fail secret in planned content packs to nothing and exits non-zero. Each path derives from the brief-section → lifeboat-path mapping in `internal/core/lifeboat/mapping.go`: a section grounds into its file where the source supports it, and `coverage.{json,md}` record every section that stays blank, what was searched, and the question a human must answer. `graveyard/` is a section of its own (adr-35).

```
<dest>/                                 # operator-chosen, outside the source repo (adr-35)
├── _provenance.json                    # the lifeboat marker and re-pack gate key (written last): schema_version, generator, source name and root SHA, tiers present, manifest_sha256 over every other file, record_manifest_sha256 over the record-derived families alone (docs/adrs, activity/issues, rescue/intents, rescue/specs, graveyard/abandoned.json), omissions, pass_b_exemption (present only when no transcript tier grounded the package, so an unmarked lifeboat marshals as it always has; embark reads it and says so)
├── coverage.json                       # per-section status (grounded|partial|blank), evidence, what was searched
├── coverage.md                         # rendered
├── brief/                              # the brief, section by section, grounded from the source
│   ├── 01-product/ … 06-delivery/      # press-release, context, mental-model, scope, personas, framing; constraints; evidence; surfaces; internals; delivery
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

The synthesis sub-verbs add the rest over an already-packed lifeboat: `press-release` writes `press-release.{json,md}`, `principles` writes `principles.{json,md}`, `review` writes the verdict artefact, and `graveyard` validates and writes the lesson JSON. None of these exist at pack time.

The lifeboat is written out-of-tree to `<dest>` (adr-35), so the source repo has nothing to gitignore.

## 6. Per-phase acceptance

Each phase passes when **both gates** succeed:

1. **Review gate**: the `lifeboat-reviewer` review on phase outputs returns a registered verdict (`SHIP` / `NEEDS_WORK` / `MAJOR_RETHINK`, per `05-internals/01-agents.md § Verdict-tag protocol`) with specific findings (not vague approval); the gate passes on `SHIP`. The review reaches a model through the oracle seam, **host-delegated by default** (per [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)); an opt-in oracle adapter (native / CLI / API / MCP) runs it when wired — never blocks.
2. **Round-trip gate**: stage outputs feed cleanly into the next stage's expected inputs (e.g., a packed lifeboat verifies against its manifest and is consumed by the synthesis sub-verbs — `review`, `principles`, `press-release`, `graveyard` — without parse errors).

Acceptance is checked across the validation corpus (`.abcd/corpus.json` — a design target; the file is not yet in the tree), with documented per-repo exemptions where a feature genuinely doesn't apply.

## 7. Acceptance

> **Open question (adr-35):** the shipped `history.jsonl` line records schema version, event, timestamp, `manifest_sha256`, source name, source root SHA, destination, file count, and bytes — so the destination a bare-invocation readout would need is already persisted. **Decide:** does bare invocation resolve `<source-root-sha>` from the cwd (as `probe` and `plan` already default to it) and read the voyage log? The design-target marker on the first bullet below is gated on that decision.

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:disembark`, **then** the dispatcher prints the available sub-verbs (`pack <source-repo> <dest>`, `probe`, `plan`, `coverage`, `review`, `graveyard`, `press-release`, `principles`; later phase: `to-spec-kit`) and flags only — bare invocation never mutates state. *(Design target, gated on the adr-35 open question above: once the `history.jsonl` line's destination field and the cwd `<source-root-sha>` resolution are decided, bare invocation also reads the voyage log at `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl` and shows when the source last disembarked and where that snapshot was written, plus suggested next actions.)*
- **Given** any source repo, **when** any sub-verb runs against it (`probe`, `plan`, or a full pack), **then** the source tree is unchanged afterwards (adr-35: disembark is read-only). The evidence is a fingerprint, not a content hash: the test accumulates every entry's relative path, mode and size under the source before and after the run and asserts the two match, skipping `.git`, whose internal bookkeeping is not the source of truth. Two mutations therefore sit outside the assertion's sight: a rewrite that preserves a file's size, and any write under `.git`. There is no `home`, and no path under `<source-repo>` is ever a destination.
- **Given** a corpus repo with an intent corpus, ADRs, and a memory backend present, **when** `/abcd:disembark pack <source-repo> <dest>` runs to completion, **then** `<dest>` contains all sections in [§ 5](#5-output-shape) and the review returns a registered verdict of `SHIP` with specific findings.
- **Given** a corpus repo where one source is sparse (e.g., a repo with no intent corpus), **when** a full pack runs, **then** the run **succeeds**: the affected section is omitted from the brief rather than fabricated, and `coverage.{json,md}` records it with `status: blank`, what was searched, and the question a human must answer — a blank is a first-class result, not a failure or an exemption footnote (adr-35).
- **Given** the user runs `/abcd:disembark probe <source-repo>`, **when** the command completes, **then** all adapters' `probe()` runs in parallel, the coverage report is rendered to stdout (text, or JSON with `--json`) with every brief section marked `grounded` / `partial` / `blank` plus what was searched, nothing is written into `<source-repo>`, and the run takes a small fraction of the time a full pack would take (no LLM dispatches).
- **Given** `probe` is run across the validation corpus, **when** the per-repo coverage reports are aggregated, **then** the aggregate reports the section-coverage delta between a rich-record repo and a git-only repo — the experiment's readout, and the evidence the packer's section list is built to (itd-88, adr-35).
- **Given** the user runs `/abcd:disembark plan`, **when** the command completes, **then** the source inventory runs end-to-end, the would-be writes are listed as file paths, and nothing is written to `<source-repo>` or `<dest>`.
- **Given** a destination that is neither absent, nor an empty directory, nor one carrying a parseable `_provenance.json`, **when** a pack targets it, **then** the run **refuses** and writes nothing — abcd never overwrites a directory it did not produce (adr-35's destination safety gate; there is no `.bak`).
- **Given** any pack run completes, **when** the lifeboat is written, **then** a new line is appended to `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl` with the run's `schema_version`, event, timestamp, `manifest_sha256`, source name, source root SHA, destination, file **count**, and bytes written (per [`03-embark.md § 7`](03-embark.md#7-voyage-layout--embarkdisembark-provenance-and-history)). The line carries no oracle backend or verdict — nothing produces them yet, and an empty field would be a lie (adr-35); its `manifest_sha256` matches the lifeboat's own `_provenance.json`, whose hash pins every other packed file.
- **Given** a `<dest>` carrying a parseable `_provenance.json` from a previous run, **when** a new snapshot lands there, **then** the previous snapshot is replaced AND its manifest remains in the voyage log — there is never a `lifeboat-v1/` / `lifeboat-v2/` directory; history is preserved in the manifest log, not in stale snapshots.
