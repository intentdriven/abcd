---
schema_version: 1
id: "iss-2610262045100283"
slug: "the-prompt-router-is-the-python-file"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1256 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:36"
remedy: "in itd-39's Hook extension bullet, replace the Python file name with the Go verb abcd hook prompt-router, as dcn-2607110000000003 ruled; the memory chapter has the sibling stale line (iss-2610262041454448)"
related_intents: [itd-89, itd-39]
finding_ref: cfk-4d0d70e148f60077
---

The prompt router is the Python file hooks/prompt_router_hook.py in itd-39 and the shipped hidden verb `hook prompt-router` in itd-89

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:36` — "a hidden, operator-internal verb following the shipped `hook prompt-router` pattern"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"

itd-89 models `abcd hook session-end` on the shipped `hook prompt-router` pattern, which makes the router a hidden binary verb. itd-39 names the same router as `hooks/prompt_router_hook.py` and plans to extend that file. One component has two names in two different forms, and only one of them exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1256.
