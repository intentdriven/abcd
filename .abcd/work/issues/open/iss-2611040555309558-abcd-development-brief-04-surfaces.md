---
schema_version: 1
id: "iss-2611040555309558"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md"
remedy: "State in the load section that the check needs --site and name its two values."
---

.abcd/development/brief/04-surfaces/27-implement.md:265-266 says that without its wait flag the check exits 0 on every status and never refuses. That holds for statuses, but `abcd implement load` with no flag exits 2, naming --site's values preflight and eval-harness. The chapter mentions `--site` at lines 139 and 163 and in its appendix at line 720, without saying it is needed or what it takes; commands/implement.md:915 gives both.
