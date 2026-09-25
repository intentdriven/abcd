---
schema_version: 1
id: "iss-2609251535090117"
slug: "the-generic-account-position-check-in-the-identity-matchers"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "standsAsAccountName folds only the window each test asks about (endsWithFold over the home literal's and each prefix's length) and stops its '@' search at a 64-byte local part; underAbsoluteRoot takes its token start from pathTokens, one forward walk per line. The six generic-login and nested-home shapes in TestScanLineWorkIsLinear were watched failing at 13.6-15.9x per quadrupling on a scratch copy of the parent commit and pass at the fix; a 128 KB line of a generic login scans in about 60 ms. The quadratic check never reached a release: it came in with this lane."
impact: internal
resolved_by:
  commit: "7d7255e1"
---

The generic-account position check in the identity matchers is quadratic in line length. standsAsAccountName (internal/adapter/scanner/identity.go) lower-cases the whole line prefix before every match to test a home-literal suffix and each account-root prefix, and underAbsoluteRoot walks back to the path token's start for every non-boundary home match. A line dense in a generic login (dev, git, test, user, build, app, node: all on genericAccountNames) under a login on that list therefore costs the square of its length: the review measured 80, 160 and 320 KB at 0.92, 3.34 and 13.6 s against 31, 60 and 124 ms for a non-generic control, so one 4 MB transcript line holding a tool result costs about half an hour in the launch gate or the store-before-commit redactor. The identity matchers sat outside every count-based cost guard (iss-2609240203462704), which is why no test saw it.

## Grounds

- pursued: every per-match position test reads a bounded window, so a line's identity work grows with its length; TestScanLineWorkIsLinear failing on a generic-login shape, or a wall-clock probe growing 4x per doubling, would show it wrong.
