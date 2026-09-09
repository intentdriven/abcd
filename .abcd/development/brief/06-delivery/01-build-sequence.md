# Build Sequence

The build follows the delivery order **MVP → the companion harness → Claude Code**, with
**install + launch first**. It is a dependency DAG, not a linear list: work is
picked up in dependency order, with parallelism where dependencies allow, which
is what the autonomous `run` seam of milestone 5 is designed to automate. The
canonical intent set and its bundling into product
phases live in the phase docs and the intent index — see
[`roadmap/phases/README.md`](../../roadmap/phases/README.md) and
[`intents/README.md`](../../intents/README.md); this file is the
canonical **build-milestone** detail (what each milestone stands up in the Go
core, the adapters, and the front doors).

abcd ships as a Go binary ([adr-21](../../decisions/adrs/0021-rebuild-in-go.md))
with a transport-agnostic core
([adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)) behind thin front
doors, and **no external tool as a hard dependency**
([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). The
first milestone is **install + launch**; every capability after it is a native
default behind an already-wired interface, with an optional external adapter.

## 0. Go scaffold + core/adapter skeleton

Stand up the module layout before any command logic:
`cmd/abcd/main.go`; `internal/core/` (one package per capability);
`internal/adapter/` for the pluggable seams; `internal/surface/cli` (Cobra). The
plugin manifest (`.claude-plugin/plugin.json`, `marketplace.json`) and the
markdown surfaces (`commands/`, `agents/`) that shell to the binary load cleanly.

A seam is stood up by the milestone that first consumes it, never on day one:
"wired or it isn't done" forbids the stub-interface scaffold this milestone was
first drawn around. `internal/adapter/scanner` is the one seam standing — the
native secret/PII scan, with `internal/adapter/gitleaks` as its config-selected
external plug-in. The oracle, history, spec, and run seams, and the `registry/`
and `config/` wiring packages beside them, are planned rather than present, and
the claim is gated rather than trusted: the `index_drift` record-lint rule holds
every path in the planned-seams region of
[`internal/README.md`](../../../../internal/README.md) to being absent from the
tree, so a seam that ships cannot go on being described as planned. History and
spec ship as core packages (`internal/core/history`, `internal/core/spec`) with
no adapter interface behind them yet.

## 1. Install (`ahoy`) + launch (curated release)

The first user-visible milestone, proving the CLI front door reaches the core and
the packaging boundary holds.

- **Install** — `/abcd:ahoy` end-to-end: `abcd ahoy install`, which is idempotent
  and carries the configuration as flags (`--visibility`, `--docs-target`,
  `--oracle-backend`, `--scan-deep`, `--bin-dir`, `--dev`, `--attribution`)
  writing `.abcd/config.json` and `.abcd/config/`; the visibility-driven gitignore
  policy; the CLAUDE.md/AGENTS.md marker block + rules loader (itd-3); and the
  user-scope history store bootstrap under `~/.abcd/history/`. There is no
  `abcd init` and no `config get|set` pair: install is the one write path, and
  `abcd ahoy` bare, `ahoy doctor`, and `ahoy dry-run` are the read-only halves.
- **Launch** — `/abcd:launch` prepares a **curated GitHub Release** from the
  single repo ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md))
  and never publishes one: `abcd launch --dry-run` previews the bundle read-only,
  `abcd launch ship` derives the version and writes the dated `CHANGELOG.md`
  heading, and `.github/workflows/auto-release.yml` is what turns that heading
  into a tag and cuts the Release. Packaging **excludes `.abcd/**` from the
  release artifact** (a dry-run proves nothing under `.abcd/` leaks). There is
  **no dev→public mirror** — the repo is the marketplace.

## 2. Native history, capture, memory

- **history seam** — the native local redacted transcript store
  ([adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)): root-SHA-keyed,
  gitignored, redacted on capture with the two-stage redaction model.
  specstory is an opt-in import over the same store. This is the research and
  benchmark corpus abcd studies its own flows against.
- **capture** — `/abcd:capture` issue ledger (itd-4) into the
  `.abcd/work/issues/` ledger.
- **memory** — the `.abcd/memory/` curated substrate (itd-36); vendor memory
  harvest is an opt-in, read-only source over it.

## 3. Intent + brief + review via host-delegated oracle (+ MCP front door)

