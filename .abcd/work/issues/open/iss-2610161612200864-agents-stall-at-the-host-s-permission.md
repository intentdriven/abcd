---
schema_version: 1
id: "iss-2610161612200864"
slug: "agents-stall-at-the-host-s-permission"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker interview 2026-10-16 (abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a bundled rule, recalled on shell work: never put rm inside a bash -c or sh -c script built from variables. Write the script to a file in one command and run it in the next (the guard already requires that split), and use literal paths with /bin/rm -f --, so the host can check each command and need not ask. Pin the rule with a test."
---

Agents stall at the host's permission prompt when they write a command the host cannot check. On 2026-10-16 a background agent ran a bash -c script built from variables that deleted files. The host could not check it, so it asked 'This shell -c script runs rm and could not be checked. Do you want to proceed?', and the agent waited for a person. No bundled rule teaches the forms the host can check.
