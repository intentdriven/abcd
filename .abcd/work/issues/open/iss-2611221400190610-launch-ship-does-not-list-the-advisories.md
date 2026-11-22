---
schema_version: 1
id: "iss-2611221400190610"
slug: "launch-ship-does-not-list-the-advisories"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review of an integration branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch"
remedy: "Have launch ship, in its dry run and its report, list every advisory under embargo whose recorded fix commits the release being cut holds, by its private record and never its advisory id, as ready to publish once the release is published, naming the publish command that shows its plan; the publish itself stays the person's act."
---

launch ship does not list the advisories under embargo that the release it cuts makes publishable. Bare abcd embargo lists each advisory whose recorded fix a published release holds, but a person cutting a release with launch ship is not told which advisories the cut will make ready to publish, so publishing them after the cut is still a step to remember.
