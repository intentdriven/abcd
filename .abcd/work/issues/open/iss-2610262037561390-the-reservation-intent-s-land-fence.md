---
schema_version: 1
id: "iss-2610262037561390"
slug: "the-reservation-intent-s-land-fence"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 760 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:434"
remedy: "none (filed automatically)"
related_intents: [itd-2609091034175565]
finding_ref: cfk-baf1496f0f8dc244
---

The reservation intent's land fence renews for the merge queue's timeout plus a margin where a mirror exists; the implement chapter renews it for a day, with a mirror or without one.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:434` — "when it holds it is renewed for a day, with a ruleset mirror at the lane's base or without one"
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:92` — "(45 here) plus a margin, or a day where no mirror exists (decision 40)"

The land fence line in itd-2609091034175565's In Scope renews the reservation with a lease of at least check_response_timeout_minutes (45 here) plus a margin, and for a day only where no mirror exists. 27-implement.md renews it for a day whether or not a ruleset mirror sits at the lane's base. The intent's own decision 40 agrees with the brief and says this line was revised to match, but the line still reads otherwise, so a reader of the scope gets a hold of 45 minutes plus a margin that the delivered fence does not use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 760.
