---
schema_version: 1
id: "iss-2611060835493836"
slug: "no-test-or-lab-run-takes-a-stop-across"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runstop/runstop.go"
remedy: "Have the live proof cover it, or add a manual step on a machine with two accounts."
decision_needed: "The remedy leaves open which of two routes to take and I could not tell whether it stays test-only: should the live proof itself take a stop across two real accounts (and if so, how may it obtain and act as a second real account, and with what privileges), or should this instead be a manual step a person runs on a two-account machine?"
---

No test or lab run takes a stop across two real accounts: the tests stand the second account in through the owner the judgement is handed (spc-2611041705580524). A fault only a real account boundary shows would first appear on a person's machine.
