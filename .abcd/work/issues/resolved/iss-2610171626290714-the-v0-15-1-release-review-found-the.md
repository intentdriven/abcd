---
schema_version: 1
id: "iss-2610171626290714"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/README.md:151 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
refines: [iss-2610151422099918]
resolution: "no prose change: the help-groups table's release row already lists embargo and launch, fixed by e4c9d8e1e under iss-2610151422099918 before this record was captured"
impact: internal
resolved_by:
  commit: "e4c9d8e1e"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/README.md:151 out of step with the shipped code. It says: The help-groups table places only `launch` in the people block's release group In the code: `abcd --help` lists both `embargo` and `launch` under Release:, and surface.json records `abcd embargo` with group `release`; embargo appears in no row of the table
