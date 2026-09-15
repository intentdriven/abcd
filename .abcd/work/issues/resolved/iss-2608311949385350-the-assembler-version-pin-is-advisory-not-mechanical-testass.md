---
schema_version: 1
id: "iss-2608311949385350"
slug: "the-assembler-version-pin-is-advisory-not-mechanical-testass"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The stamped assembler version is now computed from the rendered include table's digest, so a table change moves it whether or not anyone remembers to. The literal-digest comparison the old gate used is replaced by a structural property and a mutation proof."
impact: fix
resolved_by:
  intent: "itd-198"
  spec: "spc-68"
---

The assembler version pin is advisory, not mechanical: TestAssemblerVersionCoversTheIncludeTable compares sha256 of the rendered include table to a standalone literal digest and never reads AssemblerVersion, so changing the table and updating only the literal is green with the version unmoved

## Grounds

- pursued: an attestation that can name a version not describing its own table is the shape brief invariant 16 forbids, and a gate asking a human to move a constant is not a gate. What would show this wrong is a table change that moves the digest without changing what any reading receives, making the version noisy rather than informative.
