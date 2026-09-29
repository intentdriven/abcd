---
schema_version: 1
id: "iss-2609291610432030"
slug: "the-transcript-store-never-narrows-what-an-earlier-binary"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/location.go"
deferred_after: "v0.11.1"
deferral_reason: "deferred by lane histMode, run A 2026-09-29: the lane brief admits a narrowing pass only as a small use of an existing fsutil primitive, and none narrows a directory (tightenLock is unexported and lock-specific), so it needs a new primitive and a ruling on EnsureRealDir's never-narrow stance; new captures are written 0o600 and Migrate narrows each record it rewrites, so the exposure is bounded to records stored before the fix under a chain level an earlier layout created wider"
---

The transcript store never narrows what an earlier binary laid out wider: a chain level that already exists keeps its mode (fsutil.EnsureRealDir creates at 0o700 and leaves an existing directory alone, internal/core/history/location.go, Resolve), and a record already on disk at 0o644 stays at 0o644 until Migrate happens to rewrite it. Capture now writes every new record 0o600 (iss-2609012029343438), which protects nothing already stored: on a machine where an earlier layout created ~/.abcd/transcripts or a level below it 0o755, every record captured before the fix stays readable by other local accounts. The remedy is a narrowing pass on the store's own levels only (the base, the root-sha level and records/, never ~/.abcd or a checkout's .abcd/.work.local), each opened O_NOFOLLOW|O_DIRECTORY, owner checked with fstat against this account, and narrowed with fchmod on the descriptor, the shape of fsutil's unexported tightenLock for lock files. No exported fsutil primitive does that for a directory, so it is a new primitive plus a stance change for EnsureRealDir's never-narrow contract, not a mode constant.
