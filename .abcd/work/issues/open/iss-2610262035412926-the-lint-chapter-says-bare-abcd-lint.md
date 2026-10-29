---
schema_version: 1
id: "iss-2610262035412926"
slug: "the-lint-chapter-says-bare-abcd-lint"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 590 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:144"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does the agents-and-hosts help block hold verbs no person types, or verbs no person needs to type? Answer 1: no person needs to type them; reword README.md:163 and keep lint, guard, banlist, identity, docs and site in the block. Answer 2: literally no person types them, so move the ones a person does type to the people groups. Recommend answer 1: the chapters rightly say a person can run lint or guard, and the product thinker's 2026-10-09 placement was by what a person needs to do their job."
finding_ref: cfk-302bf7f4a3928fa3
---

The lint chapter says bare `abcd lint` runs when a human types it; the surfaces register lists `lint` with the agents, a block it reserves for commands no person types.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:144` — "The v1 rule set backs onboarding and runs when a human types it"
- End B: `.abcd/development/brief/04-surfaces/README.md:163` — "only a command no person types is listed with the agents"

16-lint says the v1 rule set backs onboarding and runs when a human types it. The surfaces README says every verb was placed by who types it, and only a command no person types is listed with the agents. Its help table puts `lint` in the agents-and-hosts block. A verb a human types breaks the README's placement rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 590.
