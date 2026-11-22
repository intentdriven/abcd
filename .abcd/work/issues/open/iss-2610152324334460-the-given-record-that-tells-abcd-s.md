---
schema_version: 1
id: "iss-2610152324334460"
slug: "the-given-record-that-tells-abcd-s"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "fix round 4 of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Drop given records by liveness as well as age (never drop a record whose session has a live entry), or raise the bound with a measured cost; test a resume after eviction."
decision_needed: "The given record decides whether a session's name is trusted as the person's own or held as abcd's, and the remedy leaves the retention rule open, so: should given records be kept by liveness (never dropped while their session has a live entry) or should the 512 bound simply be raised, and what counts as a live entry for deciding whose name is trusted on resume or fork?"
---

The given record that tells abcd's session names from the person's keeps 512 entries; a session whose record was dropped and is then resumed or forked is read as carrying the person's own name, so it is not held, and after a restart its name can be handed to a second running session.
