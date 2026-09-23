---
id: itd-73
shipped_in: v0.4.0
slug: derived-versioning
spec_id: spc-10
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
impact: additive
---

# abcd Cuts the Version — You Never Type One

## Press Release

> **Ship intents; abcd derives the version.** You work in features and intents, never in `vX.Y.Z`. Each intent declares one thing about itself — whether it *adds*, *breaks*, or *fixes* — and when `/abcd:launch` cuts a release, abcd reads the intents that shipped since the last release and computes the SemVer for you: any breaking change is a major bump, any new capability a minor, a fix-only release a patch. The working tree stays unversioned (ADR-19); the number appears only on the release artefact, generated, never edited.
>
> "I stopped thinking about version numbers entirely," said Kira, a maintainer. "I write intents and mark each one additive, breaking, or fix. abcd cuts `v2.0.0` when something actually breaks — and it will not let me ship a minor that quietly removed a command. The version finally means what SemVer promises, without anyone choosing it."

## Why This Matters

A version number is not a decision to make — it is a fact about what changed. Asking a person to pick `v1.4.0` forces release-thinking (which number?) onto what should be product-thinking (what does this feature do to compatibility?). abcd already makes the intent the unit of the *why*; the version should fall out of the intents in a release, not be authored beside them.

SemVer remains a real contract — consumers of the plugin rely on a major bump meaning "something broke". So the one irreducible input, *is this change breaking?*, is captured where the judgement naturally belongs: on the intent, once, when it is written. abcd aggregates those judgements at release time and mechanically guards against a mislabel, so the derived number keeps SemVer's promise without anyone thinking in versions.

## What's In Scope

- An `impact` classification on every intent: `additive` (backward-compatible capability), `breaking` (incompatible surface or behaviour change), or `fix` (defect repair). Set once when the intent is shaped; it is a product judgement, not a version.
- Version derivation in `/abcd:launch`: gather the intents shipped since the previous release, take the highest-severity `impact`, and compute the next SemVer — any `breaking` → major, else any `additive` → minor, else patch.
- A surface-diff guardrail: `launch` snapshots the `/abcd:*` command, flag, and manifest surface and compares it to the previous release. A removed or changed surface with no `breaking` intent in the release fails the launch — a mislabel cannot silently ship a compatibility lie.
- The derived version lands only on the release artefact — the `v*` tag, the GitHub Release, and the marketplace manifest — consistent with ADR-19 (unversioned working tree) and ADR-20 (manifest lockstep).

## What's Out of Scope

- Pre-1.0 and pre-release channel semantics (alpha/beta/rc suffixes) — a later refinement once the derivation is trusted.
- Per-consumer compatibility ranges or deprecation windows — downstream concerns, not the cut.
- Changelog *prose* generation — the changelog is auto-recorded from the shipped intents as a separate capability; this intent decides only the version number.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a set of intents shipped since the last release whose highest `impact` is `additive`, **when** `/abcd:launch` computes the next version, **then** the minor component is incremented and the patch reset (for example `v1.3.2` → `v1.4.0`) with no human input.
- **Given** at least one shipped intent declares `impact: breaking`, **when** `launch` computes the version, **then** the major component is incremented and minor and patch reset (for example `v1.4.0` → `v2.0.0`).
- **Given** the shipped intents are all `impact: fix`, **when** `launch` computes the version, **then** only the patch component is incremented.
- **Given** the release removes or changes a `/abcd:*` command or flag but no shipped intent declares `impact: breaking`, **when** `launch` runs the surface-diff guardrail, **then** the launch fails with a report naming the changed surface — the mislabel is blocked, not shipped.
- **Given** an intent is captured or shaped, **when** its `impact` field is absent or is not one of `additive`, `breaking`, or `fix`, **then** `internal/core/lint` flags it as a blocker — every intent carries a valid impact before it can ship.
- **Given** a release is cut, **when** the version is computed, **then** the number is written only to the release artefact (tag, GitHub Release, marketplace manifest) and the working tree carries no version, honouring ADR-19.

