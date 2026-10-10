---
schema_version: 1
id: "iss-2610090925390900"
slug: "guard-allows-shellopts-ps4-substitution"
severity: "major"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane G sibling sweep (peer probe, reproduced at 016c1510a)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Treat an environment prefix (bare or through env) that sets SHELLOPTS or PS4 on a shellFamily shell as interpreter-reads-stream and judge PS4's command substitutions with the inline rules, proved by a guard test watched fail first; sweep every variable bash expands or imports at startup (PS0, PS1, PS2, PS4, PROMPT_COMMAND, SHELLOPTS, BASHOPTS)."
resolution: "the guard judges the text a line hands bash through a prompt variable: each command substitution in a decoded PS0/PS1/PS2/PS4 value, and a PROMPT_COMMAND value as a command line, wherever the line assigns it (prefix, env operand, declaration builtin), since bash -x, SHELLOPTS, BASHOPTS and -i each reach it"
impact: fix
---

abcd guard allows a bash -c whose environment prefix sets SHELLOPTS=xtrace and a PS4 holding a command substitution: bash expands PS4 before tracing the first command, so a blocker in PS4 runs although the -c payload is harmless. Reproduced at main 016c1510a: `SHELLOPTS=xtrace PS4='$(pkill node)' bash -c true` (and the `env` form) judges allow while `pkill node` blocks. Sibling of the BASH_ENV finding iss-2610090821484829, found in the security-drain-2026-10-09 sweep; kept uncommitted until its fix lands.
