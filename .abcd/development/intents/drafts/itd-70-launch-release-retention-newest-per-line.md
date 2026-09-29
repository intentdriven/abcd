---
id: itd-70
slug: launch-release-retention-newest-per-line
severity: minor
related_issues: [iss-282]
---

# Drawn-out intent

Every `launch ship` prunes superseded releases under a newest-per-line (MAJOR.MINOR) retention policy: publishing vX.Y.Z removes the superseded vX.Y.(Z-1) tag and (once cut) its GitHub Release + assets, while the terminal release of every other line survives.

_Drawn out from a human brief edit by the brief-change derivation gate (itd-61 / spc-75). Grill + flesh out before planning._
