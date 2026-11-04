---
schema_version: 1
id: "iss-2611040337241546"
slug: "the-bash-tool-runs-zsh-and-abcd-s-shell"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Teach the safe forms in the SHELL rules: braces around a variable before a colon, rc=$? with no pipe, and a non-interactive removal; the guard's teaching half names the shell it read."
---

The Bash tool runs zsh, and abcd's shell guidance does not say so: a variable before a colon is mangled, PIPESTATUS is empty so a tail's exit is read as the command's, and an interactive rm alias hung a call for two minutes, all on 2026-11-04.
