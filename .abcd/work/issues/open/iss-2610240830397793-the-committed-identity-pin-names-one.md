---
schema_version: 1
id: "iss-2610240830397793"
slug: "the-committed-identity-pin-names-one"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "ahoy install review, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/identity.json"
remedy: "move the identity pin out of the repository to each person's own machine (under ~/.abcd.noindex), so the committed hook and setup check a contributor against their own pin and never print or propose another person's identity; keep refusing machine identities in every repository"
---

the committed identity pin names one person, so it misleads every other contributor: .abcd/config/identity.json pins REPPL's identity, ahoy install reports any other contributor's identity as a required git_identity.mismatch gap and proposes they commit as the pinned person, and the committed .githooks/pre-commit refuses their commits with 'fix: git config user.name "REPPL" && git config user.email ...', telling them to commit under the maintainer's name. That contradicts the attribution rule that every commit is honestly authored by the human who made it, in a repository that takes outside contributions.
