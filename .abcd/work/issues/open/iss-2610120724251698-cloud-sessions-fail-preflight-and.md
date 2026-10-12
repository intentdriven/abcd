---
schema_version: 1
id: "iss-2610120724251698"
slug: "cloud-sessions-fail-preflight-and"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "security fix run and release preparation, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Collect the cloud sessions' exact preflight and identity errors, then document (or script) the setup a cloud session needs before it commits: the human author identity it must use, the pinned toolchain, and which preflight gates need network or local state; prove with one cloud session that commits and passes preflight"
---

Cloud agent sessions started on 2026-10-12 to work drain-eligible issues all failed abcd's preflight and had git identity problems, so none pushed a branch (the person reported this; the exact errors are with the launching session abcd-0b). abcd's gates assume the local development machine: every commit must name a human author (the attribution gate refuses a machine identity or none), and preflight resolves the go.mod-pinned toolchain and other local prerequisites. A cloud session gets neither by default, so abcd cannot yet be worked from a cloud session.

## What is known (2026-10-12, from the launching session)

- The sessions' branches were fix/interview-tests-load-budget-cloud, fix/question-check-says-when-it-skips-cloud and fix/fidelity-gate-skips-the-index-cloud; none reached the forge.
- Each brief set the git identity used on main's recent commits (a human name with a users.noreply.github.com privacy mailbox), which the attribution gate accepts, before the first commit; each was told to stop and report if preflight failed on something it did not touch.
- Likely causes, unconfirmed until the transcripts' errors are copied in: a commit made before the identity step under the environment's default identity (a vendor name or a bare noreply@ mailbox, which the gate refuses), or the environment overriding git config; and a preflight gate needing local state or network the cloud environment lacks.
