---
schema_version: 1
id: "iss-2610111749099986"
slug: "guard-blocks-zsh-fpath-prepend-idiom"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "Fable review of security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Treat an unfixed fpath element that is exactly $fpath or $FPATH as the account's own directories, so a search that finds no fixed file falls through to the existing unseen-account-function reading instead of blocking; prove with Check tests that both idioms allow and a fixed directory holding a blocker still blocks"
resolution: "An fpath element that is the array itself ($fpath, \"${fpath[@]}\", $fpath[@], $FPATH) is its value before the assignment: the line's where it set one, else the account's own directories, which is also what fpath+= appends to; the autoload search passes over the account's directories and reads the fixed ones around them. Proved by TestFpathAccountIdiomAllows, watched failing first, with fixed directories holding a blocker still blocking."
impact: fix
resolved_by:
  commit: "d5b4ca8d7"
---

zsh's common idioms `fpath=(~/.zfunc $fpath); autoload -Uz compinit; compinit` and `fpath+=(dir); autoload -Uz compinit` now block as script-unread, because the autoload search reaches the unfixed $fpath element before finding compinit, although a function found only in the account's own directories is what the guard already treats as unseen when FPATH is untouched. Found by the Fable review of lane G, 2026-10-11 (autoloadTargets).
