---
schema_version: 1
id: "iss-2610290730539347"
slug: "nothing-limits-who-may-make-a-private"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Make promoting a private lab or ideate verdict to public a maintainer-only act, a maintainer being an account with Write, Maintain or Admin access: the promote verb reads the caller's repository role from the forge and refuses anyone with Read or Triage before it writes, naming the forge's write permission as the boundary that holds, as the maintainer-gate intent sets out."
blocked_by: [iss-2610170717491755]
---

Nothing limits who may make a private lab or ideate verdict public: once a promote path exists, any account able to run it could move private material into the committed research store. A check inside abcd alone cannot hold, because a contributor controls their own machine and can copy the files by hand; the boundary that holds is the forge, where only an account with write access can merge. A contributor's pull request is also public the moment it is opened, before any review, so the forge keeps such material out of the project's record but cannot stop a contributor disclosing it there.
