---
schema_version: 1
id: "iss-2609252007448074"
slug: "abcd-source-ledger-flip-is-the-person-s-act-under-adr-41"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/source/ledger.go"
resolution: "Fixed: the default hazard registry carries abcd-source-ledger-flip, a blocker on abcd source ledger with --flip (either spelling, --corpus stepped over), whose successor leaves the flip to the person. Residual, stated on the guard page: it matches the program by basename abcd, not go run ./cmd/abcd or a renamed copy (the guard is a mistake filter, adr-42)."
impact: fix
resolved_by:
  commit: "5b52293e9"
---

abcd source ledger --flip is the person's act under adr-41 gate 2 (a line is flipped to public citation only by the human), but the shell guard's default hazard registry does not name it, so an agent can run it unprompted; ledger.go refuses a confidential or non-citable source mechanically, so the exposure is misattributed provenance, not leakage. Register the command in the guard's default registry once the guard lane's registry edit has landed.

## Grounds

- pursued: an agent's abcd source ledger --flip is blocked by the bundled registry; a spelling the tests name that the hook still allows would show it wrong
