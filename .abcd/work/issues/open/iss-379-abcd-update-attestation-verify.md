---
schema_version: 1
id: "iss-379"
slug: "abcd-update-attestation-verify"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: in-process SLSA attestation verification in abcd update (sigstore-go dependency needs sign-off)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

In-process build-provenance (SLSA attestation) verification in abcd update. Trigger: the trust gap the README concedes is worth closing (e.g. after the public flip). itd-130 ships at the same-origin checksums bar (itd-105); the release workflow already attests binaries and checksums.txt, so abcd update could verify the attestation before the swap for a stronger root of trust than same-origin checksums — at the cost of a heavy dependency (sigstore-go). Twin of itd-108's deferred offline signing key (minisign): both close the 'the forge is the identity root' gap that an attestation alone does not.