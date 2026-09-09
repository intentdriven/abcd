---
id: spc-2609091722230648
slug: transcripts-already-on-disk-are-recovered-into-the-right-rep
intent: itd-2609091718566731
origin: researcher-authored
production_mode: hand-written
---
# The recovery half: transcripts on disk are ingested into a named destination, and composite records are repaired in place

## Summary

This spec delivers
[itd-2609091718566731](../../intents/planned/itd-2609091718566731-transcripts-already-on-disk-are-recovered-into-the-right-rep.md):
the corpus stops starting on the day it was fixed.

Two verbs land, both recovery, both reporting by construction. **`history
ingest`** brings transcripts that are on disk and were never captured into the
store of a repository the caller NAMES, so that repository's own redaction
configuration governs its own transcripts and can never be applied to another's.
**`history migrate`** repairs the records already in the store that were filed
under the pre-lineage composite session id, recovering the full parent session
from each record's own body.

Both rest on
[itd-2609090559376002](../../intents/planned/itd-2609090559376002-sub-agent-transcript-capture.md):
the lineage fields, the widened idempotency key and the fail-closed `Capture`
are that intent's, and this one adds no second write path beside them. The
per-repo root-SHA keying of
[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md) stands unchanged.

## Scope

In: `internal/core/history` — `Ingest` and its `Destination`, `IngestOptions`
and `IngestResult` (`ingest.go`); `LoadConfig` and `Config` (`config.go`);
`Migrate`, `MigrateOptions`, `MigrateResult` and the `LineageLookup` seam
(`migrate.go`). The `history` sub-tree in `internal/surface/cli`
(`history_recovery.go`, wired from `history.go`). The per-repo configuration
file `.abcd/config/history.json`. The plugin page `commands/history.md`, the
brief chapter [`04-surfaces/11-history.md`](../../brief/04-surfaces/11-history.md),
the regenerated surface snapshot and the generated command reference.

Out: the scanner and its detectors; the two-stage redaction discipline; the
store's keying and provisioning; the live capture path and its hook; the
harness's retention policy; reconstruction and telemetry.

## Approach

### The destination is an operand, and it is the whole security argument

```go
func Ingest(dest Destination, sources []string, opts IngestOptions) (IngestResult, error)

type Destination struct {
	RepoRoot string `json:"repo_root"`
	RootSHA  string `json:"root_sha"`
}
```

`Destination` carries the repository root and its root-commit SHA. The scanner
is constructed from `dest.RepoRoot` and from nothing else, so a repository's own
`pii.json` and `gitleaks.json` govern its own transcripts. This is a seam rather
than a default because the working directory is exactly the wrong authority
here: an operator recovering a backlog is not standing in the repository the
transcripts belong to, and deriving the destination from where they happen to be
would redact one repository's transcripts under another's rules — a privacy
fault, not a misfiling.

The CLI matches: `--into` is REQUIRED and has no default, the run prints which
repository it wrote into, and an empty value is a hard refusal that names
`--into .` as a perfectly good answer. `TestIngestRefusesWithoutAnExplicitDestination`
holds the core half and `TestHistoryIngestRefusesWithoutAnExplicitDestination`
the surface half; `TestHistoryIngestWritesIntoTheNamedRepositoryNotTheWorkingDirectory`
is the one that would catch a regression to a cwd default.

`sources` are explicit file or directory paths. There is no implicit "scan the
harness's store" mode: a vendor path baked into core would be the on-disk-layout
dependency the intent's mechanism rules out. A repository that ingests regularly
declares its roots in `.abcd/config/history.json` so the operator does not
retype them, and the configuration is the only place a path lives. Directory
sources are walked for `.jsonl` files to a bounded depth (`ingestDefaultDepth`,
8); a symlinked source is refused outright and a symlinked directory is never
descended; every read goes through the same guarded, capped reader the hook path
uses.

`IngestResult` reports four populations — `Captured`, `Skipped` (each with a
reason), `Orphans`, and `Failed` — and the `--json` envelope normalises all four
to arrays so a consumer never has to distinguish absent from empty. Nothing is
silent.

