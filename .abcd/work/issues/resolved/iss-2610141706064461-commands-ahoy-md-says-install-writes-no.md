---
schema_version: 1
id: "iss-2610141706064461"
slug: "commands-ahoy-md-says-install-writes-no"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ahoy.md"
remedy: "Name the GitHub attestations or git API or the Sigstore TUF repository."
refines: [iss-2610141706058429]
resolution: "commands/ahoy.md now says install writes no entry when the provenance check fails or cannot reach GitHub's attestations or git API or the Sigstore TUF repository, naming the git API lookup of the tag's commit that the page left out. The detector test was watched fail and pass but is not committed: the hold could not be widened to a test file from the stage (the reservation push had no credential), so its source is kept in the lane directory and the report says so."
impact: fix
resolved_by:
  commit: "ec86723145d849ad0880bfc970568d4c365cdbe4"
---

commands/ahoy.md says install writes no entry when the check fails or cannot reach the GitHub attestations API or the Sigstore TUF repository, omitting that a failed GitHub git API lookup of the tag commit refuses too (internal/core/update/provenance.go); docs/how-to/install.md already names both APIs.

## Grounds

- pursued: the ahoy page's sentence on when install writes no entry names the attestations API, the git API and the Sigstore TUF repository, as docs/how-to/install.md does and as internal/core/update/provenance.go reaches them; a provenance check that makes no request to GitHub's git API, or a page sentence still omitting it, would show it wrong
