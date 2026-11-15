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
resolution: "The implement chapter's load section now says the check without its wait flag needs its site flag, names the two values preflight and eval-harness, and says it exits 2 without one. The flag is named in words because the chapter's prose spells no flag (itd-147 ac-5); the appendix carries the spelling --site."
impact: internal
resolved_by:
  commit: "ab8143531572d4fc268e6c1b553c9119667663ff"
---

.abcd/development/brief/04-surfaces/27-implement.md:265-266 says that without its wait flag the check exits 0 on every status and never refuses. That holds for statuses, but `abcd implement load` with no flag exits 2, naming --site's values preflight and eval-harness. The chapter mentions `--site` at lines 139 and 163 and in its appendix at line 720, without saying it is needed or what it takes; commands/implement.md:915 gives both.

## Grounds

- pursued: a reader of the load section learns that a bare `abcd implement load` exits 2 and which two site values it takes; a run of `abcd implement load` with one of the two named values that exits 2, or a bare run that exits 0, would show it wrong.
