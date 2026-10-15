---
id: dcn-2610040000000004
slug: decided-without-a-question-lane-e8term3
date: 2026-10-04
---
2026-10-04 — Decided without a question (lane e8term3 of autonomous run A, spc-2610030911534855 step 3, on sec-e8term2 finding 5): the plain-Terminal answer loop's relay of an outside SIGINT, SIGTERM or SIGHUP (`term.RawSession` restores the terminal, resets the signal and raises it again, so no deferred function runs) stays as it is, and no `Hooks.Ending` is added. Nothing that draws a question runs under `signal.NotifyContext` or holds a lock or a temporary file across a question: the one `NotifyContext` in the tree is `abcd build`'s runner loop, which asks nothing, `ahoy install` asks every question outside its config, rewrite and history locks, and `ahoy remote apply` and `site setup` hold none. For these front doors the relay is parity with an unhandled signal; `ask.Terminal`'s documentation says a caller under a `NotifyContext` must not draw. Alternative not taken: a `Hooks.Ending` callback, which would have no caller.
