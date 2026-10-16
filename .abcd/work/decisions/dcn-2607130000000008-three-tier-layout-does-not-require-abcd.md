---
id: dcn-2607130000000008
slug: three-tier-layout-does-not-require-abcd
date: 2026-07-13
---
2026-07-13 (itd-85 M3): three-tier-layout does NOT require .abcd/.work.local/ to
be present (diverges from the plan's literal "present and gitignored") — it is
created on demand and a fresh clone has none; requiring presence would flag every
clean checkout. The load-bearing assertion is "if present, gitignored". Mechanics
revision, premise intact.
