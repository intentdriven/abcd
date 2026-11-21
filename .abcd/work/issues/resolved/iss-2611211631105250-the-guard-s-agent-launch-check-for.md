---
schema_version: 1
id: "iss-2611211631105250"
slug: "the-guard-s-agent-launch-check-for"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Count allexport assignments and local -x as exports in the guard's export walk"
resolution: "The guard's export walk counts allexport assignments, in every spelling, and local -x as exports."
impact: additive
resolved_by:
  commit: "2f4e6ee86"
---

The guard's agent-launch check for opencode counts an export only when it is written with export, declare or typeset, so a variable exported through the shell's allexport option or a local -x before an opencode command is not seen.
