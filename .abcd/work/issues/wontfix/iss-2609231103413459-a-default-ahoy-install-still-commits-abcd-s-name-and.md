---
schema_version: 1
id: "iss-2609231103413459"
slug: "a-default-ahoy-install-still-commits-abcd-s-name-and"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
deferred_after: "v0.9.0"
deferral_reason: "The scope of the naming ruling (does it cover the hooks and fence, and rename-with-migration vs sanction) is the product thinker's; asked in the run's interview handover, section F, 2026-09-23."
wontfix_reason: "Ruled a sanctioned exception, not a defect: the product thinker's ruling F of 2026-09-23 (run A interview, 14:38Z) holds that the 2026-09-11 naming ruling covers the name-guard hooks and the .gitignore fence, and that they keep their markers and naming as the one allowed mention of abcd in an adopted repository, because the hooks run the binary and the fence tells people not to hand-edit it. No rename and no migration. The ruling covers the markers, not abcd's record ids: the hook template's ids and design-record path were dropped in 68fcb6a3, the docs name the exception in bc65f8a6, and DECISIONS.md records the ruling."
---

A default ahoy install still commits abcd's name, and abcd-internal record references, outside the .abcd/ namespace the adopter accepts: the two name-guard hooks (.githooks/pre-commit and .githooks/pre-merge-commit: the '# abcd-name-guard: v1' marker, prose naming abcd, 'abcd ahoy' and 'abcd banlist', and the itd-74 / spc-20 ids) and the .gitignore fence ('# BEGIN ABCD', '# abcd-managed block — do not hand-edit. Run /abcd:ahoy to refresh.'). Reproduced 2026-09-23 on a fresh git repo with 'abcd ahoy install --yes --adopt --visibility private': grep -ril abcd over the tracked result names both hooks and .gitignore. The 2026-09-11 ruling (iss-2609110944498549) is that ahoy install must not write abcd by name into a repository it adopts, and prepare-this-repo's acceptance promises no abcd-internal content in any committed artefact. The fix for that record removed the conventions-file block from the default (docs target skip); these writes remain. Major: this is the unmet remainder of iss-2609110944498549 (major), whose acceptance 1 (no committed file contains abcd after a default install) this record now holds; it carries that severity rather than a lower one, so the remainder still meets the release-cut guard. The hooks are not the files a repository's contributors read first, and the hooks run the abcd binary, so part of the naming is functional, but that bears on the fix, not on whether the ruling is met. What a fix has to settle: the hook marker and the fence are what detection classifies by (classifyGuardHook, gitignoreBlockDrifts), so a rename is a migration over every already-adopted repo, and the hooks are security-sensitive; whether the name in a hook that invokes the tool is in scope of the ruling at all is the product thinker's call.

## Grounds

- declined: Ruled a sanctioned exception, not a defect: the product thinker's ruling F of 2026-09-23 (run A interview, 14:38Z) holds that the 2026-09-11 naming ruling covers the name-guard hooks and the .gitignore fence, and that they keep their markers and naming as the one allowed mention of abcd in an adopted repository, because the hooks run the binary and the fence tells people not to hand-edit it. No rename and no migration. The ruling covers the markers, not abcd's record ids: the hook template's ids and design-record path were dropped in 68fcb6a3, the docs name the exception in bc65f8a6, and DECISIONS.md records the ruling.
