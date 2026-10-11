---
schema_version: 1
id: "iss-2610070927488590"
slug: "zsh-the-user-shell-abcd-s-own-sessions"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the 2026-10-07 branch clean-out (lab-261007061316-2413a98)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Add a SHELL hazard to the guard registry for an unbraced parameter immediately followed by a colon and one of r h t e s q l u (a zsh history modifier) inside a word git or a path consumes, naming the braced form ${name}:… as the fix; warn rather than block, since the shape is also legal text in bash."
---

zsh, the user shell abcd's own sessions run in, reads $name:r, $name:h, $name:t, $name:e and $name:s as history modifiers on an unbraced parameter, so a refspec or a path written "$sha:refs/heads/x" expands to the sha with its last suffix stripped followed by 'efs/heads/x'. On 2026-10-07 this broke a git fetch into a lab snapshot inside a chain whose next step, after a ';', dropped the stash being archived; the objects were restored from the repository within two minutes. The shell-hazard registry teaches rm-after-cd and similar shapes but not this one, and bash does not have it, so a command tested in bash passes and fails in zsh.
