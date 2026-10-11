---
schema_version: 1
id: "iss-2608221456593173"
slug: "codeowners-omits-site-src-including"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: ".github/CODEOWNERS"
resolution: "CODEOWNERS now covers site-src/ and docs/requirements.txt."
impact: internal
resolved_by:
  commit: "8b1e227"
---

CODEOWNERS omits site-src/ (including install.sh.tmpl, the byte source of the served /install.sh) and docs/requirements.txt, so a PR confined to those merges with zero required code-owner review while shipping the install script to users.