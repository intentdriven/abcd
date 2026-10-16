---
schema_version: 1
id: "iss-2610161612203555"
slug: "an-agent-abcd-starts-without-a-terminal"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker interview 2026-10-16 (abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner"
remedy: "When abcd launches a headless agent through its runner, pass a permission policy for that launch only: a permission mode, plus an approval tool backed by abcd's guard registry, which allows what the guard judges safe and refuses with the guard's reason. Never edit the person's host settings (~/.claude). Interactive sessions are unchanged."
resolution: "A headless claude launch whose role's contract grants the shell now carries the guard as the harness's permission prompt tool (abcd guard approve over MCP, set per launch on the command line): a shell command the guard reads whole and clears runs, anything else is refused with the guard's reason. A per-launch hook could not carry it, since bare mode skips every hook (dcn-2610161624025493)."
impact: additive
resolved_by:
  commit: "f8a0f8866"
---

An agent abcd starts without a terminal has nobody to answer the host's permission prompts. A command the host cannot check is then refused, and the lane stalls or fails. The product thinker chose to have abcd answer those prompts for the agents it starts, using its existing guard rules.

## Grounds

- pursued: a headless lane's shell commands stop stalling on prompts nobody answers, while hazards, protected-path writes and critical-path removals stay refused; a lane refused on a command the guard reads whole and clears, or a hazard that runs unasked, would show it wrong
