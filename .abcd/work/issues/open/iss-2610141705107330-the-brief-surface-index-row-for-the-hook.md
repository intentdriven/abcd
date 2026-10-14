---
schema_version: 1
id: "iss-2610141705107330"
slug: "the-brief-surface-index-row-for-the-hook"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Make the hook row say six entrypoints and name question-answered with the event that runs it."
---

The brief surface index row for the hook verb says five host hook entrypoints and names prompt-router, prompt-router-reset, session-end, subagent-stop and session-start, but abcd hook --help lists six: question-answered (PostToolUse on the question tool, wired through hooks/post-tool-use.sh) is neither counted nor named; it is documented only in 05-internals/03-configuration.md:1118-1119. Raised by three checkers (CLI verb tree Opus, hook entrypoints Opus and Sonnet).
