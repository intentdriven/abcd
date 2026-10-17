---
schema_version: 1
id: "iss-2610171626347542"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/37-reserve.md:84-93 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/37-reserve.md:84-93 out of step with the shipped code. It says: Each reservation commit's trailers name the record (Abcd-Reservation), holder, copy, footprint, lease (Abcd-Claimed-At, Abcd-Expires-At) and the Assisted-by label; a release adds Abcd-Released-At; a hand-back adds Abcd-Handed-Back and Abcd… In the code: A drain lane's renewal of the reservation also writes an Abcd-Stage trailer (internal/core/claim/stage.go keyStage, one of worktree|brief|implement|validate|land, validated by validateStage on read). The reserve chapter never names it; its only documented home is .abcd/development/brief/04-surfaces/08-abcd.md:241-242.
