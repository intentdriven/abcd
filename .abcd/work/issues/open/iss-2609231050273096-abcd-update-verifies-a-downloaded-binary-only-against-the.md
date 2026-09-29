---
schema_version: 1
id: "iss-2609231050273096"
slug: "abcd-update-verifies-a-downloaded-binary-only-against-the"
severity: "major"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
deferred_after: v0.11.1
deferral_reason: "ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling E5 (2026-09-23) asks for a check independent of the release page and leaves the mechanism open. The release already publishes SLSA build-provenance attestations over the binaries and checksums.txt (release.yml, actions/attest), so the candidates are verifying that attestation (a sigstore verification dependency needing sign-off, or a runtime dependency on gh attestation verify) or a digest committed in the repository's history (no new Go dependency, but the release chain must first commit per-binary digests; the catalog pins only the plugin archive, adr-2609231048308186). Owed: that choice, then a lane of its own."
---

abcd update verifies a downloaded binary only against the checksums.txt published in the same GitHub release (internal/core/update/update.go), so it catches corruption in transit but not a release page whose binary and checksums.txt were both replaced; the binary needs an independent check next cycle, such as a digest committed in the repository's history the way the plugin catalog now pins the plugin archive (adr-2609231048308186), or verification of the release's signed build-provenance attestation (a new verification dependency needs sign-off first). Ruled by the product thinker on 2026-09-23 (E5, 10:27Z): harden next cycle; this release keeps the same-page check.

## Deferral 2026-09-29

Deferred past v0.11.1: ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling E5 (2026-09-23) asks for a check independent of the release page and leaves the mechanism open. The release already publishes SLSA build-provenance attestations over the binaries and checksums.txt (release.yml, actions/attest), so the candidates are verifying that attestation (a sigstore verification dependency needing sign-off, or a runtime dependency on gh attestation verify) or a digest committed in the repository's history (no new Go dependency, but the release chain must first commit per-binary digests; the catalog pins only the plugin archive, adr-2609231048308186). Owed: that choice, then a lane of its own.
