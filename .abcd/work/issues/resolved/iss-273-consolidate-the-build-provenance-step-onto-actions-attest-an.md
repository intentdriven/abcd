---
schema_version: 1
id: "iss-273"
slug: "consolidate-the-build-provenance-step-onto-actions-attest-an"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "release.yml, its scaffold template and site.yml attest build provenance with actions/attest at the pin already in the tree, no predicate input (SLSA provenance, the wrapper's own path) and the wrapper's NODE_OPTIONS env. The wrapper is gone from the tree. Predicate equivalence confirmed from the pinned wrapper's action.yml and actions/attest's src/main.ts. TestBuildProvenanceUsesActionsAttestDirectly pins it."
impact: internal
resolved_by:
  commit: "a7fa2853207f7991f54a2750e304a98bf10d3083"
---

Consolidate the build-provenance step onto actions/attest and drop the wrapper. Upstream states that as of version 4, actions/attest-build-provenance is simply a wrapper on top of actions/attest, and that new implementations should use actions/attest instead. This repo ALREADY invokes actions/attest directly at .github/workflows/release.yml:248, in the same job that uses the wrapper at :296 -- so it currently runs attest 4.2.2 directly and, after the gh-316 bump, 4.2.1 transitively through the wrapper. Consolidating would remove a trust layer, eliminate that version skew, and halve the dependabot churn that keeps tripping iss-209 (every pinned-action bump edits the generated workflow only, breaking template parity and reddening CI until a human runs make scaffold-sync). actions/attest generates SLSA provenance natively in src/provenance.ts, so the migration looks feasible; the work is to map the wrapper's inputs onto the underlying action's and confirm the emitted attestation predicate is unchanged. Found by the security review of the actions/attest-build-provenance 4.1.1 to 4.2.2 bump, which approved that bump on its merits.

## Grounds

- pursued: the release and site attestations keep the SLSA provenance predicate with one action and one version; shown wrong by a published attestation whose predicateType is not https://slsa.dev/provenance/v1 or by gh attestation verify failing on the next release
