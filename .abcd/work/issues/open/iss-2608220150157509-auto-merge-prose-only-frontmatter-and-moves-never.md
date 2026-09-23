---
schema_version: 1
id: "iss-2608220150157509"
slug: "auto-merge-prose-only-frontmatter-and-moves-never"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcdev-site decision interview 2026-08-22"
found_at: ".github (auto-merge convention)"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: mechanical CI rung refusing auto-merge arming on frontmatter or record-path diffs (protocol applied by eye today)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Auto-merge for docs PRs is armable only when the diff is prose-only, defined precisely: no YAML frontmatter changes AND no file adds, moves, or deletes under the record stores — because record state lives in both frontmatter (status flips) and directory position (lifecycle bucket moves). A state change is a gate crossing (adversarial-review-scales-with-blast-radius) and always waits for the human; prose fixes are below the review floor. Adopted as documented protocol now (applied by eye when arming); the mechanical rung — a CI check that refuses auto-merge arming when the diff touches frontmatter or record paths — is a later seed per script-first-mvp