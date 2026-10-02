---
schema_version: 1
id: "iss-2610020704152047"
slug: "ahoy-install-s-gitignore-block-repeats-abcd-work-local-when"
severity: "nitpick"
category: "ux"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/gitignore.go"
remedy: "Before writing the fenced block, omit any entry the file already carries outside the fence with the same meaning (a git check-ignore probe of the entry, or a normalised line match), and keep the fence present even when empty so detection still finds it; test: a .gitignore already listing .abcd/.work.local/ outside the fence gets no second copy, and detection reports no gitignore drift."
---

ahoy install's .gitignore block repeats .abcd/.work.local/ when the file already lists it outside the block. In a scratch repository whose .gitignore held 'node_modules/' and '.abcd/.work.local/', a private-visibility install at tip appended a fenced block carrying '.abcd/.work.local/' again, so the file holds the same entry twice and the repository's own line is left as the redundant one. The fence writer does not consult the lines outside it.
