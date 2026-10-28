---
schema_version: 1
id: "iss-2610252211549081"
slug: "readme-md-130-says-abcd-view-facilitator"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D8"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Replace README.md:130 with: 'abcd --view facilitator shows the full board: the repository, the record, the work tiers, the presence line, and the intents being built or ready to build, each with its spec, then a count of the rest.'"
---

README.md:130 says: 'abcd --view facilitator shows the full board: the repository, the record, the work tiers, the presence line and every intent with its spec.' The full board lists only the Now and Next rows, each with its spec id (internal/core/board/board.go:295, statusRow at :348). The Later group, every planned intent the readiness gate reports not READY plus every draft (internal/core/statusblock/statusblock.go:20-22), is drawn as one count line, 'Later: N intents' (board.go:308). Shipped intents are not on the board at all. A reader looking for a not-ready or draft intent on the full board will not find it there.
