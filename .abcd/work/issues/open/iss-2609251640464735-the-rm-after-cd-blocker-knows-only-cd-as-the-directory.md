---
schema_version: 1
id: "iss-2609251640464735"
slug: "the-rm-after-cd-blocker-knows-only-cd-as-the-directory"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

The rm-after-cd blocker knows only cd as the directory change a delete is chained after, so a recursive forced delete chained after pushd or popd is allowed, though either fails the way cd does and the delete then runs wherever the shell already was. Found by review-guard finding 6 (pre-existing).
