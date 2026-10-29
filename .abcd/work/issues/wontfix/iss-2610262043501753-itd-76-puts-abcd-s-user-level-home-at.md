---
schema_version: 1
id: "iss-2610262043501753"
slug: "itd-76-puts-abcd-s-user-level-home-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1165 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609301918174237-a-provider-off-this-machine-takes-a-file.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-2609301918174237, itd-76]
finding_ref: cfk-b85116982ea1baeb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/library.md puts the sources corpus at `~/.abcd.noindex/sources/`; itd-76 was true when written and a later shipped ruling changed the home."
---

itd-76 puts abcd's user-level home at ~/.abcd/; itd-2609301918174237 puts the owner's machine-layer settings under ~/.abcd.noindex/

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609301918174237-a-provider-off-this-machine-takes-a-file.md:41` — "in `~/.abcd.noindex/config.json` under `oracle.bundles`, keyed by the project's root-commit SHA"
- End B: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default"

itd-76 calls ~/.abcd/ abcd's user-level home and keeps the sources corpus there. itd-2609301918174237 keeps the owner's personal, machine-layer settings in ~/.abcd.noindex/config.json. These are two names for abcd's one per-user home, so a builder cannot tell from the record which directory user-level state belongs in.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1165.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/library.md puts the sources corpus at `~/.abcd.noindex/sources/`; itd-76 was true when written and a later shipped ruling changed the home.
