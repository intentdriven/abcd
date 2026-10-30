---
schema_version: 1
id: "iss-2610262219399974"
slug: "in-githooks-commit-msg-a-top-level-line"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "commit-msg hook review fixes, fix/catch-ci-failures-before-push"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Find why the build line loses go_bin's value: a greedy reduction that keeps each variant valid shell leaves 57 lines (the go_bin assignment, about a dozen if, for and case blocks, the build line), from which no single block can be removed, while 32 trivial if blocks alone do not trigger it, so the cause is likely how shell states merge across those branches; fix that reading and add the reduced script as a script-corpus regression case."
---

In .githooks/commit-msg, a top-level line 'if git rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1; then' placed after the hook's RS004 setup made the guard's script reading block the hook's later '"$go_bin" build -buildvcs=false -o "$work/abcd" ./cmd/abcd' line as kill-by-holder (a program name from a command substitution reading as a refused command), where the hook without that line reads as a warn only. The same if without the redirection, or the check read into a variable first, reads as before; a six-line script holding only go_bin, that if and the build is allowed, and respelling is_scissors without && inside [[ ]] does not clear it, so the earlier context that triggers it is not isolated.
