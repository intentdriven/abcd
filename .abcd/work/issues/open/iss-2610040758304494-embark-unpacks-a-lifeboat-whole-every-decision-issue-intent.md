---
schema_version: 1
id: "iss-2610040758304494"
slug: "embark-unpacks-a-lifeboat-whole-every-decision-issue-intent"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
remedy: "Add a selection to the unpack (by record id, by family, or from a list file) that writes only the chosen records, reports each reference the selection leaves dangling, and keeps the whole-write refusal for the chosen set; test: an embark of a fixture lifeboat with a two-record selection writes exactly those two and names each reference to an unselected record."
---

embark unpacks a lifeboat whole: every decision, issue, intent, spec and retrospective it carries is written, and any conflict refuses the entire write (brief 04-surfaces/03-embark.md). A downstream user starting a new project from its predecessor's lifeboat wanted only selected records carried across and had no way to choose them, so embark could not serve that use. The whole write is the designed behaviour, so this is a capability gap rather than a defect, and intent-shaped: routing it to an intent is the product thinker's call.
