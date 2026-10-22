---
id: dcn-2610180937175629
slug: an-api-error-stop-is-recorded-and-a-turn
date: 2026-10-18
---
2026-10-18 — An API-error stop is recorded, and a turn ending on a prose question while away is blocked once (facilitator-on-delegation, itd-2610180912189125 review round 1): the plugin's hook manifest gains a StopFailure entry, which the host fires instead of Stop when a turn ends on an API error such as a rate limit and whose output it ignores, so it cannot continue the session; while the person is away abcd records the stop and the host's error type in the away record, and the slice's scope conditions name that such a turn is recorded, not resumed, except by a wake the session had already scheduled. While away, a stop whose last_assistant_message ends on a line ending in a question mark is blocked once per message with the reroute's reason (decide and record it, or queue it when it is an act only the person may approve), before the verdict's other rows apply (grounds: the plan review of 2026-10-18, finding 11; code.claude.com/docs/en/hooks StopFailure and Stop input; iss-2610170619247185, whose session ended its turn on a question)
