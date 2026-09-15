---
id: itd-69
slug: plugin-metadata-lockstep-update
spec_id: null
kind: bundle-member
bundle: spc-83-operator-surfaces
suggested_kind: standalone
reclassification_history: []
prd_path: null
prd_grandfathered: true
grandfathered: true
grandfathered_at_phase: phase-6-launch
glossary_terms_used:
  - core/brief
  - distribution/version
severity: minor
builds_on: [itd-67]
---

# Plugin Metadata Stays Consistent Across Every Duplicated Surface

## Press Release

> **abcd guards against version and changelog drift across its duplicated plugin
> metadata surfaces.** The plugin's version and changelog live in more than one
> place — both `plugin.json` locations and `.claude-plugin/marketplace.json` — so
> a launch/version-bump workflow that updates one and forgets another leaves the
> published surfaces disagreeing. abcd ships a read-only consistency checker that
> proves the canonical metadata locations agree, refusing when they drift. The
> version WRITES stay with the launch/bump workflow; this is the anti-drift
> invariant that proves the writes landed everywhere they had to.
>
> "A half-bumped plugin is worse than an un-bumped one — the marketplace says one
> thing and the plugin says another, and nobody notices until an install breaks,"
> said Kira, a maintainer publishing a new build. "The lockstep check catches the
> disagreement before it publishes."

_Drawn out from a human brief edit by the brief-change derivation gate
(itd-61 / spc-75)._

## Why This Matters

Launch and version-bump workflows must update plugin metadata in lockstep at the
canonical plugin metadata locations, including both `plugin.json` files and
`.claude-plugin/marketplace.json`, so version and changelog state cannot drift
across duplicated surfaces. Where the version is WRITTEN is owned by the launch
flow (spc-77 / spc-80); what no surface owned before was the invariant that the two
manifests actually AGREE after a write. A drifted pair publishes a broken plugin:
the marketplace advertises one version, the plugin declares another, and the
mismatch surfaces only when a downstream install fails. A cheap read-only checker
turns that latent drift into an early, loud refusal.

## What's In Scope

- A read-only consistency checker (module + CLI) that proves the ADR-pinned
  metadata path set agrees across `plugin.json` and
  `.claude-plugin/marketplace.json`, with an explicit `--tree dev|public`
  argument (no auto-detection).
- A policy ADR recording the lockstep invariant, the per-tree pinned path list,
  and the `--allow-dirty`-must-not-bypass rule (enforced downstream by
  spc-79/spc-80 wiring).
- The published-marketplace changelog-entry schema the bump step consumes.

## What's Out of Scope

- Version WRITES — owned by spc-77 / spc-80; this checker never bumps a version.
- Preflight WIRING of the checker into the launch gate suite — spc-79 / spc-80.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a public tree whose `plugin.json` and `marketplace.json` versions
  disagree across the ADR-pinned path list, **when** the checker runs with
  `--tree public`, **then** it refuses with per-field drift lines and a non-zero
  exit distinct from the consistent and contract-unreadable exits.
- **Given** a dev tree that (per adr-19) must carry ABSENT version keys, **when**
  the checker runs with `--tree dev`, **then** a present version key is reported
  as drift and a correctly-absent key passes.
- **Given** the checker binary, **when** it is inspected, **then** it exposes no
  dirty/skip bypass flag — manifest consistency cannot be waved through at its
  own layer.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. The grill/PRD
bypass for this ungrilled intent is handled via the grandfather fields
(`prd_grandfathered` for GR002; two-key `grandfathered` + `grandfathered_at_phase`
for GR001). Full record in the spec's process-exception note.
