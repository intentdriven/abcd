---
schema_version: 1
id: "iss-2609231050273096"
slug: "abcd-update-verifies-a-downloaded-binary"
severity: "major"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
deferred_after: v0.11.1
deferral_reason: "ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling E5 (2026-09-23) asks for a check independent of the release page and leaves the mechanism open. The release already publishes SLSA build-provenance attestations over the binaries and checksums.txt (release.yml, actions/attest), so the candidates are verifying that attestation (a sigstore verification dependency needing sign-off, or a runtime dependency on gh attestation verify) or a digest committed in the repository's history (no new Go dependency, but the release chain must first commit per-binary digests; the catalog pins only the plugin archive, adr-2609231048308186). Owed: that choice, then a lane of its own."
remedy: "Waits on the route ruling owed on E5 (the attestation route or the committed-digest route): give abcd update (internal/core/update/update.go) a check independent of the release page: if the attestation route, verify the release's SLSA provenance before the swap, pinned to this repository and to release.yml as signer workflow, either in process (sigstore-go, needs the person's sign-off, iss-379) or by shelling out to gh attestation verify (a runtime dependency that refuses when gh is absent); if the committed-digest route, have the release chain commit per-binary digests to the repository and compare against the digest read from the tagged tree. Prove it with a test in which a replaced binary and a matching replaced checksums.txt are refused before the swap."
---

abcd update verifies a downloaded binary only against the checksums.txt published in the same GitHub release (internal/core/update/update.go), so it catches corruption in transit but not a release page whose binary and checksums.txt were both replaced; the binary needs an independent check next cycle, such as a digest committed in the repository's history the way the plugin catalog now pins the plugin archive (adr-2609231048308186), or verification of the release's signed build-provenance attestation (a new verification dependency needs sign-off first). Ruled by the product thinker on 2026-09-23 (E5, 10:27Z): harden next cycle; this release keeps the same-page check.

## Deferral 2026-09-29

Deferred past v0.11.1: ruling still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling E5 (2026-09-23) asks for a check independent of the release page and leaves the mechanism open. The release already publishes SLSA build-provenance attestations over the binaries and checksums.txt (release.yml, actions/attest), so the candidates are verifying that attestation (a sigstore verification dependency needing sign-off, or a runtime dependency on gh attestation verify) or a digest committed in the repository's history (no new Go dependency, but the release chain must first commit per-binary digests; the catalog pins only the plugin archive, adr-2609231048308186). Owed: that choice, then a lane of its own.

## Remedy grounds (2026-09-29)

- The threat is a release page whose binary and checksums.txt are replaced together; both candidate routes break that because neither reads its trust from the same page. gh attestation verify checks a signed attestation against --repo and --signer-workflow and works offline with --bundle: https://cli.github.com/manual/gh_attestation_verify (consulted 2026-09-29); sigstore-go describes itself as stable and passing the sigstore-conformance suite: https://github.com/sigstore/sigstore-go (consulted 2026-09-29).
- No route is picked: the choice and the dependency sign-off are owed rulings (rulings-owed C). Rejected: an offline signing key (minisign, itd-108), which adds key custody rather than removing the same-origin trust.
