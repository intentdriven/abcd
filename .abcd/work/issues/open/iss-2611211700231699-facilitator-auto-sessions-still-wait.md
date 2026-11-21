---
schema_version: 1
id: "iss-2611211700231699"
slug: "facilitator-auto-sessions-still-wait"
severity: "major"
category: "process"
source: "user-observation"
found_during: "planning itd-2611211024360819 during the autonomous run, 2026-11-21"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config.json"
remedy: "Find out first where the waits come from: a lab that puts a session under facilitator auto through each known trigger (a page that reserves an act to the person, a review the session started itself, a peer's relay, a question with one defensible answer, a turn that ends while work is owed) and records what it does. Then make the setting something abcd holds and checks: say it at every prompt while it is on, keep one list of the acts reserved to the person with what a session does about each while the person is away (queue it and carry on), and check a stop or a question against that list, refusing a stop that names the person while owed work remains. Grounds: dcn-2611070734169709, enforcement-claims-are-facts and memory-graduates-to-record."
---

With the technical facilitator on auto, sessions still stop and wait, and nothing in abcd tells a session at the moment it stops that it may not. On 2026-11-21 one session planning an intent waited three ways. It held the product thinker's sign-off behind a review it had started itself, and ended three turns on a status line while that review ran. After every line of the draft had been confirmed, it left the sign-off for the person's return, because the intent page reserves the plan act to the person and the ruling of 2026-11-07 (dcn-2611070734169709) lets a reserved act wait. And it did not act on the coordinating session's relay that the person was away, since a relay is not the person's word. The person had to come back and say it again, the third time after 17 October and 7 November. The setting is one field in .abcd/config.json. No hook reads it when a session stops or asks, the rules loader sends nothing about it, abcd mode prints managed whatever it is set to, and what is reserved to the person is prose spread over the command pages. The rule lives in one decision and in each session's local notes, which is where it has failed three times.
