---
schema_version: 1
id: "iss-2608220150152535"
slug: "docs-requirements-transitives-unpinned"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: "docs/requirements.txt"
resolution: "docs/requirements.txt claim corrected to name the unpinned transitive dependencies"
impact: internal
resolved_by:
  commit: "554f97f"
---

docs/requirements.txt line 1 claims pinned so every site build renders docs identically but only mkdocs-material is pinned; every transitive dependency floats from PyPI at build time so two builds of one commit can render differently and dependabots pip cooldown never sees the transitives