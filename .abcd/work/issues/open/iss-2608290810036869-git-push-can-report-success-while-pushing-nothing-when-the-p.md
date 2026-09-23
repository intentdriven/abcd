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
---

git push can report success while pushing nothing, when the pre-push preflight outlasts the SSH idle timeout. The hook runs the full preflight, which takes minutes, and the connection is opened before the hook runs, so the server closes it mid-hook: the push dies with a connection-closed message, the remote ref is unchanged, and the preflight's own passing output scrolls past the failure so the whole thing reads as a clean run. Hit twice in one session. The second time the push was also piped to another command, which returns that command's exit status, so a failed push reported zero. Remedies that worked: set SSH keepalive options for the push, never pipe the push, and confirm with a remote ref listing that the tip actually moved. Worth considering whether the hook should run before the connection is opened, or whether a push wrapper should verify the remote tip afterwards; the silent half is the defect, not the slowness.