---
schema_version: 1
id: "iss-2611060603312990"
slug: "spec-close-ships-naming-an-open-issue"
severity: "major"
category: "process"
source: "user-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core"
remedy: "Run issue-drift's check inside spec close: refuse before anything moves, naming each open issue and the two ways out (move the link or resolve it)."
resolution: "abcd spec close now asks issue-drift's question before anything moves, and refuses naming each unresolved issue and the ways out; a closing build lane asks the same, not counting the issues its own receipts resolve."
impact: fix
---

abcd spec close ships an intent whose related_issues still name an open issue the intent does not deliver, and only issue-drift at preflight says so, hours later. In the v0.16.2 cut the stop intent shipped naming the open iss-2610121528179959; the link had to be moved to the planned restart intent by hand on both records and the unpushed close commit amended.
