---
schema_version: 1
id: "iss-2608311230307436"
slug: "the-ingest-stage-marker-s-records-list-is-written-and-never"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The comment now says what the rollback does: it removes by the item-id filename grammar over the run's own directory, and the marker is evidence a person reads rather than an input the rollback consults."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest stage marker's Records list is written and never read: write() says the marker exists so a rollback removes exactly the files this ingest wrote, but rollbackRun deletes by filename grammar over the whole directory and never opens stage.json. The behaviour is currently safer than the comment, but the comment is load-bearing for the rollback's containment reasoning and is false.

## Grounds

- pursued: a comment that describes code that does not exist is worse than none, and the rollback's containment reasoning rested on it; a reader finding the comment and the code disagreeing again would show this wrong
