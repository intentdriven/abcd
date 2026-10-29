---
schema_version: 1
id: "iss-2610262036000485"
slug: "a-draft-filed-for-later-wants-every"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 616 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md:18"
remedy: "Supersede the draft by itd-2610091010348892 (planned: an opted-in project's record built at each release on its own Pages) after carrying its two open questions (the sample project shared with itd-2610040754440360; default publish versus private) into that intent; `abcd site setup` (itd-2609061543533170) already ships the pages."
related_intents: [itd-2610040754453237]
finding_ref: cfk-9f2e5a740002e597
---

A draft filed for later wants every managed project to publish the public record site; 22-site already sets up and publishes that site for any managed repository.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2610040754453237-any-team-that-manages-its-project-with.md:18` — "Any team that manages its project with abcd publishes the public record site for their own project"
- End B: `.abcd/development/brief/04-surfaces/22-site.md:48` — "Setting up takes a repository abcd manages from the checkout to a live address"

itd-2610040754453237 says any team that manages its project with abcd publishes the public record site for its own project as abcd publishes its own, and records it as a separate intent for later, not now. 22-site says setting up takes a repository abcd manages from the checkout to a live address, and that every repository gets the same record pages, the timeline among them. The same ground is claimed by a shipped surface and by an unplanned draft, so it is either delivered or still owed, and the record says both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 616.
