---
schema_version: 1
id: "iss-2608301237450133"
slug: "rawheadingtitleends-materialises-every-bound-match-in-the-re"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
resolution: "The raw heading bound scan walks with a single-match search over an advancing offset and stops at the first hard bound, instead of materialising every candidate bound in the remainder for every opener."
impact: fix
resolved_by:
  intent: "itd-183"
---

rawHeadingTitleEnds materialises every bound match in the rest of the document per opener making the scan quadratic and hanging the assembler

Found by the round-9 adversarial security review of build/itd-183.
REGRESSION INTRODUCED BY 2225d6cb -- the parent used FindStringIndex, which
stops at the first match.

`rawHeadingTitleEnds` calls FindAllStringSubmatchIndex over the whole remainder
of the document for EVERY heading opener, even though it breaks at the first
hard bound. The scan becomes quadratic with a large constant. Measured:

```
body size   parent 044ac6ed   HEAD 2225d6cb
8 KB        1 ms              508 ms
32 KB       7 ms              5.7 s
128 KB      48 ms             92.5 s
4 MiB (cap) 33.9 s            did not finish
```

~1900x at 128 KB. A committed .md of repeated `<h2>` up to the 4 MiB
MaxFileBytes cap the code itself sets does not finish. Severity is availability
rather than leak, because the floor is fail-closed: a hang is a denied
assembly. The live corpus is unaffected -- measured sub-second at all four
positions.

Remedy: restore early termination. Walk with FindStringSubmatchIndex over an
advancing offset and stop as soon as `hard` is set (and stop tracking `soft`
once found). The "both readings" semantics are unchanged; only the
materialisation of the whole match list goes away.
