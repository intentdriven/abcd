---
schema_version: 1
id: "iss-2610290855523061"
slug: "tracked-files-name-the-organisation-s"
severity: "major"
category: "process"
source: "user-observation"
found_during: "product thinker ruling relayed by the coordinator session, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "ACKNOWLEDGEMENTS.md"
remedy: "Replace every member login and name in tracked files with a role, the organisation or a standard persona; delete the code-owners file, the mailmap and the leftover identity pin, which carry nothing else; keep the LICENSE copyright line by the product thinker's decision; history and commit author fields untouched."
resolution: "Tracked files no longer name any organisation member: the code-owners file, mailmap and identity pin are removed, the citation names the organisation, scanner and test fixtures use personas, and four dated review files are redacted under a narrow append-only exception."
impact: fix
resolved_by:
  commit: "795949ce5"
---

Tracked files name the organisation's members and collaborators: about 63 files carry a member's GitHub login or display name, in credits, the plugin manifests, the code-owners file, the mailmap, the committed identity pin, test fixtures, attribution cases, research notes and records. The product thinker ruled on 2026-10-29 that a login appears only in commit metadata (the author and committer of a pushed commit), never in the content of prose or code, and that people are named by role. The one exception the product thinker kept is the copyright line in LICENSE, which still names its holder.

## Grounds

- pursued: we expect a scan of every tracked file outside the licence to find no member's login or name; any hit outside commit metadata shows it wrong