### Resolving the owning repository: per session, then per file

The design this spec was planned from resolved ownership per FILE. The corpus
refuted it before the code shipped, and the fix is
`iss-2609090723023943`: a sub-agent handed its own worktree records that
worktree as its `cwd`, the harness deletes the worktree when the agent stops, so
by recovery time the directory the transcript names does not exist. Resolving
per file orphans exactly the isolated implementation lanes — the transcripts
worth the most.

So the resolution runs a session-level pass first and a per-file pass only for
what it could not place:

1. **`session-cwd`.** Collect the distinct `cwd` values recorded on the
   MAIN-THREAD files of that session and resolve each through `ahoy.Detect`. One
   root SHA places the whole session, sub-agents included; two or more mark it
   ambiguous.
2. **`store`.** Failing that, the store itself: the session notes under
   `~/.abcd/history/<sha>/sessions/` and the `session_id` of every stored record
   are indexed once per run, and a session exactly one store claims is placed by
   it. This is the rung that recovers a session whose directories are all gone.
3. **`file-cwd`.** Only for a file whose session neither rung placed: resolve the
   file's own recorded `cwd` values.

Then, per transcript: an ambiguous owner is skipped with reason
`ambiguous-owner` — a transcript is never split between stores. A resolved owner
that is not the destination is skipped with reason `owned-elsewhere`, reporting
the root SHA, which names a repository without naming it. A file that names no
session, or two agents, is skipped as a file-integrity refusal (`no-session-id`,
`ambiguous-agent-id`). Anything left unplaced is an orphan.

The harness's project-directory name is never decoded. It is not reversible to a
filesystem path — a directory named for a repository and a path with a separator
in the same position produce the same mangled name — so the recorded `cwd` is
the only sound signal, and it is the one this uses. The name survives only as an
opaque LABEL, which is what adoption claims.

A knowing divergence, documented in the code: `probeTranscript` reads the whole
file rather than the bounded prefix plus final line the plan called for, because
a `cwd` can appear anywhere in a transcript and a prefix read placed fewer files.
The invariant it does keep is that one transcript is resident at a time.

### Orphans are ignored, reported, and adopted only by name

The default for a transcript whose repository cannot be found on this machine is
to **ignore it and say so**. It is listed in `Orphans` with its project
directory name as given and its recorded working directory home-redacted, and
nothing is written.

Adoption is opt-in and per repository, declared in `.abcd/config/history.json`:

```json
{
  "schema_version": 1,
  "ingest_roots": [],
  "adopt_projects": [],
  "on_orphan": "ignore"
}
```

`adopt_projects` lists the project directory names this repository claims;
`--adopt` adds names for one run. A transcript under a claimed name is stored in
this repository's store, under this repository's redaction configuration, and its
record carries `lineage_source: ingest` together with `adopted_project`, so the
adoption is a property of the artefact rather than of a run's output. Adoption is
reachable only when the owner is unresolved AND unambiguous, so an ambiguous
transcript is never adopted around its ambiguity.

`on_orphan` accepts `ignore` (the default) or `prompt`. **Core never prompts:**
under `prompt` it still ingests nothing and returns the orphan list, and the CLI
front door is what asks the operator on stderr and re-invokes `Ingest` with the
chosen names in `opts.Adopt`. The transport-agnostic boundary is not negotiable,
and an interactive question is a transport concern —
`TestHistoryIngestPromptsForOrphansOnlyUnderThatPolicy` is the armed detector.

`LoadConfig(repoRoot)` reads the file under the same discipline the scanner's
per-repo configuration uses: rooted at the repository, size-capped at 64 KiB, a
symlinked leaf refused. An ABSENT file is not an error — it is
`DefaultConfig()`, `{schema_version: 1, on_orphan: "ignore"}` — but a file that
is present and unreadable, malformed, or carries an unknown `on_orphan` is.

### `history migrate`: the composite records

