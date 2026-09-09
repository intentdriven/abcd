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
---

The shared file lock's backoff cap limits staging to roughly ten writers a second, so a burst of simultaneous sub-agent completions loses transcripts. The lock helper backs off exponentially to a hundred-millisecond ceiling, so the rate a contended lock admits is set by that ceiling and not by how short the critical section is. Measured here, sixteen simultaneous stages take 2.12 seconds, consistently across three runs. Extrapolating the same rate, a burst past roughly forty to fifty simultaneous completions begins exceeding the five-second staging lock timeout, and a stage that times out is refused: the transcript it carried is not written anywhere, which is the loss the capture work exists to prevent. Sessions that fan out widely are exactly the sessions whose delegated reasoning is most worth keeping, so the ceiling bites hardest where the value is highest. The condition is pre-existing in the locking helper rather than introduced by sub-agent capture, but nothing reached the lock concurrently before, so it was unreachable in practice until now. The remedies are design-shaped and should be chosen rather than assumed: raise the timeout, lower the backoff ceiling, sharded locks keyed per agent, or a lock-free append with reconciliation at drain.
