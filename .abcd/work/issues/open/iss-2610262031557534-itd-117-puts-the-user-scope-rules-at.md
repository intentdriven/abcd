---
schema_version: 1
id: "iss-2610262031557534"
slug: "itd-117-puts-the-user-scope-rules-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 195 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-9b9ca5b2dc722db8
---

itd-117 puts the user-scope rules at ~/.abcd/rules.json; invariant 21 names abcd's user-level home ~/.abcd.noindex.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "the `.noindex` suffix the home carries, `~/.abcd.noindex`"

itd-117 says a machine's shared conventions 'now live once in `~/.abcd/rules.json`' as the user scope. Invariant 21 says the space abcd owns is the home carrying the .noindex suffix, '`~/.abcd.noindex`', and the brief places every other operator-level path under it. The corpus uses two names for one user-level home, so a reader following itd-117 is sent to a path the brief no longer names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 195.
