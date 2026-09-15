---
schema_version: 1
id: "iss-2608311238236490"
slug: "the-cold-reading-assembler-s-exclusion-floor-carries-no-entr"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Exclusions gains an entry for the local ledger tier .abcd/.work.local, citing brief invariant 14, so every manifest asserts the tier was refused and assertExclusions enforces it; AssemblerVersionCore goes 1.2.0 to 1.3.0 because an added floor entry is an added promise rather than a rewording, and the charter's rendered table carries the row. TestManifestNamesEveryExcludedFamily holds the oracle's family list against the floor the manifest asserts, at every assembling position."
impact: additive
---

The cold-reading assembler's exclusion floor carries no entry for the local ledger tier. internal/core/reading/include.go's Exclusions names .abcd/development/decisions, .abcd/work/issues, .abcd/work/DECISIONS.md and the readings family individually, but nothing for .abcd/.work.local — so every manifest the assembler writes is silent about the tier brief invariant 14 exists to keep out, and assertExclusions has nothing to enforce there. The tier IS excluded, by absence from the positive walk and by the .abcd deny segment, so this is a disclosure gap rather than a leak: a reader checking the manifest's asserted exclusions cannot tell that framing traces and declined construals were refused, and invariant 16 says an attestation never states less than the examination behind it establishes. Found while transcribing the exclusion table for itd-186's read-block eval, whose own family list does name the tier; the eval's local-tier falsifiers correctly need no Exclusions row removed, which is how the asymmetry showed up.

## Grounds

- pursued: a reader checking the manifest can see that framing traces and declined construals were refused, which invariant 16 requires an attestation not to understate; it would be shown wrong by a refused family that the manifest still does not name.
