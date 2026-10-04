---
schema_version: 1
id: "iss-2610040758394205"
slug: "the-bundled-intents-rule-tells-every-managed-repository"
severity: "minor"
category: "inconsistency"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Replace the fixed-three rule with an open alphabetical sequence a repository extends, and have it point at a repository-declared roster for names and roles where one exists; test: no bundled default rule caps the persona names, and the rule names the repository's roster when one is declared."
---

The bundled INTENTS rule tells every managed repository 'Personas are always Alice, Bob, Carol — never other names.' (internal/core/rules/defaults/rules.json), a fixed roster of three that a repository can change only by overriding the whole INTENTS rules field in .abcd/rules.json. abcd's own repository does exactly that to point at its own persona registry (iss-2608211849583208), so its sessions never meet the cap. A downstream owner wanted a fourth persona and the rule forbade it; the lab proposed an open naming sequence (Alice, Bob, Carol, Dave, and on) with each persona's role declared by the repository.
