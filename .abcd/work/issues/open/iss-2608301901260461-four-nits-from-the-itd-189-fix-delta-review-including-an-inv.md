---
schema_version: 1
id: "iss-2608301901260461"
slug: "four-nits-from-the-itd-189-fix-delta-review-including-an-inv"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-fix-delta-ruthless"
found_at: "internal/core/lint/schema_test.go"
---

four nits from the itd-189 fix delta review including an inverted test failure message and an over broad negative loop

Four nits from the itd-189 fix-delta ruthless review, settled at the ship commit.

1. `schema_test.go` -- a failure message is inverted and its verb is useless: it
   fires precisely when the gate does NOT read the value as absent, and prints a
   constant `false` inside that branch.
2. `schema_test.go` -- the negative loop rejects the substrings "skipped" and
   "refuses" across EVERY `record_schema` finding on those two paths rather than
   the duplicate-key one, so an unrelated future message on an admission or a
   disposition trips it with a misleading reason.
3. `commands/capture.md` -- the spellings list names "an alias" as carrying
   nothing. An alias to a defined anchor carries the anchor's value; it passes
   the gate regardless, so only the "carry nothing" half is the overclaim.
   Adjacent to but distinct from iss-2608301808197261 item 4.
4. `schema_test.go` -- a comment calls its list "the whole of what the
   enumeration claims", but `commands/capture.md` also lists a block scalar
   holding nothing, which the pin omits because it is `valueEmpty`'s business
   rather than `isAbsentValue`'s. The two lists the test claims to bind are not
   the same set.

5. Added from the fix-delta SECURITY review, which offered it as a nit rather
   than a finding: `commands/capture.md` does not name `Null`/`NULL` or U+00A0
   among the refused spellings, although both ARE refused. It errs in the safe
   direction -- an author using them gets a clear refusal rather than a silent
   pass -- and the doc's category framing ("a YAML null", "whitespace")
   arguably covers them. Folded here rather than given an id.

