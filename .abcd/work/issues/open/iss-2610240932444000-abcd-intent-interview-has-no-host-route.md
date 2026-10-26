---
schema_version: 1
id: "iss-2610240932444000"
slug: "abcd-intent-interview-has-no-host-route"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "/abcd:build iss-2610140652230444 session, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/written.go"
remedy: "When no runner route reaches the planning interviewer and the verb runs inside a host session, hand the interview to that session the way implement step hands a host-routed role (one ask or done turn per call, written to the same answers record), and make the no-route refusal name both routes it would take: the host-run interview through /abcd:intent and a runner in the person's own config."
---

`abcd intent interview` never hands the interview to a host session, and its refusal names only a runner, so a person sent to the verb is not told that the session they are in may conduct the interview.

The refusal (`internal/core/interview/written.go:169-175`, check at `:180-190`) passes only when `roles.planning-interviewer.runner` names a runner this machine enabled, or `runner.fallback_host` is set; repository routes are skipped (`internal/core/runner/config.go:247-252`). The verb dispatches with `HostSession` unset (`written.go:342-345`), and the host route never hands off to a host session in this verb (`internal/core/runner/dispatch.go:83-103`), while `implement step` does hand a host-routed role to the session that drives it (`internal/core/implement/loop/drive.go:111-113`).

The host-run interview exists as written procedure (`commands/intent.md:319-499`; `:553-554`: "in a host session the interview is yours"), but nothing reaches it from the verb, and the refusal does not mention it. On 2026-10-24 the session itself sent the person to the terminal verb instead of conducting the interview, which this refusal did nothing to correct. A person on a subscription (no API key, iss-2610240932215918) is therefore told to configure a route they cannot run, and is not told the route they can.

Surfaced 2026-10-24 (the person ran the command from the session's own advice and got the refusal; "jesus. too complicated. Not working").
