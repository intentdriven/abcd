---
id: dcn-2607210000000005
slug: the-mapping-table-s-per-tier-status
date: 2026-07-21
---
2026-07-21 — The mapping table's per-tier status columns are a **ceiling**,
not merely a prediction. Every conventions adapter already honours its row
(`convGlossarySource` returns partial where the row says partial;
`convPlatformSource` returns grounded where it says grounded), so itd-95's
and itd-96's three new adapters cap at `StatusPartial` — the value all three
rows predict — and carry signal strength in `Confidence` instead. Rejected:
returning `StatusGrounded` for a dedicated `NAMING.md` or a prose-bearing
`ARCHITECTURE.md`, which would have made the rendered brief table wrong and
required editing `mapping.go`, the brief-to-lifeboat contract both intents
put out of scope. Every acceptance bar asks only for "non-blank", so the
ceiling satisfies them. Revisit only by amending the mapping row first.
