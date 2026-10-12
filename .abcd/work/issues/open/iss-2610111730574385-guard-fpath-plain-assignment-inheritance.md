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
---

An FPATH already exported in the hook's environment and then changed on the line by a plain assignment (no export) is not handed to a child zsh, so the guard's autoload reader, which now tracks FPATH set on the line, can judge the function file from the line's value while the child shell searches the inherited one. Left open by lane G on 2026-10-11 while fixing the FPATH and autoload gap (iss-2610111600022820).
