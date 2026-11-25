---
schema_version: 1
id: "iss-2611251855001172"
slug: "promote-doubles-the-full-stop-in-the"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
remedy: "Trim the closing full stop from the title line before the seed sentence adds its own, with a test on a title that ends in one."
---

capture promote doubles the full stop in the draft it seeds when the first line of the issue ends in one: the seed reads `Graduated from ... filed.. Read that issue record`. Three of three drafts promoted on 2026-11-25 carried it, and drafts of September and November already in the store do too; the code is the same on main.
