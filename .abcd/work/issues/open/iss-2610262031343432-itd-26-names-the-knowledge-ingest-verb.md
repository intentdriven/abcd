---
schema_version: 1
id: "iss-2610262031343432"
slug: "itd-26-names-the-knowledge-ingest-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 147 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-0afc06e6222389b6
---

itd-26 names the knowledge ingest verb /abcd:memory ingest, while the brief puts ingest under /abcd:library

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge)"
- End B: `.abcd/development/brief/01-product/02-context.md:13` — "curate a queryable memory store (`ingest` external sources / `ask` queries / `lint` health-checks)"

itd-26 says one shared substrate serves two user-facing verbs, /abcd:loot for code and /abcd:memory ingest for knowledge. The context chapter places the memory store's ingest, ask and lint under /abcd:library. The draft names the consuming verb by a command the brief does not give it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 147.
