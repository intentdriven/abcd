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
---

commands/ahoy.md says install writes no entry when the check fails or cannot reach the GitHub attestations API or the Sigstore TUF repository, omitting that a failed GitHub git API lookup of the tag commit refuses too (internal/core/update/provenance.go); docs/how-to/install.md already names both APIs.
