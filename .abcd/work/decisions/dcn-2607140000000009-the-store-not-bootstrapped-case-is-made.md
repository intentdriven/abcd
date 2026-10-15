---
id: dcn-2607140000000009
slug: the-store-not-bootstrapped-case-is-made
date: 2026-07-14
---
2026-07-14 (M1, iss-95 — maintainer decision) — The store-not-bootstrapped case
is made LOUD, not self-bootstrapped by the hook (rejects having `hook session-end`
create `~/.abcd/history/`, which would put a dir-creating trust-boundary act inside
a fail-closed hook and contradict the `ownedDirsReal` symlink discipline). Reality
check: `ahoy install` ALREADY bootstraps the store (`bootstrapHistory`, plus the
per-repo transcripts dir), and detection ALREADY emits `history.bootstrap_missing`
as a required gap that bare `abcd`, `ahoy`, and `ahoy doctor` surface — so an
installed user is never in the silent state. The only genuinely silent path is the
`SessionEnd` hook itself, which by harness contract has NO output channel (its exit
code and stdout are ignored), so it cannot speak at session end. "Loud" therefore
lives where a channel exists: a SessionStart notice (SessionStart hook output is
surfaced) that warns once when the store is absent, pointing at `/abcd:ahoy install`.
Scoped as an M1 follow-up; keeps the hook fail-closed-silent and moves the loudness
to the one event that can be heard.
