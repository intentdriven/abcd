---
schema_version: 1
id: "iss-2608301519255156"
slug: "the-gate-never-judges-schema-version-s-value-so-fifteen-spel"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-3-security"
found_at: "internal/core/lint/schema.go"
---

the gate never judges schema_version's value so fifteen spellings the ledger reader refuses are lint-green

Found by the round-3 security review. PRE-EXISTING and REPRODUCES ON MAIN —
the reviewer ran the identical probe there and found the same silences in a
superset of 30. Recorded, not chased.

`checkRecordRequiredFields` asks presence and non-blankness only (schema.go:733:
`if present && !r.valueEmpty(field, f) { continue }`); there is no typed-value
check anywhere in the store's required-field path. So `schema_version: 2`,
`"1"`, `"null"`, `[a]`, `{}`, `&a`, `!!null`, `# nothing`, a block scalar with
content, a trailing comment and padded quotes are all lint-green, while
`validateStrict` (capture/validate.go:32) returns `unsupported schema_version`
and the record is skipped — invisible to `capture list`, `capture status` and
every other ledger surface. `found_during: [a]` is the same shape.

This is a FOURTH instance of "the gate checks presence where the reader checks
type", and it belongs on one record with iss-2608300224316569 (`lapsed_at: []`)
rather than being chased on this branch. The A/B against 24860b61 is identical
on all 252 combinations, so round 3 neither caused nor worsened it.
