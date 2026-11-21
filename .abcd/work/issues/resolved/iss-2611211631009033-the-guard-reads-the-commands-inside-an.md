---
schema_version: 1
id: "iss-2611211631009033"
slug: "the-guard-reads-the-commands-inside-an"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Walk exports in the order the commands run, with an eval's commands where the eval stands"
resolution: "The guard's export walk takes commands in the order they run, so an eval's export holds for the commands after it."
impact: additive
resolved_by:
  commit: "2f4e6ee86"
---

The guard reads the commands inside an eval after the rest of the line, so an export made inside an eval is judged after an opencode command that follows the eval.
