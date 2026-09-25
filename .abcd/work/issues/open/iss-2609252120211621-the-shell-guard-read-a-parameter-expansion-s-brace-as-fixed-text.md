---
schema_version: 1
id: "iss-2609252120211621"
slug: "the-shell-guard-read-a-parameter-expansion-s-brace-as-fixed-text"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
---

The shell guard read a parameter-expansion word that carries a command substitution (a dollar-brace default or alternative holding one) with the closing brace as fixed text after the output, so the program name it could be had to end in a brace and a dash-word only a flag that did: a blocked command whose name or flag was the substitution inside such a default allowed (review4-guard finding 3). A plain variable with no substitution in it is the half iss-2609251824244354 defers.
