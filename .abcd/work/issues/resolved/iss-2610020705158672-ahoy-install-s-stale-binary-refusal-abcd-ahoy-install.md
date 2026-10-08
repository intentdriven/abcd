---
schema_version: 1
id: "iss-2610020705158672"
slug: "ahoy-install-s-stale-binary-refusal-abcd-ahoy-install"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "triage of a peer report: ahoy install --adopt on a private consumer repo, reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
remedy: "exit non-zero (2, like every other refusal) when the stale-binary guard refuses, and add a test pinning the exit code"
resolution: "ahoy install exits 2 on its stale-binary refusal, after rendering the result; TestAhoyInstallRefusalExitsTwo pins the code (with iss-2610031915386832's fix)"
impact: fix
resolved_by:
  commit: "13fc2644bebf57c095d3beeb6f23715bb101d117"
---

ahoy install's stale-binary refusal ('abcd ahoy install — refused') exits 0, so a script or agent piping the install sees success while nothing was installed

## Grounds

- pursued: a refused install, the stale-binary case included, exits 2 with its reason on stdout; an install refusal exiting 0 would show it wrong
