---
schema_version: 1
id: "iss-82"
slug: "rules-inject-savestate-inline-atomic"
severity: "minor"
impact: internal
category: "tech-debt"
source: "user-observation"
found_during: "2026-07-12 iss-79 correctness review"
found_at: "internal/core/rules/inject.go"
resolution: "Routed SaveState through fsutil.WriteFileAtomic (adds temp+parent fsync); broadened TestNoInlineAtomicWriteSequences to flag os.CreateTemp+os.Rename. Detector watched fail on inject.go then pass."
---

Sixth divergent inline atomic write, and a detector-coverage gap: internal/core/rules/inject.go SaveState open-codes a durable write as os.CreateTemp + Write + Close + os.Rename with NO fsync and no parent-dir fsync -- weaker crash-safety than even the storeOriginal inline write iss-79 consolidated. It is not routed through fsutil.WriteFileAtomic, and the iss-79 detector (TestNoInlineAtomicWriteSequences) keys on os.O_EXCL + os.Rename so it does NOT flag the CreateTemp+Rename idiom. Fix: route SaveState through fsutil.WriteFileAtomic, and broaden the canonical-primitive detector to also flag inline os.CreateTemp + os.Rename sequences (watch it flag inject.go, then drain). Deferred from iss-79 to keep that change scoped to storeOriginal. Acceptance corpus: inject.go SaveState; the broadened detector must flag it before the fix and pass after. Found during iss-79 correctness review.