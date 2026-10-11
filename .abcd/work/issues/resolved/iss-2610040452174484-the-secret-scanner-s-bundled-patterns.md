---
schema_version: 1
id: "iss-2610040452174484"
slug: "the-secret-scanner-s-bundled-patterns"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A, lane e8scanner (iss-2610040202190813), checking gitleaks' openai-api-key rule"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/patterns.go"
remedy: "Add a bundled hard_fail pattern for sk-admin- shaped exactly as the existing sk-proj- and sk-svcacct- rules (\\bsk-admin-[A-Za-z0-9_-]{40,}, no trailing \\b), on the grounds of gitleaks' openai-api-key rule (primary source: gitleaks cmd/generate/config/rules/openai.go), pinned by a runtime-built fixture; shown wrong by a committed file the new rule flags that is not a key (sweep the tree first)."
resolution: "The canonical pattern set learns OpenAI's sk-admin- key, shaped as the sk-proj- and sk-svcacct- rules, hard_fail."
impact: fix
resolved_by:
  commit: "c6e34252829b0e34bba63e236b4f73810a35f868"
---

The secret scanner's bundled patterns know OpenAI's sk-proj- and sk-svcacct- keys but not its sk-admin- admin keys, the third prefix gitleaks' openai-api-key rule names (cmd/generate/config/rules/openai.go: sk-(?:proj|svcacct|admin)-...). An admin key reaches no bundled pattern: the plain sk- rule cannot match it either, since 'admin' is followed by '-'. Every store-before-commit redactor, the launch scan and the guided connect's key check share the gap.

## Grounds

- pursued: an sk-admin- key is reported as token:openai_admin and masked by every redactor; shown wrong by such a key passing the scanner, or by a committed non-key it flags (git grep finds no sk-admin- in the tree).
