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
remedy: "Make abcd workable from a cloud container: run as a non-root user with gitleaks installed (document the container setup), or make the tests that rely on permission refusals and on gitleaks say plainly that they need a non-root user or gitleaks instead of failing; prove with one container session whose preflight passes and mints a receipt"
related_intents: [itd-2610120738075547]
---

Cloud agent sessions started on 2026-10-12 to work drain-eligible issues all failed abcd's preflight and had git identity problems, so none pushed a branch (the person reported this; the exact errors are with the launching session abcd-0b). abcd's gates assume the local development machine: every commit must name a human author (the attribution gate refuses a machine identity or none), and preflight resolves the go.mod-pinned toolchain and other local prerequisites. A cloud session gets neither by default, so abcd cannot yet be worked from a cloud session.

## What is known (2026-10-12, from the launching session)

- The sessions' branches were fix/interview-tests-load-budget-cloud, fix/question-check-says-when-it-skips-cloud and fix/fidelity-gate-skips-the-index-cloud; none reached the forge.
- Each brief set the git identity used on main's recent commits (a human name with a users.noreply.github.com privacy mailbox), which the attribution gate accepts, before the first commit; each was told to stop and report if preflight failed on something it did not touch.
- Likely causes, unconfirmed until the transcripts' errors are copied in: a commit made before the identity step under the environment's default identity (a vendor name or a bare noreply@ mailbox, which the gate refuses), or the environment overriding git config; and a preflight gate needing local state or network the cloud environment lacks.

## One session's report (pasted by the person, 2026-10-12)

The session for fix/question-check-says-when-it-skips-cloud made its fix and record commits and stopped before pushing, as briefed: `make preflight` failed 11 tests across the scanner, ahoy, history, launch and surface/cli packages, none touching its change. The container runs everything as root and has no gitleaks installed, which plausibly explains the ahoy, history and interview failures: tests that prove abcd refuses a path other accounts can write cannot hold when root bypasses permissions, and gitleaks-backed checks have no binary. It confirmed the interview failure also occurs on a clean copy of origin/main there. The commit author name came out as REPPL (the repository's pre-commit identity) with the briefed mailbox, which the attribution gate accepts, so identity was not the blocker. Without a preflight receipt the pre-push hook refuses the push, and the session rightly would not use --no-verify, so nothing reached the forge.

So the finding is narrower than first recorded: abcd's preflight cannot pass in a container that runs as root or lacks gitleaks. A cloud session needs a non-root user and gitleaks (or preflight needs to say plainly which gates need them) before abcd can be worked from one.
