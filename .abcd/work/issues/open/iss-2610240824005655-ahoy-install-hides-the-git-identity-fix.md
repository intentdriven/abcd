---
schema_version: 1
id: "iss-2610240824005655"
slug: "ahoy-install-hides-the-git-identity-fix"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "ahoy install from a terminal, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "ask the identity question on its own at a terminal, quoting the current identity and the pin, independent of the config-change approval"
---

ahoy install hides the git identity fix behind the generic settings approval: the identity mismatch is listed only as one line under 'Change the settings listed above?', a no there means the identity question is never asked, and nothing says that declining the settings approval also declines the identity pin the person wanted to keep.
