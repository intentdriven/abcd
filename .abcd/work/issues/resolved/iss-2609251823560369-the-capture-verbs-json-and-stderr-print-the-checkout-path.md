---
schema_version: 1
id: "iss-2609251823560369"
slug: "the-capture-verbs-json-and-stderr-print-the-checkout-path"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "the capture verbs' ledger identity names a checkout outside HOME by its directory name, never its absolute path, and renderLedger refuses a non-object result instead of silently dropping the ledger member"
impact: fix
resolved_by:
  commit: "a6c630c39"
---

The capture verbs' --json and stderr print the checkout path through RedactHome only, so a checkout outside HOME is printed in full (internal/surface/cli/cli.go renderLedger), an absolute local path in output that may be pasted elsewhere; renderLedger also splices the ledger member by byte surgery on the trailing brace (review-capture 4).

## Grounds

- pursued: a capture verb run in a checkout outside HOME prints no absolute path in --json or on stderr; TestTheLedgerIdentityNeverPrintsAnAbsoluteCheckout failing, or an absolute checkout in either render, would show it wrong
