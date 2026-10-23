---
id: itd-14
slug: prompt-registry-versioning
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-5]
builds_on: [itd-15]
severity: minor
---

# Prompts Are Versioned Like Code

## Press Release

> **abcd treats agent prompts as versioned artefacts.** Each prompt carries a version number, a changelog, and a known-good baseline. Editing a prompt requires running its golden-test fixtures and showing a diff in output quality before the change is accepted. Reverts are first-class. Prompt evolution becomes traceable, not a series of unverified edits.
>
> "We had a moment where someone tweaked a prompt and our oracle audits started returning weird verdicts," said Bob, staff engineer. "Golden tests caught it eventually. abcd now makes prompt changes a deliberate act with diff-on-update, so we'd have caught it at edit time."

## Why This Matters

abcd ships three prompt-quality layers (B+C+D from the brief): golden-test fixtures per agent, prompt linter, and periodic SOTA-audit oracle prompt. These catch rot but don't prevent it — a prompt edit can sneak in unaudited and break things downstream.

Prompt registry + versioning is the heavier rigour layer (option E from the brief, deliberately deferred). It becomes worthwhile once there are several months of prompt evolution to learn from. Treating prompts as code (with version, diff, review, revert) is the long-term right answer for a 13+ agent system.

## What's In Scope

- Per-prompt version field (`## Prompt Version: X.Y`)
- Per-prompt changelog (`agents/<name>/CHANGELOG.md`)
- Diff-on-update workflow: prompt edit → run golden tests → show output diff → confirm
- Revert capability via git + version field bookkeeping
- Promotion of `## Last SOTA Audit:` footer (added in the baseline) to a structured changelog entry

## What's Out of Scope

- Storing prompts in a database rather than `.md` files (overengineered)
- Cross-prompt dependency tracking (e.g., "this prompt assumes that prompt's output format")
- A/B testing prompts in production (separate concern)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** any agent prompt at `agents/<name>.md`, **when** the prompt linter runs (`lint_prompts.py`), **then** the prompt declares both `prompt_version: X.Y.Z` (per itd-5) AND a `## Changelog` section listing every version with date and one-line summary — missing changelog is a `PQ` lint blocker.
- **Given** an agent prompt is edited, **when** the user attempts to commit the change without bumping `prompt_version`, **then** the pre-commit hook refuses the commit and points to the version-bump rule.
- **Given** a prompt edit bumps `prompt_version`, **when** the pre-commit hook runs, **then** it executes the agent's golden-test fixtures AND surfaces a side-by-side diff of pre-edit vs post-edit fixture output for the user to confirm before the commit completes.
- **Given** a prompt change has been merged, **when** a regression is later discovered, **then** the user can run `abcd prompts revert <agent> --to <version>` and the prompt file is rewritten back to the named version's content (sourced from git history) AND the changelog records the revert.
- **Given** `## Last SOTA Audit:` footers exist on existing agents, **when** the registry migration runs, **then** every footer is converted to a structured `## Changelog` entry with `audit:` event type AND the original footer date is preserved.
- **Given** itd-15 (self-dogfooded SOTA audit) is shipped, **when** abcd self-disembarks, **then** the SOTA audit's findings emit `## Changelog` entries (with `audit:` event type) on the affected agents — the registry is the canonical surface for both manual and audit-driven updates.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [models, prompts and measurement SOTA note](../../research/notes/2026-10-23-models-prompts-measurement-sota.md);
the versioning consensus it checks against practice is in the
[prompting baseline](../../research/prompting/01-general-best-practices.md).

- **The substance of the first two criteria has shipped in a different
  shape.** The `agent_contract` lint (`internal/core/lint/agentcontract.go`)
  requires a semver `prompt_version`, refuses a prompt changed without a bump
  at preflight and in CI rather than in a pre-commit hook, and demands a
  changelog entry per bump in one log kept outside `agents/` because a harness
  loads that tree whole (iss-110), not a `## Changelog` section per agent.
  Criterion 5 has no `Last SOTA Audit` footers left to convert. What remains
  is the golden diff on a bump (3), the revert verb (4) and the `audit:` entry
  shape (6).
- **Versions are immutable, deployment is a pointer, rollback moves it.**
  Langfuse gives each version an id, uses labels as movable pointers, returns
  404 rather than silently falling back when a label is missing, and defines
  rollback as moving the `production` label (Langfuse prompt version control
  docs).
- **Regression runs on the pull request and posts a diff.** promptfoo's
  GitHub Action evaluates edited prompts against their previous versions on
  every PR touching them, posts a result link and caches model calls; its
  page documents no gate (promptfoo GitHub Action docs).
- **A prompt asset names its model and parameters** (Prompty frontmatter:
  model id, provider, options), and an optimised prompt is a saved build
  artefact (DSPy GEPA saves the optimised instructions as JSON).
- **"Only formatting" is a behaviour change**: spreads of up to 76 accuracy
  points from formatting alone (Sclar et al., ICLR 2024), so a PATCH bump
  cannot be exempt from the golden run.
- **The same prompt drifts when the model does**: 84 % to 51 % on the same
  questions between two releases of one model (Chen, Zaharia and Zou, 2023),
  so a version record names the model its golden run used.
- **Contrarian**: for eighteen in-repo markdown prompts, git plus the lint is
  the registry; the value left here is the golden diff and a revert that writes
  its own entry.

**Path 2**: native, shaped to Langfuse's version-and-pointer model and
promptfoo's PR-time before/after. The version stays in frontmatter; a changelog
entry gains the model id and fixture-set hash of its golden run and an
`audit:` event type; `abcd prompts revert` rewrites from git and writes the
entry; the golden diff is a host-run review recorded as a receipt the entry
cites, surfaced before the commit completes only where the host offers a hook
that can wait — a deterministic pre-commit hook cannot run fixtures. The seam:
the entry and receipt are shaped so one promptfoo eval result or Langfuse
version record maps onto them field by field. No dependency. The independent
fit-challenge runs at plan time.

## Open Questions

- Where does the prompt version live — frontmatter field, footer, or git tag?
- How aggressive is the "block edit until tests pass" check — pre-commit hook, or manual workflow?
- How does this interact with itd-15 (self-dogfooded SOTA audit)?
- (SOTA 2026-10-23) The first acceptance criterion names `agents/<name>/CHANGELOG.md` and `lint_prompts.py`; the shipped lint keeps the log outside `agents/` (iss-110) and is Go. Does the criterion follow the shipped shape, or is a per-agent log inside the tree still wanted?
- (SOTA 2026-10-23) Which golden-run facts the changelog entry records beside the version: model id, fixture-set hash, receipt path — and whether a PATCH bump that ran no golden run is refused or merely marked.

## Audit Notes

- 2026-10-02 — Declared mutual pair with itd-15 (the technical facilitator's ruling CY2 of 2026-10-02, verbatim: "SAME, DECLARED PAIRS — each direction names the piece it needs"). This record's builds_on itd-15 needs the self-dogfooded SOTA audit's per-agent findings, which this record's sixth criterion writes into the affected prompts' changelogs as `audit:` entries. itd-15's builds_on itd-14 needs the structured per-prompt changelog that replaces the `## Last SOTA Audit:` footer, which the audit's low-risk self-applied footer updates write into. Both edges stand. The record has no field record-lint's edge_cycle rule reads as a declared pair, so the linter still reports the cycle at warn until it does (iss-2610020836255174). Each piece is read from the two records' own scope, criteria and decisions; the ruling names none. Recorded by lane rd2 of autonomous run A.
