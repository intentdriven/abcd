---
schema_version: 1
id: "iss-2610111600039059"
slug: "guard-reads-bash-env-literally"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-2c9x-jxfm-w3wc, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Treat a BASH_ENV or ENV value that holds a command substitution, a backquote or a parameter expansion as not a fixed path and block it (expanding only a leading tilde, as now), so the guard never judges text bash would expand first; prove with Check tests, watched failing first, that the substitution and $HOME forms block and the fixed-path BASH_ENV control still blocks on its file's contents"
resolution: "A BASH_ENV or ENV value that holds a dollar sign or a backquote is now a script-unread block, both where a shell is handed it and where a shell script run directly reads BASH_ENV, because the shell expands the value before it opens the file; the leading-tilde handling and the fixed-path reading are unchanged. Proved by TestStartupVariableTheShellExpandsBlocks (internal/core/guard/startupexpand_test.go), watched failing first; the fixed-path control still blocks on its file's contents."
impact: fix
---

The guard judges `BASH_ENV` as literal text, but bash expands the value (command substitution, parameters, tilde) before using it as a filename, so `BASH_ENV='$(<blocker>)' bash -c true` runs the blocker during that expansion, and `BASH_ENV='$HOME/f' bash -c true` runs a file the guard never reads.

Private security advisory GHSA-2c9x-jxfm-w3wc (draft, severity high). Follows GHSA-r2w5-wf2r-jmf8, which closed the fixed-path form. Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `shellTargets` (internal/core/guard/script.go) passes the stored BASH_ENV text unchanged to `startup`, which resolves it as a path. Reproduced at that commit with `abcd guard check --command`: the `$(...)` form and the `HOME=<dir> BASH_ENV='$HOME/evil.sh'` form are both allow (exit 0); the control `BASH_ENV=<dir>/evil.sh bash -c true` blocks (exit 1).
