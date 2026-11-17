---
schema_version: 1
id: "iss-2611170028005924"
slug: "a-commit-whose-resolves-refs-or-delivers"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "code review of the consent-flag worked-example test"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landings/admitted.go"
remedy: "have the commit-msg hook refuse a record line (Resolves, Refs, Delivers) outside the final trailer paragraph, naming the fix (move it next to Assisted-by), and have the landings reader take every trailer-shaped line of the trailing record lines rather than only the last paragraph, so history already written reads as meant"
---

A commit whose Resolves, Refs or Delivers line sits in a paragraph before the one holding Assisted-by carries no such trailer to git, and the landings reader, which takes only the message's last paragraph, records no resolves or delivers for it; 68 of the last 300 commits on main are written that way
