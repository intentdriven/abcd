---
schema_version: 1
id: "iss-2608301327012166"
slug: "checkrecordjoins-does-not-consult-the-retired-set-so-one-rul"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-2-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "checkRecordJoins takes the retired set checkRecordSchema already builds and resolves a pruned handle through the successor's declaration, exactly as the cross-reference loop does, so one rule gives one answer about one pruned handle."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

checkRecordJoins does not consult the retired set so one rule gives two answers about one pruned handle

Found by the round-2 adversarial ruthless review of build/itd-189.
INTRODUCED BY THIS BRANCH.

`checkRecordJoins` (schema.go:755) does not consult the `retired` set that
`checkRecordSchema` builds for cross-references, so one rule gives two answers
about one pruned handle.

Probe-verified: with `adr-5` pruned (per decisions/adrs/README.md),
`related_adrs: [adr-5]` on one record is accepted, while
`surprises/srp-4.md` with `occasioned_by: adr-5` yields a blocker --
"occasioned_by names 'adr-5', which is not a record in the corpus".

A nit rather than a fix because reachability is low: the realistic join
vocabulary is rdi / adm / iss, not adr.

Remedy: pass `retired` into `checkRecordJoins` and skip a handle it holds,
exactly as the cross-reference loop at schema.go:458 already does.
