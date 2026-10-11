---
schema_version: 1
id: "iss-2610031215428557"
slug: "investigate-whether-the-question-chip"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "planning interview, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/badge.go"
remedy: "On abcd's own Terminal drawing, colour the chip with the same fixed role pair the status-line badge uses (one source, internal/livery), with the word and inverse video carrying the meaning and NO_COLOR, TERM=dumb and a piped stream drawing none (itd-200, adr-49); on the host's question tool, leave the chip plain and rely on the status-line badge, re-checking only if the host gains a colour field for its header."
---

Investigate whether the question chip ("Product Q1", "Tech Q2") can carry abcd's fixed role colours, the ones the status-line badge already uses for whom a question is for (product thinker amber #f0c052, technical facilitator light grey #e8e8e8, each with dark text; internal/core/statusline/badge.go rolePairs, drawn from internal/livery). The product thinker asked on 2026-10-03, pointing at a screenshot of Claude Code's question tool, where the chip is drawn by the host in its own highlight and abcd supplies only the header text. Findings so far: on the host path abcd cannot colour the chip, because the question tool renders the header as plain text, an escape sequence in it would show literally or be stripped, and brief invariant 13 plus the question check refuse escapes in that text; the role colour already shows on the host path in the status-line badge while the mode names the addressee. On abcd's own plain-Terminal drawing (itd-2610030810370060, internal/surface/cli/ask), abcd draws the chip itself and can colour it.
