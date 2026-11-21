---
schema_version: 1
id: "iss-2611211631207098"
slug: "the-opencode-config-blocker-names"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Add OPENCODE_CONFIG_DIR to the agent-launch opencode config entry"
resolution: "OPENCODE_CONFIG_DIR is in the agent-launch opencode config entry."
impact: additive
resolved_by:
  commit: "62edeabc9"
---

The opencode config blocker names OPENCODE_CONFIG and the other settings variables but not OPENCODE_CONFIG_DIR, which opencode also reads its permission rules from.
