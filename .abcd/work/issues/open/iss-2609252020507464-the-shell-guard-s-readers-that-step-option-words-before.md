---
schema_version: 1
id: "iss-2609252020507464"
slug: "the-shell-guard-s-readers-that-step-option-words-before"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

The shell guard's readers that step option words before command position, or read a shell's -c, took an unknown dash-word (a dash glued to a command substitution) as one fixed thing: never a value flag, never -c. The wrapper walk (sudo, env, nice, timeout, exec, doas, stdbuf, xargs), the operand walk that finds a subcommand (git, gh), the exec-string scan (su -c) and the shell -c reader each read it so, and a hazard behind such a word allowed or only warned, which runs it. Found by review3-guard finding 2.