## Open Questions

- Where does `impact` live for a bundle — on each member, or once on the bundle? Leaning: each member declares its own, and the bundle's impact is the maximum of its members, mirroring the release aggregation.
- Does a `breaking` discipline change — an acceptance gate that newly fails existing specs — count toward the version, or only surface and behaviour intents? Leaning: yes, a newly-enforced discipline that breaks consumers is `breaking`.
- Should the surface-diff guardrail also catch *behavioural* breaks behind an unchanged surface, or is that explicitly the author's `impact` call? Leaning: the surface-diff catches structural breaks only; behavioural breaks remain the intent author's `breaking` judgement.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0d18ea2d4682 -->
Fidelity review — receipt rcp-0d18ea2d4682 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:505eb999a5cd60215427c766ccdda96b4433fb36d00128305d7392a67a0a9417
Input attestations: diff:tree at de3ba5fa (spc-10 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: DeriveNext maps a highest impact of additive to minor++ with patch reset at >= 1.0 (1.2.3 -> 1.3.0 in the table test) with no human input; the concern is the pre-1.0 row, where additive bumps the PATCH (0.3.0 -> 0.3.1), and the repo sits at 0.9.0 so every cut to date has used that diverged row (ADR-37 as amended)
  evidence: internal/core/changelog/version.go:54 — "case bump == ImpactAdditive:"
  evidence: internal/core/changelog/version_test.go:31 — "{"stable additive bumps minor", "1.2.3", ImpactAdditive, "1.3.0", true},"
  evidence: internal/core/changelog/version_test.go:36 — "{"pre-1.0 additive bumps patch", "0.3.0", ImpactAdditive, "0.3.1", true},"
  evidence: internal/core/changelog/version.go:25 — "The pre-1.0 row is load-bearing, not a shortcut."
- ac-2 — MET_WITH_CONCERNS: a breaking impact bumps the major with minor and patch reset at >= 1.0 (1.2.3 -> 2.0.0 in the table test); the concern is the pre-1.0 row, where breaking bumps the MINOR and no input can ever derive 1.0.0 — the first stable version is an out-of-band human roll the verb does not offer
  evidence: internal/core/changelog/version.go:49 — "case bump == ImpactBreaking:"
  evidence: internal/core/changelog/version_test.go:30 — "{"stable breaking bumps major", "1.2.3", ImpactBreaking, "2.0.0", true},"
  evidence: internal/core/changelog/version_test.go:35 — "{"pre-1.0 breaking bumps minor", "0.3.0", ImpactBreaking, "0.4.0", true},"
  evidence: internal/core/changelog/version_test.go:66 — "func TestDeriveNextNeverDerivesFirstStable"
- ac-3 — MET: an all-fix cut increments only the patch component, at >= 1.0 and pre-1.0 alike, per the table test
  evidence: internal/core/changelog/version.go:57 — "default: // ImpactFix, the only remaining bump-driving member."
  evidence: internal/core/changelog/version_test.go:32 — "{"stable fix bumps patch", "1.2.3", ImpactFix, "1.2.4", true},"
  evidence: internal/core/changelog/version_test.go:37 — "{"pre-1.0 fix bumps patch", "0.3.0", ImpactFix, "0.3.1", true},"
- ac-4 — MET: GuardSurface diffs the current surface snapshot against the one at the last release tag; a break with no breaking record in the cut fails with a reason naming every changed command, flag and manifest entry, and the emit step turns that into a surface-guard refusal of the cut
  evidence: internal/core/changelog/guard.go:96 — "func GuardSurface(root string, current surface.Snapshot) (SurfaceGuard, error) {"
  evidence: internal/core/changelog/guard.go:137 — "g.Reason = failureReason(g.BaseTag, g.Breaks)"
  evidence: internal/core/release/emit.go:207 — "if guard.Status != changelog.SurfaceGuardPassed {"
  evidence: internal/core/changelog/guard_test.go:390 — "func TestGuardSurfaceReportsEveryBreak"
- ac-5 — MET_WITH_CONCERNS: intent_impact_valid is armed as a blocker in the real config and flags an invalid or internal impact in every bucket and an absent one in shipped/; the concern is that an ABSENT impact passes at capture and shaping (drafts/, planned/) — create.go leaves impact optional on a draft — so the bar bites at the move into shipped/, narrower than 'captured or shaped'
  evidence: internal/core/lint/lint.go:1754 — "func checkIntentImpact(tree intentTree, cfg RuleConfig) []Finding {"
  evidence: internal/core/lint/lint.go:1769 — "if r.bucket == "shipped" {"
  evidence: internal/core/lint/impact_test.go:71 — "func TestIntentImpactValid"
  evidence: internal/core/lint/impact_test.go:240 — "func TestImpactRulesArmedInRealConfig"
  evidence: internal/core/intent/create.go:225 — "impact is optional on a draft"
- ac-6 — MET_WITH_CONCERNS: the render stamps the derived number into the payload copies only and the working-tree manifests are proved unchanged and version-absent; the concerns are that the in-tree CHANGELOG dated heading does carry the version (ADR-37), and that the stamped marketplace manifest lives only in a payload staged with --payload-dir that release.yml never publishes — the tag and GitHub Release carry it, the marketplace manifest users install does not
  evidence: internal/core/launch/render_test.go:94 — "func TestRenderPayloadLeavesSourceTreeUnversioned"
  evidence: internal/core/launch/render.go:386 — "editManifest reads a manifest from the PAYLOAD (never the source tree)"
  evidence: internal/surface/cli/ship.go:99 — "func publishedVersion(repoRoot string) string {"
  evidence: .github/workflows/release.yml:327 — "gh release create "${TAG}" bin/abcd-* bin/checksums.txt"

Gap audit:
- honoured:
  - every intent declares an impact of additive, breaking or fix, enforced by lint
    evidence: internal/core/lint/lint.go:131 — "intentImpactValues = joinImpacts(changelog.ImpactAdditive, changelog.ImpactBreaking, changelog.ImpactFix)"
    evidence: internal/core/lint/impact_test.go:240 — "TestImpactRulesArmedInRealConfig"
  - the version is derived from the highest impact of the records shipped since the last tag
    evidence: internal/core/changelog/version.go:38 — "func DeriveNext"
    evidence: internal/core/changelog/impact.go:109 — "func MaxImpact(impacts []Impact) Impact {"
  - a surface-diff guardrail blocks a mislabelled break and names the surface
    evidence: internal/core/changelog/guard.go:96 — "func GuardSurface"
    evidence: internal/core/changelog/guard_test.go:390 — "TestGuardSurfaceReportsEveryBreak"
  - the working tree stays unversioned (ADR-19)
    evidence: internal/core/launch/render_test.go:94 — "TestRenderPayloadLeavesSourceTreeUnversioned"
- diverged:
  - breaking -> major, additive -> minor — delivered with a pre-1.0 row (breaking -> minor, additive -> patch) that the criteria do not state and that every cut so far has used
    evidence: internal/core/changelog/version.go:21 — "prev is 0.x breaking -> minor++, patch = 0"
  - impact flagged as a blocker when captured or shaped — an absent impact is flagged only in shipped/
    evidence: internal/core/lint/lint.go:1769 — "if r.bucket == "shipped" {"
  - the number lives only on the release artefact — the CHANGELOG dated heading in the tree is the carrier the cut reads
    evidence: internal/surface/cli/ship.go:99 — "func publishedVersion"
- missing:
  - the marketplace manifest carrying the derived version as a published artefact — the stamped payload is optional and unpublished
    evidence: internal/surface/cli/ship.go:224 — "if payloadDir != "" && ingested.Written {"
    evidence: .github/workflows/release.yml:327 — "gh release create "${TAG}" bin/abcd-* bin/checksums.txt"