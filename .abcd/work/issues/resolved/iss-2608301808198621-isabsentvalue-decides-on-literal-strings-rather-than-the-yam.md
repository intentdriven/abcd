---
schema_version: 1
id: "iss-2608301808198621"
slug: "isabsentvalue-decides-on-literal-strings-rather-than-the-yam"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-delta-security"
found_at: "internal/core/lint/schema.go"
resolution: "isAbsentValue now consults internal/core/frontmatter.IsEmptyValue, which decides emptiness by the class of YAML node a value spells (null, empty flow collection, empty string) after reading off its tag, anchor and alias properties. The shared reader answers once for every field and every gate; no YAML dependency was added."
impact: fix
---

isAbsentValue decides on literal strings rather than the yaml null and empty class so chasing spellings cannot close it

Found by the itd-189 delta security review, and separated from
`iss-2608301808193750` on purpose: one is a false sentence to correct now, this
is the design question underneath it.

`isAbsentValue` compares against literal strings -- `v == "!!null"` and friends
-- and `isEmptyFlowCollection` anchors on the first and last byte. The reviewer's
sentence is the one to keep: **the new constant is a spelling test, not a null
test.** `!!null null` and `!<tag:yaml.org,2002:null>` are the SAME YAML node as
`!!null`, and the predicate accepts two of the three.

So the eleven spellings are not eleven bugs. They are one wrong altitude, and
adding an eleventh case would leave a twelfth.

**This is the same shape adr-56 just ruled on for the exclusion floor**, one
workstream over: a control recognising a construct by its spelling rather than
by what it IS, closing spellings one at a time across nine rounds while the next
unenumerated one stays open. The records there say it outright -- a spelling arms
race needs a design, not more patterns -- and the ruling was to change the
altitude rather than extend the list.

The complication here is that there is no YAML library in the module (the
reviewer established this while refuting a different finding), so "decide on the
tag and anchor class" is real work rather than a swapped call. That is why this
is captured rather than fixed in the round that found it.

Two options for whoever takes it, neither chosen here: teach the shared scalar
reader the tag, anchor and alias productions so emptiness is decided once for
every field; or narrow what the gate claims to what a string comparison can
honestly establish, and say so where the operator reads it. The first is
adr-56's branch-2 analogue applied to this surface; the second is its branch-3.

## Grounds

- pursued: we expect deciding on the node's class rather than on its spelling to close the spellings nobody enumerated at the same time as the ones that were, so no future round has to add an eleventh literal; a YAML null, empty collection or empty string that the shared reader still calls populated, or a populated value it calls empty, would show it wrong.
