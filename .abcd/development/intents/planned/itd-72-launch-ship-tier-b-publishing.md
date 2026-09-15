---
id: itd-72
slug: launch-ship-tier-b-publishing
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-28]
prd_path: null
prd_grandfathered: true
grandfathered: true
grandfathered_at_phase: phase-6-launch
glossary_terms_used:
  - core/intent
  - core/spec
  - distribution/release
  - distribution/version
  - distribution/end-user
blocked_by: [itd-67]
severity: major
builds_on: [itd-65, itd-66]
---

# Publishing A Release Is One Atomic, Refusable `ship`

> _Retroactive intent record for spc-80. spc-80 builds the Tier B (publishing)
> half of the itd-67 consolidated PRD — see
> [itd-67](itd-67-installable-versioned-plugin.md), whose grilled PRD defines
> both tiers. This record exists so the spec resolves to an intent under
> `planned/`/`shipped/`; the two-key grandfather marks that it did not pass
> through its own grill (the grilled provenance lives on itd-67)._

## Press Release

> **abcd publishes a new plugin release with one command — or refuses, loudly.**
> `ship` computes the bump tier from the last launch baseline, writes the
> version and changelog into the curated release, prunes superseded payload
> files, stages the delta for confirmation, and then publishes branch + tag
> atomically. If nothing changed, it refuses. If the repo is not the one it
> expects, it refuses. If the push fails, nothing half-lands: a recoverable
> marker preserves the local commit and `main` is never force-reset.

## Why This Matters

Tier A (itd-67) makes the abcd repo installable and versioned; without
Tier B every subsequent release is a hand-run of renders, bumps, changelog
edits, prunes, and pushes — each a chance to publish a half-state. The
release is a single-repo curated cut from packaging ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)),
not a mirror push. `ship` makes that cut a single audited act with every
safety refusal built in.

## What's In Scope

- Bump-tier computation from the `.launch/launch-report.json` baseline
  (phase-complete since baseline → minor; otherwise patch; explicit
  `--version` always wins and is the only major path)
- First-release bootstrap: an absent baseline seeds `v0.1.0` and prepares the
  initial baseline record
- No-change refusal on the canonical pre-bump payload digest (`--force` to
  override)
- Repo identity check (marketplace name, plugin name, remote slug,
  protected branch) before any write
- Overlay write + tracked-manifest prune, payload-scoped only, with
  traversal/symlink-escape defenses
- Auto-changelog from git tags + the launch report, idempotent
- Staged-delta confirmation, non-TTY refusal without `--yes`
- Atomic branch+tag push or fail-closed with a recoverable
  `failed-release.json` marker; stale-verdict revalidation immediately before
  promotion

## What's Out of Scope

- The payload resolution itself (spc-78's render manifest is consumed, never
  re-derived)
- The pre-flight gate suite (spc-79 owns it; `ship` invokes it)
- Release retention pruning of prior tags/Releases (itd-70)

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by
> `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** no baseline exists in the repo, **when** `ship` runs,
  **then** the seeded version is `v0.1.0`, every surface is treated as added,
  and the initial baseline record is prepared.
- **Given** the pre-bump payload digest equals the baseline's, **when** `ship`
  runs without `--force`, **then** it refuses as a no-change release and
  publishes nothing.
- **Given** the repo's marketplace name, plugin name, remote slug, or
  protected branch does not match the launch config, **when** `ship` runs,
  **then** it refuses before any write.
- **Given** a prune entry resolving outside the payload root (absolute,
  `..`-traversal, or symlink escape), **when** the overlay+prune stage runs,
  **then** the entry is rejected and nothing outside the payload root is
  deleted.
- **Given** `--publish` in a non-TTY session without `--yes`, **when** `ship`
  reaches confirmation, **then** it refuses to publish.
- **Given** the atomic branch+tag push fails partway, **when** `ship` exits,
  **then** the protected branch is never force-reset and a recoverable
  `failed-release.json` marker preserves the local release commit.
- **Given** the gate verdict was computed against different manifest bytes or
  git state, **when** `ship` revalidates immediately before promotion,
  **then** the stale verdict is rejected and nothing is pushed.

## Audit Notes

_Populated by intent-fidelity-reviewer when intent moves to shipped/._