- **intent** — `/abcd:intent` (itd-1, itd-27, itd-34), with brief and
  press-release composition. Creation is bare quoted text (`abcd intent
  "<text>"`, with `new` kept as a deprecated alias); the shipped sub-verbs are
  `plan`, `link`, `ready`, and `audit`. Shipping runs the other way round: an
  intent moves `planned/` → `shipped/` as the close-hook of `abcd spec close
  <spc-N>`, so there is no `intent ship`. `grill` is a design target (itd-27); the
  admission gauntlet that ships is `/abcd:ideate`.
- **review** — the oracle seam, **host-delegated by default**
  ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)): abcd emits a
  prompt, the host's subagent dispatch runs it, abcd consumes the structured
  result. Native / CLI / API / MCP oracle adapters are opt-in for an operator who
  wants abcd to reach a model directly; the default install needs no API keys.
- **MCP front door** *(design target)* — enable `internal/surface/mcp` as a
  second thin door over the unchanged core
  ([adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)).
  `internal/surface/` holds one door, `cli`; the second is added once a surface is
  worth exposing, and the core needs no rework for it because it is
  transport-agnostic already.

## 4. Native minimal spec engine + ccpm

- **spec seam** — the native minimal store
  ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)):
  directory-as-truth ([adr-3](../../decisions/adrs/0003-directory-as-truth-for-lifecycle.md))
  plus a dependency graph over specs and tasks — enough to plan, sequence, and
  track work with no external tool. Directory-as-truth ships:
  `internal/core/spec` reads a spec's status from its folder, and `abcd spec
  close <spc-N>` moves it and ships its linked intent. The dependency graph and
  the sequencing over it are a design target; readiness today is per-intent,
  through `abcd intent ready <itd-N>`.
- **ccpm backend** *(design target)* — the companion harness `ccpm` as the
  primary deeper backend, read and
  written at the **convention level**
  ([adr-24](../../decisions/adrs/0024-companion-harness-peer-via-conventions-and-mcp.md)) —
  a peer over conventions + MCP, never a code dependency. **flow-next is not
  built.**

## 5. Autonomous run seam *(design target)*

The `run` seam
([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)): iterate
ready work, gate each step on a **receipt**, enforce a **safety guard**. The thin
native Go loop is to be the always-available fallback; Claude Workflows and the
companion harness's agent loop are opt-in adapter loops behind the same seam
contract. It is **not a Ralph port** — the receipt-gated, report-not-block
iteration boundary is the seam contract every adapter loop inherits.

Nothing of this ships: the binary registers no `run` verb, `internal/adapter/run`
is on the planned-seams list, and the operator surface over it
(`run status`/`pause`/`resume`/`preflight`) is itd-29 in `intents/planned/`.

## 6. Lifeboat round-trip

- **probe before pack** — `abcd disembark probe <source-repo>` ships **before a
  packer exists at all**
  ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)): it
  produces a coverage report over a corpus of repos of mixed record quality, and
  the section list that survives that aggregate is what the packer is then built
  to. A `blank` section is a first-class result, not a failure.
- **disembark** — `abcd disembark pack <source-repo> <dest>`: the lifeboat
  pipeline
  (Pass A/B/C) reads the **source repo's** settled artefacts through the source
  readers ([`../05-internals/02-adapters.md`](../05-internals/02-adapters.md))
  over the native spec / history / memory stores, synthesises the lifeboat **at
  the operator-chosen `<dest>`**, and runs the host-delegated oracle audit. The
  source repo is **never written to** — disembark is read-only and out-of-tree,
  so it can be pointed at a dead or archived project abcd has never installed
  into. Operations live at the operator level under
  `~/.abcd/voyage/<source-root-sha>/`, never in the source tree. Writing to
  `<dest>` passes the **destination safety gate**: absent, an empty directory, or
  one carrying a parseable `_provenance.json` — abcd never overwrites a directory
  it did not produce.
- **embark** — scaffold a target repo from a lifeboat, read from wherever
  disembark landed it.
- The **round-trip** (disembark on a corpus repo → embark into an empty target)
  is the integration milestone that exercises every seam end-to-end.

## Validation cadence

After **every milestone**, run `/abcd:disembark <corpus-repo> <dest>` (or the
relevant preview sub-verb: `probe` for the read-only coverage report, `plan` for
the full file set a pack would write without writing anything) against each repo
of the validation corpus. There is no default destination and no in-tree home:
the corpus repos are read, never written. Catch regressions early — and read the **coverage aggregate**
across the corpus, which is the experiment's own readout
([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)).
Acceptance is recorded in the gitignored local tier, `.abcd/.work.local/logs/`,
which a 2026-07-12 adjudication made the home for runtime output (iss-73).
`.abcd/logbook/` is a retired location, held retired by an armed detector:
`TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source
under `internal/` so much as names it.
