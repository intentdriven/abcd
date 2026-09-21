---
schema_version: 1
id: "iss-2609211738504433"
slug: "a-planned-intent-with-no-spec-cannot-be-given-one-by-any-verb"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "the autonomous run's coverage check, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli (intent plan, intent link); the spec store"
---

A planned intent with no spec cannot be given one by any verb. Fourteen intents sit in planned/ with spec_id null (itd-6, 7, 24, 27, 28, 34, 36, 42, 48, 50, 53, 58, 63, 69), planned before the spec seam existed; the readiness gate reports each as not ready, and the autonomous run treats them as skips with a planning brief each. The way to a spec is closed on every side: intent plan on an already-planned record does the identity step alone and touches no spec; intent link writes a spec_id only for a spec that exists; the spec store has no mint of its own, only close. So the only path is a hand move of the record from planned/ back to drafts/ and a fresh plan, which no verb performs and the lifecycle does not name. Wanted: intent plan on a planned record whose spec_id is null mints and links the spec the way it does for a draft, without moving a bucket, on the same human sign-off; the readiness gate's remedy names that command instead of one that cannot run.
