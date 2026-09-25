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
---

The capture verbs' --json and stderr print the checkout path through RedactHome only, so a checkout outside HOME is printed in full (internal/surface/cli/cli.go renderLedger), an absolute local path in output that may be pasted elsewhere; renderLedger also splices the ledger member by byte surgery on the trailing brace (review-capture 4).
