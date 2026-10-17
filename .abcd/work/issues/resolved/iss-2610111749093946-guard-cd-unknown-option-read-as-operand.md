---
schema_version: 1
id: "iss-2610111749093946"
slug: "guard-cd-unknown-option-read-as-operand"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "Fable review of security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Skip zsh's q and s and pushd's n as cd options, and treat any other word starting with - that the reader does not recognise as making the operand unfixed rather than reading it as a directory name; prove with Check tests"
resolution: "cdOperand now reads zsh's -q and -s as cd options and pushd's -n as an option that adds to the stack without changing directory, so the directory stays as it was; any other word starting with - that the reader does not know leaves the operand unfixed and the directory unknown, never read as a directory name or searched through CDPATH. commands/guard.md says so. Proved by TestCdOptionsBeforeTheOperand."
impact: fix
---

`cdOperand` takes an option it does not know (zsh's `cd -q sub`, `pushd -n sub`) as the operand, so with CDPATH set the option word is 'searched' and the line blocks with a misleading reason (observed: cd -q sub && bash s.sh blocks). Found by the Fable review of lane G, 2026-10-11.
