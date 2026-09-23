---
schema_version: 1
id: "iss-2608210738367948"
slug: "ahoy-setup-routine-git-identity"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "itd-130 session; bughunt routine attribution"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M2: reconciled when spc-34 is amended to pin the identity at setup (itd-131 widened by iss-193))."
---

New-repo / ahoy setup does not establish the git identity autonomous routines commit under. ahoy DETECTS a mismatch between the working git user.name/user.email and the committed identity pin (.abcd/config/identity.json, iss-62) but never SETS the identity, and nothing configures the human identity for an autonomous cloud routine's environment — so a routine defaults to Claude <noreply@anthropic.com> and every commit it makes fails the attribution gate (see attribution-gate-has-no-local-mirror). The ahoy/onboarding gap for a repo that will run routines: scaffold or verify the routine-environment git identity (git config user.name/email, or GIT_AUTHOR_*/GIT_COMMITTER_* env) so routine commits are authored by the human of record from the start. Bridges iss-62 (identity pin), itd-91 (declared AI-attribution preference), and the cloud-routine-git-identity memory. Candidate ahoy-install sub-step or a new-repo setup check.