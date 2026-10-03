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
found_at: "internal/surface/cli/cli.go"
remedy: "Read the window width on an interactive terminal through golang.org/x/term (the dependency itd-2610030810370060 already signed off), falling back to 80 columns otherwise, and wrap every board row to it with a hanging indent through one wrap helper extended from the banner's fixed-width wrapWords (internal/surface/cli/banner.go), never a second copy; proven by a golden board test at 80 columns on a fixture with a long title and the receipts line, plus the non-TTY no-escape-bytes assertion adr-49 requires."
---

The bare status board ignores the terminal's width: in an 81-column window the receipts line and every long intent title hard-wrap back to column 0, breaking the board's hanging indent (product thinker's screenshot, 2026-10-03). The same render indents rows at 2, 4, 6 and 10 columns, prints Go's raw true/true/[development work work.local] for the first three rows, and lists the next-up intent under both Now and Next, so it reads as a duplicate.
