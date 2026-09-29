---
schema_version: 1
id: "iss-2609290929129725"
slug: "rm-rf-and-rm-rf-and-home-home-are-allowed-although-globs"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
resolution: "rm-rf-root-or-home names /*/, ~/*/, $HOME/*/, ${HOME}/*/ and ~/.*/, $HOME/.*/, ${HOME}/.*/, so a glob of every directory under the root or in the home blocks as /* and ~/* do, and the fold reaches them (/tmp/../*/). TestTrailingSlashGlobsOfTheRootOrTheHome pins them."
impact: fix
resolved_by:
  commit: "4ef5013bd"
---

rm -rf /*/ and rm -rf ~/*/ (and $HOME/*/, ${HOME}/*/, ~/.*/) are allowed, although /*/ globs every directory under the root and ~/*/ every directory in the home, so each deletes what /* or ~/* deletes less the plain files: rm-rf-root-or-home names /, /*, ~, ~/ and ~/* but no trailing-slash glob, while rm-rf-working-directory names */ and ./*/ for the working directory. Found while pinning ~/**/ for the ** reading.

## Grounds

- pursued: a recursive delete of every directory under the root or in the home blocks however the trailing slash is spelled; such a spelling that allows would show it wrong
