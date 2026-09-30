---
schema_version: 1
id: "iss-2608210738367948"
slug: "ahoy-setup-routine-git-identity"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "itd-130 session; bughunt routine attribution"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed B): Does itd-131's widened identity pin (M2) cover the routine environment's human git identity, or does it wait on the routines ruling (M6)?"
remedy: "Waits on ruling B (does itd-131's pin cover the routine environment) and the routines ruling M6: detection already ships with itd-131, so only establishment remains; if the pin covers it, have ahoy install print the GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL values from .abcd/config/identity.json for a routine environment, proven by a commit made under them passing scripts/check-attribution.sh; if it waits on M6, fold the establishment into iss-2608210932052003 and resolve this record as its duplicate."
---

New-repo / ahoy setup does not establish the git identity autonomous routines commit under. ahoy DETECTS a mismatch between the working git user.name/user.email and the committed identity pin (.abcd/config/identity.json, iss-62) but never SETS the identity, and nothing configures the human identity for an autonomous cloud routine's environment — so a routine defaults to Claude <noreply@anthropic.com> and every commit it makes fails the attribution gate (see attribution-gate-has-no-local-mirror). The ahoy/onboarding gap for a repo that will run routines: scaffold or verify the routine-environment git identity (git config user.name/email, or GIT_AUTHOR_*/GIT_COMMITTER_* env) so routine commits are authored by the human of record from the start. Bridges iss-62 (identity pin), itd-91 (declared AI-attribution preference), and the cloud-routine-git-identity memory. Candidate ahoy-install sub-step or a new-repo setup check.

## Remedy grounds (2026-09-29)

- itd-131 shipped the detect half (internal/core/ahoy/detect.go names a routine's tool identity and hands establishment to the runner record, iss-2608210932052003), so the remedy is narrowed to the half that is still missing; no outside practice was needed.
- Rejected: writing the identity into the routine prompt, which itd-131 records as fragile.
