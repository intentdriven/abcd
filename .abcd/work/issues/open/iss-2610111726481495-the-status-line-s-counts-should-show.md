---
schema_version: 1
id: "iss-2610111726481495"
slug: "the-status-line-s-counts-should-show"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "orchestrator session, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/compose.go"
remedy: "Replace the intents and issues elements with ready counts, labelled as the product thinker ruled on 2026-10-11: 'itd <N> ready / iss <M> fixable', where N is the planned intents the readiness gate passes (the board's Now plus Next) and M is the drain-eligible issues (abcd drain --dry-run's eligible count). Because both exceed the status line's 500ms bound, read them from a small figure that the verbs which change records (capture, intent plan, spec close, drain) or the session-start hook refresh, never written by the status line itself, keeping its write-nothing rule; when the figure is missing or older than the newest record change, drop the element with a note rather than show a stale or slow count."
---

The status line's counts should show what can run autonomously now, not the backlog. Today 'itd 139 · iss 332' counts every intent not yet shipped (drafts plus planned) and every open issue (internal/core/statusline/compose.go:139-160), which the product thinker said is not the number they care about. They want planned intents ready to build, and issues that need no design decision. Both exist already: the board's Now/Next block runs the readiness gate over planned intents (Next: 15 READY at main 22492dbf4, plus the Now rows), and 'abcd drain --dry-run' classifies open issues (64 eligible, 264 handed back, 4 without a remedy). Measured on this machine, the board took 4.9s and the drain dry run 0.55s; the status line's own work stops at 500ms, so neither can be computed on every refresh.
