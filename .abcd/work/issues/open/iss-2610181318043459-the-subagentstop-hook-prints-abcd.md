---
schema_version: 1
id: "iss-2610181318043459"
slug: "the-subagentstop-hook-prints-abcd"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "live capture of spc-2610180914469489, 2026-10-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/history.go"
remedy: "Treat a transcript path that does not exist as nothing to capture and stay silent, keeping the error for a transcript that exists but cannot be read"
---

The SubagentStop hook prints 'abcd history: cannot read transcript' when the host keeps no transcript file, as in a session started with --no-session-persistence: the history capture treats a transcript path that does not exist as an error and reports it on every subagent stop, although there is nothing to capture. Seen in the live capture of the overnight-stall fix on 2026-10-18.
