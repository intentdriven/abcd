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
resolution: "The shell-settings write now re-checks the file with Lstat and SameFile just before the replace, so a link swapped in after the plan is refused rather than followed."
impact: internal
resolved_by:
  intent: "itd-2610240919030303"
  commit: "65cb33df8"
---

the shell-settings edit checks the profile file and then replaces it atomically, leaving a short window in which a symlink swapped in at the path would be replaced by a regular file rather than refused (the replace renames over the leaf, so it never writes through the link, but the person's link is lost).
