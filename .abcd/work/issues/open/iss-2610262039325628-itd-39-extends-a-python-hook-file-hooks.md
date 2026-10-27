---
schema_version: 1
id: "iss-2610262039325628"
slug: "itd-39-extends-a-python-hook-file-hooks"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 867 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-4c3d6668e245440f
---

itd-39 extends a Python hook file `hooks/prompt_router_hook.py`; the configuration chapter says the prompt router is the binary's `abcd hook prompt-router` entrypoint

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:969` — "`abcd hook prompt-router` is the `UserPromptSubmit` entrypoint the hook manifest wires."

itd-39's scope extends `hooks/prompt_router_hook.py` from rules to rules plus memory. The configuration chapter says the `UserPromptSubmit` entrypoint the hook manifest wires is `abcd hook prompt-router`, and its plugin layout lists `bootstrap.sh`, one shell shim per event and `hooks.json` under `hooks/`, with no Python router. The intent's scope extends a file that does not exist in the design the brief describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 867.
