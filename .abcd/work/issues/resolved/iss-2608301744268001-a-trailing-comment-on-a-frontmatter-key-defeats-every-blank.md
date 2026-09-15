---
schema_version: 1
id: "iss-2608301744268001"
slug: "a-trailing-comment-on-a-frontmatter-key-defeats-every-blank"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-189-round-5-build"
found_at: "internal/core/frontmatter/frontmatter.go (Fields)"
resolution: "frontmatter.Fields strips a trailing comment before it hands a value on: a comment starts at a hash preceded by whitespace and outside single or double quotes, so a hash inside a quoted value or with no whitespace before it stays part of the value. The strip is in the ONE scanner, so every gate and every store inherited it."
impact: fix
---

a trailing comment on a frontmatter key defeats every blank spelling the record_schema gate refuses because the shared same-line scanner strips no comments

Found by the round-5 ruthless review of the fix for iss-2608301649337965, which
closed three of the four spellings that record enumerates and left the fourth.

`frontmatter.Fields` is a same-line scanner that strips no comments, so the
value it hands every gate is the raw remainder of the line. `isAbsentValue`
anchors each of its emptiness tests on the LAST byte — `isNull` compares the
whole string, and `isEmptyFlowCollection` requires the closing bracket at the
end — so a trailing comment defeats all of them at once:

```
grounds: {}  # todo
grounds: []  # todo
grounds: ~ # todo
grounds: "" # todo
```

Probed end to end through `Lint`: each draws ZERO `record_schema` findings on an
admission that is otherwise well formed. `admittedProposals` never reads
`grounds`, so it keys `(rdg-1, rdi-2)` as admitted and `rdi-2` drops out of
`report.Undispositioned` — the exact mechanism iss-2608301649337965 describes,
reached by a fourth spelling.

It is NOT scoped to `grounds`, and that is what makes it a scanner question
rather than a predicate one: `severity: minor # todo` reads as the value
`minor # todo` to the enum leg, and every same-line scalar in every store is
read the same way.

Remedy is a comment strip in the ONE scanner, and it is not a one-liner: YAML
starts a comment only at a `#` preceded by whitespace and outside quotes, so a
naive split truncates a legitimate value containing a hash. It belongs where the
scanner lives, not in `isAbsentValue`, or the same escape stays open for every
gate that reads a value some other way.

## Grounds

- pursued: we expect a comment stripped in the shared scanner to close the escape for every gate at once, rather than for the one predicate the defect was reported against; a value containing a legitimate hash that the scanner truncates, or a commented blank that still reaches a gate as a value, would show it wrong.
