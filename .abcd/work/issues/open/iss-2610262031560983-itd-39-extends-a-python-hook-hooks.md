---
schema_version: 1
id: "iss-2610262031560983"
slug: "itd-39-extends-a-python-hook-hooks"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 197 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "In itd-39, replace hooks/prompt_router_hook.py (the Hook extension bullet) with the binary's `abcd hook prompt-router`, and say the Python script never ships."
related_intents: [itd-39]
finding_ref: cfk-5ac269b7a8343812
---

itd-39 extends a Python hook, hooks/prompt_router_hook.py; 02-dependencies says abcd ships as a single Go binary with no required external tools.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"
- End B: `.abcd/development/brief/02-constraints/02-dependencies.md:3` — "abcd has no required external tools. It ships as a single Go binary"

itd-39's scope says the prompt-router hook '`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory', which makes the hook that serves recall a Python script. 02-dependencies states 'abcd has no required external tools. It ships as a single Go binary'. A planned intent built on a Python hook would add an interpreter dependency the brief rules out, or it names a hook the shipped product does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 197.
