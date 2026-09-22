---
id: itd-17
superseded_by: itd-2609221009495079
slug: model-effectiveness-tracking
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-8, adr-25]
builds_on: [itd-5]
severity: minor
---

# Pick the Right Oracle for the Job, Automatically

> **Superseded by itd-2609221009495079** on 2026-09-22, on the product thinker's ruling after an independent research pass: a learned per-request router for lane models is not adopted (independent leaderboards show commercial routers over-selecting expensive models and drifting toward them; the choice is opaque and attackable, which the record's rule of computed facts with a falsifier cannot carry). Model per role is the tier (itd-2609170822093401); escalation is a rule on a failed fix round; the closed-option judgements go to a decision adapter measured in a lab first.


> **This intent is abcd's frontier mapping** — capability-aware dispatch across the configured oracle adapters and the host-delegated default, framed per Dell'Acqua et al. 2023 ("Navigating the Jagged Technological Frontier"). It observes the jagged frontier (which adapter or the host is strong for which task), dispatches capability-aware, and renders the frontier map on demand:
>
> - The dispatch signal is a `{task_class, agent, adapter, model_id, outcome, failure_mode_tag}` schema, with closed-enum failure-mode tags (`hallucination` / `scope_drift` / `stale_context` / `under_specification_blindness` / `format_violation`).
> - Frontier data lands as logbook events at `.abcd/logbook/frontier/<ts>/frontier-events.{json,md}`, complemented by agent-frontmatter declarations (`capability_scope` per itd-5; `known_failure_modes`); the aggregate frontier-map is rendered on demand from those source-of-truth locations. No new top-level namespace.
> - The `/abcd:frontier` command renders bare, with sub-verbs `flag` / `history` / `explain` for actions bare can't do (no `show`, per the bare-command-as-render discipline).
> - Capability-aware routing is a **selector over the configured adapters and the host default** — it picks which target handles a call, and where data is thin it defers to the host-delegated default.

## Press Release

