---
schema_version: 1
id: "iss-2609012020476120"
slug: "four-test-sites-now-assert-elapsed-wall"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/bootstrap_freshinstall_test.go"
remedy: "Waits on ruling CA1 (principle, check, or both): for a principle, land the unmerged draft from branch test/wall-clock-principle (commit 7f7b42625, 'assert the observable, not the clock') onto main with its sweep re-checked at the tip; for a check, add a test in `internal/core/lint` that walks every _test.go and refuses a comparison of time.Since or an elapsed value against a constant unless the line says it is a hang detector, proven by a fixture it refuses; either way the elapsed-time comparisons left at the base (three lines, in `internal/core/reading` and `internal/core/cite`) move onto a work count or a `testing/synctest` fake clock, or are marked as hang detectors."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Make 'a test asserts behaviour, never a duration' a principle, a lint refusing a duration assertion in a _test.go file, or both? A principle and a nine-site sweep were written on 2026-09-02 (commit 7f7b42625, branch test/wall-clock-principle) citing a ruling that is not recorded on main; that branch never merged, so nothing of it is at v0.11.1."
---

Four test sites now assert elapsed wall-clock time where they mean to assert behaviour: iss-2608292246210181 and iss-2608290810037763 in the scanner's adjacency tests, iss-2608301301041887 in the grounds lock test, and the bootstrap fresh-install self-check in internal/surface/cli. Each was found by a CI flake and each is being fixed site by site. Worth deciding whether the rule 'a test asserts behaviour or an operation count, never a duration' graduates to a principle under .abcd/development/principles/ or to a lint rule that refuses a time.Since comparison in a _test.go file, so the fifth site is refused at draft time rather than found by the next flake. This is a design decision and was not taken during autonomous-run-2026-09-01, where it was carried from the session handover; the fourth site was fixed on its own merits.

## Remedy grounds (2026-09-29)

- SOTA check: the testing/synctest documentation (https://pkg.go.dev/testing/synctest, read 2026-09-29) runs a test in a bubble whose time package uses a fake clock that advances only when every goroutine is durably blocked, generally available since Go 1.25; go.mod declares 1.26.7, so it costs no dependency (its Sleep helper is 1.27, outside the pin).
- The branch draft cites a ruling not recorded on main, so landing it still waits on CA1.
- Rejected: raising the ceilings, which keeps measuring the machine rather than the code.
