---
schema_version: 1
id: "iss-2610040452174484"
slug: "the-secret-scanner-s-bundled-patterns-know-openai-s-sk-proj"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A, lane e8scanner (iss-2610040202190813), checking gitleaks' openai-api-key rule"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/patterns.go"
remedy: "Add a bundled hard_fail pattern for sk-admin- shaped exactly as the existing sk-proj- and sk-svcacct- rules (\\bsk-admin-[A-Za-z0-9_-]{40,}, no trailing \\b), on the grounds of gitleaks' openai-api-key rule (primary source: gitleaks cmd/generate/config/rules/openai.go), pinned by a runtime-built fixture; shown wrong by a committed file the new rule flags that is not a key (sweep the tree first)."
---

The secret scanner's bundled patterns know OpenAI's sk-proj- and sk-svcacct- keys but not its sk-admin- admin keys, the third prefix gitleaks' openai-api-key rule names (cmd/generate/config/rules/openai.go: sk-(?:proj|svcacct|admin)-...). An admin key reaches no bundled pattern: the plain sk- rule cannot match it either, since 'admin' is followed by '-'. Every store-before-commit redactor, the launch scan and the guided connect's key check share the gap.
