---
schema_version: 1
id: "iss-2608300935218982"
slug: "quoted-empty-required-field-passes-the-gate"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-189 adversarial security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (checkRecordRequiredFields, isAbsentValue)"
resolution: "isAbsentValue now decides absence on the value the YAML scalar carries — quotes stripped with the rule's own issueScalar and the result re-trimmed — so an empty, whitespace-only or quoted-whitespace required field is refused in every store that shares the check, the pre-existing issue ledger included. That is the whole of what this change closed: the empty flow mapping and the explicit null tag are two further spellings of the same blank, and they are iss-2608301649337965's to close. A block-scalar header is judged by what its block holds rather than by the header byte."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

record_schema's required-field check tests absence on the raw value with quotes intact, so a quoted-empty grounds or occasioned_by ("", '', or "  ") passes as present while the writer-side model trims and refuses it; a groundless admission is lint-green and the report leg then counts its proposal as admitted, silencing the very report built to catch it. The iss store has the identical pre-existing gap (found_during: "" passes). Strip surrounding quotes before the absence test, using the rule's own issueScalar, and add the quoted-empty spellings to the tests.
