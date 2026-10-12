---
schema_version: 1
id: "iss-2610111730513071"
slug: "guard-misses-zsh-path-array"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Track `path` as PATH only for a zsh shell (a zsh -c payload, or a line the guard knows runs under zsh), never for bash; prove with Check tests that a zsh line setting path=( ) and sourcing a bare name judges the file in that directory, and a bash line using a variable named path is unchanged"
---

The guard does not track zsh's `path` array, which zsh ties to PATH: a zsh line that assigns `path=(<dir>)` and then runs or sources a bare script name searches <dir>, while the guard's script reader judges the file its own PATH search finds. Left open by lane G on 2026-10-11 while fixing the script-lookup gap (iss-2610111600031244), because in bash `path` is an ordinary variable name and tracking it there would misjudge bash lines.
