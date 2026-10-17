---
schema_version: 1
id: "iss-2610170926360632"
slug: "after-every-fallback-to-the-tracking"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "review of PR #1068"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/hand-back.sh"
remedy: "Say the report could not go with the work, and give the reason, in the closing line, rather than naming a push"
---

After every fallback to the tracking issue, the cloud hand-back's closing line says the push failed, so the report was posted as a comment, even where no push was tried: when no person's identity is set, when the report's path is a symlink, when the commit failed, when the committed hooks are not active, and when an unchanged report's commit carries another Assisted-by. The stderr line before it gives the real reason, but the closing line names the wrong one, and so does the heading the posted comment gives that reason, "Why the push failed".
