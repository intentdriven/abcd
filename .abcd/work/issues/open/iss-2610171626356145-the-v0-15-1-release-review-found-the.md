---
schema_version: 1
id: "iss-2610171626356145"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/39-status.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/39-status.md:28 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/39-status.md:28 out of step with the shipped code. It says: The verb fetches 'each status branch whose name is a GitHub username in lower case'; only a non-lower-case-username name is left unfetched. In the code: The binary fetches only the first 500 such branches (internal/core/claim/status.go maxStatusBranches = 500); branches past 500 are counted and not fetched, and their tracking copies deleted, as `abcd status --help` states. The brief never mentions the 500 cap.
