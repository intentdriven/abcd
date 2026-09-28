---
schema_version: 1
id: "iss-2609270036253187"
slug: "two-more-paths-by-which-a-process-search-s-output-reaches-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

Two more paths by which a process search's output reaches a kill are not read by kill-by-search: an unquoted here-document whose body holds a command substitution, redirected into an xargs kill (a here-string of the same substitution blocks), and a command substitution inside a command that reads a pipe, which inherits that pipe as its standard input (an xargs kill in the substitution reads the search piped into its command). A third, a search handed to a shell string as a positional parameter the string's kill reads, is not read either: it is a pid carried through a variable, the half DECISIONS 2026-09-25 (c) defers with iss-2609251824244354, unless it is read the way xargs's input to a string is (DECISIONS 2026-09-27), which is a call for the next guard lane. Found while fixing iss-2609270028388291.
