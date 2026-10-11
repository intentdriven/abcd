---
schema_version: 1
id: "iss-2608211432389791"
slug: "ahoy-md-install-mode-omits-shadowed"
severity: "nitpick"
category: "documentation"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "commands/ahoy.md"
resolution: "Document the (shadowed on PATH) install_mode suffix in commands/ahoy.md"
impact: fix
resolved_by:
  commit: "1c3ae21"
---

commands/ahoy.md enumerates install_mode as three values but detectInstallMode also appends a (shadowed on PATH) suffix, so the documented value list is incomplete