# abcd Phases

The **phase** is abcd's sequencing layer — an ordered stretch of work that ends
in a **milestone**. Phases replace plugin-version language (`v1`, `v2`) as the
way the project organises *what ships together, in what order*. See
[adr-9](../../decisions/adrs/0009-phase-as-product-layer.md) for why this layer
exists and [`01-product/03-mental-model.md`](../../brief/01-product/03-mental-model.md)
for where it sits among brief / phase / intent / spec.

**Two things are called a phase.** This one, and the brief's numbered
plumbing-phases that a phase bundles from — the glossary entry
[`phase`](../../brief/glossary/core/phase.md) names both senses, fixes one
spelling for each, and records that the two numberings are not a mapping.

## What a phase document is

Each phase is a **product-reflection point**. A phase doc opens with the product
thinker's re-statement, in user terms, of what the phase is expected to make
true — coarser than a single intent's press release, finer than the
whole-project brief. Every phase doc carries:

| Section | What it holds |
|---|---|
| `## Expectation` | Working-backwards prose: what is true for the user at phase end. Press-release voice, at phase granularity. The reflection point. |
| `## Milestone` | The concrete engineering done-cut — what must pass for the work to be "finished". |
| `## Phase Acceptance` | Given/When/Then bullets — the *user-truth* cut and the phase-audit target. Roll-up only: asserts emergent, cross-intent truths or phase-spanning journeys; never copies an intent's own acceptance. Same format as intent `## Acceptance Criteria`, one grain up. |
| `## Scope` | Which intents and which brief plumbing-phases this phase bundles. |
| `## Maps against` | Traceability — which brief sections this phase realises; which intents deliver the expectation. |
| `## Dependency rationale` | Why this phase sits where it does; what it must run after. |
| `## Open questions` | Anything not yet decided. |

`## Expectation` (prose) and `## Phase Acceptance` (structured) mirror an
intent's press-release-then-`## Acceptance Criteria` shape, one grain up. The
product thinker authors both. `## Milestone` answers "is the work finished?";
`## Phase Acceptance` answers "is the expectation met?" — see
[adr-9](../../decisions/adrs/0009-phase-as-product-layer.md) and its amendment.

A phase doc carries **no status** — per [adr-5](../../decisions/adrs/0005-brief-is-current-state.md),
status is never stored in design docs. The [roadmap dashboard](../README.md)
renders phase progress by reading the native spec store (and the companion harness `ccpm` when
that backend is attached) via the Go CLI.

## How phases connect to the rest of the model

- **Intents → phase.** The intent → phase mapping is editorial and lives here,
  in each phase doc's `## Scope`. Intent files themselves carry no phase field
  (per adr-9 — phase-grain mapping is editorial, not per-item linkage).
