---
schema_version: 1
id: "iss-2610262031557361"
slug: "itd-200-installs-the-status-line"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 196 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-7fa225db94eedb9b
---

itd-200 installs the status-line defaults under ~/.abcd/; invariant 21 names the user-level home ~/.abcd.noindex.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override, and"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "the `.noindex` suffix the home carries, `~/.abcd.noindex`"

itd-200 says the status line's defaults move into the user-level home under ~/.abcd/ at install, and its scope lists 'The user-level defaults under `~/.abcd/`'. Invariant 21 names the home abcd owns as '`~/.abcd.noindex`', with the suffix the desktop indexer honours. The user-level home has two names in the corpus.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 196.
