---
schema_version: 1
id: "iss-2608271711538151"
slug: "requirements-txt-publishes-into-the"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "docs/requirements.txt"
resolution: "requirements.txt and dotfiles excluded from the built site via mkdocs exclude_docs"
impact: fix
resolved_by:
  commit: "a032377d"
---

docs/requirements.txt is published verbatim into the built site: mkdocs.yml sets docs_dir: docs with no exclude_docs, so the Python pin file ships as a public page of the rendered site. Fix is one line — exclude_docs: requirements.txt in mkdocs.yml. Moving the file was considered and rejected: CODEOWNERS, two site.yml lines and an out-of-tree wrangler build command all name the current path.