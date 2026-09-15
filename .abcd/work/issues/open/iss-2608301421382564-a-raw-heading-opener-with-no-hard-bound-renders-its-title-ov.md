---
schema_version: 1
id: "iss-2608301421382564"
slug: "a-raw-heading-opener-with-no-hard-bound-renders-its-title-ov"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-10-ruthless"
found_at: "internal/core/reading/project.go"
---

a raw heading opener with no hard bound renders its title over the whole remainder so the scan is quadratic in a shape the linearity test does not cover

Found by the round-10 adversarial ruthless review of build/itd-183. The CLASS is
pre-existing -- the parent computes the same title over the same remainder --
but round 10 multiplied its constant, because a title is now rendered from each
masking under each of two readings of a removed tag.

When a heading opener has no hard bound, `rawHeadingTitleEnds` returns
`len(rest)`, so `renderedTexts` and `site.Slug` run over the whole remainder
once per opener. Measured on this round, `<p role="heading">x` repeated with no
close and no h-tag anywhere:

```
openers   bytes    elapsed
2 000     38 KB    3.3 s
4 000     76 KB    13.5 s
```

Clean quadratic; the reviewer measured 57 s at 8 000 openers, and at the 4 MiB
MaxFileBytes cap this is hours. The parent has the same class roughly 1.5x
faster.

`TestRawHeadingScanStaysLinearInTheOpenerCount` does NOT cover this shape: its
openers each close immediately, so 8 000 of them cost 0.1 s against a 10 s bound.
The test's name over-claims, and the shape it misses is the expensive one.

Remedies to weigh together: cap the length of text a title may be read from (a
heading is not kilobytes long), or give an unbounded opener a bound of the
document's own end-of-line rather than end-of-file. Seed material for the
exclusion floor's own intent, with the test's name to correct alongside it.
