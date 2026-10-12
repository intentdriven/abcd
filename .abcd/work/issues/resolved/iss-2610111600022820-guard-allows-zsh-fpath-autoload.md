---
schema_version: 1
id: "iss-2610111600022820"
slug: "guard-allows-zsh-fpath-autoload"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-jh4q-4cgv-5crr, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Track FPATH and fpath as file-selecting variables and read autoload as a command that loads a function file: when the line fixes the directories and the name, judge the file zsh would load; when either is not fixed by the line, block; prove with Check tests, watched failing first, that both reproduction spellings block with a blocker in the function file and a benign function file still allows"
refines: [iss-2610111559070779]
resolution: "FPATH and fpath are tracked as one file-selecting variable (an fpath=(...) array assignment is now recorded by the tokenizer and read as the value), and autoload, functions -u and typeset -fu are read as loading a function file: a path operand is that file, and a name is searched in the FPATH the line set and the file found is judged; a name, option or directory the line does not fix blocks as script-unread, and a line that leaves FPATH alone still loads the account's own functions unseen. Proved by TestAutoloadFromLineFPATHIsRead (internal/core/guard/autoload_test.go), watched failing first; the benign function file still allows."
impact: fix
---

The guard allows a zsh line that loads a function file through `FPATH` or `fpath` and `autoload`: zsh reads the file and runs its body when the function is called, and the guard never opens it, so a bundled blocker in that file runs as the operator with exit 0.

Private security advisory GHSA-jh4q-4cgv-5crr (draft, severity high). Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `trackedVars` (internal/core/guard/script.go) lists BASH_ENV, ENV, HOME, ZDOTDIR and PATH, not FPATH; `targetsOf` skips `autoload` as a command that reads no file. Reproduced at that commit with `abcd guard check --command` and a blocker in `<dir>/fp/p`: `FPATH=<dir>/fp zsh -c 'autoload p; p'` and `zsh -c 'fpath=(<dir>/fp); autoload p; p'` are both allow (exit 0).
