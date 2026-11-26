---
id: dcn-2611260259547907
slug: a-guard-killed-by-a-signal-holds-the
date: 2026-11-26
---
2026-11-26 — A guard killed by a signal holds the command. When the PreToolUse hook's guard ends with an exit above 128 (a signal, which a resource limit or the host's timeout produces), the hook exits 2 with its FAILED TO RUN line instead of exit 1, so a command the guard was still judging does not run unjudged. Any other unexpected exit, a missing binary included, keeps the loud exit-1 path, so a guard that cannot start never blocks every command. Grounds: a killed guard had started on this command and may have been about to refuse it, while one that never ran says nothing about it. To reverse: drop the exit-above-128 branch in hooks/pre-tool-use.sh at every place it judges a guard's exit (the old-folder rung, the dogfood build's fall-back to the installed release, and the plugin-root and PATH rung), and the tests TestGuardShimHoldsWhenTheGuardIsKilled, TestGuardShimHoldsAKilledGuardBesideTheOldFolder and TestGuardHoldsWhenTheReleaseIsKilledBehindTheDogfoodBuild with it.
