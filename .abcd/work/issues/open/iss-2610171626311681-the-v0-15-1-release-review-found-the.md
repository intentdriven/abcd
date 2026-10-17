---
schema_version: 1
id: "iss-2610171626311681"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/01-ahoy.md:516 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/01-ahoy.md:516 out of step with the shipped code. It says: The ~/.abcd.noindex inventory 'is stated as a table under The two .abcd/ scopes in 05-internals/03-configuration.md; the two are one list and must agree.' In the code: 05-internals/03-configuration.md 'The two .abcd/ scopes' (line 596ff) states the user-scope inventory as a prose paragraph, not a table, and it omits two entries the 01-ahoy tree lists and the binary writes: interviews/ (internal/core/interview/interview.go homeRecordsLeaf) and backups/ (internal/core/ahoy/statusline_…
