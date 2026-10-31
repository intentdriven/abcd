---
id: itd-23
slug: spec-kit-interop
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Lifeboats Speak To External Tools

## Press Release

> **abcd lets lifeboats round-trip with external spec/issue tools.** A lifeboat produced by `/abcd:disembark to-spec-kit <path>` exports cleanly to GitHub Spec Kit format (and, optionally, to Linear/Jira/Notion shapes via adapters); a lifeboat consumed by `/abcd:embark from-spec-kit <path>` ingests external specs into intent-shaped content. The lifeboat is no longer a dead-end abcd-only artefact — it's the durable representation that lives between formats. Cross-tool collaborators can hand work to abcd-using developers and back again without rewriting.
>
> "Our product team writes specs in Spec Kit; engineering uses abcd," said Iris, product manager. "abcd's interop means I drop a Spec Kit directory and abcd embarks it as a starter intent set with the press releases auto-drafted from the spec content. When engineering ships, the lifeboat exports back to Spec Kit so the loop closes. We finally bridge the format gap."

## Why This Matters

Legacy abcd shipped `spec-export` and `spec-import` for [GitHub Spec Kit][spec-kit] interop, but those tools targeted the earlier feature-spec format. abcd uses intents (press-release format), so the old importers do not apply directly. The interop concept is still valuable — abcd users collaborate with non-abcd teams who use Spec Kit, Linear, Jira, or Notion as their canonical artefact.

The right format-of-record question (which external tools, which adapter shapes, what fidelity guarantee on round-trip) needs fresh design. This interop is now more tractable than before: adapters are a first-class abcd pattern ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md), [adr-24](../../decisions/adrs/0024-companion-harness-peer-via-conventions.md)), so a Spec Kit ↔ intent adapter is one more instance of an established shape rather than bespoke plumbing.

See [`research/legacy-harvest.md`](../../research/legacy-harvest.md) Pass 2 (skills) and Pass 5 (v0 scripts) for the deferral context.

## What's In Scope

- **Spec Kit ↔ intent adapter**: read Spec Kit format and emit intent press releases; emit Spec Kit format from a shipped intent's press release + acceptance criteria + audit notes.
- **Adapter pattern reused from existing infrastructure**: the same vendor-agnostic-adapter shape abcd uses for its bundled dependencies (adr-22) and for the memory and reviews adapters (brief § 6.7).
- **Round-trip fidelity test**: corpus test imports a Spec Kit project, embarks, ships an intent, exports back to Spec Kit, diffs against original — measures information loss.
- **`/abcd:embark from-spec-kit <path>`** sub-verb — ingest external spec-kit content as starter intents.
- **`/abcd:disembark to-spec-kit <path>`** sub-verb — export shipped intents to spec-kit format alongside the lifeboat.
- **Stub Linear/Jira/Notion adapters** — interface defined; one or two reference implementations; rest community-contributable.

## What's Out of Scope

- **Real-time sync** — this is one-shot import/export, not bidirectional live sync. Real-time sync is its own product.
- **Authoring spec-kit-shaped content within abcd** — abcd users write intents; intents convert to spec-kit only at export time.
- **Conflict resolution UI** — when round-trip introduces drift, surface the diff; don't auto-merge.
- **Linear/Jira/Notion as primary backends** — abcd's source of truth remains intents in `.abcd/development/intents/`; external tools are sync targets, not primary stores.

## Acceptance Criteria

- **Given** a Spec Kit project directory with feature specs, **when** the user runs `/abcd:embark from-spec-kit <path>` in an empty repo, **then** the embark produces draft intents matching the input specs and the press release for each is auto-composed from the spec content.
- **Given** a shipped intent with acceptance criteria and audit notes, **when** the user runs `/abcd:disembark to-spec-kit <path>` , **then** the export produces a valid Spec Kit directory readable by spec-kit's own tooling.
- **Given** the corpus round-trip test, **when** an intent goes Spec Kit → embark → ship → export → Spec Kit, **then** the diff between original and final spec-kit content is bounded (specific bound TBD; documented in test).
- **Given** an unsupported external format (e.g. raw Markdown, RTF), **when** the user attempts import, **then** the command fails with a specific error pointing at the supported-formats list.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md), which reuses the June 2026 [spec-kit-vs-flow-next](../../research/notes/spec-kit-vs-flow-next.md) note.

- Spec Kit 1.0 (v1.0.0 released 2026-08-21; v1.2.3 current on 2026-10-22) records the active feature in `.specify/feature.json` ("Commands resolve the feature from that state, not from the checked-out Git branch"); the artefacts stay Markdown: `spec.md`, `plan.md`, `tasks.md`, `checklists/requirements.md`; a new `/speckit-converge` appends gap tasks until the code matches the spec (Spec Kit repository, releases and quickstart). The site claims 38 integrations.
- The spec template is the mapping target, and it is overridable: prioritised user stories with Given/When/Then acceptance scenarios, `FR-001…` functional requirements with inline `[NEEDS CLARIFICATION]`, `SC-001…` measurable success criteria, key entities, assumptions; tasks are `- [ ] T010 [P] [US1] …` lines in phases (spec and tasks templates). The site advertises presets and extensions that tune the templates per project, so the importer declines a spec missing `FR-`, `SC-` or Given/When/Then rather than assuming the stock template.
- Kiro specs are the same three-file shape (`requirements.md`, `design.md`, `tasks.md`); BMAD's v7 preview ships as skills installed into a host (stable v6.12 still uses its npm installer); OpenSpec (MIT, "30+ AI assistants") keeps current specs in `openspec/specs/` and per-change folders under `openspec/changes/<name>/` (proposal, spec deltas as `## ADDED Requirements` with `#### Scenario:` WHEN/THEN blocks, design, tasks), a second documented target for the normalised record; Tessl's format is closed beta with three variants (search snippets only).
- The June note's architecture reading holds: Spec Kit owns authoring, not execution state; the interop is an authoring-side peer format.
- No documented import format was found for Linear, Jira or Notion in this pass; three stub adapters with no caller are dead scaffolding under "wired or it isn't done".
- Spec-kit tasks have a home in abcd after all: a planned intent's spec in the spec store, not the intent, which answers the first open question.

**Path 2 — one native normalised exchange record (narrative, prioritised Given/When/Then scenarios, numbered requirements, measurable criteria, an ordered task list with story tags) shaped to Spec Kit 1.0's templates, pinned to the fields it reads rather than to a release, import first.** The independent fit-challenge runs at plan time.

## Open Questions

- **Format-of-record for round-trip stability**: spec-kit's acceptance-criteria field maps cleanly to abcd's `itd-1` BDD acceptance criteria, but spec-kit's "tasks" don't map to anything in abcd (tasks live in the native spec store, not in intents). What does the round-trip do with task content?
- **Linear/Jira/Notion priority order**: which one is the next adapter after spec-kit? Depends on user demand.
- **Persona handling**: spec-kit doesn't have a persona registry; abcd does (`personas.json`). On import, default to a persona; on export, drop the persona attribution? Or transmit as a custom field spec-kit ignores?
- **Versioning**: spec-kit may evolve its format; abcd's adapter must pin to a specific spec-kit version and gracefully decline newer/older.
- (SOTA 2026-10-23) Does import read `.specify/feature.json` to pick the active feature, or every feature folder it finds?
- (SOTA 2026-10-23) Should the Linear/Jira/Notion stubs leave scope until a documented import format and a caller exist?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

[spec-kit]: https://github.com/github/spec-kit "GitHub Spec Kit — AI-aided spec-driven development format"