The records already written under composite identifiers are the migration's
input. The count is per-machine local data rather than a fixture; the store this
was designed against held 176 of them out of 267 records.

`Migrate(rootSHA string, opts MigrateOptions)` walks the store for a `session_id`
matching the composite shape `<truncated-parent>--agent-<agent>` and, for each:

- takes the agent id from the suffix;
- **repairs the truncated parent id from the record's own body**, which still
  carries the full session identifier on its transcript lines, and requires the
  recovered value to begin with the stored prefix. A body that disagrees, or one
  where no session identifier can be found, leaves the record untouched and is
  reported. The truncation is lossy, so the prefix is a check and never a source;
- recovers `agent_type`, `parent_agent_id`, `spawn_depth` and
  `spawn_tool_use_id` through the `LineageLookup` seam when a
  `--sidecar-root` (or a declared `ingest_root`) holds the harness's per-agent
  metadata, and otherwise stamps `spawn_attribution` to say the lineage is
  UNKNOWN — so an empty agent type reads as unrecoverable rather than as a
  main-thread record;
- frames every recovered scalar through the same redaction pass the body gets
  before it is written, because a lineage scalar learned from a file a
  contributor can edit is externally-supplied text landing in frontmatter;
- rewrites the frontmatter atomically and does not touch the body.

Three properties matter. `source_sha256` is computed over the raw source and is
not recomputed, so a migrated record still dedups against a re-capture of the
same bytes. The filename is left alone: a rename would break any path a reader
already holds and buys nothing, because listing reads frontmatter. And the verb
**reports by default and writes only under `--apply`**, because the store holds
the only copy of these records. Re-running it is a no-op.

`migrate`'s destination is implicit — the store of the repository the operator is
standing in — where `ingest`'s is explicit. The two are not inconsistent:
`migrate` rewrites records already filed in one store and chooses nothing, while
`ingest` decides which store bytes enter.

### Surfaces

Two new user-facing sub-verbs. Each carries, in the same change: a section in
`commands/history.md`; rows in the brief's history chapter sub-verb table, which
`surface_coverage` checks in both directions and refuses on; a regenerated
surface snapshot (`cmd/abcd-gen-surface`) and command reference
(`cmd/abcd-gen-cli-ref`). The `history` sub-tree moved out of
`internal/surface/cli/cli.go` into its own file before the verbs were added, so
they landed in a file that is about one thing.

## How the Acceptance Criteria are satisfied

