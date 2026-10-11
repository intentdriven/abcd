---
schema_version: 1
id: "iss-2610110253059544"
slug: "gitutil-version-probe-has-no-deadline"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "abcd-60 drain run 2026-10-11, lane for iss-2610090642407850"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil"
remedy: "Run the git version probe under the caller's context (or a short fixed deadline) with the same kill-on-cancel handling every other gitutil call gets, release the lock on timeout without caching a result, and add a test with a fake git that hangs on 'version', proving the first call returns by its deadline and leaves no child behind."
---

gitutil's one-time git version probe (internal/gitutil lazyfetch.go) runs with no deadline and under a process-wide lock. So the first git call of a process whose git hangs blocks in that probe, and every other git call waits on the lock behind it. For the status line this means a hung git ends the verb at its 500ms overall limit with the wrong message, and the hanging git version child is never killed, so one is left behind on every refresh. Found by the 2026-10-11 drain lane for iss-2610090642407850: a test's hanging fake git, answering the probe, blocked every git in its test process for 30s.
