---
schema_version: 1
id: "iss-2610111826193257"
slug: "concurrent-sessions-gate-their"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "orchestrator session, 2026-10-11, peer report from abcd-a5"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Give abcd the gate the sessions keep hand-writing: 'abcd implement load --wait' blocks until the one-minute load is under a per-machine limit (~/.abcd.noindex/load-limits) and no other go test -race run is live, judged from the process table by argv with the check's own process and ancestors excluded (never a pattern over its own command line), and releases waiters one at a time through a machine-wide lock under ~/.abcd.noindex so a falling load admits one preflight, not all; then teach the LOAD rule domain to use it instead of shell loops, and prove the self-match and the herd with tests that fail against the hand-written forms."
---

Concurrent sessions gate their preflights and race runs with hand-written shell waiters, because abcd has no wait-for-capacity primitive: 'abcd implement load' (make preflight's load-check) only warns, never waits. On 2026-10-11 a waiter written as 'pgrep -f "go test -race"' matched its own bash command line, which contains that string, so it always saw a race run going and never opened; when such waiters were stopped or edited they released together, and the machine (16 cores) reached a one-minute load of about 500 with four race runs at once, one peak near 920, while unrelated preflights timed out (internal/surface/cli hit its 10-minute test limit at TestSiteVerbsSayWhichLabelsTheyAdded). Reported by peer session abcd-a5; the anchored forms that work are pgrep -f '(^|/)go test -race' and ps -Ao command | grep -c '^go test -race'. Two separate failures: the self-match, and many waiters with the same release condition opening at once (a thundering herd).
