---
id: itd-9
slug: schema-migration
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
---

# Old Lifeboats Still Unpack

## Press Release

> **abcd reads lifeboats produced by older releases.** When you embark from a lifeboat created with an older abcd version, abcd detects the schema delta and migrates internal structures automatically — your old lifeboats keep working without re-disembarking. If a lifeboat is too old to migrate cleanly, abcd refuses with a clear "re-disembark required" message and a one-line command to do it.
>
> "We had lifeboats archived from six months ago," said Dave, compliance lead. "I expected them to be obsolete. abcd just unpacked them — and warned me about the two fields that had moved."

## Why This Matters

abcd stamps `schema_version: 1` on every lifeboat artefact (`meta.json`, `config.json`, `_provenance.json`, etc.) but ships no migrators initially — premature to write migrators before there is a schema change to migrate from. The moment a schema change ships, lifeboats produced under the prior schema risk becoming silently incompatible.

This intent commits abcd to taking lifeboat portability seriously: schemas evolve, but old lifeboats stay valuable. Either we migrate them forward, or we tell users explicitly what's needed to refresh.

## What's In Scope

- Schema versioning convention across all lifeboat JSON artefacts
- Per-schema migrators (e.g., `1 → 2` for any field that changed)
- `embark` detection of schema version + auto-migration
- "Re-disembark required" failure mode with explicit command suggestion
- Migration log captured in `embark-report.{json,md}`

## What's Out of Scope

- Migrating lifeboat content (principles, press releases) — only structural schema
- Two-way compatibility (older abcd reading newer-schema lifeboats — not supported)
- Migration tools for in-flight `.abcd/work/` working artefacts (covered separately if needed)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a lifeboat produced by an older abcd version with `schema_version: 1` on every JSON artefact, **when** the user runs `/abcd:embark from <path>`, **then** embark detects the schema delta, runs the registered migrators, unpacks the lifeboat into the target repo, and writes a migration log to `embark-report.{json,md}` listing each schema upgrade applied.
- **Given** a lifeboat too old to migrate cleanly (e.g. its schema is below the lowest registered migrator's source version), **when** embark runs, **then** the command refuses with a "re-disembark required" error AND the error message includes the exact one-line command the user should run on the source repo (e.g., `/abcd:disembark pack <source> <dest>`).
- **Given** a newer-schema lifeboat, **when** an older abcd binary tries to embark from it, **then** the binary fails fast with a "lifeboat is from a newer abcd version; upgrade abcd to embark this lifeboat" message — two-way compatibility is explicitly not supported and the failure mode is clear.
- **Given** a successful migration during embark, **when** the migration log is written, **then** it records: source schema version, target schema version, list of migrators applied (in order), per-artefact field changes, and any non-fatal warnings.
- **Given** the registered v1→v2 migrator, **when** it runs against a v1 artefact, **then** it produces a v2 artefact that round-trips cleanly through the v2 schema validator (no unknown fields, no missing required fields, all enums valid).
- **Given** a lifeboat is migrated during embark, **when** the resulting `~/.abcd.noindex/voyage/<source-root-sha>/embark/provenance.json` (the operator-level voyage store adr-35 moved it to) is written, **then** it records the migration history (`was_schema: 1`, `now_schema: 2`, `migrators_applied: [...]`) so future audits can reconstruct what was changed.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md).

- The repository already carries seven independent schema versions (`probe.go` `SchemaVersion = 2`, `EmbarkSchemaVersion`, `GraveyardSchemaVersion`, `LessonsSchemaVersion`, `PrinciplesSchemaVersion = 2`, `PressReleaseSchemaVersion`, `ReviewSchemaVersion`) and refuses newer-than-known; no reader accepts an older revision yet.
- Upcasters are the read-time pattern: Axon stores a revision with each event, an upcaster maps revision *x* to *x+1*, upcasters chain, and the store is never rewritten ("the complete event history remains intact"). A lifeboat is immutable in the same sense: migrate on read, never in place.
- Kubernetes states the rule and the storage split: conversion is "a star with the internal form at the center", "it must be possible to round-trip your change … with no loss of information", adding a field with a sane default needs no new version, and a new version should not become preferred and stored in one release so a rollback still decodes. Its round trip serves an API server that converts both ways; abcd's chain is one-way (newer-than-known refused), so the half that applies is that each up-step loses no information.
- Protobuf's field discipline: never change or reuse a field number, reserve removed numbers and names, old readers "simply ignore the new field".
- Go's decoder ignores unknown keys by default and `DisallowUnknownFields` makes them an error; the lifeboat readers choose the error on purpose (`graveyard_lessons.go` calls it to "reject smuggled extra fields" and refuses any `schema_version` but its own), so Fowler's Tolerant Reader applies here only in bounded form: accept an older revision's known fields and default the missing ones, while unknown keys stay refused where a reader refuses them today; Parallel Change (expand, migrate, contract) is the writer's half.

**Path 2 — native per-artefact upcaster chains shaped to Axon's revision model, with Kubernetes' no-information-loss rule applied to each one-way up-step (no downcasters, since newer-than-known stays refused): bounded tolerance per artefact as the first rung (an older revision's known fields accepted and missing ones defaulted, unknown keys still refused where the reader refuses them today, the lessons reader's smuggled-field guard among them), one single-step migrator per revision beside its artefact type, the lifeboat on disk untouched, "re-disembark required" when the chain has no step starting at the artefact's revision, the voyage store's provenance recording each step.** The independent fit-challenge runs at plan time.

## Open Questions

- Which schema version delta triggers re-disembark vs auto-migrate? Major-version bumps only?
- Where do migrators live in the native layout?
- Should `_provenance.json` record the migration history (was-v1, now-v2)?
- (SOTA 2026-10-23) Should the probe's "is its schema version one this build understands" and embark's check be one function before any migrator lands?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
