---
id: itd-69
shipped_in: v0.1.0
slug: plugin-metadata-lockstep-update
spec_id: spc-2609211957080006
kind: standalone
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
impact: additive
---

# Plugin Metadata Stays Consistent Across Every Duplicated Surface

> **Closed as delivered on 2026-09-21** on the product thinker's ruling: the lockstep check (`launch.CheckLockstep`, run by every `launch --dry-run`) shipped in v0.1.0 with per-field drift lines, the dev-tree absent-key rule and no bypass flag, while this record sat planned with no spec.


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

## Mechanism

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

## Decisions

Ruled by the product thinker on 2026-09-21: close as delivered.

## Open Questions

_None open._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2cf45c57ec66 -->
Fidelity review — receipt rcp-2cf45c57ec66 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:deb80d504ca7b06cab66460f9f830abbd03ac701f9552ad994fb4c4488af723a
Input attestations: diff:internal/core/launch/lockstep.go, its two test files and internal/surface/cli at chore/audit-run-a-1 80b44890 (git ls-tree -r; spc-2609211957080006 closed, itd-69 shipped in v0.1.0)@sha256:11f3757706d47eee66e74275f84735bfaf8c3b41d56a37a624339073319e4fe9;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: CheckLockstep under TreePublic compares the version field across the pinned manifest paths, returns one Drifts line per disagreeing field, and its ExitCode distinguishes agreement (0), drift (1) and an unreadable contract (2); tests cover agreement, drift, non-semver and the unreadable contract; the concern is that no `--tree public` front door exists — the tree is a Go parameter, the public check runs only inside the payload render, and a public checkout cannot be checked from the CLI
  evidence: internal/core/launch/lockstep.go:21 — "TreePublic LockstepTree = "public""
  evidence: internal/core/launch/lockstep.go:28 — "Drifts []string `json:"drifts,omitempty"`"
  evidence: internal/core/launch/lockstep.go:31 — "ExitCode int `json:"exit_code"` // 0 ok, 1 drift, 2 unreadable"
  evidence: internal/core/launch/lockstep.go:51 — "func CheckLockstep(tree LockstepTree, repoRoot, versionLocationPath string) LockstepResult {"
  evidence: internal/core/launch/lockstep_test.go:45 — "func TestLockstepDrift(t *testing.T) {"
  evidence: internal/core/launch/lockstep_test.go:63 — "func TestLockstepBlockedContractUnreadable(t *testing.T) {"
  evidence: internal/core/launch/render.go:444 — "res.Lockstep = payloadLockstep(TreePublic, dest, vlPath)"
- ac-2 — MET_WITH_CONCERNS: under TreeDev the check requires the version keys absent per adr-19 and reports a present key as drift with exit 1 while a correctly absent key passes with exit 0, as TestLockstepDevKeysAbsent holds; dry-run and ship run it over the source tree; the same concern applies — the dev tree is selected by the calling verb, not by a `--tree dev` flag
  evidence: internal/core/launch/lockstep.go:19 — "TreeDev LockstepTree = "dev""
  evidence: internal/core/launch/lockstep_test.go:74 — "func TestLockstepDevKeysAbsent(t *testing.T) {"
  evidence: internal/core/launch/dryrun.go:114 — "lockstep := CheckLockstep(TreeDev, req.RepoRoot, vlPath)"
  evidence: internal/core/launch/ship.go:80 — "lockstep := CheckLockstep(TreeDev, req.RepoRoot, vlPath)"
- ac-3 — MET: the checker takes no skip or dirty argument and the launch verb exposes no flag that reaches it; dry-run folds every drift line and an unreadable contract into WouldRefuseOn unconditionally, ship folds the same into BlockReasons, and the render refuses public-payload drift with ErrPayloadDrift — `--allow-dirty` concerns the working tree, not manifest agreement
  evidence: internal/core/launch/dryrun.go:282 — "func wouldRefuseOn(bundle Bundle, scan scanner.ScanResult, lockstep LockstepResult, retention RetentionPlan, smoke SmokeReport) []string {"
  evidence: internal/core/launch/ship.go:102 — "report.BlockReasons = wouldRefuseOn(bundle, scan, lockstep, report.Retention, report.Smoke)"
  evidence: internal/core/launch/render.go:446 — "return res, fmt.Errorf("%w: %s", ErrPayloadDrift, strings.Join(lockstepDetail(res.Lockstep), "; "))"

Gap audit:
- honoured:
  - per-field drift with three distinct exit outcomes
    evidence: internal/core/launch/lockstep.go:31 — "ExitCode int `json:"exit_code"` // 0 ok, 1 drift, 2 unreadable"
  - the dev tree carries absent version keys and a present one is drift
    evidence: internal/core/launch/lockstep_test.go:74 — "func TestLockstepDevKeysAbsent(t *testing.T) {"
  - no bypass at the checker's own layer
    evidence: internal/core/launch/ship.go:102 — "report.BlockReasons = wouldRefuseOn(bundle, scan, lockstep, report.Retention, report.Smoke)"
- diverged:
  - the checker runs with `--tree public` or `--tree dev`
    evidence: internal/core/launch/lockstep.go:51 — "func CheckLockstep(tree LockstepTree, repoRoot, versionLocationPath string) LockstepResult {"
    evidence: internal/core/launch/render.go:444 — "res.Lockstep = payloadLockstep(TreePublic, dest, vlPath)"
- missing: (none)

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. The grill/PRD
bypass for this ungrilled intent is handled via the grandfather fields
(`prd_grandfathered` for GR002; two-key `grandfathered` + `grandfathered_at_phase`
for GR001). Full record in the spec's process-exception note.

## Grounds

- pursued: the record is being closed for work v0.1.0 carried so the store matches what ships; shown wrong if the lockstep check is found not to meet the criteria this record keeps
