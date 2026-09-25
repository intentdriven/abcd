---
schema_version: 1
id: "iss-2608291915432717"
slug: "nested-third-party-home-path-is-reported-by-no-detector"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/scanner/identity.go"
resolution: "home_path_other evaluates a /Users or /home segment after a path byte when the path token is absolute, so /Volumes/Backup/Users/<name>/x and /mnt/data/home/<name>/x are reported and redacted. The record's second sentence, on a home under a deeper root, was itself redacted when captured and its path cannot be recovered; a layout whose name is the third segment (a grouped /home/<group>/<name>) keeps only the first two segments masked, which this change does not alter."
impact: fix
resolved_by:
  commit: "8636491e444b42cff5a4d4ee87c8b39dec5600d8"
---

ultra-v0.6.8 follow-up observation (security review of the branch): a third party's home path nested under a longer path — /Volumes/Backup/Users/alice/x, or any prefix before /Users or /home — is reported by no detector in internal/adapter/scanner/identity.go and is committed verbatim by memory and history. leadingBoundaryOK refuses the generic-home match when the byte before /Users is a path-segment byte, home_path_self correctly declines it, and local_username does not match a foreign name. The caller's OWN name in the same shape is still caught hard_fail by local_username, so this is the warn-class home_path_other gap only. The generic regex also covers two segments at most, so a home under a deeper root ([redacted-path]/alice) is only ever matched by its first two. Pre-existing on main, identical before and after the branch. Fix: let home_path_other evaluate a /Users|/home segment that follows a path byte, or widen the regex to a nested root.

## Grounds

- pursued: a third-party home under a longer absolute root is caught by home_path_other; a nested home left verbatim by memory or history would show it wrong
