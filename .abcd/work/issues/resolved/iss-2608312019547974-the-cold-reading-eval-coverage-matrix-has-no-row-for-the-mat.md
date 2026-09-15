---
schema_version: 1
id: "iss-2608312019547974"
slug: "the-cold-reading-eval-coverage-matrix-has-no-row-for-the-mat"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "the coverage matrix gains two rows — the basename-suffix match form and the per-item kind attestation — each with a falsifier watched red on a scratch copy. Behind them is a kind oracle transcribed by hand (materialClasses, one row per member of the closed vocabulary) checked over the manifest and the bundle, and a _test.go file planted in the baseline corpus, without which deleting the suffix row leaves every fixture manifest identical but for one item's class."
impact: internal
---

The cold-reading eval coverage matrix has no row for the MatchSuffix match form and no row asserting a manifest item's kind names its actual material class, so the per-item kind attestation added by itd-198 has no falsifier behind it

## Grounds

- pursued: the kind an item is attested as is now judged by something independent of the assembler, so a mis-stated material class fails; it would be shown wrong by a corpus that stops carrying one of the nine classes, which the eval reports rather than skipping.
