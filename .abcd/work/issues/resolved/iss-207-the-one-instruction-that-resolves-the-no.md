---
schema_version: 1
id: "iss-207"
slug: "the-one-instruction-that-resolves-the-no"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "first manual plugin install test (2026-08-10)"
found_at: "hooks/bootstrap.sh"
resolution: "The bootstrap notice and the README now name the plugin-root binary by path, so the one-time PATH setup instruction runs in the no-binary-on-PATH state it addresses; a bare abcd ahoy install on either surface is held by test."
impact: fix
---

The one instruction that resolves the no-binary-on-PATH state cannot be run in that state. hooks/bootstrap.sh's success notice ends 'For the abcd command in your own terminal, run `abcd ahoy install` once.' and README.md repeats it: 'run `abcd ahoy install` once to put the plugin-root binary on your PATH.' But abcd is not on PATH — that is exactly the condition the sentence addresses — so the command as written fails with 'command not found'. Observed consequence on the first manual install (2026-08-10): the agent, unable to run the printed command, invented 'cd ~/.claude/plugins/marketplaces/abcd-marketplace && go run ./cmd/abcd ahoy install' and told the user to run that instead — a source-build instruction reaching into the harness's plugin cache, which is not the documented install path and requires a Go toolchain. Fix direction: print the absolute plugin-root path in the notice (the script already holds it as $binary) and give README the same concrete form, so the sentence is copy-pasteable in the state it describes. Related to iss-205, which is the same root confusion on the command surface.