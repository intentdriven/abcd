---
id: dcn-2607280000000006
slug: itd-102-implementation-repo-positioning
date: 2026-07-28
---
2026-07-28 — itd-102 implementation: repo positioning lives in a NEW `internal/core/positioning` package, not `internal/core/identity` — that package is the git commit-author gate (`.abcd/config/identity.json` pin, pre-commit hook) and shares nothing with repo self-description but the English word; folding them would put two unrelated concerns behind one name. The user-facing verb stays `abcd identity` (spc-19). For the same collision the registry is `.abcd/positioning.json` at the `.abcd/` top level (beside docs-lint.json / record-lint.json / rules.json), NOT `.abcd/config/identity.json`. Rejected: extending the commit-identity package; naming the verb `abcd positioning` (spc-19 fixes the verb).