- **ac-1 (ingest under the destination's own configuration; twice adds
  nothing).** `Ingest`'s explicit `Destination`, the scanner built from
  `dest.RepoRoot` alone, and the unchanged `(source_sha256, session_id, agent_id,
  kind)` idempotency key. Tests:
  `TestIngestRedactsUnderTheDestinationsOwnConfiguration` (two fixture
  repositories with different rules, asserting the destination's applied and the
  source's not), `TestIngestStoresOnlyWhatTheDestinationOwns`, and
  `TestIngestIsIdempotent`.
- **ac-2 (an unidentifiable owner is skipped and reported).** The resolution
  ladder's refusals. Tests: `TestIngestRefusesAnAmbiguousOwner`,
  `TestIngestIgnoresAndReportsOrphans`, `TestIngestSkipsAFileThatIsNotOneTranscript`,
  with `TestIngestPlacesTheSessionBeforeTheFile` and
  `TestIngestPlacesASessionFromTheStoreWhenNoDirectorySurvives` bounding the
  refusal so it cannot be satisfied by refusing everything.
- **ac-3 (a configured adoption stores and records it).** `adopt_projects` plus
  the `adopted_project` stamp on the record. Test:
  `TestIngestAdoptsOnlyProjectsNamedByTheDestination` asserts both directions —
  the claimed project is stored and stamped, and an unclaimed one is not stored
  at all.

`migrate` carries no acceptance criterion of its own. It is in this spec because
it is the same capability from the record side: history that already exists,
brought up to the standard capture now sets, by the same operator in the same
run. Its bar is its own test set below.

## Tests

`internal/core/history`: `TestIngestRefusesWithoutAnExplicitDestination`,
`TestIngestStoresOnlyWhatTheDestinationOwns`,
`TestIngestPlacesTheSessionBeforeTheFile`,
`TestIngestPlacesASessionFromTheStoreWhenNoDirectorySurvives`,
`TestIngestIgnoresAndReportsOrphans`,
`TestIngestAdoptsOnlyProjectsNamedByTheDestination`, `TestIngestIsIdempotent`,
`TestIngestRefusesAnAmbiguousOwner`,
`TestIngestRedactsUnderTheDestinationsOwnConfiguration`,
`TestIngestEnrichesASubAgentFromTheLineageRung`,
`TestIngestSkipsAFileThatIsNotOneTranscript`; configuration loading
(`TestLoadConfigDefaultsWhenAbsent`,
`TestLoadConfigReadsTheDeclaredRootsAndClaims`,
`TestLoadConfigRefusesAnUnknownOrphanPolicy`,
`TestLoadConfigRefusesMalformedJSON`, `TestLoadConfigRefusesASymlinkedLeaf`);
migration (`TestMigrateRecoversTheParentSessionFromTheBody`,
`TestMigrateReportsWithoutApplying`,
`TestMigrateRefusesABodyThatDisagreesWithThePrefix`,
`TestMigrateIsARepeatableNoOp`, `TestMigrateKeepsTheFilenameAndTheBody`,
`TestMigrateParsesTheWorkflowShapedComposite`,
`TestMigrateSaysLineageIsUnknownWithoutASidecar`,
`TestMigrateEnrichesFromTheHarnessSidecar`,
`TestMigrateRedactsTheLineageItLearns`,
`TestMigrateLeavesMainThreadRecordsAlone`,
`TestMigrateValidatesTheLineageBeforeItFramesIt`,
`TestFrameLineageRefusesAScalarWithALineBreak`).

`internal/surface/cli`: `TestHistoryIngestRefusesWithoutAnExplicitDestination`,
`TestHistoryIngestWritesIntoTheNamedRepositoryNotTheWorkingDirectory`,
`TestHistoryIngestPromptsForOrphansOnlyUnderThatPolicy`,
`TestHistoryIngestRefusesWithNoSource`,
`TestHistoryMigrateReportsByDefaultAndWritesOnlyOnApply`,
`TestHistoryMigrateRecoversLineageFromADeclaredSidecarRoot`.

Gates: `surface_coverage` over the two new sub-verbs, the regenerated snapshot
and command reference, and `make record-lint` over this spec and its intent.

Real-corpus run: 869 transcripts stored for this repository and 197 for a second
one on the same machine, all 176 composite records repaired, 13 transcripts
refused outright by the fail-closed scanner over network addresses it could not
redact, leaving the store holding 1104 records.

## Uncertainties

- **`SessionOwner` is exported and has no caller.** Its logic is duplicated
  inline by the session-placement pass, which returns empty rather than erroring
  on a multi-store claim. Either it gains a front door or it goes; against the
  "wired or it isn't done" boundary, exported-with-no-caller is the wrong state
  to leave it in.
- **`IngestOptions.MaxDepth` is unreachable from the CLI.** No flag exposes it,
  so the front door can only ever walk to depth 8. The default has been adequate
  on every corpus tried; whether the knob should be exposed or removed is
  unsettled.
- **The whole-file probe read is a knowing divergence** from the bounded prefix
  read the plan called for. It placed more transcripts, which is why it stands,
  but its cost has only been measured at the one backlog scale above.
- **The two skip reasons that are file-integrity rather than ownership
  refusals** (`no-session-id`, `ambiguous-agent-id`) have no test that asserts on
  the reason string specifically.

## Out of scope

The scanner's detectors and the two-stage redaction discipline. The store's
keying and provisioning. The live sub-agent capture path. The harness's
retention policy. Reconstruction and telemetry. Renaming the records that
migration touches, recomputing their source digests, and any change to
`source_kind`. A scheduled or unattended recovery run.
