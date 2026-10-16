---
schema_version: 1
id: "iss-2610160857121531"
slug: "the-brief-says-abcd-update-is-the-only"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "release provenance documentation fixes, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Name ahoy install's provenance check as a second network reach in the update chapter and wherever the brief lists abcd's network reaches, with the destinations it contacts"
refines: [iss-2610160608419844, iss-2610141706058429, iss-2610141706064461]
resolution: "The update chapter, the brief's one list of network reaches, now says an ahoy installation reaches GitHub's API and Sigstore's TUF repository through the same provenance check before it promotes or adopts the PATH copy"
impact: fix
resolved_by:
  commit: "50dea413e"
---

The brief says abcd update is the only command that reaches the network, but ahoy install also reaches GitHub's attestations API and Sigstore's TUF repository when it checks a downloaded binary's provenance (internal/core/update/provenance.go)
