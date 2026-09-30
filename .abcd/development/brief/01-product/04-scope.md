# Scope

The scope this brief covers is the intents on the lifecycle shelves: what has shipped (`intents/shipped/`), what is committed (`intents/planned/`), and the uncommitted bench (`intents/drafts/`). No stored unit sits above the intent for sequence ([adr-2609212115255771](../../decisions/adrs/2609212115255771-phases-and-milestones-are-retired-sequencing-is-dependencies.md)): order comes from each record's dependencies (`blocked_by`, `builds_on`) and its shelf, the bare `/abcd` board renders it as the Now / Next / Later block, and an intent that must land by a cut names that release in `target_release`. The phase documents under `roadmap/phases/` are history.

## What the intents deliver

**User-facing commands.** The generated [CLI reference](../../../../docs/reference/cli/commands.md) lists every command and sub-verb the binary registers; these are the ones at the centre of the brief (a row marked as a design target is not shipped):

- `/abcd:ahoy` — install/update — see [`04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)
- `/abcd:disembark` — pack a lifeboat — see [`04-surfaces/02-disembark.md`](../04-surfaces/02-disembark.md)
- `/abcd:embark` — unpack a lifeboat — see [`04-surfaces/03-embark.md`](../04-surfaces/03-embark.md)
- `/abcd:launch` — public promotion — see [`04-surfaces/04-launch.md`](../04-surfaces/04-launch.md)
- `/abcd:intent` — press-release intent capture — see [`04-surfaces/05-intent.md`](../04-surfaces/05-intent.md)
- `/abcd:capture` — issue ledger — see [`04-surfaces/06-capture.md`](../04-surfaces/06-capture.md)
- `/abcd:memory` — multi-upstream curated knowledge substrate (per itd-36) — see [`05-internals/07-memory.md`](../05-internals/07-memory.md)
- `/abcd` — top-level where-am-i status board (per itd-20) — see [`04-surfaces/08-abcd.md`](../04-surfaces/08-abcd.md)
- `/abcd:reflect` — release retrospective, a design target (itd-24, planned) — see [`04-surfaces/09-reflect.md`](../04-surfaces/09-reflect.md)

**Operator-internal commands** (wiring rather than user-facing surface): `/abcd:run` — the itd-29 autonomous-run operator surface (`status`/`pause`/`resume`/`preflight`; read-mostly over the pluggable autonomous-run seam ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)), v1 never starts or kills the loop) — is not built: itd-29 is superseded (by itd-2609201916151817, planned), and no `run` verb or `commands/run.md` is on any shipped surface. The operator-internal verbs the binary does register are `changelog`, `completion`, `hook`, `rules`, `spec` and `statusline`, and the class is defined by the absence of a `commands/` file rather than by the presence of one. See [`04-surfaces/README.md`](../04-surfaces/README.md) for the user-facing-vs-operator-internal boundary and the record that delivered each.

**Intents: derived, never hand-counted here.** The shelves are the single source
of what is shipped, committed and benched, and this page keeps no static copy of
them (a hand-kept count re-drifts the moment a record moves). The set spans the
three kinds: standalone capabilities, bundle members, and the disciplines, whose
full active roster lives in [`intents/disciplines/`](../../intents/disciplines/);
disciplines have no user moment, and they impose acceptance gates on every other
spec per the three-kinds taxonomy in
[`01-product/03-mental-model.md`](03-mental-model.md) and itd-34.

See [`intents/README.md`](../../intents/README.md) for the intent index. The phase documents under [`roadmap/phases/`](../../roadmap/phases/README.md) are history: [adr-2609212115255771](../../decisions/adrs/2609212115255771-phases-and-milestones-are-retired-sequencing-is-dependencies.md) retired the phase and the milestone as units of the record. Capture history lives in `git log` and each intent file's own provenance, never in this page (per [adr-5](../../decisions/adrs/0005-brief-is-current-state.md)).

**Plumbing infrastructure** (sixteen agents — the canonical roster is the catalog in [`05-internals/01-agents.md`](../05-internals/01-agents.md) — 11 adapters, harness shim, prompt-quality stack, hooks): see [`05-internals/`](../05-internals).

## What comes later

**Work not yet under way lives as press-release intents**: the uncommitted bench in `.abcd/development/intents/drafts/`, plus the committed intents in `planned/` that no lane has taken (per [adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md), lifecycle and scheduling are orthogonal). The bare `/abcd` board places each in its Now / Next / Later block. The canonical out-of-scope list is at [`06-delivery/03-out-of-scope.md`](../06-delivery/03-out-of-scope.md).

The bench is the live `drafts/` corpus; it is enumerated — and kept non-drifting via a filesystem-derived command rather than a hand-count — in the canonical [`06-delivery/03-out-of-scope.md`](../06-delivery/03-out-of-scope.md). itd-31 and itd-32 are superseded (preserved as historical record in `intents/superseded/`).

Each intent captures the press-release-shaped scope and acceptance criteria, and those criteria are its end condition. A draft enters work by being planned: `/abcd:intent plan <itd-N>` mints its spec and moves it to `planned/`. Its dependencies (`blocked_by`, `builds_on`) and the readiness gate (`/abcd:intent ready`) decide when a lane can take it, and an intent that must land by a cut names the release with `/abcd:intent target`, which the preview and the cut report and never refuse on. It reaches `shipped/` when its linked spec is closed: closing the spec is what ships the intent, and there is no separate ship command.

The brief does not get re-versioned. What has shipped is defined by which intents are in `shipped/`, and the checkpoint is the derived release, whose changelog is composed from the records that reached a terminal folder; this brief stays the canonical current-state design record.
