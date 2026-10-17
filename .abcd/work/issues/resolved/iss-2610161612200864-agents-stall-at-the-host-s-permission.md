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
resolution: "The guard refuses a bash -c or sh -c script built from variables that runs rm, rmdir, shred or unlink (shell-c-variable-delete), with the rewrite in the refusal: write the script to a file in one command and run it in the next, or delete on the line itself with literal paths through /bin/rm -f -- or a \"${VAR:?}\"-guarded variable. The SHELL rules teach the same words before shell work, so an agent writes a command each check can read and no host has to ask a person."
impact: fix
resolved_by:
  commit: "6703d2ac9a12229b57c08214476007878c5d1fa1"
---

Agents stall at the host's permission prompt when they write a command the host cannot check. On 2026-10-16 a background agent ran a bash -c script built from variables that deleted files. The host could not check it, so it asked 'This shell -c script runs rm and could not be checked. Do you want to proceed?', and the agent waited for a person. No bundled rule teaches the forms the host can check.

## Grounds

- pursued: an agent told why and how to rewrite a variable-built -c delete before it runs stops writing commands a host must ask a person about; a host prompt on a -c delete an agent wrote after this rule was taught would show it wrong
