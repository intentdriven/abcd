---
schema_version: 1
id: "iss-2609300536292775"
slug: "capture-blocked-by-takes-only-issue-ids"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "teachingprep-autonomous-run-2026-09-23"
origin: researcher-authored
production_mode: hand-written
---

capture --blocked-by takes only issue ids: An issue blocked by a spec had to name its blocker in its body instead, and a run script told agents to add the flag only if the CLI accepts it. The flag should accept spec (and intent) ids.
