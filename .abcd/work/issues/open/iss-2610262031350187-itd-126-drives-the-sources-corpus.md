---
schema_version: 1
id: "iss-2610262031350187"
slug: "itd-126-drives-the-sources-corpus"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 149 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-126]
finding_ref: cfk-e437c28cfce85306
---

itd-126 drives the sources corpus through a source verb, while the brief keeps the corpus under /abcd:library

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17` — "`abcd source ingest` imports the repo's shared entries into the local corpus."
- End B: `.abcd/development/brief/01-product/02-context.md:13` — "keep the sources corpus (`init` / `add` / `declassify` / `ledger` / `sync-banlist` / `cite-check`, per itd-76)"

itd-126 has abcd source share write citation data and abcd source ingest import shared entries into the local corpus. The context chapter says /abcd:library keeps the sources corpus through init, add, declassify, ledger, sync-banlist and cite-check. One corpus is given two verb names, and ingest means the memory store under library but the sources corpus under source.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 149.
