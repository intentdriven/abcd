---
schema_version: 1
id: "iss-2610031207397996"
slug: "the-bare-status-board-ignores-the-terminal-s-width-in-an"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "board readability review with the product thinker, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/board_status.go, internal/surface/cli/board_reviews.go, internal/surface/cli/cli.go"
remedy: "Read the window width on an interactive terminal through golang.org/x/term (the dependency itd-2610030810370060 already signed off), falling back to 80 columns otherwise, and wrap every board row to it with a hanging indent through the one wrap primitive, internal/textwidth.Wrap (the banner's wrapWords is a one-line alias of it), never a second copy; print the first three rows as words, not Go's raw values; indent rows evenly; and list the next-up intent once, editing the brief chapter 04-surfaces/08-abcd.md, which states the duplicate as designed, and the status-block test in the same change; proven by a golden board test at 80 columns on a fixture with a long title and the receipts line, plus the non-TTY no-escape-bytes assertion adr-49 requires."
related_intents: [itd-2610031214560142]
---

The bare status board ignores the terminal's width: in an 81-column window the receipts line and every long intent title hard-wrap back to column 0, breaking the board's hanging indent (product thinker's screenshot, 2026-10-03). The same render indents rows at 2, 4, 6 and 10 columns, prints Go's raw true/true/[development work work.local] for the first three rows, and lists the next-up intent under both Now and Next, so it reads as a duplicate.

## Note (2026-10-03)

Linked to itd-2610031214560142, the product thinker's board, which is drawn through the same wrap helper and is built after this fix. The duplicated next-up row is owned here, not by the board draft (the technical facilitator's ruling, that draft's decision 9), so the remedy names the brief-chapter edit it needs. The remedy now names internal/textwidth.Wrap as the primitive, and found_at names the renderers that wrap, from the record-discipline review of 2026-10-03 (reports/review-board-records.md in the local tier).
