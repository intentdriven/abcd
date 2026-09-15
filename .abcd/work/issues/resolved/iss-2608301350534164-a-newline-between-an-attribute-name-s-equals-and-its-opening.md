---
schema_version: 1
id: "iss-2608301350534164"
slug: "a-newline-between-an-attribute-name-s-equals-and-its-opening"
severity: "major"
category: "security"
source: "user-observation"
found_during: "itd-183-round-10-security"
found_at: "internal/core/reading/project.go"
resolution: "maskMarkupData gains a second return naming the shape when the blank skip after an equals sign reaches a newline before the opening quote, and verifyRedaction refuses it as 'an attribute value that opens on the line after its equals sign'. The HTML-whitespace skip the record proposed is deliberately not taken: a resolved mask on that shape is comprehension, which the 2026-08-30 ruling declined."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

a newline between an attribute name's equals and its opening quote declines the markup mask on both readings so a value carrying a greater-than truncates the heading opener

Found by the round-10 adversarial security review of build/itd-183. NOT
introduced by this branch -- present on the round-9 parent too, and recorded
rather than chased under the ruling that round 10 closes only what the branch
introduced.

`maskMarkupData` finds an attribute value by skipping blanks after the `=`, and
a blank is space, tab, carriage return or form feed -- never a newline, because
the same helper reads YAML lines where a newline ends the line. So a tag written
across the break is not recognised as carrying a value at all, on EITHER
masking, and the unmasked opener again stops at the `>` inside the value:

```
<h2 title=
"a>b">Audit Notes</h2>
```

Every renderer reads that as the excluded heading. Verified leaking on this
round and on the parent.

This is the same root cause as iss-2608301350527962 -- the opener parse depends
on the mask, and a declined mask leaves it mis-parsed -- reached by a different
route. Remedy: an HTML-whitespace skip local to the markup mask, which a YAML
line scan has no business sharing. Seed material for the exclusion floor's own
intent.

## Grounds

- pursued: we expect refusing the shape to close it where widening the skip class would have resolved it, because a mask that declines silently is the failure and a mask that guesses is the comprehension this intent does not buy; an attribute value opening on the next line that still assembles, or a heading admitted past a mask that declined, would show it wrong
