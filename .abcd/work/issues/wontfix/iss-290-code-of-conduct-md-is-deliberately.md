---
schema_version: 1
id: "iss-290"
slug: "code-of-conduct-md-is-deliberately"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
found_at: ".abcd/docs-lint.json"
wontfix_reason: "deliberate deferral recorded by the maintainer: no Code of Conduct while the repo is single-maintainer; revisit on external adoption"
---

CODE_OF_CONDUCT.md is deliberately deferred: it is not in the stray_root_docs allowlist, and allowlisting it in the same change that adds it would put the evaluator inside the loop; adding it is a two-change dance (allowlist first, file second) for whenever the contributor base warrants one