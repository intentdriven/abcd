---
schema_version: 1
id: "iss-2610151422107714"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite .abcd/development/brief/02-constraints/04-naming.md:222 to match the shipped surface: The per-repo meta.json that ahoy install writes (internal/core/ahoy/apply.go) holds root_commit, name, github and corpus only; no code in internal/core/ahoy reads or writes an `aliases` field, and the index.json entry struct (internal/core/ahoy/store.go) has…"
---

The brief at .abcd/development/brief/02-constraints/04-naming.md:222 says: `aliases`: array of prior names a repo has had, 'Recorded in per-root-sha `meta.json`' (unmarked, so a delivery claim per the truth rule). The shipped surface differs: The per-repo meta.json that ahoy install writes (internal/core/ahoy/apply.go) holds root_commit, name, github and corpus only; no code in internal/core/ahoy reads or writes an `aliases` field, and the index.json entry struct (internal/core/ahoy/store.go) has none either. (v0.14.2 release crosscheck, false-claim, raised by opus in a32.)
