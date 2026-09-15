---
schema_version: 1
id: "iss-2608301350537306"
slug: "maskmarkupdata-searches-the-whole-remainder-of-the-document"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-10-security"
found_at: "internal/core/reading/project.go"
resolution: "The cursor onto the next newline advances with the walk instead of restarting, since the walk's own offsets only ever move forward."
impact: fix
resolved_by:
  intent: "itd-183"
---

maskMarkupData searches the whole remainder of the document for a newline once per attribute assignment making the mask quadratic in file size

Found by the round-10 adversarial security review of build/itd-183, and fixed in
the same round. REGRESSION INTRODUCED BY THIS ROUND'S OWN FIRST COMMIT.

Bounding an attribute value to its own line searched the whole remainder of the
document for a newline once per attribute assignment. Measured over a single
line of `="x"` assignments:

```
assignments   bytes    per-assignment search   monotone cursor
200 000       800 KB   0.80 s                  2.3 ms
400 000       1.6 MB   3.07 s                  3.6 ms
800 000       3.2 MB   12.28 s                 7.3 ms
```

Clean quadratic against a 4 MiB MaxFileBytes cap that bounds ONE file rather
than how many of them a repository holds, so k such files cost k times over. As
with the bound scan, the severity is availability rather than leak -- a hang is
a denied assembly -- but silently, which is the staging a fail-closed floor
cannot afford.

Remedy: the walk's own offsets only ever advance, so the cursor onto the next
newline can advance with them instead of restarting.
