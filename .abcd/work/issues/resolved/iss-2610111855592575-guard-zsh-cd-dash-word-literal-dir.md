---
schema_version: 1
id: "iss-2610111855592575"
slug: "guard-zsh-cd-dash-word-literal-dir"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G round 5, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "When a lone dash word is not a known option and a directory of that name exists, add it as zsh's candidate (excluding zsh's -N and +N stack forms, which stay unknown); update the test that expects the allow; prove with a Check test over a planted -x/s.sh"
resolution: "A lone word starting with a dash with a letter neither shell takes (cd -x, pushd -x, cd -qx) is now zsh's one operand, the directory zsh enters by that name where one exists, so the scripts after it are read there; zsh's -N stack words name no directory and stay unknown, and +N stays unknown as before (TestCdLoneUnknownDashWordIsZshsDirectory; TestCdOptionsBeforeTheOperand's cd -x case now expects the block)."
impact: fix
---

A bare `cd -x` (an option word neither shell takes, with no operand) leaves the directory unknown in the guard, but zsh 5.9 enters a directory literally named `-x` when one exists (verified 2026-10-11 with cd and pwd only), so a following `bash s.sh` runs the script there unread. An existing test expects the allow; zsh's -N directory-stack words need care.
