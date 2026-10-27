---
schema_version: 1
id: "iss-2610262042201742"
slug: "itd-76-puts-abcd-s-user-level-home-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1062 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-dd148aace909232c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename as the other itd-76 home records."
---

itd-76 puts abcd's user-level home at `~/.abcd/`; the memory chapter puts the user scope under `~/.abcd.noindex/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"
- End B: `.abcd/development/brief/05-internals/07-memory.md:14` — "`~/.abcd.noindex/memory/` | **Personal preferences** and cross-project principles that have no single repo home"

itd-76 keeps the local-only corpus in abcd's user-level home, at ~/.abcd/sources/ by default. The memory chapter's two-scope table puts the user-level store at ~/.abcd.noindex/memory/. The two documents give different roots for the same user-level home, so a corpus built from itd-76 ends up outside the directory the brief treats as the user scope.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1062.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same rename as the other itd-76 home records.
