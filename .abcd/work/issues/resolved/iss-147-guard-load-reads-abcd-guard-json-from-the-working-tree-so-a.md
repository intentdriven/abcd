---
schema_version: 1
id: "iss-147"
slug: "guard-load-reads-abcd-guard-json-from-the-working-tree-so-a"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Load refuses a working-tree guard.json edit that switches the guard off or changes a blocker's tier or pattern unless HEAD carries it (ErrUncommittedOverride); the committed registry stays in force, the hook says so, the check exits 2. Where git cannot confirm HEAD, the edit is refused."
impact: fix
resolved_by:
  commit: "e5dcc48acc5f8ebaaa54ba32d0a67ff6f874247d"
---

guard.Load reads .abcd/guard.json from the WORKING TREE, so a disabled:true (or a retiered blocker) takes effect on the very next command, before anyone reviews it — spc-16 and the shipped docs both say the only escape is a committed, reviewable override, and nothing enforces the committed half. Reachable in one move: an agent writes .abcd/guard.json and the guard itself allows that write. Mitigated at the front door in itd-103 wiring (a disabled registry now warns UNGUARDED on every command, and abcd ahoy reports OFF) but not enforced. Proper fix is core-side: refuse a disabled:true that is not in HEAD, or drop the committed claim from spc-16. Found by the security reviewer on the itd-103 wiring branch.

## Grounds

- pursued: weakening the guard takes a committed edit; shown wrong by an uncommitted disable or retier taking effect (TestUncommittedWeakeningIsRefused)
