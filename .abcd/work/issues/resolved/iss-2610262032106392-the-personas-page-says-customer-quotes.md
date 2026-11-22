---
schema_version: 1
id: "iss-2610262032106392"
slug: "the-personas-page-says-customer-quotes"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 232 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:3"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): May a press-release quote be attributed to one of abcd's own roles ('a product thinker', 'a technical facilitator') instead of a registry persona? Answers: (a) yes, amend 05-personas.md and itd-79; (b) no, rewrite every such quote to the registered name. Recommended (a): about 47 intents already do this and this one is shipped, so rewriting it is not available."
related_intents: [itd-2609211116005482]
finding_ref: cfk-e400a736dc47dfda
resolution: "05-personas.md now allows a press-release quote attributed to a role alone, with no name, as itd-79 already does, citing dcn-2610271112330536; any name used must still be registered and role-matched."
impact: internal
resolved_by:
  commit: "fc6fb8edda5cf6c8d718bcab9afb5352145aa111"
---

The personas page says customer quotes use named placeholder personas from the registry; itd-2609211116005482 attributes its quote to an unnamed product thinker.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:3` — "Customer quotes use placeholder personas from `.abcd/development/personas.json`"
- End B: `.abcd/development/intents/shipped/itd-2609211116005482-abcd-build-next-picks-the-readiest.md:22` — "said a product thinker who had planned forty-eight intents and could build one a day"

The brief says quotes are voiced by registry personas, chosen by role, and rules out generic unnamed phrasing. itd-2609211116005482's quote is attributed to 'a product thinker' with no registered name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 232.

## Grounds

- pursued: the personas page and itd-79 agree that a nameless-role quote is allowed, so itd-2609211116005482's 'a product thinker' quote no longer contradicts the brief; a brief page or itd-79 still requiring a registered name for every quote would show it wrong, and TestPersonasPageAllowsAQuoteAttributedToARoleAlone with TestAQuoteMayBeAttributedToARoleAlone check both.
