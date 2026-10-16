---
id: dcn-2608310000000006
slug: the-position-regime-disagreement-is
date: 2026-08-31
---
2026-08-31 — The position/regime disagreement is refused in
`reading.LoadDefinition`, not cross-checked in the ingest verb. Ground: the
locator is the one thing that claims to resolve a position to its regime, and
a resolver returning a confidently wrong answer would hand the supply-regime
gate the wrong licence silently. Rejected: a cross-check inside the ingest
path, which would be a second table of one fact and leave the primitive broken
for the status render.
