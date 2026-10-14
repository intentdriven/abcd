---
schema_version: 1
id: "iss-2610121214249290"
slug: "guard-append-prefix-hides-command"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security fix lane G3 follow-up, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Read NAME+=VALUE as an assignment word wherever isAssignment is read (command position, prefixes, brace expansion, carried values), keeping isName and the name split correct, and prove with Check tests that FOO+=x before a blocker blocks as the blocker"
resolution: "One rule, splitAssignment, now says what an assignment word is (NAME=, NAME+=, NAME[SUB]= and NAME[SUB]+=, as bash 3.2, bash 5 and zsh 5.9 read them before a command) and every reading that recognised NAME= asks it: the walk to command position and Tier 2, carried values, the names a line assigns, the script reader's shell state, the git config prefix read and the prompt variables. Proved by TestAnAppendOrSubscriptPrefixIsAnAssignment, watched failing first, with FOO+=x echo hi and other ordinary lines allowed."
impact: fix
resolved_by:
  commit: "593ca7467"
---

A command prefixed by an append assignment is not judged: FOO+=x rm -rf / is allowed while FOO=x rm -rf / blocks, because isAssignment accepts only NAME=VALUE, so the walk to command position takes FOO+=x for the command and the hazard for its operands; bash and zsh read NAME+=VALUE as an assignment prefix. Found in security lane G3 follow-up, 2026-10-12, while reading PATH+= for iss-2610120953445472, whose fix reads += only in the script reader's shell state.
