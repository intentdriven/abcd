---
schema_version: 1
id: "iss-2608301646042379"
slug: "the-control-character-refusal-claims-no-record-field-can-hol"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-security"
found_at: "internal/core/grounds/grounds.go"
resolution: "The refusal and both doc claims now name the class the check refuses, below U+0020, which is what yamlScalar will not write."
impact: fix
resolved_by:
  intent: "itd-179"
---

the control character refusal claims no record field can hold the rune when DEL C1 and bidi all round trip through committed scalars

Found by the round-5 security review. Branch-introduced: the whole `grounds`
package is, existing on neither `main` nor `experiment/cold-reading`.

The CHECK is `r < 0x20`, and it is exactly congruent with `yamlScalar`'s guard.
The comment directly above it says so, correctly. The MESSAGE generalises to a
rule the store does not have:

```
grounds text carries the control character U+%04X, which no record field can hold
```

The reviewer wrote U+007F, U+009B, U+2028, U+200B and U+202E into committed
`grounds:` and `wontfix_reason:` scalars through `capture resolve` and
`capture wontfix`, and read every one back through `capture list` unchanged. So
a caller who reads the refusal learns a rule about record fields that does not
exist, and the next reviewer reads the same sentence as a guarantee. The
package doc repeats the claim.

Remedy is the narrowing, not the widening. Widening the check to DEL and C1
would put this floor out of step with `yamlScalar`, which is the gate/reader
split this repository hunts as its own defect class; the honest fix is to say
what the check does, which is to refuse the class the frontmatter serialiser
refuses.

Class: a message asserting a mechanism that is not the case. Sixth instance
this cycle, and the second on this branch.

## Grounds

- pursued: We expect narrowing the wording rather than widening the check to be the right close, because the check deliberately mirrors the frontmatter serialiser and widening it would put the floor out of step with the reader it exists to match.
