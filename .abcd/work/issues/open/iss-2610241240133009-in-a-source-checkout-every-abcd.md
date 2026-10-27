---
schema_version: 1
id: "iss-2610241240133009"
slug: "in-a-source-checkout-every-abcd"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "branch sweep survey, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "give the source checkout a make target that builds the current tree to a fixed local-tier path (for example .abcd/.work.local/bin/abcd, rebuilt when the tree changes) and point the dogfooding rule at it instead of go run; until then name the collapse in the DOGFOODING rule and have scripts read --json exit_code rather than the process status"
---

In a source checkout every abcd invocation is go run ./cmd/abcd, and go run turns every nonzero exit into 1: the binary's exit 2 (guard unevaluable, a usage error) and exit 3 (reserve: back off) both arrive as 1, the same code as a guard block. Measured 2026-10-24: guard check on an unterminated quote exits 2 as a built binary and 1 under go run. The dogfooding rule therefore makes the documented exit-code contract unreadable to the agents it governs, and a script that branches on exit 3 never sees it.
