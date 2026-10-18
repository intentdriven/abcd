---
schema_version: 1
id: "iss-2610171626323993"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/05-intent.md:412 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "the press-release template's Grounds comment now says the gate reports it on an advisory row that never withholds readiness"
impact: internal
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/05-intent.md:412 (press-release template, ## Grounds comment) out of step with the shipped code. It says: Grounds is 'Required at the readiness gate for a press-release intent'. In the code: internal/core/intent/ready.go groundsCheck builds its row with Advisory: true, so a missing Grounds section never withholds readiness. The same page says so at line 317: 'the grounds row ... advisory and never withhold readiness'. Only bucket, acceptance_criteria, spec_link, spec_body and open_questions gate.
