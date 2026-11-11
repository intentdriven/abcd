---
schema_version: 1
id: "iss-2611101302324425"
slug: "the-prompt-hook-redacts-and-stores"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "coordinating session, 10 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/user-prompt-submit.sh"
remedy: "Give the transcript step its own small time budget inside the hook and stop at it, keeping what was done so the next prompt continues instead of restarting; or move redaction of a large staged transcript out of the prompt hook to session end or a detached worker. The rules and notices are written first, so a slow transcript can never discard them."
---

The prompt hook redacts and stores staged session transcripts on every prompt, inside the hook's 120-second limit. On 10 Nov 2026 one large staged transcript needed about three minutes of processor time, so the host ended the hook at 120 seconds on every prompt, discarded its output (the rules and the away notice with it), and the next prompt started the same work from the beginning. The person saw a timed-out line under every message until one run was let finish outside the host (187 seconds; the run after it took 13). Work that cannot finish inside the limit is never finished and is paid for again at each prompt.

This refines iss-2610161555125041, where the same hook overran the same limit under machine load: the cause here is the transcript step, which starts again at each prompt, and that record's remedy (one deadline for the whole hook, a slow stage skipped and named) would cover it too.
