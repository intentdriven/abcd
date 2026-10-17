---
schema_version: 1
id: "iss-2610171626374131"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md"
remedy: "Rewrite the passage at .abcd/development/brief/05-internals/01-agents.md:137 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
refines: [iss-2609091956001547]
---

The v0.15.1 release review found the brief at .abcd/development/brief/05-internals/01-agents.md:137 out of step with the shipped code. It says: The intent-auditor 'has three roles ... Each role has its own verb ... Only the first ships.' In the code: Two roles ship: the same chapter marks role 2 (`abcd intent consistency`) as '(shipped, itd-48)' at line 162, the binary registers `intent consistency`, and agents/intent-auditor.md declares Role 2 (cross-document). Only role 3 (`intent shape`) is unshipped.
