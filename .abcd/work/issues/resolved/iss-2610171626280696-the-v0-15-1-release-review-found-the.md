---
schema_version: 1
id: "iss-2610171626280696"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite the passage at .abcd/development/brief/02-constraints/04-naming.md:54-59 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "the naming chapter's plain exemption list now registers announce and status; embargo, land, memory and reserve were added by ea9a638df"
impact: internal
---

The v0.15.1 release review found the brief at .abcd/development/brief/02-constraints/04-naming.md:54-59 out of step with the shipped code. It says: The remaining surfaces are exempt ... `banlist`, `build`, ... `statusline`, and `update`. They are registered here so the exemption is on the record (and line 9-10: an exemption is a decision rather than an omission). In the code: Six shipped top-level verbs appear nowhere in the naming register, neither as maritime nor as exempt: `announce`, `embargo`, `land`, `memory`, `reserve`, `status` (all in root --help --agent and commands/; documented in 04-surfaces/38-announce.md, 37-embargo.md, 38-land.md, 37-memory.md, 37-reserve.md, 39-status.md).
