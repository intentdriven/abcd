---
schema_version: 1
id: "iss-2610090925399967"
slug: "guard-allows-exported-bash-function-env"
severity: "major"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane G sibling sweep (peer probe, reproduced at 016c1510a)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Judge the body of any BASH_FUNC_*%% assignment in an env or prefix position before a shellFamily shell with the inline rules (or block it as interpreter-reads-stream), proved by a guard test watched fail first that the env form blocks and a plain `env FOO=1 bash -c true` stays allow."
duplicates: [iss-2610090925390900]
resolution: "env and sudo now step every '=' operand before the command as an assignment, as they apply it, and the body of a BASH_FUNC_<name>%% function they export is judged with the inline rules; a prefix form does not run in bash, which takes no % in a name"
impact: fix
---

abcd guard allows a bash -c whose environment carries an exported function (a BASH_FUNC_<name>%% variable, through env): bash imports it at startup, so a function named like the -c command runs a blocker in its body although the -c payload is harmless. Sibling of the BASH_ENV finding iss-2610090821484829, found in the security-drain-2026-10-09 sweep; kept uncommitted until its fix lands.
