---
schema_version: 1
id: "iss-2610241158083013"
slug: "the-shell-settings-edit-checks-the"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "setup redesign build, shell-settings lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "the shellpath package's Apply step (branch feat/setup-redesign-shellpath)"
remedy: "re-check the leaf with an lstat immediately before the rename and refuse if it changed, or open the file with O_NOFOLLOW and compare its identity before and after"
---

the shell-settings edit checks the profile file and then replaces it atomically, leaving a short window in which a symlink swapped in at the path would be replaced by a regular file rather than refused (the replace renames over the leaf, so it never writes through the link, but the person's link is lost).
