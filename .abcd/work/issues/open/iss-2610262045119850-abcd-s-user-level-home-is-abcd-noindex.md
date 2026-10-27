---
schema_version: 1
id: "iss-2610262045119850"
slug: "abcd-s-user-level-home-is-abcd-noindex"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1257 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-39, itd-76]
finding_ref: cfk-7e9d91118f4c1a7c
---

abcd's user-level home is ~/.abcd.noindex/ in itd-39 and ~/.abcd/ in itd-76

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35` — "the hook queries the `index.md` of both memory scopes (`<workspace>/.abcd/memory/` and `~/.abcd.noindex/memory/`)"
- End B: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default"

itd-39 puts the user's memory scope at `~/.abcd.noindex/memory/`. itd-76 puts the sources corpus in abcd's user-level home at `~/.abcd/sources/` by default. These are two different paths for the one user-level home, so a reader cannot tell which directory the user scope lives in.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1257.
