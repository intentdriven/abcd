---
schema_version: 1
id: "iss-2609100531063912"
slug: "positive-result-a-full-adversarial-review-loop-ran-end-to-en"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core"
---

Positive result: a full adversarial review loop ran end to end inside abcd's record shapes without the tooling getting in the way. Recorded because a stress that finds nothing is evidence about the tool, and it is only legible if someone writes it down. An autonomous run filed, planned and implemented one intent without the human interview, then put it through two adversarial reviews. The reviews produced eleven findings about the repository under review and none about abcd. Applying those findings needed no change to any record shape: a scope condition was reworded under its existing stamp rather than being reissued, one acceptance criterion was added, and the readiness gate stayed green across the review-driven edits. Closing the spec moved both the spec and its intent in one step with correct paths. The parts of abcd exercised hardest here, the shapes that carry a claim and the verbs that move a record between states, held under a workload they were not specifically designed for, namely a review loop with no human in it. This is the second positive finding from the same experiment, alongside the observation that every record verb worked from worktrees across 27 branches with no merge conflict in the ledger. Both bound the negative findings: what failed was consistently at the edges of the tool rather than in its centre.
