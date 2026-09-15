---
schema_version: 1
id: "iss-2609012035115696"
slug: "ghsa-5mmm-3whv-3rqp-cwe-693-launch-copied-skip-listed-payloa"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/dryrun.go"
resolution: "Fixed before this run by 30f62e82 (fix(launch): close nested/case-fold deny bypass and unscanned-payload gap), shipped in v0.6.7 and deepened by 1f71e608 and 95b98abc; recorded so the advisory has a ledger marker."
impact: internal
resolved_by:
  commit: "30f62e828f4c51f04de8589dd7722f29215ca69c"
---

GHSA-5mmm-3whv-3rqp (CWE-693): launch copied skip-listed payload files (.svg among them) into the bundle unscanned, so a secret in a skip-listed file passed every gate. Fixed before this run by 30f62e82 (.svg removed from the default skip set and wouldRefuseOn failing closed on Unscanned; v0.6.7), deepened by 1f71e608 (every skip-listed file byte-scanned; v0.6.9) and 95b98abc (explicit byte-rule table); the class record is resolved/iss-2608291807454357 (GHSA-9wv7-88w3-f77m, the same skip-by-name mechanism), whose fixing commit names this advisory as precedent. This record binds the advisory id to its fixing commits so an advisory-keyed search hits. Evidence at v0.7.0: internal/core/launch/dryrun.go scanRefusals and internal/adapter/scanner/scanner.go ScanBundle; tests TestSvgPayloadSecretRefuses, TestUnscannedPayloadRefuses, TestUnscannedRefusalCarriesWhy and TestBinaryPayloadSecretRefuses in scan_coverage_test.go.

## Grounds

- pursued: the defect is closed on main; the record exists to bind the advisory to its fixing commit
