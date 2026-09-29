---
schema_version: 1
id: "iss-2609290745243990"
slug: "rm-rf-root-or-home-reads-a-parent-segment-only-at-the-root"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

rm-rf-root-or-home reads a parent segment only at the root: rm -rf ~/../* deletes the home, and rm -rf ~/../../*, $HOME/../../*, /tmp/../* and /etc/../* delete the root, bare and in sh -c, yet each allows, because cleanSeparators takes out only a leading /../ and never folds a .. after a named directory, the home or a tilde. A root or home delete is allowed.
