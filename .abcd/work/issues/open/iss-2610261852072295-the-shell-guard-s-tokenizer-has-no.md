---
schema_version: 1
id: "iss-2610261852072295"
slug: "the-shell-guard-s-tokenizer-has-no"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "fix/catch-ci-failures-before-push: script corpus warn after the commit-msg hook gained its RS004 run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Teach tokenize that [[ at command position opens one compound command that ends at its ]]: && and || inside it join tests and do not end a segment, and < and > inside it are comparisons, not redirections; a regression test holds this repro, and maxScriptCorpusWarns goes back to 0."
---

The shell guard's tokenizer has no notion of [[ … ]]: it cuts [[ $n -gt 1 && $a == *x* && $d == "$c" ]] at each && into separate segments, so $d == "$c" ]] reads as a command whose program the line does not fix. The guard then guesses $d may be a shell (a guessed sh -c payload) and re-reads its words as a shell string with the variables' values written in, so a quoted value such as c=" >8 " becomes a redirect to a file 8. A shell run of any script later in the same script then warns script-written-then-run (a write the guard cannot place). Found on .githooks/commit-msg's is_scissors test (cut_line holds ' ------------------------ >8 ------------------------') once the hook ran check-issue-resolution.sh through bash after it; minimal repro: a script of c=" >8 ", that [[ ]] line, then bash /abs/script.sh. Traced in a debug build: payload.go's kindShell ref with guessed:true, payload '==  >8  ]]', from a segment whose tokens are [<unknown> == ' >8 ' ]].
