---
schema_version: 1
id: "iss-2611040554393460"
slug: "05-intent-md-508-says-an-intent-cannot"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "State that the plan gate requires a bullet and does not check its form, or make the check read the form."
---

05-intent.md:508 says an intent cannot be planned without an Acceptance Criteria section carrying at least one Given-When-Then bullet, and that everything in planned/ and shipped/ has therefore passed it. hasAcceptanceCriteria (internal/core/intent/intent.go:163) counts top-level -/* bullets only: a draft whose whole section is one plain bullet with no Given, When or Then plans with exit 0.
