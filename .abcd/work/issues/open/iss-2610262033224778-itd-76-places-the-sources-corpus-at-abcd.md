---
schema_version: 1
id: "iss-2610262033224778"
slug: "itd-76-places-the-sources-corpus-at-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 395 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-dfa5b365b2fd5173
---

itd-76 places the sources corpus at `~/.abcd/sources/`; 01-ahoy lists it under the user-scope home `~/.abcd.noindex/`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:472` — "the local sources corpus the library verb maintains"

itd-76 calls abcd's user-level home `~/.abcd/` and puts the local-only sources corpus at `~/.abcd/sources/` by default. 01-ahoy's inventory of the user-scope directory, `~/.abcd.noindex/`, lists `sources/` as the local sources corpus the library verb maintains. Two names for one user-level home send a builder to two different directories for the same corpus.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 395.
