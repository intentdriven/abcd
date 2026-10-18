---
schema_version: 1
id: "iss-2610170619247185"
slug: "an-autonomous-session-sat-idle-overnight"
severity: "critical"
category: "lapse"
source: "user-observation"
found_during: "autonomous overnight run 2026-10-16"
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2610170638184486, itd-2610180912189125]
found_at: "the agent loop between turns in an autonomous run"
lapsed_at: "2026-10-16T19:48:14Z"
remedy: "Make an autonomous run abcd state rather than chat: an 'abcd mode autonomous --until <time>' state per checkout that the prompt and session-start hooks inject (so compaction and peer relays cannot drop it); a Stop hook that, while that state holds, refuses to end a turn that has owed work (its reservations, its behind or failing pull requests, its unpushed branches) and names the next item, bounded by the host's stop_hook_active flag; the question check rerouting a product question in that state to 'abcd decide' with only website-only steps queued for the morning; and idle time per session in 'abcd peers' so a coordinator sees a stall."
---

An agent session told to work autonomously overnight sat idle for about ten of the eleven and a half hours between 20:48 BST on 2026-10-16 and 07:13 BST on 2026-10-17, while it held owed work it could have done alone.

What happened. At 18:58 BST a peer session relayed the product thinker's word that they had left for the day and that every session should work on autonomously; a standing instruction from 2026-10-15 already ran the autonomous run through Saturday 2026-10-17, with auto-merge allowed. The product thinker then came back to the terminal for about an hour. At 20:48 BST the session ended its turn on a question to them (whether to take over an unowned pull request) and listed three more items as waiting on their answer. At 21:42 BST its context was compacted; the summary carried the waiting items forward as 'awaiting the product thinker' and did not carry the autonomous instruction at all. A Claude Code session runs only when something sends it input, so nothing ran until a peer's message at 02:59 BST. The session then chose to wait on that peer's pull request instead of picking up other work, ran a two-hour watcher, reported, and stopped again until the product thinker asked for a status at 07:13 BST. Idle spans: 20:48 to 02:59, 03:00 to 05:00 (a watcher only), 05:01 to 07:13.

Work that was owed and needed no one: merging main into two of its pull requests that had fallen behind; folding finished interview answers into two drafts; a review pair, preflight and push for the reservation sweep and for two dependency bumps; taking over the unowned pull request; a design re-review pair. Each could have been decided and recorded under the standing instruction.

Why. (1) The session treated its own decisions as the product thinker's, against a standing instruction it had in memory. (2) The autonomous state lived only in chat: relayed by a peer, then overwritten in the session's reading by the product thinker's return, then dropped by the compaction summary. (3) Nothing in abcd notices a session that stops with owed work during an autonomous run: abcd mode knows managed, facilitator and product-thinker but no autonomous state, the plugin has no Stop hook, and abcd peers shows no idle time, so the coordinating session could not see the stall. (4) The evening plan's rule to send a morning list of items needing the product thinker made queueing a decision feel like doing it.

Second instance, same night. The coordinating session held three drain steps, built and reviewed at about 03:00 BST, behind the merge of the pull request they were stacked on, although their merge-from-main, fixes and preflight could have run in parallel the whole time; only the order in which they land was fixed. The product thinker caught it in the morning. So the lapse is wider than idling on a question: it is also serialising work on a merge order when only the landing order is constrained. The remedy's Stop hook covers it if a stacked branch whose base is still open counts as owed work: rebase on the base's tip, fix and preflight now, and land after the base.
