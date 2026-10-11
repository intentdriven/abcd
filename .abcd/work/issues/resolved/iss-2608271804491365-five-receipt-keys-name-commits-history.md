---
schema_version: 1
id: "iss-2608271804491365"
slug: "five-receipt-keys-name-commits-history"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/work/reviews"
resolution: "receipt-keys translation note added to the reviews charter naming the five history-rewritten keys"
impact: internal
resolved_by:
  commit: "a032377d"
---

five sha-keyed semantic-gate receipt directories under work/reviews/ name commits unreachable from HEAD: three are old-history ids from the 2026-08-06 attribution rewrite (translatable via the sha-map) and two are squashed-away branch heads with no current-history equivalent. Do not rename receipt directories; add a Receipt keys paragraph to the reviews charter recording the translation route and naming the five, so a reader stops bisecting for commits that history rewrote. Related: iss-2608271707587825 promotes the translation tables to research/data — repoint this paragraph when that lands.