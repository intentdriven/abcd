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
---

intent plan's comment and surface claim a refusal leaves the record byte-identical with no spec minted, but the spec is minted before the intent write: with planned/ read-only, intent plan refuses (exit 2) and leaves an orphan spc file under specs/open/ that a retry reuses. Self-healing, not atomic; the claim overstates it.
