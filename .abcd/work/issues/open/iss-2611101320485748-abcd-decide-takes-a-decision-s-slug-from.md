---
schema_version: 1
id: "iss-2611101320485748"
slug: "abcd-decide-takes-a-decision-s-slug-from"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/decide/decide.go"
remedy: "Give decide a --slug flag as capture has, and leave a leading role or date phrase out when deriving the slug; a test decides a text that opens with a role and finds the slug drawn from the subject."
---

abcd decide takes a decision's slug from the first words of its text and has no way to set it, so a text that opens with who decided gets a slug naming the role rather than the subject. On 9 Nov at 02:50 a decision whose text opened with the role got the slug technical-facilitator.
