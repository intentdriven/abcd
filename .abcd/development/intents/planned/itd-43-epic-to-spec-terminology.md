---
id: itd-43
slug: epic-to-spec-terminology
spec_id: spc-8
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-26]
prd_path: null
severity: minor
---

# abcd Speaks One Word for a Specced Block of Work, and That Word Is "Spec"

## Press Release

> **abcd speaks one word for a specced block of work — `spec` — everywhere a product thinker or contributor reads: no `epic` left behind in a heading, a review type, a schema field, or the glossary.** The native spec store is `spec` throughout, and abcd's `spec_id` intent-frontmatter field already carries the concept. Where abcd's *surfaces* still say `"epic"` — the reviews subsystem (`epic-review` type, `## Epic:` headings, `epic_id` review-directory identifiers), and prose across the brief, docs, and command help — this intent makes the vocabulary single. Every one of those surfaces is abcd-owned; there is no vendored external epic/spec boundary to preserve, so the rename runs clean through the reviews subsystem, the schemas, and the glossary.
>
> "I'd renamed the field and thought I was done — then a contributor opened the glossary's `epic` term file and asked which word was real," said Kira, framework author. "abcd's whole pitch is that each concept has one canonical term. Having `spec` in the schema and `epic` in the glossary was exactly the drift the glossary exists to prevent. One sweep, one word, and the term file is the source of truth again."

## Status

This intent is the *remaining* terminology sweep. The atomic
`epic_id`→`spec_id` intent-frontmatter field rename is a separate concern (it
has to be atomic: schema + data + code together, or intent-lint fails); the
broader, non-atomic surface/prose/glossary sweep enumerated under *What's In
Scope* below is what this intent carries.

## Why This Matters

abcd's [terminology discipline](../../brief/glossary/) exists to kill exactly one failure: the same concept named two ways, drifting until two readers mean different things. Right now abcd commits that failure about its own core noun. The schema and the intent corpus say `spec_id`; the glossary half is delivered — `brief/glossary/core/spec.md` carries `term: spec` with `epic` in `forbidden_synonyms` (spc-8) — but the reviews subsystem still classifies `epic-review`. A framework that enforces ubiquitous language cannot itself be bilingual about its central term.

The `epic_id`→`spec_id` field rename was done separately and first, on purpose — it had to be atomic (schema + data + code, or intent-lint validation fails). What remains does **not** break anything: it is inconsistency, not breakage, which is why it is its own intent rather than an emergency fix. But unaddressed it erodes the glossary's authority and confuses every new contributor.

The reviews subsystem, the schemas, and the spec store are all abcd-owned — there is no vendored external plugin whose `epic`/`spec` aliases must be preserved, so the rename is a coherent internal sweep rather than a negotiation across a boundary. The single-source-of-truth rule decides the order: `brief/glossary/core/spec.md` is canonical for the concept, so it was renamed first and everything else conforms to it.

## What's In Scope

- **Rename the canonical term file** — delivered with spc-8: the term file lives at `brief/glossary/core/spec.md` with `term: spec` and `epic` in `forbidden_synonyms`, and the `GL002` lint catches regressions.
- **Reviews subsystem rename** — the review-index, review-postprocess, and review-verify surfaces: `epic_id` parameters → `spec_id`, the `--epic` CLI flag, the `## Epic:` rendered heading, the `epic_id` JSON field, and the `epic-review`/`epic` review-type tokens. All of it is abcd-owned, so the review-type token becomes `spec-review` throughout with no external token to accommodate.
- **`issue.schema.json`** — moot in the Go rebuild (spc-8): no `*.schema.json` exists in the tree, and the native validator already uses `related_specs` exclusively.
- **`grill-report.schema.json`** — moot in the Go rebuild (spc-8): the file does not exist.
- **Prose sweep** — `intents/README.md`, the brief (`04-surfaces/`, `02-constraints/`, etc.), `docs/reference/{commands,facilitator,review-schema}.md`, `commands/intent.md`, the grill `SKILL.md` boundary message, project READMEs: `epic` as a noun → `spec`.
- **The native spec store's README** — moot in the Go rebuild (spc-8): `.abcd/development/specs/` carries no README.

## What's Out of Scope

- **The `epic_id`→`spec_id` intent-field rename** — already completed in a prior, separate change (it had to be atomic; this intent is the non-atomic remainder).
- **Renaming the `spc-` ID prefix** — `spc-N-slug` is the spec identifier format; this intent renames the *concept word*, not the ID scheme.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** the rename is complete, **when** a contributor greps abcd-owned files for `"epic"` as a standalone noun, **then** no live reference remains — only historical git-tracked records.
- **Given** the glossary, **when** a contributor looks up the concept, **then** it resolves to `brief/glossary/core/spec.md` with `term: spec`, and `epic` appears there only as a `forbidden_synonyms` entry.
- **Given** the reviews subsystem is renamed, **when** a review is classified, **then** classification succeeds against the `spec-review` type the native reviews surface emits — one internal token, no desync.
- **Given** the issue ledger, **when** an issue links to a spec, **then** it uses `related_specs` — met by the Go rebuild's validator (spc-8); no `*.schema.json` remains to update.
- **Given** the prose sweep is complete, **when** `internal/core/lint` runs, **then** no forbidden-synonym (`GL002`) violation for `epic` is raised by any abcd-owned intent or doc.

## Open Questions

- Sequencing against the `intents/README.md` v1/v2/v3 → phase migration (logged separately in a working-log entry): both rewrite `intents/README.md`. Run the README migration first and this sweep second, or merge them into one README pass?
- ~~Should the term file be renamed or kept as a stub?~~ Answered by the tree (spc-8): the rename happened with no stub, and the definition body carries `term: spec`.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Follows: the `epic_id`→`spec_id` intent-field rename (intent.schema.json, prd.schema.json, all 41 intent files, internal/core/lint, commands/intent.md) — the atomic part, done first; this intent is the non-atomic remainder.
- Sequenced with: the `intents/README.md` v1/v2/v3 → phase migration (logged in a dated working-log entry, 2026-05-16 session) — both rewrite the same README; order or merge them.
- Triggered by: the native spec store adopting `spec` as its term ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)), which leaves abcd's older `epic` surfaces inconsistent.
