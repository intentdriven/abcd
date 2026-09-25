---
schema_version: 1
id: "iss-2609252214217550"
slug: "the-shell-guard-blocks-valid-bash-as-command-unparsable-when"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard blocks valid bash as command-unparsable when a double-quoted ${…} carries double quotes of its own. Inside double quotes bash parses a parameter expansion to its own closing brace: a double quote in it opens a nested string instead of closing the outer one, and a single quote pairs with the next. The tokenizer's double-quote branch and closingDoubleQuote (internal/core/guard/tokenize.go) ended the string at the first nested quote, so an apostrophe inside the nested quotes read as an unterminated single quote: echo "${MSG:-"don't"}", "${1:-"it's"}", printf '%s\n' "${NAME:-"O'Brien"}" and "${X//"'"/x}" returned ErrUnparsableCommand, which the pre-tool-use hook (internal/surface/cli/guard.go) now blocks as command-unparsable, where the round before they ran (review5-guard finding 2).
