---
schema_version: 1
id: "iss-2610262032008964"
slug: "the-mental-model-says-the-now-next-later"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 207 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103568351]
finding_ref: cfk-4958a26d2c8c422e
---

The mental model says the Now/Next/Later status block's intent is planned; that intent is on the shipped shelf and says the board shows the block.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:3` — "the status block that renders it is itd-2609212103568351, planned"
- End B: `.abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:21` — "`abcd` and the site's Status page show Now, Next and Later, computed from the record and maintained by nobody."

The brief names itd-2609212103568351 as the record that renders the status block and calls it planned. The corpus holds that record under intents/shipped/, and its press release says the bare board and the site's Status page show Now, Next and Later. The block cannot be both unbuilt and shipped, so a reader of the brief would take a delivered surface for future work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 207.
