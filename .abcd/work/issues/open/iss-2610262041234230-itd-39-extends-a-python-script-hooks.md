---
schema_version: 1
id: "iss-2610262041234230"
slug: "itd-39-extends-a-python-script-hooks"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 997 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-b8278f5582a61b6f
---

itd-39 extends a Python script `hooks/prompt_router_hook.py`; the memory chapter says the prompt router is a binary sub-command, not a script file in `hooks/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory."
- End B: `.abcd/development/brief/05-internals/07-memory.md:181` — "It is a sub-command of the abcd binary reached through the hook manifest, not a script file in `hooks/`."

itd-39's scope extends the prompt router by editing `hooks/prompt_router_hook.py`. The memory chapter says the router is a sub-command of the abcd binary, reached through the hook manifest, and explicitly not a script file in `hooks/`. The file the intent tells an implementer to change does not exist under the brief's description of the router.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 997.
