---
schema_version: 1
id: "iss-2611211733065807"
slug: "person-says-away-and-nothing-sets-the"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "lab-261121170045-71e07e7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add one bundled rules domain that recalls on the words a person uses to say they are leaving (away, back at a time, overnight, do not wait, carry on without me) and tells the session, in one rule, to set the away mode on the person's word with the end time they gave and to say its terms back. Where the person names no end, the rule has the session set the longest window the mode allows and say so: the mode adds checks and grants nothing, and it ends on the person's word. Grounds: itd-2610180912189125, whose promise has the person ask a session to, and the paired probes of lab-261121170045-71e07e7."
---

When the person says they are away, nothing sets the away mode and no rule tells a session to. A prompt in which the person says they are away, that the session should not wait and that the technical facilitator is on auto injects nothing about the away mode: the one domain that recalls on a word of that prompt (the asking rules, on the word facilitator) says nothing of the mode, and the instruction to set the mode on the person's word is on the mode page alone, which a session reads only when it already means to run that command. On 2026-11-21 the person said they were away to three sessions and none set it at the time; one did at 17:27 UTC, after a lab had shown the gap. The shipped promise (itd-2610180912189125) has the person run one command or ask a session to; the second half does not hold. Measured in lab-261121170045-71e07e7 at commit 71e07e7ea, probes router-away-words-fresh and router-away-words-present.
