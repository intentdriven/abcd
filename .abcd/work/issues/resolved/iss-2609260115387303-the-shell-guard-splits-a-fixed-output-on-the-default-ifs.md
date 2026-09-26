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
resolution: "An unquoted fixed output on a command line where another segment names IFS, at any payload layer, is refused under the reserved id ifs-split-unread, because the guard splits on the default IFS only; a prefix assignment to the carrier's own command still reads as the default split, and an IFS the shell holds before the line starts is a named residual."
impact: fix
resolved_by:
  commit: "4e64a15b"
---

The shell guard splits an unquoted fixed here-document output on the default IFS (payload.go fixedOutputSegment), whatever the line assigned IFS to before it. A line that sets IFS in a command of its own (IFS=x; or IFS=x and a newline) and then runs an unquoted cat of a quoted here-document whose body spells a hazard joined by x reaches ALLOW as one word with no operand, while bash 3.2 and 5.3 split it on x and run the hazard (review7-guard finding 2, verified with a neutral word). A prefix assignment on the same command does not take effect in bash and is read correctly.

## Grounds

- pursued: the review's IFS shapes and their export, backtick, sh -c and eval twins block while the prefix and quoted shapes allow; an IFS-reassigned split bash runs that still allows would show it wrong
