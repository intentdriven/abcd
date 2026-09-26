---
schema_version: 1
id: "iss-2609260115387303"
slug: "the-shell-guard-splits-a-fixed-output-on-the-default-ifs"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
---

The shell guard splits an unquoted fixed here-document output on the default IFS (payload.go fixedOutputSegment), whatever the line assigned IFS to before it. A line that sets IFS in a command of its own (IFS=x; or IFS=x and a newline) and then runs an unquoted cat of a quoted here-document whose body spells a hazard joined by x reaches ALLOW as one word with no operand, while bash 3.2 and 5.3 split it on x and run the hazard (review7-guard finding 2, verified with a neutral word). A prefix assignment on the same command does not take effect in bash and is read correctly.
