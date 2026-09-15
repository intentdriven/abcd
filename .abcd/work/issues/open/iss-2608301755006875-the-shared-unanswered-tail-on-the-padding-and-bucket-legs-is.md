---
schema_version: 1
id: "iss-2608301755006875"
slug: "the-shared-unanswered-tail-on-the-padding-and-bucket-legs-is"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-builder"
found_at: "internal/core/lint/schema.go"
---

the shared unanswered tail on the padding and bucket legs is not true of an item whose disposition is declined or held

Reported by the round-5 builder in its own words and deliberately NOT captured
by it, on the grounds that correcting it is a three-leg message-design change
beyond the six records it was given. The judgement about scope is right. Leaving
it unrecorded is not: a known defect with no record is the thing this ledger
exists to prevent, and the builder's report is not a place a future reader
looks.

Both the padding leg and the bucket leg end with the same tail, saying the named
item "goes on being reported as unanswered with no sign that an answer was
written". That is conditional in fact. A widening item carrying a `declined` or
a `held` disposition IS answered, so for such a target the tail asserts
something the walk has not established and that may simply be false.

It is the cycle's standing class once more, and this is its seventh surface: the
leg speaks about a report line whose content turns on dispositions the leg never
read.

Scope note, which is why this is its own record rather than a reopened one. The
identical wording ships in the round-4 padding leg, so a correct fix touches
three legs and settles one question for all of them: what a gate may assert
about a report it has not consulted. The round-5 position leg already avoids the
phrasing, which is the shape the other three should follow -- it says only that
the record counts for nothing and that no line reports an answer was written,
both of which the walk does establish.

