---
id: dcn-2607170000000001
slug: burst-2-run-test-b-two-mechanics
date: 2026-07-17
---
2026-07-17 — Burst 2 (run test B), two mechanics decisions: (1) an
already-planned intent with spec_id null (itd-40, created before the
lifecycle verbs existed) is routed through a transient git mv planned->drafts
so `intent plan` can mint+link its spec fail-closed — there is no standalone
spec-create verb, and hand-authoring a spec file would bypass the mint lock's
id allocation; net record churn is the spec_id write. (2) Implementation was
delegated to an Opus 4.8 worker (recorded protocol deviation, manual test B);
the orchestrator re-ran the gate on the output, and commits carry the trailer
of the model that authored them (worker code: claude-opus-4-8; orchestrator
record work: claude-fable-5).
