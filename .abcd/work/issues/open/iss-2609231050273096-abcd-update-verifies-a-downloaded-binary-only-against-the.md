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
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): For abcd update's independent check, a digest committed in the repo's history (no new dependency) or verification of the signed build attestation (sigstore-go sign-off, iss-379)?"
---

abcd update verifies a downloaded binary only against the checksums.txt published in the same GitHub release (internal/core/update/update.go), so it catches corruption in transit but not a release page whose binary and checksums.txt were both replaced; the binary needs an independent check next cycle, such as a digest committed in the repository's history the way the plugin catalog now pins the plugin archive (adr-2609231048308186), or verification of the release's signed build-provenance attestation (a new verification dependency needs sign-off first). Ruled by the product thinker on 2026-09-23 (E5, 10:27Z): harden next cycle; this release keeps the same-page check.
