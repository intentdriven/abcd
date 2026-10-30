---
schema_version: 1
id: "iss-2610231057580368"
slug: "scripts-check-reviews-cases-sh-case"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "drain lane run-2610230715022943 preflight, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-reviews-cases.sh"
remedy: "Make each case's match read the whole output before testing it (capture the output to a variable, then test it, or use grep without -q), so an early-exiting grep cannot fail the pipeline under pipefail; prove it with a case run under deliberate load."
---

scripts/check-reviews-cases.sh case 'RD004 a NUL byte after the sha' failed once inside make preflight on a loaded machine (load about 30 on 16 cores, 2026-10-23) although the output carried RD004, then passed on three standalone reruns; a likely cause, unconfirmed, is grep -q under set -o pipefail closing the pipe early and failing the printf that feeds it, which a busy machine makes more likely.
