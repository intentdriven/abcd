# Build Sequence

The order things get built, and what a user can do after each step. It is a
dependency DAG rather than a linear list: work is picked up in dependency order,
with parallelism where dependencies allow. The canonical intent set and its
bundling into product phases live in the phase docs and the intent index — see
[`roadmap/phases/README.md`](../../roadmap/phases/README.md) and
[`intents/README.md`](../../intents/README.md). This file is the
**build-milestone** detail: what each milestone stands up in the Go core, the
adapters, and the front doors.

Two commitments shape every milestone. abcd ships as a Go binary
([adr-21](../../decisions/adrs/0021-rebuild-in-go.md)) with a
transport-agnostic core
([adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)) behind thin front
doors; and **no external tool is a hard dependency**
([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)), so
every capability after the first milestone is a native default behind an
already-wired interface, with an optional external adapter beside it.

## 0. Go scaffold and the core/adapter skeleton

Stand up the module layout before any command logic: the entrypoint, one core
package per capability, the adapter directory for the pluggable seams, and the CLI
front door. The plugin manifest and the markdown surfaces that shell to the binary
load cleanly.

A seam is stood up by the milestone that first consumes it, never on day one:
"wired or it isn't done" forbids the stub-interface scaffold this milestone was
first drawn around. The scanner is the one seam standing — the native secret and
PII scan, with an external backend as its config-selected plug-in. The oracle,
history, spec and run seams, and the wiring packages beside them, are planned
rather than present, and the claim is gated rather than trusted: the `index_drift`
record-lint rule holds every path in the planned-seams region of
[`internal/README.md`](../../../../internal/README.md) to being absent from the
tree, so a seam that ships cannot go on being described as planned. History and
spec ship as core packages with no adapter interface behind them yet.

## 1. Install (`ahoy`) and launch (curated release)

The first user-visible milestone. After it, a person can adopt abcd in a project
and see what a release would ship — and it proves the CLI front door reaches the
core and the packaging boundary holds.

