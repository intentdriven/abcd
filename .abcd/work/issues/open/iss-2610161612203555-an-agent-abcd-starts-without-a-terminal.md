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
---

An agent abcd starts without a terminal has nobody to answer the host's permission prompts. A command the host cannot check is then refused, and the lane stalls or fails. The product thinker chose to have abcd answer those prompts for the agents it starts, using its existing guard rules.
