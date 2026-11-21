---
schema_version: 1
id: "iss-2611202034242491"
slug: "the-installer-s-tree-check-tools"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run of 20 Nov 2026, review of itd-2611041815429986 step 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
remedy: "Hold each link step's real location (the step's directory resolved, joined with its base name) to the tree, as the needs judge does since itd-2611041815429986 step 1, with a test of a repository link reached through an outside directory link for each."
resolution: "each link step is now held to the tree at its real location in tools.WithinTree and gitleaks admitBinary"
impact: fix
resolved_by:
  intent: "itd-2611041815429986"
---

The installer's tree check (tools.WithinTree) and the secret scanner's binary admission (gitleaks admitBinary) judge a program by its written path and final target only, never by each step's real location, so a PATH directory that links into a repository can hand them a program the repository chose under a trusted name.
