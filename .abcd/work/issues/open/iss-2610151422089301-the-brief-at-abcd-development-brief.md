---
schema_version: 1
id: "iss-2610151422089301"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/21-update.md:77 to match the shipped surface: At least two more refusals arrive after dispatch: `unverified-provenance` (after the checksums match) and `older-than-mark` (after resolving the latest tag); 'one further refusal' understates the shipped refusal set (commands/update.md lists both)."
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:77 says: One further refusal arrives after the dispatch, once the release checksums are in hand: a regular file that none of the three proofs vouches for (the only post-dispatch refusal). The shipped surface differs: At least two more refusals arrive after dispatch: `unverified-provenance` (after the checksums match) and `older-than-mark` (after resolving the latest tag); 'one further refusal' understates the shipped refusal set (commands/update.md lists both). (v0.14.2 release crosscheck, false-claim, raised by opus in a16.)
