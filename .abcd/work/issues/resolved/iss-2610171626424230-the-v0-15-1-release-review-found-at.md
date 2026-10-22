---
schema_version: 1
id: "iss-2610171626424230"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Rewrite the passage at README.md:69; to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found at README.md:69; commands/announce.md: INCOMPLETE. The README says '`abcd status` needs git 2.29 or later, which reads a fetch's branches from its standard input (`git fetch --stdin`)'. `abcd announce` fetches the status branches by name through the same path: internal/core/claim/announce.go:95-96 sets ns.only and calls fetchWith, and remote.go:765-766 switches to `fetch --stdin` whenever a namespace has ns.only. So announce needs git 2.29 too once origin holds any status branch, and on an older git it refuses with an error that names 'the status fetch'. commands/announce.md does not mention…
