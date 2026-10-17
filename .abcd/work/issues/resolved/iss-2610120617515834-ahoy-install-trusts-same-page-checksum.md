---
schema_version: 1
id: "iss-2610120617515834"
slug: "ahoy-install-trusts-same-page-checksum"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "E5 fix lane, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/owned_copy.go"
remedy: "Verify the release's SLSA build provenance before promoting, through the same in-process check abcd update now uses (internal/core/update/provenance.go; one canonical primitive), refusing on any failure; prove with the recorded-bundle tests used for update"
resolution: "ahoy install verifies the cached release binary's SLSA build provenance through abcd update's in-process check (update.VerifyReleaseProvenance, wired by the CLI as InstallOptions.VerifyProvenance) before promoting it to the PATH copy, and refuses on any failure, a missing check included"
impact: fix
---

abcd ahoy install promotes an owned copy of the abcd binary (internal/core/ahoy/owned_copy.go) after checking it only against checksums.txt from the same release page, the weakness iss-2609231050273096 fixed for abcd update: whoever can change the release page can swap the binary and its checksum together. Found by the E5 fix lane on 2026-10-12.
