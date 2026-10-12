---
schema_version: 1
id: "iss-2610111730551327"
slug: "guard-zsh-source-order-differs"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Apply zsh's lookup order (working directory first, then PATH) when the guard knows the line runs under zsh, and block when the shell is not known and the two orders would pick different files; prove both with Check tests over a planted pair of files"
refines: [iss-2610111730513071]
---

zsh's `source` (and `.`) of a bare name looks in the current directory before PATH, while bash and the guard's script reader search PATH first, so on a zsh line the guard can judge a different file than zsh runs when both a working-directory file and a PATH file share the name. Left open by lane G on 2026-10-11 while fixing the script-lookup gap (iss-2610111600031244).
