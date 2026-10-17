---
schema_version: 1
id: "iss-2610111559070779"
slug: "guard-allows-zsh-equals-process"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-8m66-666c-8995, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Recognise a word-initial =( as a process substitution in the tokenizer (never the =( of a bash array assignment such as a=(1 2)) and give source, . or a shell pointed at it the same interpreter-reads-stream block as <(...); prove with Check tests, watched failing first, that both zsh spellings block, the <(...) control still blocks, source =(blocker) still blocks on the inner command, and a=(1 2) stays allowed"
resolution: "The tokenizer reads a word-initial =( as a process substitution, as it reads <( and >(, so source, . or a shell pointed at one gets the interpreter-reads-stream block; an array assignment's =( follows its name and stays a word. Proved by TestZshEqualsProcessSubstitutionIsAStream (internal/core/guard/equalsprocsub_test.go), watched failing first."
impact: fix
---

The guard allows `zsh -c` lines that bypass it through process substitution spelled `=(...)`: zsh writes the substitution to a temporary file and sources it, while the tokenizer reads `=` as a plain word, so a bundled blocker inside an `echo` or `cat` runs as the operator with exit 0.

Private security advisory GHSA-8m66-666c-8995 (draft, severity high). Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `tokenizeAt` (internal/core/guard/tokenize.go) enters process substitution only for `<(` and `>(`; `targetsOf` (internal/core/guard/script.go) returns no file for the operand `=`. Reproduced at that commit with `abcd guard check --command`: `zsh -c 'source =(echo "<blocker>")'` and `zsh -c 'source =(cat <file>)'` are allow (exit 0), while the `source <(echo "<blocker>")` control blocks as interpreter-reads-stream (exit 1).