- **Specs → phase.** Phase membership is reconstructed **editorially from each
  phase doc's `## Scope`** — this is the live mechanism. A spec's `phase:` anchor
  is a native-store field: the phase-audit reviewer that reads it and the `PA001`
  verify-exists lint (a `phase:` naming a non-existent phase is an error) are a
  design target for the Go binary — specified in the predecessor store's spc-66,
  not yet shipped. What stays **deferred** is making `phase:` a *standing
  convention* — the corpus anchor backfill onto the unanchored specs is a
  separate planning act. Until that backfill lands, no spec is expected to carry
  the anchor and its absence on the corpus is not drift; a spec listed in no
  phase doc's `## Scope` is implicitly unscheduled and correctly carries no
  anchor (the spec analogue of adr-9's unscheduled-intent rule). `PA001` only
  validates an anchor that IS present — a missing anchor is legal.

## Phase index

| Phase | Milestone | Document |
|---|---|---|
| Phase 0 — Foundations | Go core + adapter seams with native defaults; disciplines in force | [phase-0-substrate.md](phase-0-substrate.md) |
| Phase 1 — Install and launch | `/abcd:ahoy` installs cleanly on any folder; `/abcd:launch` cuts a curated single-repo release | [phase-1-ahoy.md](phase-1-ahoy.md) |
| Phase 2 — History, capture, and memory | Native transcript store, `/abcd:capture` ledger, and `/abcd:memory` substrate | [phase-2-capture.md](phase-2-capture.md) |
| Phase 3 — Intent, brief, and review | `/abcd:intent` + `grill` harden an intent; review runs via the host-delegated oracle; MCP front door opens | [phase-3-intent.md](phase-3-intent.md) |
| Phase 4 — Native spec and task engine | `intent → plan → ship` over the native store, with the companion harness `ccpm` as the deeper backend | [phase-4-spec-engine.md](phase-4-spec-engine.md) |
| Phase 5 — Autonomous run seam | A pluggable run seam (native loop / Workflows / the companion harness's loop) drives specs unattended | [phase-5-run-seam.md](phase-5-run-seam.md) |
| Phase 6 — Lifeboat round-trip | `/abcd:disembark` + `/abcd:embark`; a faithful lifeboat round-trips | [phase-6-lifeboat.md](phase-6-lifeboat.md) |
| Phase 7 — Provenance ledger and cold reading | Origin and grounds stamped at the point of commitment; the cold-reading instrument built, unrun | [phase-7-ledger.md](phase-7-ledger.md) |

Phases 1–7 are organised by **user-capability moment** — each one ends in a
milestone a contributor can demo. **Install and launch (Phase 1) is the first
milestone**, and the delivery order is MVP → the companion harness → Claude Code. Phase 0 is
the exception: it delivers no product-capability user moment, and is numbered 0
to say so honestly — it is the floor the capability phases stand on. (It does
carry one user-typed verb, the `/abcd:intent audit` discipline-audit sub-verb —
the substrate inspecting itself, not a product capability; see
[phase-0-substrate.md](phase-0-substrate.md) for the qualifier.) Phases are sequenced
but their *contents* may run in any dependency-respecting order — see each
phase's `## Dependency rationale`.

## Beyond Phase 7

Phase 6 closes abcd's loop with the lifeboat round-trip and Phase 7 makes the
record show where human judgement went; neither is the end of the work. Most of the intent corpus is not yet phased — an intent listed in no
phase doc's `## Scope` is implicitly unscheduled (per adr-9), and that is a valid
state. Further phases will be authored as that work is committed to.

This section is deliberately a placeholder, not a forecast: naming which intents
land in which future phase before the phase is planned would only go stale. See
[intents/README.md](../../intents/README.md) for the full corpus; an intent moves
into a phase's `## Scope` when the phase that bundles it is written.

One unscheduled cluster is deliberate rather than pending, and is recorded as
such: the **launch deepenings** — the pre-flight gate suite (itd-65), payload
render parity (itd-66), the installable versioned plugin (itd-67), release
retention (itd-70), tier-b publishing (itd-72), and derived versioning
(itd-73). Phase 1 owns only the curated-release cut; the deepenings enter a
phase's `## Scope` when sequenced, per
[adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md). Their
critical/major severities keep them at the top of the derived-priority queue
(itd-78) without a phase anchor.

A second cluster is now unscheduled on the same terms: the **lifeboat coverage
experiment** (itd-88). Per
[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md), the
probe-and-coverage half of the lifeboat is **pulled out of Phase 6** and
sequenced ahead of it. Phase 6's stated dependency on Phases 3, 4 and 5 was
checked against the binary and found mostly false, and the experiment's whole
point is to test the brief's structure *before* a packer is written to it — so
it cannot sit behind the phase whose section list it exists to revise. Phase 6
retains the packer, embark, and the round-trip, and is built to whatever section
list the experiment leaves standing. itd-88 enters no phase's `## Scope` —
scheduled and planned are orthogonal axes (per
[adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md)), so
an intent can be delivered without ever being sequenced into a phase.

## Related Documentation

- [Roadmap](../README.md) — status dashboard
- [Intents](../../intents/README.md) — the intent registry
- [Build sequence](../../brief/06-delivery/01-build-sequence.md) — the brief's plumbing-phase DAG, which phases bundle from
- [adr-9](../../decisions/adrs/0009-phase-as-product-layer.md) — the phase layer decision
