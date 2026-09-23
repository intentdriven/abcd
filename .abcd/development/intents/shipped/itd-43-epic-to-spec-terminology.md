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
impact: additive
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

abcd's [terminology discipline](../../brief/glossary/) exists to kill exactly one failure: the same concept named two ways, drifting until two readers mean different things. abcd names its own core noun one way: the schema and the intent corpus say `spec_id`, `brief/glossary/core/spec.md` carries `term: spec` with `epic` in `forbidden_synonyms` (spc-8), and GL002 keeps the live prose on the one word. A framework that enforces ubiquitous language cannot itself be bilingual about its central term.

The `epic_id`→`spec_id` field rename is a separate, earlier change, on purpose — it had to be atomic (schema + data + code, or intent-lint validation fails). The sweep this intent carries does **not** guard against breakage: the drift it removes is inconsistency, not breakage, which is why it is its own intent rather than an emergency fix. Left alone, that drift erodes the glossary's authority and confuses every new contributor.

The reviews subsystem, the schemas, and the spec store are all abcd-owned — there is no vendored external plugin whose `epic`/`spec` aliases must be preserved, so the rename is a coherent internal sweep rather than a negotiation across a boundary. The single-source-of-truth rule decides the order: `brief/glossary/core/spec.md` is canonical for the concept, so it was renamed first and everything else conforms to it.

## What's In Scope

- **Rename the canonical term file** — delivered with spc-8: the term file lives at `brief/glossary/core/spec.md` with `term: spec` and `epic` in `forbidden_synonyms`, and the `GL002` lint catches regressions.
- **Reviews subsystem rename** — the review-index, review-postprocess, and review-verify surfaces: `epic_id` parameters → `spec_id`, the `--epic` CLI flag, the `## Epic:` rendered heading, the `epic_id` JSON field, and the `epic-review`/`epic` review-type tokens. All of it is abcd-owned, so the review-type token becomes `spec-review` throughout with no external token to accommodate. Moot (product thinker, 2026-09-23): the Go tree has no reviews subsystem that classifies reviews by kind, so none of these surfaces exists to rename.
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
- **Given** the issue ledger, **when** an issue links to a spec, **then** it uses `related_specs` — met by the Go rebuild's validator (spc-8); no `*.schema.json` remains to update.
- **Given** the prose sweep is complete, **when** `internal/core/lint` runs, **then** no forbidden-synonym (`GL002`) violation for `epic` is raised by any abcd-owned intent or doc.

A criterion that the reviews subsystem classifies against a `spec-review` type is dropped as moot (product thinker, 2026-09-23): itd-28 was re-scoped to pin plus staleness on 2026-09-21, nothing classifies reviews by kind, and so there is no `epic-review` token to rename and no `spec-review` emitter to build.

## Open Questions

- Sequencing against the `intents/README.md` v1/v2/v3 → phase migration (logged separately in a working-log entry): both rewrite `intents/README.md`. Run the README migration first and this sweep second, or merge them into one README pass?
- ~~Should the term file be renamed or kept as a stub?~~ Answered by the tree (spc-8): the rename happened with no stub, and the definition body carries `term: spec`.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-3a24ef30ac16 -->
Fidelity review — receipt rcp-3a24ef30ac16 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:85e2f149ad89a02ed2e0e1d99761a26a2e331045a21a310856d3fdf6b95657b4
Input attestations: diff:da7b7cf4..2a759d32 (PR #668 shipped the record; the sweep itself is the tree read at cede78b8)@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: a word-bounded grep of the tree at cede78b8 outside the record folders finds the noun only in the glossary's own declarations, historical plans and reviews, a mined test corpus, and an external tool's model in the acknowledgements; the one live residue is a parenthetical inside a fenced surface diagram on the brief's intent page, which grep finds and which GL002 skips by design as fenced text
  evidence: .abcd/development/brief/04-surfaces/05-intent.md:244 — "Auto-running the reviewer off that queue is still deferred (no epic currently owns"
  evidence: .abcd/development/brief/glossary/core/spec.md:6 — "forbidden_synonyms: ["sprint", "milestone", "project", "feature", "epic"]"
  evidence: internal/core/lint/lint.go:2320 — "(fenced and inline single-backtick), YAML frontmatter, exempt path prefixes, the"
- ac-2 — MET: the term file is brief/glossary/core/spec.md with term: spec and epic listed among its forbidden_synonyms, and no core/epic term file exists
  evidence: .abcd/development/brief/glossary/core/spec.md:2 — "term: spec"
  evidence: .abcd/development/brief/glossary/core/spec.md:6 — "forbidden_synonyms: ["sprint", "milestone", "project", "feature", "epic"]"
- ac-3 — MET: the issue validator declares related_specs as the spc-N list field and the tree carries no *.schema.json file
  evidence: internal/core/capture/validate.go:144 — "{"related_specs", reSpcID, "spc-N"},"
  evidence: internal/core/capture/capture.go:106 — "RelatedSpecs []string `json:"related_specs,omitempty"`"
- ac-4 — MET: GL002 is enabled as a blocker enforcing epic over the .abcd/development root, and a test lints the live corpus with the real glossary and pins the GL002 count at zero
  evidence: .abcd/record-lint.json:405 — ""enforce": ["
  evidence: internal/core/lint/forbidden_synonyms_test.go:201 — "func TestForbiddenSynonymsRealGlossary"
  evidence: internal/core/lint/forbidden_synonyms_test.go:224 — "if n := countRule(fs, "GL002"); n != 0 {"

Gap audit:
- honoured:
  - the glossary term file is the source of truth and the lint reads it rather than a copy
    evidence: internal/core/lint/lint.go:2310 — "the glossary term files under cfg.GlossaryDir (the single source of truth for"
  - the issue ledger links to specs through related_specs
    evidence: internal/core/capture/validate.go:144 — "{"related_specs", reSpcID, "spc-N"},"
  - the reviews-subsystem criterion is dropped as moot by the product thinker's ruling, and the record says so
    evidence: .abcd/work/DECISIONS.md:2507 — "itd-43's third criterion is moot and dropped, so spc-8 closes."
- diverged:
  - no epic left behind in a heading or the brief: one noun use survives inside a fenced diagram on the brief's intent surface page, outside the detector's scope
    evidence: .abcd/development/brief/04-surfaces/05-intent.md:244 — "(no epic currently owns"
- missing: (none)
## References

- Follows: the `epic_id`→`spec_id` intent-field rename (intent.schema.json, prd.schema.json, all 41 intent files, internal/core/lint, commands/intent.md) — the atomic part, done first; this intent is the non-atomic remainder.
- Sequenced with: the `intents/README.md` v1/v2/v3 → phase migration (logged in a dated working-log entry, 2026-05-16 session) — both rewrite the same README; order or merge them.
- Triggered by: the native spec store adopting `spec` as its term ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)), which leaves abcd's older `epic` surfaces inconsistent.
