---
schema_version: 1
id: "iss-208"
slug: "a-successful-bootstrap-install-is"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "first manual plugin install test (2026-08-10)"
found_at: "hooks/bootstrap.sh"
blocked_by: [iss-204]
resolution: "The chained SessionStart command emits the bootstrap's own message first, so the single line the transcript renders on a fresh install is the checksum-verified success rather than a missing-binary complaint. The non-zero exit stays: it is the only channel that puts stderr in front of the human."
impact: fix
---

A successful bootstrap install is rendered to the user as a third 'SessionStart:startup hook error', indistinguishable from the two genuine failures above it. hooks/bootstrap.sh's notice() exits 2 deliberately, and its comment gives the sound reason: a SessionStart hook's stdout becomes model context, while only a non-zero exit puts stderr in front of the human. But the harness renders ANY non-zero SessionStart exit as a '<hook name> hook error' notice — the docs confirm exit 2 'renders in the transcript as a <hook name> hook error notice, the same way a non-blocking error does'. So there is no 'notice' channel distinct from 'error', and the checksum-verified happy path is labelled a fault. Observed on the first manual install (2026-08-10): three consecutive hook-error lines, of which the third was the install SUCCEEDING. Fix direction: fold this into the single chained SessionStart entry proposed in iss-204 and lead the visible first line with the success, since the transcript shows only the first line of stderr; the honest-failure posture for the genuinely-no-binary window is unaffected.