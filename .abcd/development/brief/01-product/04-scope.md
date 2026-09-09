# Scope

The scope this brief covers is bounded by what is bundled into the eight planned phases (Phase 0 — Foundations through Phase 7 — Provenance ledger and cold reading; see [`roadmap/phases/README.md`](../../roadmap/phases/README.md)) plus what is explicitly slated for a later phase.

## What the phases deliver

**User-facing commands (design target across the planned phases — not all shipped yet):**

- `/abcd:ahoy` — install/update — see [`04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)
- `/abcd:disembark` — pack a lifeboat — see [`04-surfaces/02-disembark.md`](../04-surfaces/02-disembark.md)
- `/abcd:embark` — unpack a lifeboat — see [`04-surfaces/03-embark.md`](../04-surfaces/03-embark.md)
- `/abcd:launch` — public promotion — see [`04-surfaces/04-launch.md`](../04-surfaces/04-launch.md)
- `/abcd:intent` — press-release intent capture — see [`04-surfaces/05-intent.md`](../04-surfaces/05-intent.md)
- `/abcd:capture` — issue ledger — see [`04-surfaces/06-capture.md`](../04-surfaces/06-capture.md)
- `/abcd:memory` — multi-upstream curated knowledge substrate (per itd-36) — see [`05-internals/07-memory.md`](../05-internals/07-memory.md)
- `/abcd` — top-level where-am-i status board (per itd-20) — see [`04-surfaces/08-abcd.md`](../04-surfaces/08-abcd.md)
- `/abcd:reflect` — phase retrospective (per itd-24) — see [`04-surfaces/09-reflect.md`](../04-surfaces/09-reflect.md)

**Operator-internal commands** (wiring rather than user-facing surface): `/abcd:run` — the itd-29 autonomous-run operator surface (`status`/`pause`/`resume`/`preflight`; read-mostly over the pluggable autonomous-run seam ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)), v1 never starts or kills the loop) — is a **design target**: itd-29 sits in `intents/planned/`, and no `run` verb or `commands/run.md` is on any shipped surface. The operator-internal verbs the binary does register are `changelog`, `completion`, `hook`, `rules` and `spec`, and the class is defined by the absence of a `commands/` file rather than by the presence of one. See [`04-surfaces/README.md`](../04-surfaces/README.md) for the user-facing-vs-operator-internal boundary and the record that delivered each.

**Phased intents — derived, never hand-counted here.** The intents each phase
bundles are named in that phase doc's `## Scope` section — the single source of
the mapping ([adr-9](../../decisions/adrs/0009-phase-as-product-layer.md)); the
phased set is the union across the eight phase docs, and this page keeps no
static copy of it (a hand-kept count re-drifts the moment a phase doc changes,
the same failure the roadmap dashboard avoids by deriving counts from disk).
The set spans two of the three kinds: standalone capabilities, plus the three
phased disciplines (itd-1 acceptance gates, itd-5 prompt-quality +
capability_scope, itd-37 modification grammar — the full active roster lives in
[`intents/disciplines/`](../../intents/disciplines/)) — disciplines have no user moment; they impose
acceptance gates on every other spec per the three-kinds taxonomy in
[`01-product/03-mental-model.md`](03-mental-model.md) and itd-34.

See [`phases/README.md`](../../roadmap/phases/README.md) for the phase plan and each phase's intent scope, and [`intents/README.md`](../../intents/README.md) for the intent index. Capture history lives in `git log` and each intent file's own provenance, never in this page (per [adr-5](../../decisions/adrs/0005-brief-is-current-state.md)).

**Plumbing infrastructure** (16 agents — the canonical roster is the catalog in [`05-internals/01-agents.md`](../05-internals/01-agents.md) — 11 adapters, harness shim, prompt-quality stack, hooks): see [`05-internals/`](../05-internals).

## What comes in a later phase

**Later-phase items live as press-release intents** — the uncommitted bench in `.abcd/development/intents/drafts/`, plus the committed-but-unscheduled intents in `planned/` (per [adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md)). The canonical out-of-scope list is at [`06-delivery/03-out-of-scope.md`](../06-delivery/03-out-of-scope.md).

The later-phase bench is the live `drafts/` corpus (per adr-34 no phase-scoped intent lives there, so nothing is subtracted); it is enumerated — and kept non-drifting via a filesystem-derived command rather than a hand-count — in the canonical [`06-delivery/03-out-of-scope.md`](../06-delivery/03-out-of-scope.md). itd-31 and itd-32 are superseded (preserved as historical record in `intents/superseded/`).

Each intent captures the press-release-shaped scope and acceptance criteria. A later-phase intent enters work by being scoped into a phase, then promoted to `planned/` via `/abcd:intent plan <itd-N>` and to `shipped/` via `/abcd:intent ship <itd-N>` (or automatically when the linked spec closes).

The brief does not get re-versioned. What has shipped is defined by which phases are complete and which intents are in `shipped/`; this brief stays the canonical current-state design record.
