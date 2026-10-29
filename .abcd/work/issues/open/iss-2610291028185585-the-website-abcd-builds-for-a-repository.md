---
schema_version: 1
id: "iss-2610291028185585"
slug: "the-website-abcd-builds-for-a-repository"
severity: "major"
category: "process"
source: "user-observation"
found_during: "product thinker ruling relayed by the coordinator session, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/contributors.go"
remedy: "Make the contributors page and its JSON name no person: the humans row becomes a count of human authors and their commits, the author name and profile fields never reach any rendered or serialised output, and the tool and bot rows and the Assisted-by model tally stay as they are; supersede the naming part of adr-47 with a new ADR."
---

The website abcd builds for a repository names every human author on its contributors page, with a forge profile link built from their noreply address, which carries their login. The product thinker ruled on 2026-10-29 that no member's login or name appears in what abcd publishes: credits name no person. The page reads git shortlog and prints each author's name and commit count, and the record JSON beside it carries the same fields, so every site abcd builds publishes its contributors' names.
