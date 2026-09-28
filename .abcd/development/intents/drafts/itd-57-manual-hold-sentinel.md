---
id: itd-57
slug: manual-hold-sentinel
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-26, adr-27]
prd_path: null
severity: minor
---

# Every abcd Voyage Ships With A Manual-Hold Sentinel That Blocks A Spec From Autonomous Pickup Until A Human Lifts It

## Press Release

> **abcd gains a built-in "park this, but keep it planned" primitive: a permanent sentinel spec, scaffolded at configure time, that any future spec can depend on to stay invisible to autonomous pickup until a human explicitly unblocks it.** Today, holding a spec back from the run seam means either deleting it (loses the work), leaving `plan_review_status` open (fragile — a reviewer can flip it), or hand-rolling a never-`done` dependency each time. With this, `abcd configure` plants a single `spc-0-manual-hold` sentinel spec (permanently open, zero tasks, a self-documenting plan) as the voyage's first spec; to park any spec, the facilitator adds a native spec dependency on it — the run seam's selector already skips specs blocked by a non-done dependency, which does the rest. To unblock, one native `spec rm-dep` lifts the hold. The spec stays fully planned, reviewed, and visible in the dependency graph the entire time — it simply cannot be picked up.

> "I wanted the session-layer activation spec written and reviewed so the design was parked — but I absolutely did not want the autonomous run starting it behind my back," said a facilitator. "Deleting it loses the plan. Leaving it half-statused is a foot-gun. I want one obvious, durable lever: this spec is held until I say so."

## Why This Matters

abcd's autonomous run seam is opt-out by absence, not by design: any open spec with ready tasks and a clean plan gate is fair game for the selector. There is no first-class way to say "this is real, planned, reviewed — and deliberately not now." The workarounds all have failure modes: deletion is lossy; an open review status is mutable by the next review pass and conflates "not reviewed" with "not wanted yet"; a hand-rolled never-done dependency works (it is exactly the mechanism here) but is undiscoverable, re-invented per use, and easy to typo into a real spec id.

A standing sentinel turns that pattern into a primitive. It is the natural home for: a spike-first spec whose activation is a strategic decision (the session-layer wiring), a security spec that must not run before a dependency lands, a spec staged for a future phase, or any "review now, run later" case. Because the block is a dependency edge, it is visible in the graph, survives task splits and replans, cannot be cleared by a worker or a review pass, and lifts with a single sanctioned command. Shipping it at configure time means every abcd voyage has the lever from day one, and the linkage lint (the done-spec ⇒ shipped-intent guard) can special-case it as never-completing-by-design.

This is voyage-agnostic: every voyage that runs an autonomous loop eventually needs to hold a planned spec back from it.

## What's In Scope

- A sentinel spec scaffolded by `abcd configure` (or the equivalent first-run path): a fixed id (e.g. `spc-0-manual-hold`), status permanently open, zero tasks, a plan body that documents its own purpose and the block/unblock commands.
- The block convention: a spec is held by adding a native spec dependency on the sentinel (`depends_on: [<sentinel>]`); the run seam's selector already skips specs with a non-done dependency, which enforces it. Unblock = native `spec rm-dep <spec> <sentinel>`.
- A `🔒 BLOCKED` banner convention for held specs so the native `spec` view makes the state unmistakable.
- Idempotent scaffolding: re-running configure never duplicates or resets the sentinel; never auto-closes it (it must stay non-done forever).
- Linkage-lint / completion-review awareness: the sentinel is exempt from "a done spec must have a shipped intent" and from any "open spec should make progress" nudges — it is permanently-open BY DESIGN.

## What's Out of Scope

- Blocking individual TASKS — the native task-block already covers that; this is spec-grain.
- A bespoke `spec hold`/`unhold` verb — v1 rides the existing native dependency mechanism; a dedicated verb is a possible later ergonomic layer, not required.
- Auto-holding specs by heuristic — holds are always a deliberate human act.
- Changing the selector's skip logic — it already skips non-done-dependency specs; this intent only provides the canonical thing to depend on.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a freshly configured abcd voyage, **when** configure completes, **then** a permanently-open, zero-task `spc-0-manual-hold` sentinel spec exists with a self-documenting plan (a test asserts presence + open status + zero tasks).
- **Given** a spec that depends on the sentinel, **when** the autonomous selector runs, **then** that spec is never offered (the existing non-done-dependency skip fires; a test proves it).
- **Given** a held spec, **when** the facilitator runs the documented unblock command, **then** the dependency is removed and the spec becomes selectable on the next iteration (a test proves the round-trip).
- **Given** configure is re-run on a voyage that already has the sentinel, **when** it completes, **then** the sentinel is unchanged (idempotent; never duplicated, never closed).
- **Given** the linkage lint / completion-review machinery, **when** it evaluates the sentinel, **then** the sentinel is exempt from done-spec and progress nudges (permanently-open by design).

## Open Questions

- Sentinel id: `spc-0-manual-hold` (sorts first) vs a non-`spc-` reserved id outside the normal numbering — which avoids colliding with the spec sequence and reads clearest in the graph?
- Should configure plant it, or should it be lazily created on first `spec hold`-style use? (Eager = always present and discoverable; lazy = no clutter in projects that never hold anything.)
- Does the existing native spec-dependency skip emit a clear "blocked by manual-hold" reason in the native selector's `next` output, or does that need a small message addition so the operator sees WHY a spec is skipped?

## Audit Notes

Captured 2026-06-12 from a facilitator decision while planning the session-layer activation spec (spc-58, predecessor store): the spec needed to be planned and double-backend reviewed but hard-blocked from run-seam pickup. The never-done-dependency mechanism was used ad hoc; this intent promotes it to a configure-time primitive. Hand-authored draft — validate via `/abcd:intent` or the record lint (`abcd docs lint`) before promotion.

## References

- The native spec store's selector — the "skip specs blocked by a non-done spec-level dependency" rule this rides (referenced by behaviour, not a fixed location).
- The native spec store's close path — the accidental-completion path the sentinel must be guarded against (a zero-task sentinel vacuously satisfies "no incomplete tasks"); the productized primitive should make `spec close` refuse the sentinel.
- The native spec store's `spec add-dep` / `rm-dep` and task-block (task-grain precedent).
- The predecessor store's spc-60 (`spc-60-manual-hold-sentinel-never-completes`) — the ad-hoc sentinel precursor this intent generalizes; its spc-62 (`spc-62-session-enforcement-wiring-n1-a1-a4b2`) — the first held consumer; its spc-48 linkage lint — the guard that had to exempt the sentinel. None of the three was carried into the Go rebuild, and the live `spc-48`, `spc-58`, `spc-60` and `spc-62` are unrelated records (amended 2026-09-26, iss-239; see the specs charter, "Two `spc-N` Namespaces").
