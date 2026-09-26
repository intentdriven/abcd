---
schema_version: 1
id: "iss-2609261013469387"
slug: "ahoy-s-hookspatharmed-reads-core-hookspath-raw-with-git"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix-lab sibling sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
resolution: "hooksPathArmed reads core.hooksPath with --type=path, so a ~/ spelling of the committed .githooks reads armed (TestHooksPathArmedResolvesBothSides)."
impact: fix
resolved_by:
  commit: "e621cd6f"
---

ahoy's hooksPathArmed reads core.hooksPath raw with git config --local --get and joins a non-absolute value to the clone, so a ~/ spelling of the committed .githooks directory (which git expands to the same directory) reads as unarmed, and the status line advises re-arming a clone whose name guard already runs. Sibling of iss-2609261004261615: judge the path git expands (--type=path).

## Grounds

- pursued: a clone armed through a ~/ spelling of its own .githooks is reported armed; a git whose --type=path expansion differs from the directory it runs hooks from would show it wrong
