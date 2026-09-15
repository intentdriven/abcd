---
id: itd-162
slug: adoption-templates-outside-record
spec_id: spc-54
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-87
---

# prepare-this-repo adopt phase is not self-contained in the abcd record: Phase 3-5 reference templates at a machine-local path outside both the abcd repo and the target (pre-commit-config.yaml, prepare-commit-msg, AGENTS.md, DECISIONS.md, NEXT.md). A fresh clone of abcd-cli onboarding a repo would not have them, so the adoption step silently degrades against loud-staging. Detector: an onboarding self-containment check -- every asset the adopt phase applies resolves from within the abcd record or the binary, never an external machine-local path. Acceptance: the Phase 3/4/5 template references in prepare-this-repo.md.

## Press Release

> _Seeded by promotion from iss-87. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-87`: prepare-this-repo adopt phase is not self-contained in the abcd record: Phase 3-5 reference templates at a machine-local path outside both the abcd repo and the target (pre-commit-config.yaml, prepare-commit-msg, AGENTS.md, DECISIONS.md, NEXT.md). A fresh clone of abcd-cli onboarding a repo would not have them, so the adoption step silently degrades against loud-staging. Detector: an onboarding self-containment check -- every asset the adopt phase applies resolves from within the abcd record or the binary, never an external machine-local path. Acceptance: the Phase 3/4/5 template references in prepare-this-repo.md.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a machine without the machine-local `~/ABCDevelopment/.agents/templates/` directory, **when** prepare-this-repo's adopt phase runs, **then** it applies the templates from the committed record or the embedded binary and the adoption completes rather than silently degrading.
- **Given** the adopt phase's assets (the pre-commit config, the prepare-commit-msg hook, AGENTS.md, DECISIONS.md and NEXT.md templates), **when** they are resolved, **then** every one resolves from within the abcd record or the binary and never from an external machine-local path.
- **Given** the onboarding self-containment check, **when** it scans prepare-this-repo, **then** no reference to the `~` machine-local templates path remains.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
