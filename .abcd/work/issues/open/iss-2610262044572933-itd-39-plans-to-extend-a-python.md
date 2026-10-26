---
schema_version: 1
id: "iss-2610262044572933"
slug: "itd-39-plans-to-extend-a-python"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1241 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-39, itd-105]
finding_ref: cfk-4fb99e4761a30b07
---

itd-39 plans to extend a Python prompt-router hook; itd-105 says the plugin's hooks call the abcd binary

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"
- End B: `.abcd/development/intents/shipped/itd-105-a-fresh-plugin-install-just-works-the.md:26` — "Today the hooks call `$CLAUDE_PLUGIN_ROOT/abcd`"

itd-39, still planned, scopes its retrieval as an extension of hooks/prompt_router_hook.py from rules-only to rules plus memory. itd-105 states that the plugin's hooks call $CLAUDE_PLUGIN_ROOT/abcd, with a POSIX-sh bootstrap as the only hook that runs without the binary. The UserPromptSubmit hook itd-39 sets out to extend is not the hook itd-105 describes the plugin running, so a builder following itd-39 starts from a file that is not the delivered hook.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1241.
