---
schema_version: 1
id: "iss-2608221342506032"
slug: "docs-requirements-txt-pins-mkdocs"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "agent-finding"
found_at: "docs/requirements.txt"
wontfix_reason: "deferral documented in docs/requirements.txt itself: transitive freeze via pip-compile --generate-hashes only if reproducible doc builds become a requirement"
---

docs/requirements.txt pins mkdocs-material by version only; the deploy render job runs pip without hash-pinning, so a --require-hashes freeze is the remaining supply-chain hardening