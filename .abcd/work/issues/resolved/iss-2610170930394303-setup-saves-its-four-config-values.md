---
schema_version: 1
id: "iss-2610170930394303"
slug: "setup-saves-its-four-config-values"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "after-update bundle step 7, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "Write each answered config value on its own and leave only the deferred ones unset, with a test where one value is deferred and another answered that fails today; keep the write atomic per file."
refines: [iss-2610031236155833]
resolution: "Setup now saves each config value it is given on its own: a value deferred with decide later, or given an answer setup does not write, stays unset and is listed as remaining, while every answered value goes into the one atomic write of .abcd/config.json, in a full ahoy install and in ahoy install --only-waiting; the answers-file walk asks on past a deferred value as the install now does."
impact: fix
---

Setup saves its four config values (visibility, docs_target, oracle_backend, scan_deep) all together or not at all, so when one of them was deferred with 'decide later' and another is answered, neither is written: the answer reaches the setup answers record but never .abcd/config.json. A full ahoy install does this today, and the new ahoy install --only-waiting inherits it, so an answer a person gives after an update can silently not take effect.

## Grounds

- pursued: an answer given beside a deferred value reaches .abcd/config.json in both install modes; a deferred value appearing in the file, or an answered one missing from it, would show it wrong
