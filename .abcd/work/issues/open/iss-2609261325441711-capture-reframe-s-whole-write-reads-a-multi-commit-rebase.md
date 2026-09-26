---
schema_version: 1
id: "iss-2609261325441711"
slug: "capture-reframe-s-whole-write-reads-a-multi-commit-rebase"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: integ3, review2-reframe OBS-A"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reframe.go"
---

capture reframe's whole write reads a multi-commit rebase differently from a --no-ff merge or a squash of the same rewrite: two rebased commits that move the construal and then the glossary give changed=[glossary] with before at the post-construal state, where --no-ff and squash give changed=[construal glossary]. The first-differing-triple rule of spc-2609020626048705 produces it, and commands/capture.md and brief 06-capture.md claim only squash equivalence. Either a line on the page naming the rebase case or a spec ruling on what previous distinct state means across a rebased series; not decided here (review2-reframe OBS-A).
