---
schema_version: 1
id: "iss-2609090828371674"
slug: "the-shared-file-lock-s-backoff-cap-limits-staging-to-roughly"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "sub-agent transcript capture implementation"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M22: sharded per-agent locks, each agent staging behind its own lock; not a timeout or backoff tune and not a lock-free append; a build lane owed, not holding the tag). Earlier deferral: The record states its own position plainly: the remedies are design-shaped and should be chosen rather than assumed. Raising the timeout, lowering the backoff ceiling, sharding the lock per agent, or moving to a lock-free append reconciled at drain are four different bargains between latency, contention and complexity, and the measurement that motivates them is a ceiling rather than a fault. Choosing among them is the maintainer's call and no reading of the evidence makes one of them obviously right."
resolution: "Each staged key (agent id, or session id for a main thread) stages behind its own lock under staging/locks/, per the M22 ruling; fsutil.WithFileLock revalidates the locked inode so a lock is retired with its staged file."
impact: fix
resolved_by:
  commit: "fcc3df07"
---

The shared file lock's backoff cap limits staging to roughly ten writers a second, so a burst of simultaneous sub-agent completions loses transcripts. The lock helper backs off exponentially to a hundred-millisecond ceiling, so the rate a contended lock admits is set by that ceiling and not by how short the critical section is. Measured here, sixteen simultaneous stages take 2.12 seconds, consistently across three runs. Extrapolating the same rate, a burst past roughly forty to fifty simultaneous completions begins exceeding the five-second staging lock timeout, and a stage that times out is refused: the transcript it carried is not written anywhere, which is the loss the capture work exists to prevent. Sessions that fan out widely are exactly the sessions whose delegated reasoning is most worth keeping, so the ceiling bites hardest where the value is highest. The condition is pre-existing in the locking helper rather than introduced by sub-agent capture, but nothing reached the lock concurrently before, so it was unreachable in practice until now. The remedies are design-shaped and should be chosen rather than assumed: raise the timeout, lower the backoff ceiling, sharded locks keyed per agent, or a lock-free append with reconciliation at drain.

## Grounds

- pursued: a burst of 64 simultaneous distinct-agent stages all succeed and a held agent lock blocks only that agent; a stage refused with lock contention during a fan-out, or an empty lock file per sub-agent accumulating in staging/locks, would show it wrong
