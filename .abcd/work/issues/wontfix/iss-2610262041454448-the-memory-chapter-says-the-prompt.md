---
schema_version: 1
id: "iss-2610262041454448"
slug: "the-memory-chapter-says-the-prompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1022 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:181"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-1677a08239531db0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. hooks/ holds only shell wrappers that exec the binary; the plan records the Python-to-Go change before the intent shipped."
---

The memory chapter says the prompt router is a binary sub-command, not a script in hooks/; the shipped rules-loader intent scopes it as hooks/prompt_router_hook.py.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:181` — "It is a sub-command of the abcd binary reached through the hook manifest, not a script file in `hooks/`."
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "`hooks/prompt_router_hook.py`"

07-memory states that itd-3's prompt router is a sub-command of the abcd binary reached through the hook manifest, and explicitly not a script file in hooks/. itd-3's scope defines the router as the Python file hooks/prompt_router_hook.py. The two documents disagree about what the shipped router is and where it lives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1022.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. hooks/ holds only shell wrappers that exec the binary; the plan records the Python-to-Go change before the intent shipped.
