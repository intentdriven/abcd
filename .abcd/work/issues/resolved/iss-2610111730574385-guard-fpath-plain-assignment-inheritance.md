---
schema_version: 1
id: "iss-2610111730574385"
slug: "guard-fpath-plain-assignment-inheritance"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Model export state for FPATH as the reader already does for PATH: a plain assignment changes the value only for the current shell, and a child zsh sees the inherited exported value; block when the two would differ; prove with Check tests"
resolution: "Verified against bash 3.2, bash 5 and zsh 5.9: with FPATH exported in the environment a plain assignment IS handed to a child shell, so the gap ran the other way from this record's text: the guard handed a child only what the line exported and read the child as loading the account's own functions, unseen. Every tracked variable the line sets is now read as handed down unless the line takes the export away (export -n, typeset +x); where it is not handed down the child reads the account's own, unseen, so the handed reading covers both. Proved by TestPlainAssignmentReachesAChildShell, watched failing first."
impact: fix
resolved_by:
  commit: "db0310cda"
---

An FPATH already exported in the hook's environment and then changed on the line by a plain assignment (no export) is not handed to a child zsh, so the guard's autoload reader, which now tracks FPATH set on the line, can judge the function file from the line's value while the child shell searches the inherited one. Left open by lane G on 2026-10-11 while fixing the FPATH and autoload gap (iss-2610111600022820).