- **Install**: `abcd ahoy install` end to end. It is idempotent, carrying its
  configuration as flags rather than a wizard, writing the config file and the
  per-surface records, applying the visibility-driven gitignore policy, injecting
  the conventions marker block and the rules loader (itd-3), and bootstrapping the
  user-scope history store. There is no `abcd init` and no config get/set pair:
  install is the write path a person reaches for, and the bare invocation,
  its `--dry-run`, `--identity` and `--remote` modes, and `doctor` are the
  read-only halves. Two further forms write: `uninstall` takes abcd back out again, and
  `remote apply` turns on the forge's own secret scanning. The full surface is
  the machine-checked table in
  [`04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md); this milestone is what
  install has to do, not the whole verb.
- **Launch**: `/abcd:launch` prepares a **curated release** from the single repo
  ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) and never
  publishes one. `abcd launch --dry-run` previews the bundle read-only, `abcd
  launch ship` derives the version and writes the dated changelog heading, and a
  workflow turns that heading into a tag and cuts the release. Packaging excludes
  the whole `.abcd/` namespace from the artefact, and a dry-run proves nothing
  under it leaks. There is no dev-to-public mirror: the repo is the marketplace.

## 2. Native history, capture, memory

- **history seam**: the native redacted transcript store
  ([adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)), keyed on the
  repo's root commit and redacted on capture before anything lands on disk. It
  lives outside every checkout at the user level, so a repo tracks none of it;
  pulling the store into a checkout, where the gitignored local tier holds it, is
  a per-machine opt-in. An external transcript tool is an opt-in import over the
  same store. This is the research and benchmark corpus abcd studies its own
  flows against.
- **capture**: the issue ledger (itd-4), so an observation reaches a durable
  record in one line.
- **memory**: the curated knowledge substrate (itd-36); a vendor memory harvest
  is an opt-in, read-only source over it.

## 3. Intent, brief, and review through the host-delegated oracle

- **intent**: `/abcd:intent` (itd-1, itd-27, itd-34), with press-release
  composition. Creation is bare quoted text; the shipped sub-verbs are the machine-checked table in
  [`../04-surfaces/05-intent.md`](../04-surfaces/05-intent.md). Shipping runs the
  other way round: an intent moves to `shipped/` as the close-hook of `abcd spec
  close` — on the close after which no open spec names it, since an intent owns
  one or more specs — so there is no `intent ship`. `grill` is a design target (itd-27); the
  admission gauntlet that ships is `/abcd:ideate`.
- **review**: the oracle seam, **host-delegated by default**
  ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)): abcd emits
  a prompt, the host's subagent dispatch runs it, abcd consumes the structured
  result. Native, CLI, API and MCP adapters are opt-in for an operator who wants
  abcd to reach a model directly; the default install needs no API keys.
- **MCP front door** *(design target)*: a second thin door over the unchanged
  core (adr-23). `internal/surface/` holds one door, `cli`; the second is added
  once a surface is worth exposing, and the core needs no rework for it because it
  is transport-agnostic already.

## 4. Native minimal spec engine

- **spec seam**: the native minimal store
  ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)):
  directory-as-truth
  ([adr-3](../../decisions/adrs/0003-directory-as-truth-for-lifecycle.md)) plus a
  dependency graph over specs and tasks, enough to plan, sequence and track work
  with no external tool. Directory-as-truth ships: a spec's status is its folder,
  and `abcd spec close` moves it and ships its linked intent once no open spec is
  left naming that intent. The dependency graph
  and the sequencing over it are a **design target**; readiness today is
  per-intent, through `abcd intent ready`.
- **A companion-harness backend** *(design target)*: read and written at the
  **convention level**
  ([adr-24](../../decisions/adrs/0024-companion-harness-peer-via-conventions-and-mcp.md)),
  a peer over conventions and MCP, never a code dependency.

## 5. Autonomous run seam *(design target)*

The `run` seam
([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)): iterate
ready work, gate each step on a **receipt**, enforce a **safety guard**. The thin
native Go loop is to be the always-available fallback, with host workflow engines
and a companion agent loop as opt-in adapter loops behind the same contract. The
receipt-gated, report-not-block iteration boundary is the seam contract every
adapter loop inherits.

Nothing of this ships: the binary registers no `run` verb, the run adapter is on
the planned-seams list, and the operator surface over it is itd-29, in
`intents/planned/`.

## 6. Lifeboat round-trip

- **probe before pack**: `abcd disembark probe` ships **before a packer exists at
  all** ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)).
  It reports one repository per run, and `abcd disembark coverage` folds those
  reports into the cross-repo table over a corpus of mixed record quality. The
  section list that survives that aggregate is what the packer is then built to.
  A blank section is a first-class result rather than a failure.
- **disembark**: the pack reads the source repo's settled artefacts through the
  source readers, synthesises the lifeboat at the operator-chosen destination, and
  runs the host-delegated audit. The source repo is **never written to**, so a
  pack can be pointed at a dead or archived project abcd has never installed into.
  Operations live at the operator level. Writing to the destination passes the
  **safety gate**: absent, empty, or carrying a provenance file abcd itself wrote.
- **embark**: scaffold a target repo from a lifeboat, read from wherever
  disembark landed it.
- The **round-trip**: disembark on a corpus repo, embark into an empty target —
  is the integration milestone that exercises every seam end to end.

## Validation cadence

After **every milestone**, run disembark against each repo of the validation
corpus, or the relevant read-only preview sub-verb: `probe` for one repository's
section coverage, `coverage` for the aggregate across them, `plan` for the full
file set a pack would write without writing anything. There is no default
destination and no in-tree home; the corpus repos are read, never written.

Catch regressions early, and read the **coverage aggregate** that `coverage`
folds those probe reports into, which is the experiment's own readout (adr-35).
Acceptance is recorded in the gitignored local tier, which a 2026-07-12
adjudication made the home for runtime output (iss-73). `.abcd/logbook/` is a
retired location, held retired by an armed detector:
`TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source
under `internal/` so much as names it.
