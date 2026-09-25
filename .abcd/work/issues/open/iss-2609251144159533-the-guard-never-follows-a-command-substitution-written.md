---
schema_version: 1
id: "iss-2609251144159533"
slug: "the-guard-never-follows-a-command-substitution-written"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The guard never follows a command substitution written inside double quotes: echo "$(gh repo delete owner/repo)" and x="$(gh repo delete owner/repo)" both answer allow, because the double-quote branch of the tokenizer consumes the whole string as one literal word, while the unquoted and backtick forms are followed into command position. Double-quoting a substitution is the idiomatic shell spelling, so this is the common form, not an evasion. The check verb's help text listed it garbled, as 'a hazard inside a top-level command substitution' with a parenthesis saying both forms ARE followed. Wanted: a substitution inside double quotes is read as its own command, as the unquoted form is.
