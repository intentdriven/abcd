---
schema_version: 1
id: "iss-2608301327013320"
slug: "an-admission-whose-proposal-names-an-item-in-a-different-run"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-2-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "checkRecordJoins gains a sameBucket obligation, declared on the admission store's proposal join: a target the corpus holds in another bucket is now a finding naming both buckets. The comment in readingoutstanding.go that claimed the run-field agreement enforced it is corrected to state what the walk actually establishes."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

an admission whose proposal names an item in a different run passes the gate silently and admits nothing

Found by the round-2 adversarial ruthless review of build/itd-189.
INTRODUCED BY THIS BRANCH (the `adm` store is new); does not reproduce on main.

`admissions/rdg-1/adm-3.md` carrying `run: rdg-1`, `proposal: rdi-7`, where
`rdi-7` lives in `readings/rdg-9/`, passes the gate with ZERO record_schema
findings (probe-verified). `admittedProposals` keys `{rdg-1, rdi-7}`, which is
never queried, because `rdi-7` is walked under `rdg-9`. If `rdi-7` carries an
accepted disposition the report then prints that it "is a widening proposal
with neither an admission nor a decline" while an admission record naming it
sits in the tree. The admission counts for nothing, permanently and silently.

That is precisely the class `checkRecordBucketField`'s own comment says the
rule exists to make loud: "a record that quietly stops counting".

The code asserts the invariant it does not establish. readingoutstanding.go:
73-75 says "An admission for an item can only live under that item's own run
(the run-field agreement below enforces exactly that)". The run-field agreement
(readingoutstanding.go:477, schema.go:785) enforces FIELD == DIRECTORY. It never
enforces DIRECTORY == THE ITEM'S OWN RUN. Those are different claims, and the
comment names the stronger one.

The reviewer tried to refute it -- could a run legitimately admit another run's
proposal? The schema says no: an admission is "meaningful only against the run
whose proposals it admits".

Remedy (class-level, two lines of comparison, no new walk): `checkRecordJoins`
already resolves the target through `index`, and `schemaRecord` carries
`bucket`. Add a field to `recordJoin` (e.g. `sameBucket bool`), set it on the
adm store's `proposal` join, and report when `index[ref].bucket != r.bucket`.
