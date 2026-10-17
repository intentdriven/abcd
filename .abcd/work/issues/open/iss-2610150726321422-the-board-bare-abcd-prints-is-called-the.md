---
schema_version: 1
id: "iss-2610150726321422"
slug: "the-board-bare-abcd-prints-is-called-the"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "planning interview of itd-2609151838312703, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/abcd.md"
remedy: "Rename the status board to the dashboard everywhere it is named (README, docs, command pages, the brief and the glossary), so bare abcd renders the dashboard in the terminal and abcd dashboard serves the same dashboard to a browser, and run abcd lint docs and the site render to prove no page still says status board."
related_intents: [itd-2609151838312703]
---

The board bare abcd prints is called the status board, but it is the dashboard in the terminal: abcd dashboard already serves the same view to a browser over Tailscale, and the board is moving to that HTML form. Two names for one thing, and the word status is now wanted for abcd status, the people's view of what other sessions announce and hold (itd-2609151838312703 decision 21). The product thinker ruled on 2026-10-15 that the status board should be renamed the dashboard. About 14 files say status board: README.md, commands/abcd.md, docs/how-to/install.md, docs/reference/cli/commands.md and the brief.
