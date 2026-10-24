---
schema_version: 1
id: "iss-2610241130157455"
slug: "plain-go-test-runs-that-sessions-start"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd build itd-82 session, 2026-10-24, three preflights waiting at the load gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload"
remedy: "the waiter names each go test run that no gate admitted (no make preflight or waiter above it) beside the load in its waiting line and as ungated_tests in --json, and the LOAD rules domain teaches running a package's tests through 'abcd implement load --wait -- go test'; counting them as occupancy would not help, since an ungated run never consults the gate"
---

Plain go test runs that sessions start outside the load gate keep the one-minute load above the gate's limit, so the gated preflights starve: on 2026-10-24 three preflights of this account waited 18, 36 and 54 minutes at 'abcd implement load --wait' (load 26 to 76 against a limit of 20 on 10 cores) while three ungated 'go test -timeout 40m' runs of other sessions (internal/surface/cli, internal/core/identity and its neighbours) ran beside them, and the session filing this had itself run internal/core/implement/... and internal/surface/cli ungated earlier the same morning. The gate counts a running make preflight and a go test -race run as occupancy, but a plain go test only as load, so the runs that skip the gate are the ones that hold it shut for the runs that use it.
