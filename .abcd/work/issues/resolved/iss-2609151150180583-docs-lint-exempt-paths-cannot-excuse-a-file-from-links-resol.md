---
schema_version: 1
id: "iss-2609151150180583"
slug: "docs-lint-exempt-paths-cannot-excuse-a-file-from-links-resol"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "peer session report 2026-09-15 (a teaching-repo session planning five intents)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "links_resolve reads its own exempt globs, so a tool-mandated mirror file is excused from the link check alone; documented on the docs surface and plugin page."
impact: fix
resolved_by:
  commit: "b19c5374"
---

docs lint: exempt_paths cannot excuse a file from links_resolve. contentExempt (internal/core/lint/lint.go, around the function at line 2675) covers only the content-authoring checks (banned_tokens, persona_registry), so a tool-mandated mirror of a root file into a subdirectory (a repo that must carry .github/copilot-instructions.md byte-identical to its root AGENTS.md, because that tool follows no pointer) raises a links_resolve blocker per relative link that resolves from the root and not from the mirror's directory, with no way to excuse it short of dropping the links. Observed on the v0.8.0 plugin binary: three blockers on one mirror. Workaround the adopter took: name companion pages as backticked paths with a shell guard checking each exists. Either let exempt_paths (or a dedicated key) excuse a path from links_resolve, or document that the link check has no exemption so authors of tool-mandated mirrors know in advance.

## Grounds

- pursued: a file named in rules.links_resolve.exempt raises no links_resolve finding while every other file still does; a broken link reported from an exempt file, or an unnamed file excused, would show it wrong
