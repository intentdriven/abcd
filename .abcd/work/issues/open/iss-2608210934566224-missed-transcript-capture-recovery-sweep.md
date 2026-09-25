---
schema_version: 1
id: "iss-2608210934566224"
slug: "missed-transcript-capture-recovery-sweep"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "plugin-update post-mortem 2026-08-21"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Transcript recovery sweep: report the ended-but-unsaved sessions it finds at next start, or save them automatically?"
---

Session-end transcript capture is best-effort and its loss is silent: a cancelled or killed SessionEnd hook (update-then-quit, crash, SIGKILL) leaves no trace that a session was never captured into the history store. Add a recovery sweep — at session start or in ahoy doctor — that compares harness transcripts against the history store index and reports (or captures) the gap, turning silent loss into a caught-on-next-start notice. abcd history capture already ingests retroactively.

Raised minor -> major on 2026-08-23. The sweep was seeded as a nicety against
rare events — update-then-quit, crash, SIGKILL. `iss-2608230817034768` shows the
loss is not rare but systematic: redaction cost rises at roughly 0.7 s per MB
against a shutdown budget somewhere near 1.5 s, so past a few MB loss is the
norm rather than a risk. Nine of this one repo's ended transcripts are already
absent from its store.

That also makes this the detector the capture fix has to land against, since no
fix to the budget can be watched fail without it — and the sweep is what would
pin the budget itself, which observational data cannot. Distinguishing "never
ended" from "ended and lost" is the whole difficulty: three of that repo's
apparent absences were simply sessions still running, and two more predate the
store's first record, which is exactly the ambiguity a sweep has to resolve
before it can report anything trustworthy.

Additional evidence (2026-08-26, second-harness adaptor lab, local tier): the
lab built and red-teamed a working sweep of exactly this shape against a host
with no session-end event — list the directory's sessions, skip live and
recent ones, export each finished session, hand it to `hook session-end`,
watermark per session id. Design points proven there that a native sweep
should reuse: a recency guard alone pins partial transcripts (an
idle-then-resumed session was exported mid-life and, staging being first-wins, its later
turns were never captured — an advancement re-check between listing and
staging fixed it); success must be keyed on the staging contract rather than
the hook's exit code, because `hook session-end` exits 0 on every path and
treating that as success watermarked failed stagings as captured, a silent
permanent loss (now iss-2608261550596333); watermark writes must be atomic,
since a torn state file reads as empty and mass re-exports the backlog; and
per-session failure isolation keeps one bad export from abandoning the batch.