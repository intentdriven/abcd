---
schema_version: 1
id: "iss-2609231103413459"
slug: "a-default-ahoy-install-still-commits-abcd-s-name-and"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
---

A default ahoy install still commits abcd's name, and abcd-internal record references, outside the .abcd/ namespace the adopter accepts: the two name-guard hooks (.githooks/pre-commit and .githooks/pre-merge-commit: the '# abcd-name-guard: v1' marker, prose naming abcd, 'abcd ahoy' and 'abcd banlist', and the itd-74 / spc-20 ids) and the .gitignore fence ('# BEGIN ABCD', '# abcd-managed block — do not hand-edit. Run /abcd:ahoy to refresh.'). Reproduced 2026-09-23 on a fresh git repo with 'abcd ahoy install --yes --adopt --visibility private': grep -ril abcd over the tracked result names both hooks and .gitignore. The 2026-09-11 ruling (iss-2609110944498549) is that ahoy install must not write abcd by name into a repository it adopts, and prepare-this-repo's acceptance promises no abcd-internal content in any committed artefact. The fix for that record removed the conventions-file block from the default (docs target skip); these writes remain. Minor rather than major: they are not the files a repository's contributors read first, and the hooks run the abcd binary, so part of the naming is functional. What a fix has to settle: the hook marker and the fence are what detection classifies by (classifyGuardHook, gitignoreBlockDrifts), so a rename is a migration over every already-adopted repo, and the hooks are security-sensitive; whether the name in a hook that invokes the tool is in scope of the ruling at all is the product thinker's call.
