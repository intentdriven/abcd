---
schema_version: 1
id: "iss-2610161632245527"
slug: "abcd-should-find-out-what-the-forge-can"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcd land lab test, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd's forge-facing verbs (land, launch, ahoy remote)"
remedy: "Before abcd picks a route that depends on what the forge allows (a private repository, a private fork into an organisation, a merge queue on the account's plan), probe it with read-only forge queries and take the route that delivers what the person asked for; where no route exists, ask the person with options instead of failing or guessing. Run a state-of-the-art pass on forge capability detection first, and promote this to an intent."
related_intents: [itd-2610161647248332]
---

abcd should find out what the forge can do instead of asking the person or assuming. On 2026-10-16, setting up a throwaway lab to test abcd land, the agent asked the product thinker whether the team organisation allowed private copies; the product thinker did not know and should not have to. A read-only query answered it (members_can_fork_private_repositories is false for the organisation's own repositories), and a real attempt showed a private fork of a repository the person owns works anyway. The general want: abcd tests the forge's functionality itself (can repositories be private, can they be forked privately, is a merge queue available) and goes with the route that delivers the wanted outcome; only when no route exists does it ask, offering the options. Likely an intent; it needs a state-of-the-art pass on forge capability detection.
