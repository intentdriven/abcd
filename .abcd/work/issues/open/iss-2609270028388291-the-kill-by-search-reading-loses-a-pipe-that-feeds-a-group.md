---
schema_version: 1
id: "iss-2609270028388291"
slug: "the-kill-by-search-reading-loses-a-pipe-that-feeds-a-group"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The kill-by-search reading loses a pipe that feeds a group or a command string from outside it. A pipe INTO a brace or paren group feeds every command in the group, but the tokenizer hands it only to the commands before the first separator inside, so a process search piped into a group whose later command is an xargs kill (after a sleep, a read, or an and-list) allows silently while the one-command group blocks. In the same class, payloadInput hands a command string only its running shell's pipe, so a here-string or a process-substitution redirect carrying a search into a shell string whose xargs kills allows, while the same redirect into a plain xargs kill blocks. Found by review-drainG2 (tokenize.go group open/close, payload.go payloadInput).
