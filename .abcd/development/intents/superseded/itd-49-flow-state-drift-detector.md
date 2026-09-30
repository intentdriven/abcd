---
id: itd-49
slug: flow-state-drift-detector
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
superseded_by: adr-26
kind_at_supersession: standalone
related_adrs: []
severity: nitpick
---

> **⚠️ Superseded by [ADR-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)** (flow-next is dropped; its runtime `.flow/` state store no longer exists, so there is no flow-state to drift against). Preserved as historical record per the supersession lifecycle.

# Flow-State Drift Becomes Visible Before It Compounds

## Press Release

> **abcd ships a standing flow-state drift detector — a read-only checker that compares the `.flow/.checkpoint-spc-*.json` runtime blocks against the `<git-common-dir>/flow-state/` store and exits non-zero on divergence.** Wired as a local pre-commit hook (pre-commit/local only — see the corrected framing under What's In Scope: both inputs are git-untracked, so a CI gate is infeasible), the detector catches the exact desync that spc-6 (predecessor store; Phase 1 reconciliation) had to repair by hand: completed specs reporting `0/N tasks done`, `flowctl validate --all` flagging done-specs INVALID because the runtime state store was never persisted. Once drift surfaces in a pre-commit run, it gets fixed in the moment instead of accumulating until the next reconciliation pass.
>
> "I committed a task closure and the pre-commit hook told me my flow-state store and my `checkpoint` file disagreed about three tasks — including one I hadn't actually finished yet," said Diana, contributor. "Five minutes of investigation showed the runtime store had stale rows from a Ralph run that crashed partway through a loop pass. I fixed it, re-committed, and moved on. The alternative would have been: don't notice, accumulate three more drifts over the next month, then spend half a day reconciling everything in a sweep like the predecessor store's spc-6."

## Why This Matters

spc-6 (predecessor store) fixed the flow-state desync that abcd-cli had accumulated: `.git/flow-state/` was never persisted (deliverables squashed into the initial commit), `flowctl validate --all` defaulted every task's status to `todo`, and three closed specs (spc-1, spc-2 and spc-4, all predecessor store) reported `0/N` because the runtime state store and the committed task JSONs disagreed silently. spc-6 reconciled by hand. spc-6's `## Boundaries / non-goals` explicitly deferred the **standing drift detector** — a read-only checker that would catch the desync from recurring — as a new feature with its own intent.

That intent is this one.

The status quo is fragile in a specific way. Before the 0.20.21 → 1.1.1 flow-next migration, abcd had a stricter-than-flow-next `flow-task-definition-drift` pre-commit hook (`check_task_drift.py`) that covered exactly this surface. The 1.1.1 migration removed it, and flow-next 1.x ships no equivalent. So the exact desync this spec repaired by hand has **no automated guard against silently recurring** (per `.work/issues.md` 2026-05-17 line 240). Until a drift detector exists, periodic manual re-verification (`flowctl validate --all` + `flowctl specs`) is the only safety net — and "periodic" means "when someone remembers", which means "after the drift has already cost someone an afternoon".

The drift detector is **project-agnostic** in the same sense spc-6 (predecessor store) was: every abcd project that uses flow-next accumulates flow-state, every one of those projects can desync the runtime store from the checkpoint files, and every one of those projects benefits from the same read-only check.

## What's In Scope

- A new script (provisional: `scripts/abcd/check_flow_state_drift.py` or
  similar — location decided at plan) that:
  - Reads `.flow/.checkpoint-spc-*.json` runtime blocks (the canonical
    per-task `status`, `claim_note`, `claimed_at` fields).
  - Reads the runtime state store at `<git-common-dir>/flow-state/tasks/*.state.json`.
  - Reports any divergence with a structured `Finding` (matching abcd's
    existing `internal/core/lint`/`lint_prompts.py` `Finding` shape) and exits
    non-zero on any finding.
  - Supports `--json` for machine consumption, `--codes` for code-contract
    discovery, and a default human-readable text output.
- A pre-commit hook wired in `.pre-commit-config.yaml`. (Corrected at plan,
  spc-41, predecessor store: NOT scoped to staged `.flow/` paths — the hook runs on every commit,
  because the store changes independent of staged files and a staged-file
  filter would blind the detector. See spc-41 (predecessor store) R5.)
- ~~A CI workflow~~ — **corrected at plan (spc-41, predecessor store): pre-commit/local only, NO
  CI gate.** Both inputs are git-untracked (`.flow/.checkpoint-*` is
  gitignored; the store lives under `.git/flow-state/`, which git never
  tracks), so a CI checkout has nothing to scan — a CI job would be
  always-green false assurance.
- Documentation in `05-internals/06-lint.md` (the lint code contract) and
  whichever `05-internals/` page covers flow-state structure.

## What's Out Of Scope

- **Auto-repair.** The detector is read-only. Repair tooling
  (`flowctl checkpoint restore` is the existing flow-next-side option, but
  it has the overwrite-defs problem that T1 of spc-6 (predecessor store) documented) is a separate
  intent if it turns out the hand-fix path is too cumbersome.
- **Coverage beyond task `status`.** The detector flags status divergence
  (which was the specific drift of spc-6, predecessor store). Per-task `claim_note` or `claimed_at`
  drift, spec-level `next_task` drift, or dependency-graph drift are
  out-of-scope for v1.
- **flow-next-side fix.** If flow-next 1.x re-introduces an equivalent hook,
  this intent should still ship abcd's stricter variant — abcd's drift
  appetite is lower than flow-next's default. Stay coordinated, but don't
  block on upstream.

## Acceptance Criteria

- *Given* a fresh repo where `.git/flow-state/` matches `.flow/.checkpoint-*.json` exactly, *when* `check_flow_state_drift.py` runs, *then* exit code is 0 and no findings are emitted.
- *Given* a repo where one task's runtime state differs from its checkpoint (e.g. checkpoint says `done`, state store says `todo`), *when* the detector runs, *then* exit code is non-zero and the finding names the task id, the divergent field, the checkpoint value, and the state-store value.
- *Given* a contributor staging a change that introduces flow-state drift, *when* they commit, *then* the pre-commit hook fires and blocks the commit with the same finding output the detector emits standalone.
- ~~*Given* a PR whose head carries flow-state drift its base does not, *when* CI runs, *then* the drift-check workflow fails the PR check.~~ (Corrected at plan, spc-41 (predecessor store): no CI gate exists or can exist — both inputs are git-untracked, so a CI checkout carries no drift to detect. The pre-commit hook is the only automated trigger.)
- *Given* the detector emits a finding, *when* a contributor reads
  `05-internals/06-lint.md`, *then* the finding's code is registered there
  (provisional code: `FS001` — exact allocation at plan).

## Open Questions

- **Strict mode vs. recoverable mode.** Should the detector also flag the
  *absence* of `<git-common-dir>/flow-state/` (which is the spc-6 (predecessor store) starting
  state — the store had been wiped), or only divergence between an existing
  store and the checkpoints? Lean: missing-store is itself a finding (an
  empty store is worse than a wrong one), but a `--allow-empty-store` flag
  exists for fresh-clone scenarios.
- **Pre-commit scope.** Run on every commit, or only when `.flow/` paths
  are staged? Tighter scope (only `.flow/`) is faster; looser scope catches
  drift introduced by non-`.flow/` work (which shouldn't happen but
  occasionally does). Lean tight. (Resolved at plan, spc-41 (predecessor store): LOOSE — the hook
  runs on every commit. The store is git-untracked and changes independent of
  staged files, so the tight scope would miss exactly the drift the detector
  exists to catch. Cost is measured and bounded instead — spc-41 (predecessor store) M5.)
- **Finding code allocation.** `FS001` for status divergence; reserve
  `FS002`–`FS005` for the deferred coverage extensions (claim_note,
  claimed_at, next_task, dependency-graph). Decide if the reservation is
  worth declaring up-front in `06-lint.md § 1` or only when those extensions
  ship.
- **Interaction with `flowctl checkpoint restore`.** If a contributor runs
  `restore` and the operation overwrites a definition field, does the
  detector flag that as drift on the next commit, or does it know to
  recognise a fresh restore? The cleanest answer is: detector flags it,
  contributor reads the diff and decides. But this could trap a contributor
  who legitimately needed `restore`.

## Related

- **spc-6** (predecessor store; Phase 1 reconciliation) — the spec that surfaced the gap and
  deferred this intent. Frame this work as "the standing version of what
  spc-6 (predecessor store) did by hand".
- **`.work/issues.md` 2026-05-17 line 240** — the canonical entry recording
  the gap.
- **spc-18 T2** (predecessor store) — the pre-commit-vs-CI parity work; this intent's hook
  should adopt the same scoping pattern spc-18 T2 (predecessor store) settles.
- **`05-internals/06-lint.md`** — the lint-code contract; this detector
  registers there alongside the existing IL/MG/PQ/TM families.
- **flow-next 0.20.21** `check_task_drift.py` — the historical reference for
  what coverage looked like before the 1.1.1 migration removed it.
