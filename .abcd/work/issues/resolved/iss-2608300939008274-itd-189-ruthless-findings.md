---
schema_version: 1
id: "iss-2608300939008274"
slug: "itd-189-ruthless-findings"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-189 adversarial ruthless review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, .abcd/work/issues/README.md, internal/core/lint/schema.go"
resolution: "The admission leg stands down when the admissions tree is unreadable, so an unread tree is named as unsafe rather than read as an absent one — the same unread-is-not-absent invariant the disposition branch already keeps; the issue ledger's README states all four sibling families and the schemas the step-2 pair declare; the held exclusion is pinned by a test; and proposal is now resolved on the same store-declared terms as occasioned_by (iss-2608300935215868)."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

itd-189 ruthless findings: a widening proposal whose admission sits behind an unreadable admissions tree is still reported outstanding with a remedy to write a disposition, contradicting the invariant the disposition side enforces (after the admitted check, break when the admissions tree is unreadable; add the mirror test); the issues README store contract still says the ledger root holds two sibling families while four roots are now declared; the held exclusion on the admission leg is untested; proposal is never resolved while occasioned_by is.
