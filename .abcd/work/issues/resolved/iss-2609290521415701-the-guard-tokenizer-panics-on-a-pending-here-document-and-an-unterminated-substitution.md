---
schema_version: 1
id: "iss-2609290521415701"
slug: "the-guard-tokenizer-panics-on-a-pending-here-document-and-an-unterminated-substitution"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A substitution suspends the here-documents pending where it opens, as bash 3.2, bash 5 and /bin/sh do: a newline inside it reads only the documents it opened, its own lines are read as commands, and its close restores the pending ones, whose bodies begin on the line after it, so the suspended command's record of its documents never points at a body already read; TestPendingHereDocumentInsideAnUnterminatedSubstitution pins 60 lines with no panic and TestPendingHereDocumentWaitsOutItsSubstitution 144 lines whose substitution holds a newline and a hazard, none below the same line without the document."
impact: fix
resolved_by:
  commit: "627a73a4d"
---

The shell guard's tokenizer panics with index out of range when a here-document is pending and a substitution opened on the same line is left unterminated: cat <<E <(x, cat <<E $(x and cat <<E with an open backtick, each followed by a newline and the line E. The newline inside the substitution reads the pending bodies and resets docOwners to nil, while the enclosing command's curDocs, restored when the unterminated substitution is closed at the end of input, still indexes it, so flushSegment's docOwners[k] panics. The hook fails closed (a panic exits 2, which the hook passes through as a block), but abcd guard check prints a Go stack trace instead of a verdict, and a panic is not a verdict the guard can explain. Present at main 285455056 and at 2fa327bb5.

## Grounds

- pursued: every pending-document shape with a process, command or backtick substitution holding a newline, closed or not, reads with no panic and at least as strictly as its sibling without the document, and a 3-minute tokenizer fuzz finds no panic; a panic, or a line that reads more leniently than its sibling, would show it wrong
