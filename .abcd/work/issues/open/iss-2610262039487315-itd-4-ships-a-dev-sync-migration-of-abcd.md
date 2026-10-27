---
schema_version: 1
id: "iss-2610262039487315"
slug: "itd-4-ships-a-dev-sync-migration-of-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 888 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:1194"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-936bba8318c0381e
---

itd-4 ships a `dev-sync` migration of `.abcd/.work.local/issues.md`; 03-configuration says no `dev-sync` verb exists.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:1194` — "No `dev-sync` verb is registered on the binary,"
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:46` — "`dev-sync` promotes existing `.abcd/.work.local/issues.md` entries to the structured ledger on first run after install"

itd-4's scope has `dev-sync` move existing `.abcd/.work.local/issues.md` entries into the structured ledger on the first run after install. 03-configuration says the whole `dev-sync` section is staged, that no `dev-sync` verb is registered and that the migration has nothing left to do. itd-4 is in shipped/, so it claims a delivered mechanism that the configuration chapter says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 888.
