---
schema_version: 1
id: "iss-2608311302045104"
slug: "the-coverage-matrix-s-count-pin-catches-a-row-deleted-but-no"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "third delta review of itd-186"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_oracle_test.go"
resolution: "TestEveryAssemblerRuleHasAFalsifier refuses two coverage rows carrying the same Rule text, so a row substituted for a duplicate of another is red rather than silently keeping the count. The other half — a row rewritten to a rule the assembler does not have — stays the declared limit the matrix header states."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

The coverage matrix's count pin catches a row deleted but not a row substituted. Replacing the structural deny row for the abcd namespace with a duplicate of the glossary row keeps the row count at 46 and the gap count at 6, orphans no sentinel class, and the lane stays green: the single most load-bearing rule in the assembler's contract leaves the matrix silently. The matrix discloses that it cannot check that it names every rule the assembler has, so this is the declared limit rather than a broken promise, but the by-duplication half of that limit closes with a three-line check that the Rule strings are distinct.

## Grounds

- pursued: the by-duplication half of the substitution hole is three lines and is closed here, watched red by swapping the structural-deny row for a second copy of the glossary row. The by-rewriting half is left open deliberately and disclosed in the matrix header: catching it needs a human reading the include table against this list. This is wrong if someone closes that half by deriving the matrix from the assembler, which would make it confirm the table it exists to test.
