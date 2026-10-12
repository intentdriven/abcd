---
schema_version: 1
id: "iss-2610120729094755"
slug: "no-cloud-environment-setup-for-abcd"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "release preparation with cloud sessions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "cloud environment setup"
remedy: "Write the cloud environment's setup (a non-root user, gitleaks, the go.mod toolchain, the human author identity, the committed hooks) as a script or a documented setup the cloud session runs first, plus an abcd check that reports what is missing before any work starts"
refines: [iss-2610120724251698]
---

Nothing prepares a cloud container to work on abcd the way the development machine is prepared: on 2026-10-12 the container ran everything as root, had no gitleaks, and needed the brief to set the git identity, so preflight failed (iss-2610120724251698). abcd documents its local prerequisites but has no setup step or check for a cloud environment.
