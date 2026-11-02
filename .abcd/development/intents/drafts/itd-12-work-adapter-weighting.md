---
id: itd-12
slug: work-adapter-weighting
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Working Notes Don't Outrank Settled Decisions

## Press Release

> **abcd weights `.abcd/work/notes/` (shared working scratch) below ADRs, specs, and memory in the principle distiller.** Quick scribbles in `.abcd/work/notes/` no longer compete on equal footing with carefully-considered architecture decisions. The distiller's confidence scoring per principle source is exposed in the lifeboat brief so reviewers can see why a particular principle made the cut.
>
> "I had a half-formed thought in `.abcd/.work.local/scratch/` that ended up appearing as a top-ranked principle in my lifeboat next to an ADR I'd spent days on," said Carol, engineering manager. "abcd now ranks them by source authority. The ADR wins; my scratch note becomes a candidate principle to revisit."

## Why This Matters

abcd promotes content from `.abcd/.work.local/` into `.abcd/work/notes/` via `dev-sync`, then feeds those into `principle-distiller` alongside ADRs, specs, memory, code-rescuer principles, and review findings. All sources are weighted equally by default. Working scratch is highly variable in quality — drafts, brainstorming, status snapshots — and shouldn't be treated as authoritative.

This intent introduces source-authority weighting and exposes the reasoning to reviewers.

## What's In Scope

- Per-source weight configuration in `.abcd/config.json` (defaults: ADRs > specs > memory > reviews > notes > code-extracted)
- Confidence scoring per emitted principle
- Source-attribution shown in `principles.md` (each principle cites its sources with weights)
- Override mechanism for users who want to flip weights for their context

## What's Out of Scope

- Auto-detecting whether a `.abcd/work/notes/` file is "polished" vs "scratch" (heuristic, fragile)
- Cross-source contradiction resolution (a separate problem)
- Removing `.abcd/work/notes/` from inputs entirely (we want low-weight inclusion, not exclusion)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a project with both an ADR and a `.abcd/work/notes/` entry covering the same architectural concern, **when** principle-distiller runs in Pass C, **then** the resulting principle in `principles.md` cites the ADR as primary source AND the notes entry as a candidate-to-revisit (not equally weighted).
- **Given** `.abcd/config.json` declares per-source weights overriding the defaults (e.g., `principle_distiller.weights.notes = 0.8`), **when** principle-distiller runs, **then** the configured weights replace the defaults AND the resulting `principles.json` records the active weight table for audit.
- **Given** every principle in the lifeboat brief, **when** a reviewer reads `principles.md`, **then** each principle is annotated with its source list (ADR / spec / memory / review / notes / code-extracted) and the per-source weight that contributed to its acceptance.
- **Given** a low-weight source (notes) and a high-weight source (ADR) emit contradictory principles, **when** the distiller resolves the conflict, **then** the high-weight source wins by default AND the suppressed low-weight principle appears under "candidate-principles-to-revisit" rather than being silently dropped.
- **Given** the default weight hierarchy (ADRs > specs > memory > reviews > notes > code-extracted), **when** any abcd documentation describes the distiller, **then** the hierarchy is documented exactly once (SSOT) at `05-internals/04-universal-patterns.md` (or wherever the distiller lives) and every other reference links back rather than re-stating.
- **Given** a user wants ADRs and specs treated as equally authoritative for a particular project, **when** they set `principle_distiller.weights.adr = 1.0` and `weights.spec = 1.0` in `.abcd/config.json`, **then** the override takes effect on the next disembark and is recorded in the principle-distiller report.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [memory-knowledge SOTA note](../../research/notes/2026-10-23-memory-knowledge-sota.md).

- **The premise has moved.** There is no `.abcd/work/notes/` tier and no `dev-sync` verb; the principle distiller's deterministic mode reads only the packed ADRs, and its delegated mode drops any principle that cannot cite a live record (`internal/core/lifeboat/synthesis_principles.go`). What survives is how a synthesis over mixed sources treats their authority.
- **Tools order sources; no surveyed harness rule system weights them numerically** (source-reliability RAG work was not surveyed). Cursor: team, then project, then user rules, and "Earlier sources take precedence when guidance conflicts"; Copilot: personal, repository, organisation, with "all sets of relevant instructions ... provided"; Claude Code: on a contradiction Claude "may pick one arbitrarily", so the remedy is periodic review and a prompt audit (vendor docs).
- **Whatever reaches the model is believed.** ClashEval (over 1,200 questions, six models) found models adopt incorrect retrieved content "overriding their own correct prior knowledge over 60% of the time", more so when less confident (Wu, Wu and Zou). The benchmark measures adoption of retrieved content over prior knowledge; it does not test whether a stated weight or label is honoured, so the step from it to "a weight will not be respected" is inference. It suggests that authority act before the model reads, through inclusion, order and label; and the tier decision is the distiller's, made deterministically, so nothing depends on the model honouring a number.
- **A model-written synthesis is checked by citation.** Clio keeps a cluster above a size floor, has a model write its summary, audits it, and validates reconstruction of known categories at 94% on synthetic data (Tamkin et al.); the distiller's cite-or-be-dropped is the local form.
- **Path 2**: native ordinal source tiers shaped to the precedence order the surveyed tools use, fixed by the record's shape (ADR; spec and brief; principle; issue ledger; working material), rendered as a provenance label on every principle, with a conflict shown as a pair (the higher tier's statement, the lower tier's beside it as a candidate to revisit) and no configuration knob; the seam is the tier table, one ordered list a later change can reorder. The divergence is from the draft, not from the outside practice: this contradicts the acceptance criteria on configurable weights and a recorded weight table; those stand as written and the contradiction is for plan time. The independent fit-challenge runs at plan time.

## Open Questions

- Are the default weights right, or do they need to be tuned per-project?
- Should weights be a flat hierarchy or a context-dependent ordering (e.g., ADRs trump specs ONLY for architecture decisions, not naming conventions)?
- How does `intent-fidelity-reviewer` interact with weighted principles — does it audit higher-weight principles more strictly?
- (SOTA 2026-10-23) Given that no surveyed harness rule system uses numeric weights and the retrieved-content evidence suggests a model would not honour one, should this intent be re-framed from "weights" to "tiers as labels, conflicts as pairs" before planning, and its acceptance criteria rewritten by the product thinker to match?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
