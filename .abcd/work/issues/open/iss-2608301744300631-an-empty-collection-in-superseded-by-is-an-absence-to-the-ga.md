---
schema_version: 1
id: "iss-2608301744300631"
slug: "an-empty-collection-in-superseded-by-is-an-absence-to-the-ga"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-189-round-5-build"
found_at: "internal/core/record/record.go (describeADR)"
---

an empty collection in superseded_by is an absence to the gate and a rendered link to record Describe so the two readers disagree about whether the record names a successor

Found by the round-5 ruthless review, on a call site the isAbsentValue widening
in iss-2608301649337965 reached without being audited.

`checkRecordSchema`'s supersession leg reports a `superseded_by` that parses to
no handle, and it tests absence with `isAbsentValue` — deliberately, because
`superseded_by: []` is this record's house spelling for an empty list. So both
empty flow collections are an absence to the gate and draw no finding.

`record.describeADR` does not share that predicate. It gates the link on
`sup != "" && !frontmatter.IsNull(sup)`, and neither `[]` nor `{}` is in the
YAML null set, so it renders `Links["superseded_by"] = "[]"` — a successor link
whose target is a bracket pair. One record, two readings: the gate says the ADR
names no successor and the dispatcher shows one.

The `[]` half is pre-existing; the `{}` half arrived with the widening, which
made the two spellings agree with each other rather than with the dispatcher.
Both are the same defect and neither is separately fixable, because the
disagreement is between the two predicates and not between the two spellings.

Remedy: give the dispatcher the same emptiness question the gate asks, so one
value gets one answer — not a second special case in `isAbsentValue`, which
would leave the dispatcher rendering `[]` as a link.
