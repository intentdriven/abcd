---
schema_version: 1
id: "iss-2608301656192369"
slug: "the-join-and-bucket-legs-strip-the-scalar-twice-so-a-doubly"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "checkRecordJoins and checkRecordBucketField now put the RAW frontmatter value to isAbsentValue rather than one issueScalar has already read, so all three legs decide emptiness on one strip. A value that is two apostrophes inside double quotes is present to every leg again: the join judges its spelling and the bucket field judges its disagreement with the directory, instead of all three standing down on an admission that admits nothing."
impact: fix
resolved_by:
  intent: "itd-189"
---

the join and bucket legs strip the scalar twice so a doubly quoted empty value is present to one leg and absent to the other and no leg speaks

Found by the round-5 ruthless review. Branch-introduced: both functions are new.

`checkRecordJoins` and `checkRecordBucketField` compute `value :=
issueScalar(f.value)` and then hand that ALREADY-STRIPPED value to
`isAbsentValue`, which strips again. `checkRecordRequiredFields` strips once. So
a doubly-quoted-empty value is PRESENT to the required leg and ABSENT to both
new legs, and no leg speaks:

```
proposal: "''"      # or   run: "''"
```

Probed: zero `record_schema` findings, and the report still reports the
proposal outstanding. The admission counts for nothing, permanently and
silently, which is the shape `checkRecordBucketField`'s own godoc calls the one
this whole rule exists to make loud.

It also contradicts the branch's own record of what a value is: `issueScalar`'s
godoc says a value that is itself two apostrophes inside double quotes IS a
value, and `TestARequiredValueMadeOfQuoteCharactersIsPresent` pins exactly that.
The two new legs disagree with the test standing next to them.

Distinct from iss-2608301649337965, which is the opposite direction in a
different function: that one reads `{}` as NOT absent, this one reads a present
value as absent.

Remedy: pass the raw `f.value` to `isAbsentValue` in both legs, so all three
legs decide emptiness on one strip. The reviewer verified the two-site change
closes both spellings and leaves the whole lint and capture suite green, which
also shows nothing currently pins either behaviour.
