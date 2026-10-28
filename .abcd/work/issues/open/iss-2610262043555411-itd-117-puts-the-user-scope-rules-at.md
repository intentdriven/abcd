---
schema_version: 1
id: "iss-2610262043555411"
slug: "itd-117-puts-the-user-scope-rules-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1171 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117, itd-77]
finding_ref: cfk-a6f862e78c03fec5
---

itd-117 puts the user-scope rules at `~/.abcd/rules.json`; itd-77 puts abcd's user-level home at `~/.abcd.noindex/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "That state lives in abcd's user-level home — `~/.abcd.noindex/` by default"

itd-117 says machine-wide conventions live in `~/.abcd/rules.json`. itd-77 says the user-level home, where abcd keeps per-person state, is `~/.abcd.noindex/` by default. The two records name different home directories for the same user-scope layer, so someone building from either would read or write a different path.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1171.
