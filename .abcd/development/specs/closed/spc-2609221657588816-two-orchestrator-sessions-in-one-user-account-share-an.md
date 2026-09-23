---
id: spc-2609221657588816
slug: two-orchestrator-sessions-in-one-user-account-share-an
intent: itd-2609221656373558
origin: researcher-authored
production_mode: hand-written
---
# two-orchestrator-sessions-in-one-user-account-share-an

## Summary

The design record for itd-2609221656373558: joining, the claim, the three modes, the bounds and the comparison.

## Scope

1. **The claim** (`internal/core/implement/claim.go`): a claim file per record under the run's state in the machine store (`~/.abcd/runs/<root-sha>/claims/<record>.json`) carrying the session, the lane, the time and a lease; a claim is taken by an atomic create that fails if the file exists, which is the exclusion; a lapsed lease is claimable and the lapse is logged (criterion 3).
2. **Joining**: a session reads the run's state and writes `session_open` with its id; nothing signals the first session (criterion 1).
3. **The modes**: `run.mode` per window (`claim` | `batch` | `split-roles`), written into the window's log line; in `batch` the assignment comes from the run file, in `split-roles` the second session takes only review, audit and landing steps (criteria 2, 7).
4. **The bounds**: the second session's configuration carries `role: second`, which refuses the release path and any lane whose files touch the reading corpus, caps it at one lane, and gives it its own ceiling; contention (a failed claim, a busy queue, a locked store) backs it off with a logged reason (criteria 4, 5, 6).
5. **The measurement**: the run log's existing events plus `claim`, `claim_denied`, `backoff` and `window_mode`; the comparison is derived at the run's end (criteria 2, 6, 7).

## Out of scope

- A third session; other accounts or machines; an automatic scheduler.

## Approach

Everything lives in the machine-scoped run state the measurement ruling already put there, so two sessions share one directory and no repository file; the claim's exclusion is an atomic file create, which needs no lock manager and survives a killed session through its lease.

## Footprint

- packages: internal/core/implement, internal/core/positioning, internal/surface/cli
- tests: the claim's exclusion under two writers; the lease lapse; each mode's window line; the second session's refusals; the backoff log; the derived comparison

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 joining | scope 2 |
| 2 modes and the log | scope 3, 5 |
| 3 the claim and the lease | scope 1 |
| 4 release and corpus refused | scope 4 |
| 5 one lane, own ceiling | scope 4 |
| 6 backoff logged | scope 4, 5 |
| 7 the comparison | scope 3, 5 |
