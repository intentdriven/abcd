---
schema_version: 1
id: "iss-2610262033124129"
slug: "itd-39-extends-a-python-prompt-router"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 373 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:663"
remedy: "In itd-39 (the 'Hook extension' scope bullet) replace the hooks/prompt_router_hook.py reference with the prompt router sub-command of the abcd binary, reached through the hook manifest, as 05-internals/07-memory.md section 9 already says."
related_intents: [itd-39]
finding_ref: cfk-f82d3aec4e4eb134
---

itd-39 extends a Python prompt-router hook; the ahoy chapter says every hook event command is a shim that finds and runs the abcd binary

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:663` — "resolving shim rather than a plain binary call."
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"

itd-39 scopes memory recall as extending `hooks/prompt_router_hook.py` (itd-3) from rules only to rules plus memory. The ahoy chapter describes the shipped hook manifest: nine event types, each event command a resolving shim rather than a plain binary call, with `UserPromptSubmit` provisioning the plugin-root binary when it is missing. So the prompt router that hook runs is the abcd binary. The planned work names a hook file the shipped surface does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 373.
