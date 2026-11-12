---
schema_version: 1
id: "iss-2611121749343164"
slug: "the-shared-default-branch-resolver"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/defaultref.go"
remedy: "Return an error beside the name and have each caller tell the two apart."
---

The shared default-branch resolver answers an empty name both for a repository with no default branch and for a failed ref read, so every caller reports a failed read as no default branch. The landings verb states it as a limit.
