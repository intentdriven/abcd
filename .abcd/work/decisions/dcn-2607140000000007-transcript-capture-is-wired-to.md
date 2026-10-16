---
id: dcn-2607140000000007
slug: transcript-capture-is-wired-to
date: 2026-07-14
---
2026-07-14 (M1, itd-89/spc-4) — Transcript capture is wired to `SessionEnd`, NOT the
`Stop` the plan specified. The plan's letter was wrong on a matter of harness fact:
`Stop` fires once per assistant TURN, and Claude Code's transcript file grows through a
session, so a `Stop`-wired capture stores a fresh, larger superset every turn — proven
by live test (one session, 4 turns → 4 records; a 100-turn session → 100 records and
O(N²) bytes). `history.Capture`'s sha256 dedup only collapses byte-IDENTICAL
re-captures, which never happens on a live transcript, so the plan's "re-capture is
idempotent" acceptance is false under `Stop`. `SessionEnd` fires once at termination and
by contract ignores exit code + stdout — a perfect fit for a fail-closed, non-blocking
side-effect hook. Verified against the harness docs (code.claude.com/docs/en/hooks).
Accepted cost, recorded not hidden: `SessionEnd` does not fire on a hard crash/SIGKILL,
so an uncleanly-killed session is not captured; the `Stop`-with-session_id-dedup
alternative that would recover that case needs a change to shipped core dedup semantics
and is deferred. This is the M1 deviation the loop is required to surface.
