---
id: itd-2609150819432059
slug: abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2609061432214212
origin: extracted-from-record
production_mode: hand-written
---

# abcd launch cannot set up the release flow for a managed repo that is not a plugin. Observed adopting abcd in a managed Go macOS app (own tag-driven release workflow building on a macOS runner with minisign, no CHANGELOG.md, no .claude-plugin/plugin.json): 'abcd launch --dry-run' fails with 'include config not found: .abcd/config/launch-payload.json'; 'abcd changelog' and 'abcd launch ship' fail reading .claude-plugin/plugin.json; 'abcd launch scaffold' writes the generic ubuntu Go template (verify/build/publish) that replaces the repo's own release workflow, and nothing creates the pieces the flow presupposes: CHANGELOG.md with its empty [Unreleased] anchor, the launch payload include config, a version location for a non-plugin artefact, and (where detectors are configured) the release-gate manifest and receipts directory. Needed: an adoption step (ahoy install or launch scaffold --init) that lays these down for a managed repo, a way to declare a non-plugin version location and payload, and a template extension point for platform-specific build/publish steps so scaffold parity does not fight a macOS build. Until then a managed repo has to hand-port the template.

## Press Release

> _Seeded by promotion from iss-2609061432214212. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2609061432214212`: abcd launch cannot set up the release flow for a managed repo that is not a plugin. Observed adopting abcd in a managed Go macOS app (own tag-driven release workflow building on a macOS runner with minisign, no CHANGELOG.md, no .claude-plugin/plugin.json): 'abcd launch --dry-run' fails with 'include config not found: .abcd/config/launch-payload.json'; 'abcd changelog' and 'abcd launch ship' fail reading .claude-plugin/plugin.json; 'abcd launch scaffold' writes the generic ubuntu Go template (verify/build/publish) that replaces the repo's own release workflow, and nothing creates the pieces the flow presupposes: CHANGELOG.md with its empty [Unreleased] anchor, the launch payload include config, a version location for a non-plugin artefact, and (where detectors are configured) the release-gate manifest and receipts directory. Needed: an adoption step (ahoy install or launch scaffold --init) that lays these down for a managed repo, a way to declare a non-plugin version location and payload, and a template extension point for platform-specific build/publish steps so scaffold parity does not fight a macOS build. Until then a managed repo has to hand-port the template.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
