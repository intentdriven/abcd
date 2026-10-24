---
schema_version: 1
id: "iss-2610241134315056"
slug: "ahoy-install-reports-clean-in-a-clone"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "setup redesign build, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
remedy: "make an unarmed committed hook set a gap that setup offers to arm (one person-approved question naming core.hooksPath and the clone's .git/config), and never report clean while the hooks it installed do not run"
refines: [iss-2610020704152920, iss-2609120447480903]
---

ahoy install reports 'clean' in a clone whose committed hooks are not armed: in abcd's own checkout on 2026-10-24 the detection pass read banlist.hooks_path as 'unarmed' (core.hooksPath unset, so .githooks/pre-commit and pre-merge-commit never run) both before and after setup, yet no gap was listed and the install summary said 'abcd is set up in this repository'.
