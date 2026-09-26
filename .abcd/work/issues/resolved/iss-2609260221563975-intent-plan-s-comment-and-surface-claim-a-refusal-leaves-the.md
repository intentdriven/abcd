---
schema_version: 1
id: "iss-2609260221563975"
slug: "intent-plan-s-comment-and-surface-claim-a-refusal-leaves-the"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
resolution: "An in-place intent plan refused after spec.Create removes the spec it minted, so the refusal leaves the record and the spec store as they were; the comment states both sides of the mint."
impact: fix
resolved_by:
  commit: "d70a6753"
---

intent plan's comment and surface claim a refusal leaves the record byte-identical with no spec minted, but the spec is minted before the intent write: with planned/ read-only, intent plan refuses (exit 2) and leaves an orphan spc file under specs/open/ that a retry reuses. Self-healing, not atomic; the claim overstates it.

## Grounds

- pursued: with planned/ read-only, intent plan on a null-spec planned record refuses and no spec names the intent afterwards, and the retry mints exactly one (TestPlanInPlaceRefusalAfterTheMintLeavesNoSpec); a spc file left under specs/open/ after such a refusal would show it wrong
