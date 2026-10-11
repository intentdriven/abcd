---
schema_version: 1
id: "iss-2608261437040448"
slug: "the-served-install-script-template"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: ".gitattributes"
resolution: "install.sh.tmpl pinned text eol=lf with the pipe-to-sh rationale"
impact: fix
resolved_by:
  commit: "66c06e15"
---

the served install script template misses the eol=lf pin its shell siblings carry