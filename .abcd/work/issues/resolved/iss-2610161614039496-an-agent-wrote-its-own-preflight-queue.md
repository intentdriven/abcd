---
schema_version: 1
id: "iss-2610161614039496"
slug: "an-agent-wrote-its-own-preflight-queue"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker interview 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a second rule to the bundled SCRATCH domain: before writing a helper script, check whether an abcd command already does it (abcd --help and the command pages); write one only when none does, keep it in the main checkout's .abcd/.work.local/scratch/, and capture the gap as an issue so abcd can add the command. Name the case that went wrong: queueing a preflight is abcd implement load --wait -- make preflight, never a hand-written waiter. Pin the rule with a test."
resolution: "The bundled SCRATCH domain now carries a second rule: before writing a helper script an agent checks whether an abcd command already does the job (abcd --help and the command pages), writes its own only when none does, keeps it in the main checkout's .abcd/.work.local/scratch/, and captures the gap so abcd can add the command. The rule names queueing a preflight as abcd implement load --wait -- make preflight, never a hand-written waiter, and points at LOAD's waiter rule. AGENTS.md's local-tier paragraph sends reused files to the main checkout's tier. TestScratchRuleSendsAHelperToAbcdsCommandsFirst pins it."
impact: fix
resolved_by:
  commit: "6e7836e35"
---

An agent wrote its own preflight queue script although abcd's waiter already did that. The script lived in the host scratchpad, a reboot wiped it, and its queued runs never started. No bundled rule tells agents to use abcd's commands before writing helpers.

## Grounds

- pursued: a prompt to script a preflight queue now injects the SCRATCH block carrying the abcd implement load --wait -- make preflight pointer, and the agent queues through the waiter; a hand-written preflight waiter or queue script written after this ships, or that prompt injecting no SCRATCH block, would show it wrong
