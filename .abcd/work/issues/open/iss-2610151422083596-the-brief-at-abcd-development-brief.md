---
schema_version: 1
id: "iss-2610151422083596"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/21-update.md:143 to match the shipped surface: The receipt also carries `attested_by`, the verified provenance signer (internal/core/update/update.go:132; commands/update.md tells the host to report it); the chapter's receipt field list omits it."
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:143 says: On a run that reached the release origin, the receipt carries beside action the origin, the tag, the asset and its digest, the target path, and the ownership proof; env_ignored when overrides were scrubbed. The shipped surface differs: The receipt also carries `attested_by`, the verified provenance signer (internal/core/update/update.go:132; commands/update.md tells the host to report it); the chapter's receipt field list omits it. (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a16.)
