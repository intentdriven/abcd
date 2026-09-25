---
schema_version: 1
id: "iss-2608290810036869"
slug: "git-push-can-report-success-while-pushing-nothing-when-the-p"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "intent-implementation-run"
found_at: ".githooks/pre-push"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M16: check before connect: preflight runs to completion before the push opens its connection (a push path that gates first); a build lane owed, not holding the tag)."
resolution: "The pre-push hook no longer runs the preflight, so it no longer holds the push's connection open: make preflight runs first and mints a receipt for HEAD, and the hook refuses in milliseconds a push whose new commit has none (ruling M16, check before connect)."
impact: fix
resolved_by:
  commit: "b5b0897e4a5c1ac920500d2ae287a05648bd8d15"
---

git push can report success while pushing nothing, when the pre-push preflight outlasts the SSH idle timeout. The hook runs the full preflight, which takes minutes, and the connection is opened before the hook runs, so the server closes it mid-hook: the push dies with a connection-closed message, the remote ref is unchanged, and the preflight's own passing output scrolls past the failure so the whole thing reads as a clean run. Hit twice in one session. The second time the push was also piped to another command, which returns that command's exit status, so a failed push reported zero. Remedies that worked: set SSH keepalive options for the push, never pipe the push, and confirm with a remote ref listing that the tip actually moved. Worth considering whether the hook should run before the connection is opened, or whether a push wrapper should verify the remote tip afterwards; the silent half is the defect, not the slowness.

## Grounds

- pursued: no push waits on a gate while its connection is open, so an idle-timeout close can no longer eat a push; a pre-push hook that still runs a multi-minute step, or a plain git push of an unpreflighted new commit that reaches the remote, would show it wrong
