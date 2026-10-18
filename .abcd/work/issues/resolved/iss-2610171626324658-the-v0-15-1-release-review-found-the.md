---
schema_version: 1
id: "iss-2610171626324658"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/07-library.md:174-177 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/07-library.md:174-177 out of step with the shipped code. It says: How a scaffolded hook finds abcd, and whether it does so by default, is a ruling owed to the product thinker (iss-2609250834251447) In the code: iss-2609250834251447 is in .abcd/work/issues/resolved/ with a recorded ruling: `ahoy install --outbound-hook` opts in, never installed by default, finds abcd only through git config --local abcd.binary, and fails open with one warning; the chapter still presents the ruling as owed
