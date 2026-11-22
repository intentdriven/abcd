---
schema_version: 1
id: "iss-2611220942278236"
slug: "nothing-links-a-cut-release-to-the"
severity: "major"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/plan.go"
remedy: "Record the fixing commit on each embargoed record (an embargo verb that takes the record and the commit), and have bare abcd embargo and launch ship list every record whose fixing commit is in a published release tag as ready to publish, queueing the person's publish step in away mode; the publish itself stays the person's act."
---

Nothing links a cut release to the embargoed records it fixes. After the fixes merged, publishing or closing each embargoed record once the release that carries its fix is out is a step a person has to remember, look up record by record and match to the release by hand.
