---
schema_version: 1
id: "iss-2610262036343040"
slug: "itd-117-puts-the-user-scope-rules-file"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 657 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-7b3aa3fb08c13b1d
---

itd-117 puts the user-scope rules file under ~/.abcd/; the history chapter calls ~/.abcd a legacy location

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:216` — "or a legacy `~/.abcd` standing with no plugin binary in the plugin root"

itd-117 says the machine's shared conventions live once in ~/.abcd/rules.json. 11-history treats a ~/.abcd directory as a legacy home the hooks report on, while the user-level home the brief chapters name is ~/.abcd.noindex/. Two names are given for the one user-level home, and a reader following itd-117 writes the rules file where the brief says nothing current lives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 657.
