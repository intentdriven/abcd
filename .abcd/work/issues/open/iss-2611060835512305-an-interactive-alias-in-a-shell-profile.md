---
schema_version: 1
id: "iss-2611060835512305"
slug: "an-interactive-alias-in-a-shell-profile"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "a stage's shell under a drive"
remedy: "Start stage shells with no aliases and with standard input closed, and test that the stage timeout or the no-movement check ends a shell held this way."
---

An interactive alias in a shell profile, such as `cp -i`, can hold a stage's shell on a prompt nobody answers, because the stage has no terminal; iss-2611040337241546 is the same hazard in a host's shell tool. Nothing yet shows whether the stage timeout or the no-movement check ends such a shell, so an unattended drive can lose hours to one overwrite prompt.

Correction on reading the wall: a drive's stage runs with its home set to the stage's own emptied folder and an allow-list of variables, so a person's shell profile does not reach it, by reading and not yet by a real run. The hazard is an agent in a host session on the person's real home, where two agents met these prompts on 6 November.
