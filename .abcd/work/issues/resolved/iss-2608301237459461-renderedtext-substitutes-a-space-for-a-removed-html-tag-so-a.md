---
schema_version: 1
id: "iss-2608301237459461"
slug: "renderedtext-substitutes-a-space-for-a-removed-html-tag-so-a"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
resolution: "Heading titles are compared under both readings of a removed tag: a tag standing for a boundary and a tag standing for none. All four heading paths refuse either way."
impact: fix
resolved_by:
  intent: "itd-183"
---

renderedText substitutes a space for a removed HTML tag so a tag inside a word splits it and the excluded heading escapes on all four heading paths

Found by the round-9 adversarial security review of build/itd-183.
REGRESSION INTRODUCED BY 2225d6cb. The cheapest of the round's regressions to
trigger, and it defeats all four heading paths at once.

`renderedText` now does `out = htmlTagRe.ReplaceAllString(out, " ")`. A tag
INSIDE a word therefore splits the word, and the slug stops matching:

```
renderedText("Audi<i>t</i> Notes")
  HEAD:   "Audi t  Notes"  -> slug audi-t-notes
  parent: "Audit Notes"    -> slug audit-notes
```

So `## Audit Note<sup>s</sup>` in any admitted file passes the section loop's
exact-title lookup, then passes `namesExcludedHeading` -> `sameRendering`, and
the section travels. No HTML-block trickery and no unclosed anything is needed;
a renderer shows exactly "Audit Notes".

Verified admitted on HEAD and refused on the parent on every path:
`## Audi<i>t</i> Notes` (section scan), `  ## Audi<i>t</i> Notes` (indented
ATX), `Audi<i>t</i> Notes\n===` (setext), `<h2>Audi<i>t</i> Notes</h2>` (raw
HTML), and the same shape against `Why This Matters` and `Open Questions`.

The round traded ONE case (`Audit<br>Notes`, which the parent leaked) for
SEVEN. This is the exact pattern the eight prior rounds were trying to break:
closed on one path, opened on its neighbour.

Remedy (class-closing): take BOTH readings, exactly as this same commit already
did for the soft heading bound. Compare the slug of `renderedText` with tags
replaced by "" AND with tags replaced by " ", and refuse if either names an
excluded heading. That closes `Audit<br>Notes` and `Audi<i>t</i> Notes`
together and cannot reopen either.
