---
schema_version: 1
id: "iss-2610160608444348"
slug: "the-drain-sentence-at-internal-core"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Change the drain sentence to 'Writes its state, user-visible drafts and each lane's reservation on origin; refuses without the rule's record.' and regenerate the pages."
---

The drain sentence at internal/core/surface/sentences.go:130, 'Writes its state and user-visible drafts', shown by abcd drain --help, commands/drain.md:3 and docs/reference/cli/commands.md, omits a remote write: in a checkout with origin, drain reserves each issue it takes by pushing abcd-reserved/<iss-N> to origin (commands/drain.md:146-162), and the 81-line help never mentions origin or a reservation. build's sentence already says 'Writes the run's state and its reservation'.
