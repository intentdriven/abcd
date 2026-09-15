---
schema_version: 1
id: "iss-2608301327012407"
slug: "the-new-unadmitted-clause-in-empty-is-the-one-surviving-muta"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-round-2-ruthless"
found_at: "internal/core/lint/readingoutstanding.go"
resolution: "Empty gains a test that enumerates the report's fault lists from the struct itself and asserts each one alone makes the report non-empty, so every clause discriminates and a field added to the report but forgotten in Empty is red. Deleting any clause now turns its case red, the Unadmitted one included."
impact: internal
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

the new Unadmitted clause in Empty is the one surviving mutation because Empty has no production caller

Found by the round-2 adversarial ruthless review of build/itd-189, and it is
the ONE survivor out of twenty-two mutations the reviewer ran against this
branch's new guards. Twenty-one died.

Deleting `len(r.Unadmitted) == 0` from `Empty()` (readingoutstanding.go:195)
leaves both `internal/core/lint` and `internal/surface/cli` green.

No failure today: `Empty()` has NO PRODUCTION CALLER -- only
`reading_outstanding_test.go:313` -- so the clause is correct-but-unreachable
rather than wrong. It becomes a real silence the day a surface gates rendering
on `Empty()`, which is exactly the kind of latent hole that is cheap now and
expensive later.

The pre-existing dead method is not this branch's doing; the untested new
clause is.
