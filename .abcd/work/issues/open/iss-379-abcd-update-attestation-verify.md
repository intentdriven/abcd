---
schema_version: 1
id: "iss-379"
slug: "abcd-update-attestation-verify"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Sign off sigstore-go for in-process attestation verification in abcd update (the E5 attestation route)?"
---

In-process build-provenance (SLSA attestation) verification in abcd update. Trigger: the trust gap the README concedes is worth closing (e.g. after the public flip). itd-130 ships at the same-origin checksums bar (itd-105); the release workflow already attests binaries and checksums.txt, so abcd update could verify the attestation before the swap for a stronger root of trust than same-origin checksums — at the cost of a heavy dependency (sigstore-go). Twin of itd-108's deferred offline signing key (minisign): both close the 'the forge is the identity root' gap that an attestation alone does not.