---
schema_version: 1
id: "iss-2609012020476120"
slug: "four-test-sites-now-assert-elapsed-wall-clock-time-where-the"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/bootstrap_freshinstall_test.go"
---

Four test sites now assert elapsed wall-clock time where they mean to assert behaviour: iss-2608292246210181 and iss-2608290810037763 in the scanner's adjacency tests, iss-2608301301041887 in the grounds lock test, and the bootstrap fresh-install self-check in internal/surface/cli. Each was found by a CI flake and each is being fixed site by site. Worth deciding whether the rule 'a test asserts behaviour or an operation count, never a duration' graduates to a principle under .abcd/development/principles/ or to a lint rule that refuses a time.Since comparison in a _test.go file, so the fifth site is refused at draft time rather than found by the next flake. This is a design decision and was not taken during autonomous-run-2026-09-01, where it was carried from the session handover; the fourth site was fixed on its own merits.
