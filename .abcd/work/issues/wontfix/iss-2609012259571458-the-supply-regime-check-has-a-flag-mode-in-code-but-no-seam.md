---
schema_version: 1
id: "iss-2609012259571458"
slug: "the-supply-regime-check-has-a-flag-mode-in-code-but-no-seam"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "iteration-2 conformance audit against the design framework v4 and the readings companion v4, 2026-09-01"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/reading/ingest_regime.go"
wontfix_reason: "Ruled 2026-09-02: the supply-regime check is the framework's section 8.9 as written, the reserved-name refusals enforced and the prose detectors flagging until their withdrawal lands; the mode of a signature is code-pinned by a test and moves by a commit, never by a setting, so no seam is built."
---

The supply-regime check has a flag mode in code but no seam that can select it, so the design's degradation path cannot be taken

The governing framework (v4, section 8.9) names a degradation path for the
supply-regime check: where a signature proves noisy in practice, the check
degrades to recording the regime and flagging a suspected violation for
researcher review rather than refusing, and the change is recorded so the
property claimed weakens from enforced to observed. The itd-185 fidelity audit
established that the signatures are noisy: fourteen of thirty-four realistic
outputs were refused, all by the disposition detector matching a bare token
followed by a colon or equals sign, which this repository's records carry
everywhere (iss-2608311518056854).

The code carries the mode. `SignatureFlag` exists beside `SignatureEnforce` in
`internal/core/reading/ingest_regime.go`, and a flagged signature produces a
review flag rather than a refusal. What does not exist is any way to select it:
the mode on each shipped signature is a Go literal, every one is `enforce`, and
no configuration, flag or record can move one to `flag`. The `generative` regime
already runs flag-only by construction, so the flag path itself is exercised.

The consequence is that the decision the framework anticipates cannot be taken
without a code change and a release, and the first ingest of a real reading
will refuse legitimate output at the measured rate. Either a committed,
reviewed seam selects the mode per signature (the reading presets file is the
nearest precedent for a committed configuration inside the dirty gate), or the
mode is ruled fixed at `enforce` and the framework's degradation clause is
declared not taken. Whichever is chosen, the record must say which, because the
framework's own words are that the property claimed weakens when this path is
taken, and a weakening that no record states is the shape brief invariant 16
forbids.

Related: iss-2608311517547712 (the provenance field is a signature-free
channel) and iss-2608311518109233 (the reserved-name tables are
mutation-vacuous), both of which bear on the same gate.

## Grounds

- declined: Ruled 2026-09-02: the supply-regime check is the framework's section 8.9 as written, the reserved-name refusals enforced and the prose detectors flagging until their withdrawal lands; the mode of a signature is code-pinned by a test and moves by a commit, never by a setting, so no seam is built.
