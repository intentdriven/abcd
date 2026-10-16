---
schema_version: 1
id: "iss-2610160608426210"
slug: "the-hook-wiring-at-abcd-development"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add the gh attestation verify provenance check to the hook-wiring paragraph at 03-configuration.md:1171-1175."
resolution: "the configuration brief's hook-wiring paragraph now says bootstrap.sh holds the download to the build-provenance policy (release workflow on main, GitHub-hosted runner, the tag's commit), refusing on a failed check and warning when gh cannot verify"
impact: fix
resolved_by:
  commit: "8ba003074e3c2d58d41ed08197fce207f89670eb"
---

The hook wiring at .abcd/development/brief/05-internals/03-configuration.md:1171-1175 says hooks/bootstrap.sh downloads the pinned asset verified against that release's own checksums. hooks/bootstrap.sh also holds the download to the build-provenance policy through gh attestation verify, so the checksums are not the whole check.
