---
schema_version: 1
id: "iss-379"
slug: "abcd-update-attestation-verify"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed C): Sign off sigstore-go for in-process attestation verification in abcd update (the E5 attestation route)?"
remedy: "If the person signs off sigstore-go: add it, fetch the release's attestation bundle for each downloaded binary, verify it in process against the embedded Sigstore trusted root with the signer identity pinned to this repository's release.yml, and refuse the swap on any failure; prove it with a recorded genuine bundle that verifies and a tampered one that is refused. If declined: close this as wontfix in favour of the route iss-2609231050273096 is given."
---

In-process build-provenance (SLSA attestation) verification in abcd update. Trigger: the trust gap the README concedes is worth closing (e.g. after the public flip). itd-130 ships at the same-origin checksums bar (itd-105); the release workflow already attests binaries and checksums.txt, so abcd update could verify the attestation before the swap for a stronger root of trust than same-origin checksums — at the cost of a heavy dependency (sigstore-go). Twin of itd-108's deferred offline signing key (minisign): both close the 'the forge is the identity root' gap that an attestation alone does not.

## Remedy grounds (2026-09-29)

- sigstore-go states it is stable, production-ready and conformance-tested, and verifies bundles, Rekor inclusion and timestamps: https://github.com/sigstore/sigstore-go (consulted 2026-09-29); the release already attests the binaries (release.yml, actions/attest), so the verifier is the only missing half.
- The dependency is the sign-off owed (rulings-owed C); rejected: a hand-rolled bundle verifier, which re-implements exactly what the dependency exists to get right.
