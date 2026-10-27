---
schema_version: 1
id: "iss-2610262037458871"
slug: "itd-117-puts-the-user-scope-rules-layer"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 747 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-ec65f4e38baaab31
---

itd-117 puts the user-scope rules layer at ~/.abcd/rules.json; 30-inbox says the rules loader reads ~/.abcd.noindex/rules.json

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/brief/04-surfaces/30-inbox.md:143` — "the stance the rules loader takes on `~/.abcd.noindex/rules.json` behind a symlinked"

Shipped itd-117 says the machine's shared conventions live in ~/.abcd/rules.json and inject into every managed repository. 30-inbox describes the rules loader's stance on ~/.abcd.noindex/rules.json behind a symlinked ~/.abcd.noindex. These are two names for the one user-scope home and its rules file. A builder or user following itd-117 would write a file that the loader, as the brief describes it, never reads.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 747.
