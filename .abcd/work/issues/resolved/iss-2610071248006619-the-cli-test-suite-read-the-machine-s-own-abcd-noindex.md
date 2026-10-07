---
schema_version: 1
id: "iss-2610071248006619"
slug: "the-cli-test-suite-read-the-machine-s-own-abcd-noindex"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the post-v0.13.2 follow-up preflight, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/layered/layered.go"
remedy: "Give internal/core/layered the same home seam rules has (a userHomeDir var and SwapUserHomeForTest), and have the cli TestMain point it at an empty home of the suite's own whenever a test has not set HOME itself."
resolution: "internal/core/layered gained the home seam rules has; the cli TestMain points it at an empty suite home unless a test sets HOME."
impact: internal
resolved_by:
  commit: "36928d867"
---

The cli test suite read the machine's own ~/.abcd.noindex/oracle-routing.json: once a routing table was accepted on the machine (written 2026-10-07 13:13, by another session), every delegated disembark verb printed a routing line to stderr and six TestDisembark* tests that parse the --json output failed, on this machine only. The suite's TestMain already kept the machine's rules.json out of reach (rules.SwapUserHomeForTest); the layered loader that resolves the routing table called os.UserHomeDir directly.
