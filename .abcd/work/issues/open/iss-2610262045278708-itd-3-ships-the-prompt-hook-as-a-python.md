---
schema_version: 1
id: "iss-2610262045278708"
slug: "itd-3-ships-the-prompt-hook-as-a-python"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1275 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-3, itd-66]
finding_ref: cfk-a8b5b12588ba95e8
---

itd-3 ships the prompt hook as a Python file in hooks/; itd-66 says hooks/ ships shell and the payload carries no Python

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "**`hooks/prompt_router_hook.py`** — `UserPromptSubmit` hook."
- End B: `.abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:85` — "no other imported entrypoint (`hooks/` and `scripts/` ship shell)"

itd-3 names the UserPromptSubmit hook as hooks/prompt_router_hook.py. itd-66 records that the payload's include roots carry no Python and that hooks/ and scripts/ ship shell, which is why it calls its Python-import criterion moot. A Python file under hooks/ cannot be shipped if hooks/ holds only shell.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1275.
