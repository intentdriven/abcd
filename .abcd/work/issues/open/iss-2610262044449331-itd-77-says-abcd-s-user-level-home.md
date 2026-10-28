---
schema_version: 1
id: "iss-2610262044449331"
slug: "itd-77-says-abcd-s-user-level-home"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1226 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-77, itd-76]
finding_ref: cfk-d3e7d5949c214df6
refines: [itd-77]
---

itd-77 says abcd's user-level home defaults to `~/.abcd.noindex/`; shipped itd-76 says it defaults to `~/.abcd/`, with the sources corpus at `~/.abcd/sources/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "That state lives in abcd's user-level home — `~/.abcd.noindex/` by default"
- End B: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"

itd-77 names itd-76's confidential-sources corpus as state that lives in the user-level home, and says that home is `~/.abcd.noindex/` by default. itd-76 says the same corpus lives in abcd's user-level home at `~/.abcd/sources/` by default. Both give the default location of the same directory, and the paths differ.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1226.
