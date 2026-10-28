---
schema_version: 1
id: "iss-2610262030556470"
slug: "the-mental-model-says-the-intent-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 61 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:3"
remedy: "none (filed automatically)"
finding_ref: cfk-3009913f18db61e2
---

The mental model says the intent for the Now / Next / Later status block is still planned; the site chapter describes the status page and the bare board rendering that block as shipped behaviour.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:3` — "the status block that renders it is itd-2609212103568351, planned"
- End B: `.abcd/development/brief/04-surfaces/22-site.md:111` — "It opens with the Now / Next / Later block the bare `abcd` board carries"

03-mental-model says the status block that renders Now / Next / Later is itd-2609212103568351 and that this intent is planned, meaning not yet delivered. 22-site describes the shipped status page as opening with the Now / Next / Later block the bare `abcd` board carries, rendered from the same read. A reader cannot tell from these two pages whether the block exists, so one of the two lifecycle claims is wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 61.
