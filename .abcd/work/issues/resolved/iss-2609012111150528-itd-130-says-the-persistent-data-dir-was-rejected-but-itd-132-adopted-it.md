---
schema_version: 1
id: "iss-2609012111150528"
slug: "itd-130-says-the-persistent-data-dir-was-rejected-but-itd-132-adopted-it"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md"
resolution: "itd-132 carries a typed reversal of itd-130's data-directory decision and itd-130 gains one dated line noting it; the maintainer confirmed reversal over refinement on 2026-09-01."
impact: internal
resolved_by:
  commit: "07479728"
---

itd-130's Decisions section records that 'the persistent-data-dir alternative is rejected, not deferred' (grilled 2026-08-20), and one day later itd-132/spc-35 adopted exactly that shape as the download cache with a copied PATH file, superseding spc-21's re-fetch criterion. itd-130's text was never revised, so the shipped record now carries a decision its successor reversed without a typed link between them. Surfaced by the itd-130 fidelity audit (receipt rcp-264f7b144576) as a diverged item. The fix is a record change, not code: itd-132 should carry a typed 'reverses' link to the itd-130 decision, or itd-130's Decisions should note the supersession with the date, and the reversal is a human's to confirm.

## Grounds

- pursued: a reader of itd-130 is warned that its rejection was reversed; if readers still take itd-130's paragraph as current, the note is in the wrong place
