---
schema_version: 1
id: "iss-204"
slug: "hooks-hooks-json-puts-bootstrap-sh-and"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "first manual plugin install test (2026-08-10)"
found_at: "hooks/hooks.json"
resolution: "Collapsed the three SessionStart entries into one chained command (bootstrap, then prompt-router-reset and session-start in the same shell), so sequencing is owned by hooks.json rather than assumed of a harness that runs sibling hooks in parallel. spc-21's false ordering warrant corrected in place."
impact: fix
---

hooks/hooks.json puts bootstrap.sh and the two binary-backed SessionStart hooks in ONE event group and relies on list order, but the harness runs all matching hooks in PARALLEL — the Claude Code hooks reference states verbatim 'All matching hooks run in parallel.' spc-21 asserts the opposite: 'Ordering within one event's hook list is preserved by the harness, so the binary-backed session hooks run after the bootstrap in the same event.' That warrant is false, and it is load-bearing. On a fresh install the two gated hooks evaluate [ -f "$CLAUDE_PLUGIN_ROOT/abcd" ] while bootstrap.sh is still downloading the ~10.7MB release binary (timeout 240), lose the race, and both print 'the plugin binary is not installed'. Observed on the first manual marketplace install (2026-08-10): both errors printed BEFORE bootstrap's own success notice, which is direct evidence of parallel execution. Consequence is not only noise — 'hook prompt-router-reset' and 'hook session-start' genuinely do not run for that session, and itd-105 AC#1 ('every hook executes successfully — no No such file or directory, no unguarded-shell warning') fails on every fresh install AND every plugin update, since each update lands in a fresh commit-stamped cache directory with no binary. Fix direction: collapse the three entries into ONE SessionStart command that runs bootstrap.sh and then the two binary calls in a single shell, so the sequencing is owned here rather than assumed of the harness.