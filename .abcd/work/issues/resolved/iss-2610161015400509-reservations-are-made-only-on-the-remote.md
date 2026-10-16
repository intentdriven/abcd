---
schema_version: 1
id: "iss-2610161015400509"
slug: "reservations-are-made-only-on-the-remote"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/claim.go"
remedy: "Refuse when origin is a fork or not the repository the pull requests target, naming the remote to set"
related_intents: [itd-2610161016394967]
resolution: "a fresh reservation asks the forge once per process whether a github.com origin is a fork, and refuses a fork naming its parent and the set-url command; renewals and releases never ask, and an unanswered forge warns and lets the act through"
impact: fix
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

Reservations are made only on the remote named origin, and nothing checks that origin is the repository pull requests target, so an account whose origin is a fork reserves on the fork, invisibly to every other account, and works issues others hold.
