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
resolution: "The contributors page and record.json publish the human authors as a count of people and of their commits; no author name, login or profile link reaches a page or the JSON, the health page's same-author panel is withdrawn, and adr-2610291037481588 records it, partly superseding adr-47."
impact: breaking
resolved_by:
  commit: "e91e1e820"
---

The website abcd builds for a repository names every human author on its contributors page, with a forge profile link built from their noreply address, which carries their login. The product thinker ruled on 2026-10-29 that no member's login or name appears in what abcd publishes: credits name no person. The page reads git shortlog and prints each author's name and commit count, and the record JSON beside it carries the same fields, so every site abcd builds publishes its contributors' names.

## Grounds

- pursued: removing the per-person list and profile fields from the published record.json is breaking for any reader of authorship.humans; a site built from a history with several identities per person shows one count per person and no name anywhere, and a name found on any built page or in record.json would show it wrong
