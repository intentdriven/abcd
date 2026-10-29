---
schema_version: 1
id: "iss-2610262043159613"
slug: "itd-79-requires-every-press-release"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1125 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-79-persona-registry.md:18"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Decision: may a press-release quote be attributed to a role alone (a nameless role such as 'an AI/agent researcher'), or must it carry the role's registered persona name? Answers: (A) the rule binds as written, so about a dozen role-only quotes in shipped, planned and draft intents get a registered name and the lint is extended to flag them; (B) amend itd-79 to allow role-only attribution, keeping the rule that any name used must be registered and role-matched. Recommend B: a nameless role invents no name, so it does not offend the rule's stated reason, the persona_registry lint already leaves it out of reach, and editing a dozen shipped records buys nothing."
related_intents: [itd-79, itd-198]
finding_ref: cfk-d79fd24f3489e09c
---

itd-79 requires every press-release quote's persona to be the role's registered name; itd-198 attributes its quote to an unnamed role.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-79-persona-registry.md:18` — "the scenario's audience determines the role, and the role's registered name is used"
- End B: `.abcd/development/intents/shipped/itd-198-an-assembly-reports-what-it-would-cost.md:21` — "said an AI/agent researcher who runs cold readings"

itd-79 states that every persona in a press-release quote comes from the registry and that 'the role's registered name is used'. itd-198's press release attributes its quote to 'an AI/agent researcher' with no name, as several later shipped intents also do. Either the rule still binds and these quotes break it, or the convention has moved to role-only attribution and the discipline no longer says so.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1125.
