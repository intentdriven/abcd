---
schema_version: 1
id: "iss-2610151422133234"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Rewrite .abcd/development/brief/05-internals/05-prompt-quality.md:13-14,56-57 to match the shipped surface: The range is armed locally too: the Makefile `record-lint` target runs `go run ./cmd/record-lint -agent-diff origin/main...HEAD` (Makefile:165), so every `make record-lint` and `make preflight` arms the check, not only CI (.github/workflows/ci.yml:477). The…"
---

The brief at .abcd/development/brief/05-internals/05-prompt-quality.md:13-14,56-57 says: The unbumped-body-edit check needs a diff range "which the continuous-integration caller supplies"; "The range comes from the CI caller". The shipped surface differs: The range is armed locally too: the Makefile `record-lint` target runs `go run ./cmd/record-lint -agent-diff origin/main...HEAD` (Makefile:165), so every `make record-lint` and `make preflight` arms the check, not only CI (.github/workflows/ci.yml:477). The brief presents the CI caller as the only source of the range; the 'never from the in-tree config' half still holds (.abcd/record-lint.json carries no range). (v0.14.2 release crosscheck, false-claim, raised by opus in a35.)
