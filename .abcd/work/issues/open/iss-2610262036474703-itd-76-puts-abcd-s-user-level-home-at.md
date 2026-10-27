---
schema_version: 1
id: "iss-2610262036474703"
slug: "itd-76-puts-abcd-s-user-level-home-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 674 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:216"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-67d2fc170e920638
---

itd-76 puts abcd's user-level home at ~/.abcd/; the history chapter treats ~/.abcd as a legacy location

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:216` — "or a legacy `~/.abcd` standing with no plugin binary in the plugin root"
- End B: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default"

itd-76 names abcd's user-level home as ~/.abcd/ and puts the sources corpus at ~/.abcd/sources/ by default. The history chapter puts its user-level store under ~/.abcd.noindex/ and calls a standing ~/.abcd 'legacy', a state its hook wrappers refuse when no plugin binary is in the plugin root. A builder following itd-76 would create the corpus in the directory the brief treats as left behind.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 674.
