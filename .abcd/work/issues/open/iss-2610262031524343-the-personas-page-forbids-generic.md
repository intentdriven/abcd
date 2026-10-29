---
schema_version: 1
id: "iss-2610262031524343"
slug: "the-personas-page-forbids-generic"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 188 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:5"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): May a press-release quote be attributed to one of abcd's own roles ('a product thinker', 'a technical facilitator') instead of a registry persona? Answers: (a) yes, amend 05-personas.md and itd-79; (b) no, rewrite every such quote to the registered name. Recommended (a): 'a technical facilitator' is one of the framework's two named roles (adr-2609151528057260) and about 47 intents already use role attributions the lint cannot see."
related_intents: [itd-2609201916151817]
finding_ref: cfk-896bc07753e23357
---

The personas page forbids generic, unnamed quote attributions, but itd-2609201916151817's quote is credited to 'a technical facilitator'.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:5` — "never use generic "a hypothetical user" phrasing (loses voice)"
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:23` — "said a technical facilitator who ran one intent through a coordinator"

05-personas codifies that press-release quotes use named placeholders from the registry, picked by role, and never generic phrasing, because that loses voice. itd-2609201916151817's press-release quote is credited to 'a technical facilitator' with no registered name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 188.
