---
schema_version: 1
id: "iss-2610251201361740"
slug: "the-session-end-hook-s-retry-of-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Count the retry budget in attempts with a short backoff (for example three tries) rather than in wall-clock time that the first try can consume, or start the clock after the first try returns; keep a hard ceiling so a session end never hangs, and say on stderr when the store was skipped."
---

The session-end hook's retry of a transient repository-root failure has a 1.2-second budget, so on a loaded machine the first detection uses it all and the hook never retries: on 2026-10-25 TestHookSessionEndRetriesATransientRootFailure failed this way at a load near 20, and the same budget applies to real session ends, where the transcript store step is then skipped.
