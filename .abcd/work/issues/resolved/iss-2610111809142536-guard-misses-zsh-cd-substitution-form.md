---
schema_version: 1
id: "iss-2610111809142536"
slug: "guard-misses-zsh-cd-substitution-form"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "Fable review of guard round 2, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Treat a cd or pushd with two operands, or with an option word the reader does not know for that shell, as leaving the directory unknown (never 'stays' and never 'enters the operand') unless the segment is known to run under bash; prove with Check tests over a planted with-n/withsub layout that the line no longer reads the wrong directory"
resolution: "Fixed in the guard's cd reader: a cd or pushd bash and zsh read differently (a second operand, bash's -e and -@, pushd -n) now leaves the line in every directory either reading enters, and the scripts after it are read in each, blocking on a hazard in any: bash's reading enters the operand, stays put for pushd -n, and refuses a second operand; zsh's substitution form replaces the first occurrence of the first word in the current directory's path with the second, where that directory exists. The directory is unknown only where neither reading names one; -q and -s are read as before. Pinned by TestCdReadsEveryShellsDirectory and TestCdHazardInEitherReadingBlocks over a planted with-n/withsub layout and by the pushd -n cases in TestCdOptionsBeforeTheOperand. The CDPATH block's reason and commands/guard.md now also name the no-candidate and too-many-entries cases."
impact: fix
---

zsh reads `cd X Y` with two operands, and an option word it does not take (`cd -e X`, `cd -@ X`, `pushd -n X`), as the substitution form: it replaces the first word in the current directory's path with the second and enters the result, so the guard's script reader judges a script in one directory while zsh runs the one in another. Found by the Fable review of guard round 2 on 2026-10-11 (reproduced with zsh 5.9: `cd <p>/with-n/dir && pushd -n sub && pwd` prints <p>/withsub/dir); the cd forms predate that round, and round 2 made pushd -n read as staying put.
