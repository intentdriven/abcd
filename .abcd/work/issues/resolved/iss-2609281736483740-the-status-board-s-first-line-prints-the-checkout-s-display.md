---
schema_version: 1
id: "iss-2609281736483740"
slug: "the-status-board-s-first-line-prints-the-checkout-s-display"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainRedact"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "The board's text first line goes through termsafe.Sanitize, so an ESC sequence or a bidi control in the checkout's directory name is masked; --json dir keeps the true name, which encoding/json escapes where it is a control byte. TestBoardFirstLineMasksControlsInTheCheckoutName was watched fail with both names printed raw, and passes after."
impact: fix
resolved_by:
  commit: "7004cc395"
---

The status board's first line prints the checkout's display name without termsafe.Sanitize, while every other board value is sanitised. Bare abcd writes 'abcd — <dir>' from fsutil.DisplayPath(st.Dir) straight to the terminal, so a checkout whose directory name carries an ESC sequence or a bidi control (U+202E) reaches the terminal raw: a crafted clone directory can recolour, retitle or reorder the board a person reads and pastes. Named FOR THE REVIEWER by fix2-drainRedact after it routed the line through DisplayPath.

## Grounds

- pursued: a checkout directory named with ESC or U+202E prints masked on the board's first line; the raw control reaching the text output, or --json dir no longer carrying the true name, would show it wrong.
