---
schema_version: 1
id: "iss-2609251640464212"
slug: "the-no-verify-blockers-miss-the-same-bypass-spelled-as"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/gitconfig.go"
---

The no-verify blockers miss the same bypass spelled as configuration: a commit or push that points core.hooksPath elsewhere for that one command, through -c, --config-env or the GIT_CONFIG environment, skips the repository hooks exactly as the no-verify flag does, and the matcher steps the -c value over unread. Found by review-guard finding 6 (pre-existing).
