---
schema_version: 1
id: "iss-2609291233468970"
slug: "a-globbed-command-name-or-flag-with-a-posix-class-allows"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix7-guardGlob"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "globMatches decides only a pattern whose one bracket expression is a plain set; any other is compared with its bracket span read as a star, so r[[:lower:]] -rf /, ba[].s]h -c, git clea[[:lower:]] -fd and git clea[\\!n] -fd block or warn as their expansion does."
impact: fix
resolved_by:
  commit: "9a430d8b9680a2f8928ab93bd109778ccc606bf8"
---

The guard's glob compare (globMatches) reports no match for a bracket expression path.Match cannot read, so a globbed command name, subcommand, flag or setting spelled with one allows where bash expands it to the dangerous word: r[[:lower:]] -rf /, /bin/r[[:lower:]] -rf /, ba[[:lower:]]h -c 'rm -rf /', git clea[[:lower:]] -fd, git clea[].n] -fd and git reset --har[[:lower:]] allow, and bash 3.2, /bin/sh and bash 5.3 each expand r[[:lower:]] to rm and ba[].s]h to bash when such a file is in the working directory (the case GHSA-3w99-pgv4-8g55 closed for plain sets). The tokenizer also removes a backslash before the compare, so r[m\]] (a set holding m and ], which every shell expands to rm) reads as r[m] followed by ]. Pre-existing at the round's base 4e8cbd387; confirmed while fixing the dot-glob finding from verify-fix6-guardGlob.

## Grounds

- pursued: every globbed command name, subcommand and flag spelled with a class, a ]- or --first set, an escaped ! or ] is compared as the word bash can expand it to (TestGlobbedWordsWithBracketExpressions); a bracket spelling any shell expands to a hazard that the guard allows would show it wrong
