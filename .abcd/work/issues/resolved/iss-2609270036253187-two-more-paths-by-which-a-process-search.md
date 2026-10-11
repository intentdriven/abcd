---
schema_version: 1
id: "iss-2609270036253187"
slug: "two-more-paths-by-which-a-process-search"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "Kill-by-search reads the three paths: an unquoted here-document whose body holds the search, redirected into an xargs kill; a substitution in a command that reads a pipe, which runs with that pipe as its input; and a shell string handed the search in its positional parameters or its own text (TestKillFedThroughAHereDocOrAnInheritedPipeIsBlocked)."
impact: fix
resolved_by:
  commit: "7cf24a47a"
---

Two more paths by which a process search's output reaches a kill are not read by kill-by-search: an unquoted here-document whose body holds a command substitution, redirected into an xargs kill (a here-string of the same substitution blocks), and a command substitution inside a command that reads a pipe, which inherits that pipe as its standard input (an xargs kill in the substitution reads the search piped into its command). A third, a search handed to a shell string as a positional parameter the string's kill reads, is not read either: it is a pid carried through a variable, the half DECISIONS 2026-09-25 (c) defers with iss-2609251824244354, unless it is read the way xargs's input to a string is (DECISIONS 2026-09-27), which is a call for the next guard lane. Found while fixing iss-2609270028388291.

## Grounds

- pursued: each path from a search to a kill through a here-document, an inherited pipe or a string parameter blocks, and the no-leak shapes stay allowed; an allowed spelling of one of the three, or a kill of a named pid that now blocks outside the recorded string over-block, would show it wrong
