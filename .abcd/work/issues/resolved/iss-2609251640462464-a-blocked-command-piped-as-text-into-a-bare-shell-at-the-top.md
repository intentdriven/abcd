---
schema_version: 1
id: "iss-2609251640462464"
slug: "a-blocked-command-piped-as-text-into-a-bare-shell-at-the-top"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
resolution: "A shell reading its script from a pipe, a here-document or a here-string is refused under the reserved id interpreter-reads-stream; the tokenizer now records which commands read a stream (segment.stdinStream), at the top level and inside payloads."
impact: fix
resolved_by:
  commit: "7e523ba9"
---

A blocked command piped as text into a bare shell at the top level is allowed: echo or printf of the command string piped into sh, or into bash -s, runs it, but pipesIntoInterpreter is consulted only inside execute-a-string payloads, and the top-level segments carry no record of which operator joined them, so the guard cannot tell a shell reading the pipe from one running a script file without tracking pipes. Found by review-guard finding 6 (pre-existing).

## Grounds

- pursued: every pipe, here-document or here-string into a bare sh/bash/zsh (with -s, -, or no script operand) blocks, and a shell given a script file or a -c string does not (TestInterpreterReadingAStreamBlocks, TestShellFamilyIsSharedNotRelisted); a stream-fed shell that allows, or a script-file run that blocks, would show it wrong
