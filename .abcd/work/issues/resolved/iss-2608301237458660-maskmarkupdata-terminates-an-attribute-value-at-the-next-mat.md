---
schema_version: 1
id: "iss-2608301237458660"
slug: "maskmarkupdata-terminates-an-attribute-value-at-the-next-mat"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
resolution: "An attribute value ends on the line it opens on, so an unbalanced quote can no longer blank the angle brackets of a heading its tag never reached."
impact: fix
resolved_by:
  intent: "itd-183"
---

maskMarkupData terminates an attribute value at the next matching quote anywhere in the document so one unclosed quote erases a raw HTML excluded heading from the scan

Found by the round-9 adversarial security review of build/itd-183.
REGRESSION INTRODUCED BY 2225d6cb -- the parent 044ac6ed refuses this input.

`maskMarkupData` ends an attribute value at the next matching quote found
anywhere in the joined document, not within the tag:

```
end := strings.IndexByte(s[q+1:], s[q])
if end < 0 { i++; continue }
maskAngles(out, q+1, q+1+end)
```

So one unclosed attribute quote blanks every `<` and `>` up to some unrelated
quote thousands of bytes later, erasing a raw-HTML excluded heading from the
scan before `rawHTMLHeading` ever sees it. `rawHeadingOpenRe` then finds no
opener, nothing refuses, no redaction span is recorded, and the heading plus
its whole section reaches the bundle under a manifest that asserts the heading
was refused.

Minimal reproduction, verified admitted on HEAD and refused on the parent:

```
---
id: x
---

<div id="

<h2>Audit Notes</h2>

private provenance

">
```

Same result for `<a href='x" >`, `<img alt="...`, `<a x='...`, and for an
opener sitting INSIDE a fenced code block -- the mask is fence-blind, so a
fenced `<a x="` masks structure outside the fence.

The code comment at project.go:625 asserts the opposite: "none, so an
unterminated comment or attribute value makes the scan see MORE". That is
untrue. The value is unterminated only relative to its tag; IndexByte pairs it
with a quote far away. This is the branch's recurring failure direction -- a
mask that answers a false positive by seeing less.

Remedy (class-closing): bound the attribute search to its own tag. Search for
the closing quote only within s[q+1:tagEnd], where tagEnd is the first `>` at
or after q IN THE RAW STRING; if no closing quote occurs before it, mask
nothing. That makes "unterminated" mean what the comment already claims, and
makes the mask incapable of crossing a line the tag never reached.
