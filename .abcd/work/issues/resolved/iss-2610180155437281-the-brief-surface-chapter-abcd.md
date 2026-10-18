---
schema_version: 1
id: "iss-2610180155437281"
slug: "the-brief-surface-chapter-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "build of the v0.15.0 deferred glossary findings (fix/deferred-glossary-flake)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Mark the PRD freeze as part of the design target and say the press-release intent is not frozen at planning, as the intent glossary entry now does."
resolution: "Said the immutability belongs to the later-phase PRD design, and that what ships freezes neither the intent nor a PRD."
impact: internal
---

The brief surface chapter .abcd/development/brief/04-surfaces/05-intent.md says both the press-release intent and the frozen PRD are immutable input artefacts post-promotion (line 489), but no shipped check freezes an intent at planning: the planning interview edits a planned intent's record (internal/surface/cli/intent_interview.go:121 admits drafts and planned), and the hold, target, edge, link, reclassify, plan and ready sub-verbs write its frontmatter, scope-condition identities or grounds (internal/core/intent/hold.go, target.go, edge.go, reclassify.go, grounds.go, lifecycle.go Plan and Link).