> **abcd learns which oracle target performs best for which agent task.** Every oracle audit (lifeboat-oracle, press-release-composer's product audit, intent-fidelity-reviewer) records its outcome — was the verdict accurate? Did revisions follow? Over time, abcd builds a per-target per-agent effectiveness profile and biases dispatch automatically across the configured oracle adapters and the host-delegated default — chosen based on data, not configuration guesswork.
>
> "I had no way of knowing which oracle was actually catching the most real issues," said Eve, ML engineer. "abcd tracks the data and just picks the right one. My oracle audits got better without me changing any config."

## Why This Matters

abcd's default oracle path is the host-delegated in-session reviewer, with configured oracle adapters available on top. There's no feedback loop — abcd never learns which target produces audits that actually correlate with real issues vs noise, so a capability-aware selector cannot improve on a static choice.

The fix: track per-target statistics (ship_count, revise_count, false_revise_count, accuracy ratios) and per-judge scoring (depth_score, calibration_score) across every oracle adapter and the host default, and bias dispatch by measured effectiveness for the requesting agent type.

## What's In Scope

- `.abcd/logbook/frontier/<ts>/frontier-events.{json,md}` — per-audit frontier events (run_id, target, agent, verdict, follow-up actions, failure_mode_tag) plus the rolling per-target per-agent statistics aggregated from them
- Dispatch selection logic: weighted by recent effectiveness for the requesting agent type, across the configured adapters and the host default
- `abcd oracle [agent]` CLI subcommand to inspect effectiveness profiles. Bare `abcd oracle` renders the full per-target per-agent table; `abcd oracle <agent>` scopes to one agent. No `stats` sub-verb — collapses to bare per the bare-command-as-render discipline (see `02-constraints/04-naming.md`).
- Manual override remains (config setting, per-call flag)

## What's Out of Scope

- Cross-project effectiveness sharing (each project has its own profile)
- Auto-retraining or fine-tuning models (abcd doesn't own the models)
- Cost-aware selection (covered by separate cost-tracking work if/when needed)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a project with N completed oracle audits across multiple targets, **when** the user runs `abcd oracle` (bare), **then** the output reports per-target per-agent rolling statistics (ship_count, revise_count, false_revise_count, accuracy ratio) aggregated from `.abcd/logbook/frontier/`.
- **Given** an oracle audit completes, **when** the audit-fix loop or downstream review action confirms or refutes the verdict, **then** a frontier event is appended under `.abcd/logbook/frontier/<ts>/frontier-events.json` recording `run_id`, `target`, `agent`, `verdict`, `confirmed/refuted`, `failure_mode_tag`, and `follow_up_action`.
- **Given** sufficient effectiveness data exists for a given agent type (`>=` configured threshold of past audits), **when** abcd selects a target for a new oracle call with `oracle.backend = "auto"`, **then** the selection is biased by recent effectiveness AND the rationale ("RepoPrompt adapter chosen for press-release audit; +12% accuracy over the host default on this agent in last 20 calls") is logged in the run report.
- **Given** insufficient effectiveness data (cold start), **when** abcd selects a target, **then** the selection defers to the host-delegated default AND the run log records "cold-start fallback".
- **Given** the user runs an oracle call with an explicit target override (`--backend rp`), **when** the call executes, **then** the override is respected unconditionally AND the call's outcome is still recorded as a frontier event for future statistics.
- **Given** effectiveness data exists for a previously-used adapter that is no longer configured (e.g., user removed the RepoPrompt adapter), **when** `abcd oracle` (bare) runs, **then** the stale rows are surfaced under "historical (target not currently available)" rather than influencing live selection.

## Open Questions

- How is "effectiveness" measured — verdict accuracy? Followed-up actions? User feedback signal?
- Cold-start: how does selection work before any data exists? (Defers to the host-delegated default.)
- Does the user see selection decisions ("using Codex for this audit because it's outperformed RP on press-release audits")?

## Field Evidence

> Empirical observations to feed the reframed plan-review (per the reframe blockquote at the top of this file). Not yet structured into the `{task_class, agent, backend, model_id, outcome, failure_mode_tag}` schema — recorded here as raw input the reframe should consume.

### 2026-05-16 — review-backend frontier, observed over the `spc-5` dual-backend plan-review loop (12 rounds, 24 reviews)

A new **`task_class: review_backend`** worth tracking once itd-17 reframes. The `spc-5` spec plan review ran RepoPrompt and Codex CLI in parallel for rounds 16–27; their behaviour was asymmetric and consistent:

| Backend | Strength (high accuracy) | Weakness (failure mode) |
|---|---|---|
| **Codex CLI** (`gpt-5.2`, high effort, repo read access) | Repo-wide reasoning — found 2 Criticals only a repo scan could find (a `pytest`-breaking stale-string match; a concurrency race). Cross-file / cross-doc drift. | **`verdict_unreliable`** — returned `SHIP` 8× while its body still listed Major/Critical findings. Also one `hallucination` (fabricated citation) and one cross-round flip-flop on an SDK detail. Slow (8–13 min). |
| **RepoPrompt** (scoped chat, sees only selected files) | Honest verdict tag (`SHIP` ⇔ clean body, every round). Tight consistency / regression review — caught contradictions the loop's own fixes introduced. Fast (1–3 min), stateful re-review. | **`selection_blind`** — sees only attached files; never raises a repo-wide issue. Cannot find anything outside the curated selection. |

**Candidate failure-mode tags this surfaces** (for the reframe's closed enum): `verdict_unreliable` (self-assessment contradicts findings), `selection_blind` (reviewer scope too narrow to see the issue). Both are distinct from the draft enum (`hallucination` / `scope_drift` / `stale_context` / `under_specification_blindness` / `format_violation`) and worth considering as additions.

**Implication for capability-aware dispatch:** for review `task_class`, the selection signal is not "which backend is better" but "which axis the review needs" — repo-breadth → Codex; scoped-consistency + a trustworthy gate verdict → RepoPrompt; high-stakes → both. The qualitative decision is recorded in adr-8; itd-17's reframe should make it data-driven.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
