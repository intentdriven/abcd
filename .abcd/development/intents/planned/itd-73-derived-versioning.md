---
id: itd-73
slug: derived-versioning
spec_id: spc-10
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
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

_None yet — this intent has not been reviewed._
