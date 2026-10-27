---
schema_version: 1
id: "iss-2610230851520489"
slug: "the-stop-hook-also-runs-while-the-person"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/mode.md"
remedy: "Add one sentence to commands/mode.md's away-mode section saying what the stop check does when the person is present."
---

The Stop hook also runs while the person is present: a turn ending with work owed is allowed with one line naming the next item; commands/mode.md describes it only as the away mode's check.
