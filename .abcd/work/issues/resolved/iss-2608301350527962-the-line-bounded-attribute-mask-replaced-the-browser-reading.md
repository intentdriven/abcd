---
schema_version: 1
id: "iss-2608301350527962"
slug: "the-line-bounded-attribute-mask-replaced-the-browser-reading"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "itd-183-round-10-security"
found_at: "internal/core/reading/project.go"
resolution: "Both maskings are taken as readings alongside the unmasked text, so a mask adds a reading and never replaces one. Each is load-bearing against a document the other misreads."
impact: fix
resolved_by:
  intent: "itd-183"
---

the line-bounded attribute mask replaced the browser reading instead of adding to it so an attribute value carrying a greater-than and a line break hides the heading opener

Found by the round-10 adversarial security review of build/itd-183, and fixed in
the same round. REGRESSION INTRODUCED BY THIS ROUND'S OWN FIRST COMMIT.

Closing the runaway attribute mask by bounding its value to its own line
REPLACED the reading a browser's tokenizer takes -- the closing quote is the
next matching quote ahead -- rather than adding to it. The two readings disagree
about exactly one shape, and the round deleted the reading that resolves it:

```
<h2 title="a>
b">Audit Notes</h2>
```

Every renderer reads that as one `h2` whose title attribute is `a>\nb` and whose
text is `Audit Notes`. The line-bounded mask declines it, because the value does
not close on its own line; and the unmasked reading cannot parse the opener at
all, because `rawHeadingOpenRe`'s `[^>]*` stops at the `>` INSIDE the value, so
the title read is `b">Audit Notes` and slugs to something else. The round-9
mask, unbounded, blanked that `>` and refused.

Verified leaking for the single-quoted form, for a `role="heading"` element, for
a harmless attribute ahead of the spanning one, and for a trailing inline
element carrying the break.

Remedy, and the doctrine the round already states: a mask may ADD a reading and
may never take one away. Both maskings are taken alongside the unmasked text and
a heading is refused if any reading names an excluded one. The conservative
reading is load-bearing in the other direction, so neither may be dropped: a
stray quote earlier in the file pairs, under the browser reading, with the
opening quote of a LATER legitimate value, and that value is then never masked.

LESSON, and it is the branch's own: substituting one reading for another is the
move that has opened a leak in five consecutive rounds. Adding is safe;
replacing is not.
