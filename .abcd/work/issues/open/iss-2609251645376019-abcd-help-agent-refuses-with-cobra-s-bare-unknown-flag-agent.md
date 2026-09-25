---
schema_version: 1
id: "iss-2609251645376019"
slug: "abcd-help-agent-refuses-with-cobra-s-bare-unknown-flag-agent"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

abcd help --agent refuses with cobra's bare 'unknown flag: --agent' (exit 2), while abcd --agent refuses naming the spelling that works; the help-subcommand path does not name --help --agent (review-helpgroups 3).
