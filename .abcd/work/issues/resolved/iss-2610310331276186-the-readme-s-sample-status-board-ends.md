---
schema_version: 1
id: "iss-2610310331276186"
slug: "the-readme-s-sample-status-board-ends"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Replace the version line of the README board example with a placeholder that names no release, so it never goes stale."
resolution: "the user-facing page or help example now states what the binary does"
impact: fix
---

The README's sample status board ends with a version line naming v0.13.4, though the board's last line names the running version and the release is newer.
