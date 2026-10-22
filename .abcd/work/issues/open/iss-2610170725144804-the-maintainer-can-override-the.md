---
schema_version: 1
id: "iss-2610170725144804"
slug: "the-maintainer-can-override-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "conductor-intent session, reserve refused at the cap on 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Let the maintainer, and only the maintainer, change the live-reservation cap (default ten, a higher number, or no limit): the value lives where an agent session cannot write it, preferably a setting on the forge that only a repository admin can change, which abcd reads, or failing that a person-only verb the shell guard refuses from an agent as it refuses the ledger flip; a plain config file an agent can edit is not enough, the refusal names the setting and who may change it, and abcd peers shows the current cap and where it came from."
---

The live-reservation cap is fixed at ten per person (claim.MaxLive in internal/core/claim/crowd.go), counted by username across every session, laptop and cloud copy. A project with one maintainer running many parallel sessions under one GitHub identity hits it with no one else to protect: on 2026-10-17 all ten were held by the maintainer's own sessions, mostly finished work waiting to merge, and a new reserve was refused. The cap is right for abcd once several people share a project, but the maintainer cannot raise or lift it today.

## Requirement from the product thinker

Restated from the product thinker on 2026-10-17: the cap must be overridable by the maintainer alone. No session or agent may raise or lift it for itself. A key in `.abcd/config.json` or `~/.abcd.noindex/config.json` does not meet this, because an agent working in the same account can edit either file.

Candidate mechanisms, to settle at planning with a state-of-the-art pass:
- A setting on the forge that only a repository admin can change, such as a repository variable or ruleset, which abcd reads when it judges the cap.
- A person-only verb that the shell guard refuses when an agent runs it, as it refuses the source-ledger flip. This verb alone does not stop an agent editing whatever file the verb writes.

## Preference and a finding (2026-10-17)

The product thinker prefers the forge setting, provided abcd can set it up for a repository without manual steps. If it needs manual steps, the person-only verb is the fallback, which they like less.

Finding, checked the same day: every session on this machine uses the maintainer's own `gh` login. That login holds admin rights on the repository (the `repo` scope). Any session can therefore change a repository variable as easily as the maintainer, so a forge setting keeps agents out only when they run under a login without admin rights, such as a fine-grained token or a separate account. Planning has to settle which credential the sessions use, or pair the forge setting with a guard that refuses an agent's write to it. Without one of those, neither option meets the requirement on this machine.
